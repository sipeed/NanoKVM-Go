package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"testing"
	"time"
)

type memoryFabricConfigAPIResponse struct {
	Code int `json:"code"`
	Data struct {
		Config    map[string]interface{} `json:"config"`
		Changed   bool                   `json:"changed"`
		Restarted bool                   `json:"restarted"`
	} `json:"data"`
}

func TestGetMemoryFabricConfigReturnsStructuredCLIOutput(t *testing.T) {
	withMemoryFabricConfigTest(t, memoryFabricConfigTestOptions{
		run: func(_ context.Context, args []string, input []byte) ([]byte, error) {
			if !configStringSlicesEqual(args, []string{"get", "--json"}) || input != nil {
				t.Fatalf("unexpected command args=%v input=%q", args, input)
			}
			return []byte(`{"revision":"rev-1","models":{"default":{"model":"gpt-5-mini","baseUrl":"https://api.example.com/v1","apiKeyConfigured":true,"maxTokens":1024,"timeoutSeconds":30}}}`), nil
		},
	})

	service := &Service{}
	c, recorder := newTestContext(http.MethodGet, "/api/ai/memory-fabric/config", "")
	service.GetMemoryFabricConfig(c)
	rsp := decodeMemoryFabricConfigAPIResponse(t, recorder.Body.Bytes())
	if rsp.Code != 0 || rsp.Data.Config["revision"] != "rev-1" {
		t.Fatalf("unexpected response: %#v", rsp)
	}
	models, ok := rsp.Data.Config["models"].(map[string]interface{})
	if !ok || len(rsp.Data.Config) != 2 || len(models) != 1 {
		t.Fatalf("unexpected public config shape: %#v", rsp.Data.Config)
	}
	defaultModel, ok := models["default"].(map[string]interface{})
	if !ok || len(defaultModel) != 5 || defaultModel["model"] != "gpt-5-mini" {
		t.Fatalf("unexpected public model shape: %#v", models)
	}
}

func TestGetMemoryFabricConfigRejectsUnknownCLIConfigFields(t *testing.T) {
	for name, output := range map[string]string{
		"compact model":       `{"revision":"rev-1","models":{"default":{},"compact":{}}}`,
		"other section":       `{"revision":"rev-1","models":{"default":{}},"capture":{"enabled":true}}`,
		"removed api key env": `{"revision":"rev-1","models":{"default":{"apiKeyEnv":"OPENAI_API_KEY"}}}`,
		"unknown model field": `{"revision":"rev-1","models":{"default":{"displayName":"Default"}}}`,
	} {
		t.Run(name, func(t *testing.T) {
			withMemoryFabricConfigTest(t, memoryFabricConfigTestOptions{
				run: func(context.Context, []string, []byte) ([]byte, error) {
					return []byte(output), nil
				},
			})

			service := &Service{}
			c, recorder := newTestContext(http.MethodGet, "/api/ai/memory-fabric/config", "")
			service.GetMemoryFabricConfig(c)
			rsp := decodeMemoryFabricConfigAPIResponse(t, recorder.Body.Bytes())
			if rsp.Code != -1 {
				t.Fatalf("unexpected response: %#v", rsp)
			}
		})
	}
}

func TestGetMemoryFabricConfigUsesRequestContext(t *testing.T) {
	withMemoryFabricConfigTest(t, memoryFabricConfigTestOptions{
		run: func(ctx context.Context, args []string, _ []byte) ([]byte, error) {
			if !configStringSlicesEqual(args, []string{"get", "--json"}) {
				t.Fatalf("unexpected args=%v", args)
			}
			if !errors.Is(ctx.Err(), context.Canceled) {
				t.Fatalf("GET context error=%v, want canceled", ctx.Err())
			}
			return nil, ctx.Err()
		},
	})

	service := &Service{}
	c, recorder := newTestContext(http.MethodGet, "/api/ai/memory-fabric/config", "")
	ctx, cancel := context.WithCancel(c.Request.Context())
	cancel()
	c.Request = c.Request.WithContext(ctx)
	service.GetMemoryFabricConfig(c)
	rsp := decodeMemoryFabricConfigAPIResponse(t, recorder.Body.Bytes())
	if rsp.Code != -1 {
		t.Fatalf("unexpected response: %#v", rsp)
	}
}

func TestGetMemoryFabricConfigRejectsSensitiveCLIOutput(t *testing.T) {
	for name, output := range map[string]string{
		"api key":  `{"revision":"rev-1","models":{"default":{"apiKey":"plain-secret"}}}`,
		"token":    `{"revision":"rev-1","accessToken":"plain-token"}`,
		"password": `{"revision":"rev-1","password":"plain-password"}`,
	} {
		t.Run(name, func(t *testing.T) {
			withMemoryFabricConfigTest(t, memoryFabricConfigTestOptions{
				run: func(context.Context, []string, []byte) ([]byte, error) {
					return []byte(output), nil
				},
			})

			service := &Service{}
			c, recorder := newTestContext(http.MethodGet, "/api/ai/memory-fabric/config", "")
			service.GetMemoryFabricConfig(c)
			rsp := decodeMemoryFabricConfigAPIResponse(t, recorder.Body.Bytes())
			if rsp.Code != -1 {
				t.Fatalf("unexpected response: %#v", rsp)
			}
			if bytes.Contains(recorder.Body.Bytes(), []byte("plain-")) {
				t.Fatalf("sensitive CLI output leaked: %s", recorder.Body.String())
			}
		})
	}
}

func TestSetMemoryFabricConfigStoppedServiceDoesNotRestart(t *testing.T) {
	secretRequest := `{"revision":"rev-1","models":{"default":{"apiKey":"new-secret"}}}`
	var calls []string
	withMemoryFabricConfigTest(t, memoryFabricConfigTestOptions{
		backup:         []byte("old config with secret"),
		backupRevision: "rev-1",
		run: func(_ context.Context, args []string, input []byte) ([]byte, error) {
			calls = append(calls, args[0])
			if !configStringSlicesEqual(args, []string{"update", "--json"}) || string(input) != secretRequest {
				t.Fatalf("unexpected update args=%v input=%q", args, input)
			}
			return []byte(`{"config":{"revision":"rev-2","models":{"default":{"model":"gpt-5-mini","baseUrl":"https://api.example.com/v1","apiKeyConfigured":true,"maxTokens":1024,"timeoutSeconds":30}}},"changed":true}`), nil
		},
		isRunning: func() (bool, error) {
			calls = append(calls, "status")
			return false, nil
		},
		restart: func() error {
			t.Fatal("stopped service must not be restarted")
			return nil
		},
	})

	service := &Service{}
	c, recorder := newTestContext(http.MethodPut, "/api/ai/memory-fabric/config", secretRequest)
	service.SetMemoryFabricConfig(c)
	rsp := decodeMemoryFabricConfigAPIResponse(t, recorder.Body.Bytes())
	if rsp.Code != 0 || !rsp.Data.Changed || rsp.Data.Restarted || rsp.Data.Config["revision"] != "rev-2" {
		t.Fatalf("unexpected response: %#v", rsp)
	}
	if !configStringSlicesEqual(calls, []string{"update", "status"}) {
		t.Fatalf("calls=%v", calls)
	}
	if bytes.Contains(recorder.Body.Bytes(), []byte("new-secret")) {
		t.Fatalf("secret leaked in response: %s", recorder.Body.String())
	}
	models, ok := rsp.Data.Config["models"].(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected public models: %#v", rsp.Data.Config)
	}
	defaultModel, ok := models["default"].(map[string]interface{})
	if !ok || len(defaultModel) != 5 {
		t.Fatalf("unexpected public model shape: %#v", models)
	}
}

func TestSetMemoryFabricConfigUsesBackgroundContextsAfterRequestCancellation(t *testing.T) {
	backup := []byte("old config")
	checkBackgroundContext := func(ctx context.Context) {
		t.Helper()
		if err := ctx.Err(); err != nil {
			t.Fatalf("background command context was canceled: %v", err)
		}
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("background command context has no timeout")
		}
	}
	withMemoryFabricConfigTest(t, memoryFabricConfigTestOptions{
		backup:         backup,
		backupRevision: "rev-1",
		revisionRun: func(ctx context.Context, args []string, input []byte) ([]byte, error) {
			checkBackgroundContext(ctx)
			if !configStringSlicesEqual(args, []string{"revision", "--raw"}) || !bytes.Equal(input, backup) {
				t.Fatalf("unexpected revision args=%v input=%q", args, input)
			}
			return []byte("rev-1\n"), nil
		},
		run: func(ctx context.Context, args []string, _ []byte) ([]byte, error) {
			checkBackgroundContext(ctx)
			if !configStringSlicesEqual(args, []string{"update", "--json"}) {
				t.Fatalf("unexpected args=%v", args)
			}
			return []byte(`{"config":{"revision":"rev-1"},"changed":false}`), nil
		},
	})

	service := &Service{}
	c, recorder := newTestContext(http.MethodPut, "/api/ai/memory-fabric/config", `{"revision":"rev-1"}`)
	ctx, cancel := context.WithCancel(c.Request.Context())
	cancel()
	c.Request = c.Request.WithContext(ctx)
	service.SetMemoryFabricConfig(c)
	rsp := decodeMemoryFabricConfigAPIResponse(t, recorder.Body.Bytes())
	if rsp.Code != 0 {
		t.Fatalf("unexpected response: %#v", rsp)
	}
}

func TestSetMemoryFabricConfigRunningServiceRestartsOnce(t *testing.T) {
	restarts := 0
	statusChecks := 0
	withMemoryFabricConfigTest(t, memoryFabricConfigTestOptions{
		backup:         []byte("old config"),
		backupRevision: "rev-1",
		run: func(_ context.Context, args []string, _ []byte) ([]byte, error) {
			if !configStringSlicesEqual(args, []string{"update", "--json"}) {
				t.Fatalf("unexpected args=%v", args)
			}
			return []byte(`{"config":{"revision":"rev-2"},"changed":true}`), nil
		},
		isRunning: func() (bool, error) {
			statusChecks++
			return true, nil
		},
		restart: func() error {
			restarts++
			return nil
		},
	})

	service := &Service{}
	c, recorder := newTestContext(http.MethodPut, "/api/ai/memory-fabric/config", `{"revision":"rev-1"}`)
	service.SetMemoryFabricConfig(c)
	rsp := decodeMemoryFabricConfigAPIResponse(t, recorder.Body.Bytes())
	if rsp.Code != 0 || !rsp.Data.Changed || !rsp.Data.Restarted || restarts != 1 || statusChecks != 3 {
		t.Fatalf("response=%#v restarts=%d statusChecks=%d", rsp, restarts, statusChecks)
	}
}

func TestSetMemoryFabricConfigNoChangeSkipsStatusAndRestart(t *testing.T) {
	withMemoryFabricConfigTest(t, memoryFabricConfigTestOptions{
		backup:         []byte("old config"),
		backupRevision: "same",
		run: func(_ context.Context, args []string, _ []byte) ([]byte, error) {
			if !configStringSlicesEqual(args, []string{"update", "--json"}) {
				t.Fatalf("unexpected args=%v", args)
			}
			return []byte(`{"config":{"revision":"same"},"changed":false}`), nil
		},
		isRunning: func() (bool, error) {
			t.Fatal("unchanged config must not query service status")
			return false, nil
		},
		restart: func() error {
			t.Fatal("unchanged config must not restart service")
			return nil
		},
	})

	service := &Service{}
	c, recorder := newTestContext(http.MethodPut, "/api/ai/memory-fabric/config", `{"revision":"same"}`)
	service.SetMemoryFabricConfig(c)
	rsp := decodeMemoryFabricConfigAPIResponse(t, recorder.Body.Bytes())
	if rsp.Code != 0 || rsp.Data.Changed || rsp.Data.Restarted {
		t.Fatalf("unexpected response: %#v", rsp)
	}
}

func TestSetMemoryFabricConfigRestartFailureRestoresBackup(t *testing.T) {
	backup := []byte("old config with secret")
	var restore []byte
	restarts := 0
	withMemoryFabricConfigTest(t, memoryFabricConfigTestOptions{
		backup:         backup,
		backupRevision: "old",
		run: func(_ context.Context, args []string, input []byte) ([]byte, error) {
			switch args[0] {
			case "update":
				return []byte(`{"config":{"revision":"new"},"changed":true}`), nil
			case "restore":
				if !configStringSlicesEqual(args, []string{"restore", "--raw", "--if-revision", "new"}) {
					t.Fatalf("unexpected restore args=%v", args)
				}
				restore = append([]byte(nil), input...)
				return nil, nil
			default:
				t.Fatalf("unexpected args=%v", args)
				return nil, nil
			}
		},
		isRunning: func() (bool, error) { return true, nil },
		restart: func() error {
			restarts++
			if restarts == 1 {
				return errors.New("new config failed")
			}
			return nil
		},
	})

	service := &Service{}
	c, recorder := newTestContext(http.MethodPut, "/api/ai/memory-fabric/config", `{"revision":"old"}`)
	service.SetMemoryFabricConfig(c)
	rsp := decodeMemoryFabricConfigAPIResponse(t, recorder.Body.Bytes())
	if rsp.Code == 0 {
		t.Fatalf("expected failed response: %#v", rsp)
	}
	if !bytes.Equal(restore, backup) || restarts != 2 {
		t.Fatalf("restore=%q restarts=%d", restore, restarts)
	}
}

func TestSetMemoryFabricConfigUnstableRestartRestoresAndConfirmsBackup(t *testing.T) {
	backup := []byte("old config")
	var restored []byte
	statusResults := []bool{true, true, false, true, true}
	statusIndex := 0
	waits := 0
	restarts := 0
	withMemoryFabricConfigTest(t, memoryFabricConfigTestOptions{
		backup:         backup,
		backupRevision: "old",
		run: func(_ context.Context, args []string, input []byte) ([]byte, error) {
			switch args[0] {
			case "update":
				return []byte(`{"config":{"revision":"new"},"changed":true}`), nil
			case "restore":
				if !configStringSlicesEqual(args, []string{"restore", "--raw", "--if-revision", "new"}) {
					t.Fatalf("unexpected restore args=%v", args)
				}
				restored = append([]byte(nil), input...)
				return nil, nil
			default:
				t.Fatalf("unexpected args=%v", args)
				return nil, nil
			}
		},
		isRunning: func() (bool, error) {
			if statusIndex >= len(statusResults) {
				t.Fatalf("too many status checks: %d", statusIndex+1)
			}
			result := statusResults[statusIndex]
			statusIndex++
			return result, nil
		},
		restart: func() error {
			restarts++
			return nil
		},
		wait: func(context.Context, time.Duration) error {
			waits++
			if waits == 2 {
				return context.DeadlineExceeded
			}
			return nil
		},
	})

	service := &Service{}
	c, recorder := newTestContext(http.MethodPut, "/api/ai/memory-fabric/config", `{"revision":"old"}`)
	service.SetMemoryFabricConfig(c)
	rsp := decodeMemoryFabricConfigAPIResponse(t, recorder.Body.Bytes())
	if rsp.Code == 0 || !bytes.Equal(restored, backup) || restarts != 2 || statusIndex != len(statusResults) {
		t.Fatalf("response=%#v restored=%q restarts=%d statusChecks=%d", rsp, restored, restarts, statusIndex)
	}
}

func TestSetMemoryFabricConfigRollbackConflictPreservesConcurrentConfig(t *testing.T) {
	restarts := 0
	restoreCalls := 0
	withMemoryFabricConfigTest(t, memoryFabricConfigTestOptions{
		backup:         []byte("old config"),
		backupRevision: "old",
		run: func(_ context.Context, args []string, _ []byte) ([]byte, error) {
			switch args[0] {
			case "update":
				return []byte(`{"config":{"revision":"new"},"changed":true}`), nil
			case "restore":
				restoreCalls++
				if !configStringSlicesEqual(args, []string{"restore", "--raw", "--if-revision", "new"}) {
					t.Fatalf("unexpected restore args=%v", args)
				}
				return nil, errMemoryFabricConfigRevision
			default:
				t.Fatalf("unexpected args=%v", args)
				return nil, nil
			}
		},
		isRunning: func() (bool, error) { return true, nil },
		restart: func() error {
			restarts++
			return errors.New("new config failed")
		},
	})

	service := &Service{}
	c, recorder := newTestContext(http.MethodPut, "/api/ai/memory-fabric/config", `{"revision":"old"}`)
	service.SetMemoryFabricConfig(c)
	rsp := decodeMemoryFabricConfigAPIResponse(t, recorder.Body.Bytes())
	if rsp.Code == 0 || restoreCalls != 1 || restarts != 1 {
		t.Fatalf("response=%#v restoreCalls=%d restarts=%d", rsp, restoreCalls, restarts)
	}
}

func TestSetMemoryFabricConfigRevisionConflictDoesNotRollback(t *testing.T) {
	withMemoryFabricConfigTest(t, memoryFabricConfigTestOptions{
		backup:         []byte("old config"),
		backupRevision: "stale",
		run: func(_ context.Context, args []string, _ []byte) ([]byte, error) {
			if args[0] != "update" {
				t.Fatalf("unexpected rollback after conflict: %v", args)
			}
			return nil, errMemoryFabricConfigRevision
		},
	})

	service := &Service{}
	c, recorder := newTestContext(http.MethodPut, "/api/ai/memory-fabric/config", `{"revision":"stale"}`)
	service.SetMemoryFabricConfig(c)
	rsp := decodeMemoryFabricConfigAPIResponse(t, recorder.Body.Bytes())
	if rsp.Code != -3 {
		t.Fatalf("unexpected response: %#v", rsp)
	}
}

func TestSetMemoryFabricConfigBackupRevisionMismatchSkipsUpdate(t *testing.T) {
	withMemoryFabricConfigTest(t, memoryFabricConfigTestOptions{
		backup:         []byte("current config"),
		backupRevision: "current",
		run: func(context.Context, []string, []byte) ([]byte, error) {
			t.Fatal("backup revision mismatch must not update")
			return nil, nil
		},
	})

	service := &Service{}
	c, recorder := newTestContext(http.MethodPut, "/api/ai/memory-fabric/config", `{"revision":"stale"}`)
	service.SetMemoryFabricConfig(c)
	rsp := decodeMemoryFabricConfigAPIResponse(t, recorder.Body.Bytes())
	if rsp.Code != -3 {
		t.Fatalf("unexpected response: %#v", rsp)
	}
}

func TestSetMemoryFabricConfigInitializesMissingConfigBeforeBackup(t *testing.T) {
	backup := []byte("initialized config")
	reads := 0
	var calls []string
	withMemoryFabricConfigTest(t, memoryFabricConfigTestOptions{
		backup:         backup,
		backupRevision: "initialized",
		read: func(string) ([]byte, error) {
			reads++
			if reads == 1 {
				return nil, os.ErrNotExist
			}
			return append([]byte(nil), backup...), nil
		},
		run: func(_ context.Context, args []string, input []byte) ([]byte, error) {
			calls = append(calls, args[0])
			switch args[0] {
			case "get":
				if input != nil {
					t.Fatalf("get input=%q", input)
				}
				return []byte(`{"revision":"initialized"}`), nil
			case "update":
				return []byte(`{"config":{"revision":"next"},"changed":true}`), nil
			default:
				t.Fatalf("unexpected args=%v", args)
				return nil, nil
			}
		},
		isRunning: func() (bool, error) { return false, nil },
	})

	service := &Service{}
	c, recorder := newTestContext(http.MethodPut, "/api/ai/memory-fabric/config", `{"revision":"initialized"}`)
	service.SetMemoryFabricConfig(c)
	rsp := decodeMemoryFabricConfigAPIResponse(t, recorder.Body.Bytes())
	if rsp.Code != 0 || reads != 2 || !configStringSlicesEqual(calls, []string{"get", "update"}) {
		t.Fatalf("response=%#v reads=%d calls=%v", rsp, reads, calls)
	}
}

func TestSetMemoryFabricConfigNonRevisionCommandErrorDoesNotBlindlyRollback(t *testing.T) {
	withMemoryFabricConfigTest(t, memoryFabricConfigTestOptions{
		backup:         []byte("old config"),
		backupRevision: "old",
		run: func(_ context.Context, args []string, _ []byte) ([]byte, error) {
			if args[0] != "update" {
				t.Fatalf("unexpected command after indeterminate update failure: %v", args)
			}
			return nil, errors.New("command failed after an unknown commit boundary")
		},
	})

	service := &Service{}
	c, recorder := newTestContext(http.MethodPut, "/api/ai/memory-fabric/config", `{"revision":"old"}`)
	service.SetMemoryFabricConfig(c)
	rsp := decodeMemoryFabricConfigAPIResponse(t, recorder.Body.Bytes())
	if rsp.Code != -2 {
		t.Fatalf("unexpected response: %#v", rsp)
	}
}

func TestSetMemoryFabricConfigRejectsSensitiveUpdateOutputAndUsesCASRollback(t *testing.T) {
	for name, secretField := range map[string]string{
		"api key":  `"apiKey":"plain-secret"`,
		"token":    `"accessToken":"plain-token"`,
		"password": `"password":"plain-password"`,
	} {
		t.Run(name, func(t *testing.T) {
			backup := []byte("old config")
			restored := false
			withMemoryFabricConfigTest(t, memoryFabricConfigTestOptions{
				backup:         backup,
				backupRevision: "old",
				run: func(_ context.Context, args []string, input []byte) ([]byte, error) {
					switch args[0] {
					case "update":
						return []byte(`{"config":{"revision":"new",` + secretField + `},"changed":true}`), nil
					case "restore":
						if !configStringSlicesEqual(args, []string{"restore", "--raw", "--if-revision", "new"}) || !bytes.Equal(input, backup) {
							t.Fatalf("unexpected restore args=%v input=%q", args, input)
						}
						restored = true
						return nil, nil
					default:
						t.Fatalf("unexpected args=%v", args)
						return nil, nil
					}
				},
			})

			service := &Service{}
			c, recorder := newTestContext(http.MethodPut, "/api/ai/memory-fabric/config", `{"revision":"old"}`)
			service.SetMemoryFabricConfig(c)
			rsp := decodeMemoryFabricConfigAPIResponse(t, recorder.Body.Bytes())
			if rsp.Code != -2 || !restored {
				t.Fatalf("response=%#v restored=%t", rsp, restored)
			}
			if bytes.Contains(recorder.Body.Bytes(), []byte("plain-")) {
				t.Fatalf("sensitive CLI output leaked: %s", recorder.Body.String())
			}
		})
	}
}

func TestSetMemoryFabricConfigRejectsUnknownConfigOutputAndUsesCASRollback(t *testing.T) {
	for name, configOutput := range map[string]string{
		"removed api key env": `{"revision":"new","models":{"default":{"apiKeyEnv":"OPENAI_API_KEY"}}}`,
		"unknown model field": `{"revision":"new","models":{"default":{"displayName":"Default"}}}`,
		"unknown section":     `{"revision":"new","diagnostics":{"healthy":true}}`,
	} {
		t.Run(name, func(t *testing.T) {
			backup := []byte("old config")
			restored := false
			withMemoryFabricConfigTest(t, memoryFabricConfigTestOptions{
				backup:         backup,
				backupRevision: "old",
				run: func(_ context.Context, args []string, input []byte) ([]byte, error) {
					switch args[0] {
					case "update":
						return []byte(`{"config":` + configOutput + `,"changed":true}`), nil
					case "restore":
						if !configStringSlicesEqual(args, []string{"restore", "--raw", "--if-revision", "new"}) || !bytes.Equal(input, backup) {
							t.Fatalf("unexpected restore args=%v input=%q", args, input)
						}
						restored = true
						return nil, nil
					default:
						t.Fatalf("unexpected args=%v", args)
						return nil, nil
					}
				},
			})

			service := &Service{}
			c, recorder := newTestContext(http.MethodPut, "/api/ai/memory-fabric/config", `{"revision":"old"}`)
			service.SetMemoryFabricConfig(c)
			rsp := decodeMemoryFabricConfigAPIResponse(t, recorder.Body.Bytes())
			if rsp.Code != -2 || !restored {
				t.Fatalf("response=%#v restored=%t", rsp, restored)
			}
		})
	}
}

func TestSetMemoryFabricConfigRejectsInvalidJSONBeforeReadingConfig(t *testing.T) {
	withMemoryFabricConfigTest(t, memoryFabricConfigTestOptions{
		readErr: errors.New("must not read backup"),
		run: func(context.Context, []string, []byte) ([]byte, error) {
			t.Fatal("invalid JSON must not run CLI")
			return nil, nil
		},
	})

	service := &Service{}
	c, recorder := newTestContext(http.MethodPut, "/api/ai/memory-fabric/config", `{"revision":`)
	service.SetMemoryFabricConfig(c)
	rsp := decodeMemoryFabricConfigAPIResponse(t, recorder.Body.Bytes())
	if rsp.Code != -1 {
		t.Fatalf("unexpected response: %#v", rsp)
	}
}

type memoryFabricConfigTestOptions struct {
	backup         []byte
	backupRevision string
	readErr        error
	read           func(string) ([]byte, error)
	run            func(context.Context, []string, []byte) ([]byte, error)
	revisionRun    func(context.Context, []string, []byte) ([]byte, error)
	isRunning      func() (bool, error)
	restart        func() error
	wait           func(context.Context, time.Duration) error
}

func withMemoryFabricConfigTest(t *testing.T, opts memoryFabricConfigTestOptions) {
	t.Helper()
	oldRead := readMemoryFabricConfigFile
	oldRun := runMemoryFabricConfigCLI
	oldRuntime := newMemoryFabricConfigRuntime
	oldWait := waitMemoryFabricConfigPoll
	if opts.read != nil {
		readMemoryFabricConfigFile = opts.read
	} else {
		readMemoryFabricConfigFile = func(string) ([]byte, error) {
			if opts.readErr != nil {
				return nil, opts.readErr
			}
			return append([]byte(nil), opts.backup...), nil
		}
	}
	runMemoryFabricConfigCLI = func(ctx context.Context, args []string, input []byte) ([]byte, error) {
		if configStringSlicesEqual(args, []string{"revision", "--raw"}) {
			if opts.revisionRun != nil {
				return opts.revisionRun(ctx, args, input)
			}
			if opts.backupRevision == "" {
				t.Fatal("backupRevision is required for PUT tests")
			}
			return []byte(opts.backupRevision + "\n"), nil
		}
		if opts.run != nil {
			return opts.run(ctx, args, input)
		}
		return oldRun(ctx, args, input)
	}
	if opts.isRunning == nil {
		opts.isRunning = func() (bool, error) { return false, nil }
	}
	if opts.restart == nil {
		opts.restart = func() error { return nil }
	}
	newMemoryFabricConfigRuntime = func() memoryFabricConfigRuntime {
		return memoryFabricConfigRuntime{isRunning: opts.isRunning, restart: opts.restart}
	}
	if opts.wait == nil {
		opts.wait = func(context.Context, time.Duration) error { return nil }
	}
	waitMemoryFabricConfigPoll = opts.wait
	t.Cleanup(func() {
		readMemoryFabricConfigFile = oldRead
		runMemoryFabricConfigCLI = oldRun
		newMemoryFabricConfigRuntime = oldRuntime
		waitMemoryFabricConfigPoll = oldWait
	})
}

func decodeMemoryFabricConfigAPIResponse(t *testing.T, data []byte) memoryFabricConfigAPIResponse {
	t.Helper()
	var rsp memoryFabricConfigAPIResponse
	if err := json.Unmarshal(data, &rsp); err != nil {
		t.Fatalf("decode response: %v: %s", err, data)
	}
	return rsp
}

func configStringSlicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
