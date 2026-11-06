package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	log "github.com/sirupsen/logrus"
)

const (
	MemfabToolSocketPath = "/run/memfab/tools.sock"

	memfabContractVersion        = 1
	memfabMaxTools               = 16
	memfabMaxToolNameBytes       = 64
	memfabMaxToolDescriptionSize = 4 << 10
	memfabMaxSchemaBytes         = 64 << 10
	memfabMaxAllSchemaBytes      = 256 << 10
	memfabMaxRequestBytes        = 256 << 10
	memfabMaxResponseBytes       = 4 << 20
	memfabDescriptorTimeout      = 2 * time.Second
	memfabToolCallTimeout        = 10 * time.Second
)

var (
	memfabToolSocketPath = MemfabToolSocketPath
	memfabToolName       = regexp.MustCompile("^[A-Za-z0-9_.-]+$")
)

type memfabToolDescriptor struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

type memfabToolsResponse struct {
	Version int                    `json:"version"`
	Tools   []memfabToolDescriptor `json:"tools"`
}

type memfabToolCallRequest struct {
	Version   int             `json:"version"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type memfabToolCallSuccessResponse struct {
	Version int             `json:"version"`
	Result  json.RawMessage `json:"result"`
}

type memfabToolCallErrorResponse struct {
	Version int                      `json:"version"`
	Error   memfabBackendErrorObject `json:"error"`
}

type memfabBackendErrorObject struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type memfabBridge struct {
	client *http.Client
}

func newMemfabBridge(socketPath string) *memfabBridge {
	transport := &http.Transport{
		DisableCompression: true,
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, "unix", socketPath)
		},
	}
	return &memfabBridge{client: &http.Client{Transport: transport}}
}

func (b *memfabBridge) listTools(parent context.Context) ([]memfabToolDescriptor, error) {
	ctx, cancel := context.WithTimeout(parent, memfabDescriptorTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://memfab/v1/tools", nil)
	if err != nil {
		return nil, errors.New("create memory tool request")
	}
	resp, err := b.client.Do(req)
	if err != nil {
		return nil, errors.New("memory tool backend unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("memory tool descriptor request failed")
	}
	var payload memfabToolsResponse
	if err := decodeMemfabResponse(resp, &payload); err != nil {
		return nil, err
	}
	if payload.Version != memfabContractVersion || payload.Tools == nil {
		return nil, errors.New("invalid memory tool descriptor contract")
	}
	return payload.Tools, nil
}

func (b *memfabBridge) callTool(parent context.Context, name string, arguments json.RawMessage) (json.RawMessage, error) {
	if !jsonObject(arguments) {
		return nil, errors.New("memory tool arguments must be an object")
	}
	payload, err := json.Marshal(memfabToolCallRequest{Version: memfabContractVersion, Name: name, Arguments: arguments})
	if err != nil || len(payload) > memfabMaxRequestBytes {
		return nil, errors.New("memory tool request is too large")
	}
	ctx, cancel := context.WithTimeout(parent, memfabToolCallTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://memfab/v1/tools/call", bytes.NewReader(payload))
	if err != nil {
		return nil, errors.New("create memory tool call")
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := b.client.Do(req)
	if err != nil {
		return nil, errors.New("memory tool backend unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		var response memfabToolCallSuccessResponse
		if err := decodeMemfabResponse(resp, &response); err != nil {
			return nil, err
		}
		if response.Version != memfabContractVersion || len(response.Result) == 0 || !json.Valid(response.Result) {
			return nil, errors.New("invalid memory tool result")
		}
		return append(json.RawMessage(nil), response.Result...), nil
	}

	var response memfabToolCallErrorResponse
	if err := decodeMemfabResponse(resp, &response); err != nil {
		return nil, err
	}
	if response.Version != memfabContractVersion || response.Error.Code == "" || response.Error.Message == "" {
		return nil, errors.New("invalid memory tool error contract")
	}
	switch resp.StatusCode {
	case http.StatusBadRequest:
		return nil, errors.New("invalid memory tool request")
	case http.StatusNotFound:
		return nil, errors.New("unknown memory tool")
	case http.StatusInternalServerError:
		return nil, errors.New("memory tool failed")
	default:
		return nil, errors.New("invalid memory tool backend status")
	}
}

func decodeMemfabResponse(resp *http.Response, target any) error {
	mediaType, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return errors.New("invalid memory tool response content type")
	}
	if resp.ContentLength > memfabMaxResponseBytes {
		return errors.New("memory tool response is too large")
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, memfabMaxResponseBytes+1))
	if err != nil {
		return errors.New("read memory tool response")
	}
	if len(data) == 0 || len(data) > memfabMaxResponseBytes {
		return errors.New("memory tool response is too large")
	}
	if err := rejectMemfabDuplicateJSONKeys(data); err != nil {
		return errors.New("invalid memory tool response")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return errors.New("invalid memory tool response")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("invalid memory tool response")
	}
	return nil
}

func jsonObject(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var object map[string]json.RawMessage
	return rejectMemfabDuplicateJSONKeys(raw) == nil && json.Unmarshal(raw, &object) == nil && object != nil
}

func rejectMemfabDuplicateJSONKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := consumeMemfabJSONValue(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return errors.New("trailing JSON data")
	}
	return nil
}

func consumeMemfabJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("invalid JSON object key")
			}
			if _, duplicate := seen[key]; duplicate {
				return errors.New("duplicate JSON object key")
			}
			seen[key] = struct{}{}
			if err := consumeMemfabJSONValue(decoder); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	case '[':
		for decoder.More() {
			if err := consumeMemfabJSONValue(decoder); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	default:
		return errors.New("invalid JSON delimiter")
	}
}

type mcpController struct {
	mu               sync.Mutex
	server           *mcp.Server
	handler          http.Handler
	bridge           *memfabBridge
	memoryEnabled    func() (bool, error)
	controlToolNames map[string]struct{}
	memoryTools      map[string]memfabToolDescriptor
}

func newMCPController(server *mcp.Server, handler http.Handler, bridge *memfabBridge, memoryEnabled func() (bool, error), controlNames map[string]struct{}) *mcpController {
	return &mcpController{server: server, handler: handler, bridge: bridge, memoryEnabled: memoryEnabled, controlToolNames: controlNames, memoryTools: make(map[string]memfabToolDescriptor)}
}

func (c *mcpController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := c.reconcile(r.Context()); err != nil {
		// Do not include backend errors: they may contain socket or process detail.
		log.Debug("Memory Fabric tools unavailable on authenticated MCP endpoint")
	}
	c.handler.ServeHTTP(w, r)
}

func (c *mcpController) reconcile(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	enabled, err := c.memoryEnabled()
	if err != nil || !enabled {
		c.removeMemoryToolsLocked()
		if err != nil {
			return errors.New("read Memory Fabric state")
		}
		return nil
	}
	descriptors, err := c.bridge.listTools(ctx)
	if err != nil {
		c.removeMemoryToolsLocked()
		return err
	}
	next, err := validateMemfabToolDescriptors(descriptors, c.controlToolNames)
	if err != nil {
		c.removeMemoryToolsLocked()
		return err
	}
	for name, descriptor := range next {
		if current, ok := c.memoryTools[name]; ok && equalMemfabDescriptor(current, descriptor) {
			continue
		}
		descriptor := descriptor
		c.server.AddTool(&mcp.Tool{Name: descriptor.Name, Description: descriptor.Description, InputSchema: append(json.RawMessage(nil), descriptor.InputSchema...)}, c.memoryToolHandler(descriptor.Name))
	}
	remove := make([]string, 0)
	for name := range c.memoryTools {
		if _, ok := next[name]; !ok {
			remove = append(remove, name)
		}
	}
	if len(remove) > 0 {
		c.server.RemoveTools(remove...)
	}
	c.memoryTools = next
	return nil
}

func (c *mcpController) memoryToolHandler(name string) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		enabled, err := c.memoryEnabled()
		if err != nil || !enabled {
			return memoryToolError("memory tool is disabled"), nil
		}
		if req == nil || req.Params == nil || req.Params.Name != name {
			return memoryToolError("invalid memory tool request"), nil
		}
		arguments := req.Params.Arguments
		if len(arguments) == 0 {
			arguments = json.RawMessage([]byte("{}"))
		}
		result, err := c.bridge.callTool(ctx, name, arguments)
		if err != nil {
			return memoryToolError(err.Error()), nil
		}
		text := string(result)
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}, StructuredContent: map[string]any{"result": text}}, nil
	}
}

func (c *mcpController) removeMemoryToolsLocked() {
	if len(c.memoryTools) == 0 {
		return
	}
	names := make([]string, 0, len(c.memoryTools))
	for name := range c.memoryTools {
		names = append(names, name)
	}
	c.server.RemoveTools(names...)
	c.memoryTools = make(map[string]memfabToolDescriptor)
}

func validateMemfabToolDescriptors(descriptors []memfabToolDescriptor, controlNames map[string]struct{}) (map[string]memfabToolDescriptor, error) {
	if len(descriptors) == 0 || len(descriptors) > memfabMaxTools {
		return nil, errors.New("invalid memory tool count")
	}
	result := make(map[string]memfabToolDescriptor, len(descriptors))
	totalSchemaBytes := 0
	for _, descriptor := range descriptors {
		if len(descriptor.Name) == 0 || len(descriptor.Name) > memfabMaxToolNameBytes || !memfabToolName.MatchString(descriptor.Name) {
			return nil, errors.New("invalid memory tool name")
		}
		if descriptor.Description == "" || len(descriptor.Description) > memfabMaxToolDescriptionSize || strings.IndexByte(descriptor.Description, 0) >= 0 {
			return nil, errors.New("invalid memory tool description")
		}
		if _, collision := controlNames[descriptor.Name]; collision {
			return nil, errors.New("memory tool collides with control tool")
		}
		if _, duplicate := result[descriptor.Name]; duplicate {
			return nil, errors.New("duplicate memory tool name")
		}
		if len(descriptor.InputSchema) == 0 || len(descriptor.InputSchema) > memfabMaxSchemaBytes {
			return nil, errors.New("invalid memory tool schema size")
		}
		totalSchemaBytes += len(descriptor.InputSchema)
		if totalSchemaBytes > memfabMaxAllSchemaBytes {
			return nil, errors.New("memory tool schemas are too large")
		}
		var schema map[string]json.RawMessage
		if err := rejectMemfabDuplicateJSONKeys(descriptor.InputSchema); err != nil {
			return nil, errors.New("invalid memory tool schema")
		}
		if err := json.Unmarshal(descriptor.InputSchema, &schema); err != nil || schema == nil {
			return nil, errors.New("invalid memory tool schema")
		}
		var schemaType string
		if err := json.Unmarshal(schema["type"], &schemaType); err != nil || schemaType != "object" {
			return nil, errors.New("memory tool schema must describe an object")
		}
		descriptor.InputSchema = append(json.RawMessage(nil), descriptor.InputSchema...)
		result[descriptor.Name] = descriptor
	}
	return result, nil
}

func equalMemfabDescriptor(left, right memfabToolDescriptor) bool {
	return left.Name == right.Name && left.Description == right.Description && bytes.Equal(left.InputSchema, right.InputSchema)
}

func memoryToolError(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: message}}, IsError: true}
}

func getMemoryFabricEnabled() (bool, error) {
	entry, err := DefaultServiceStateManager().Get(aiServiceMemoryFabric)
	if err != nil {
		return false, err
	}
	return entry.Enabled, nil
}
