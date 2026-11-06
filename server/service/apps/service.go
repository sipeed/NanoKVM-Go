package apps

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
)

const defaultAppsRoot = "/kvmcomm/apps"

var builtinSipeedServer = Server{
	ID: "sipeed", Name: "Sipeed",
	URL: "https://github.com/sipeed/NanoKVM-Go-Apps", Official: true, Builtin: true,
}

var readAppInstallRandom = rand.Read

var appIDPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type Service struct {
	root   string
	client *http.Client
	mu     sync.Mutex
}

type AppInfo struct {
	Directory string   `json:"directory"`
	Manifest  Manifest `json:"manifest"`
}

type Server struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	URL      string `json:"url"`
	Official bool   `json:"official"`
	Builtin  bool   `json:"builtin"`
}

type CatalogApp struct {
	Directory string   `json:"directory"`
	Archive   string   `json:"archive"`
	Manifest  Manifest `json:"manifest"`
}

type Catalog struct {
	Apps []CatalogApp `json:"apps"`
}

func NewService() *Service {
	return newService(defaultAppsRoot)
}

func newService(root string) *Service {
	return &Service{root: root, client: &http.Client{Timeout: 10 * time.Second}}
}

func (s *Service) List(c *gin.Context) {
	apps, err := s.list()
	respond(c, apps, err)
}

func (s *Service) Inspect(c *gin.Context) {
	data, err := uploadedZIP(c)
	if err != nil {
		respond(c, nil, err)
		return
	}
	archive, err := inspectArchive(data)
	if err != nil {
		respond(c, nil, err)
		return
	}
	id := strings.TrimSpace(c.PostForm("id"))
	if id == "" {
		id = archiveID(archive, c.PostForm("filename"))
	}
	if !appIDPattern.MatchString(id) {
		respond(c, nil, errors.New("App id must use lowercase kebab-case"))
		return
	}
	respond(c, AppInfo{Directory: id, Manifest: publicManifest(archive.Manifest)}, nil)
}

func (s *Service) Install(c *gin.Context) {
	app, err := s.installUploaded(c, io.Discard)
	respond(c, app, err)
}

func (s *Service) InstallStream(c *gin.Context) {
	streamInstall(c, func(_ context.Context, log io.Writer) (interface{}, error) {
		return s.installUploaded(c, log)
	})
}

func (s *Service) installUploaded(c *gin.Context, log io.Writer) (AppInfo, error) {
	fmt.Fprintln(log, "Reading uploaded App archive...")
	data, err := uploadedZIP(c)
	if err != nil {
		return AppInfo{}, err
	}
	fmt.Fprintln(log, "Validating app.json and archive layout...")
	archive, err := inspectArchive(data)
	if err != nil {
		return AppInfo{}, err
	}
	id := strings.TrimSpace(c.PostForm("id"))
	if id == "" {
		id = archiveID(archive, "")
	}
	if !appIDPattern.MatchString(id) {
		return AppInfo{}, errors.New("App id must use lowercase kebab-case")
	}
	values := map[string]string{}
	if raw := c.PostForm("env"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &values); err != nil {
			return AppInfo{}, errors.New("env must be a JSON string map")
		}
	}
	if err := applyEnv(&archive.Manifest, values); err != nil {
		return AppInfo{}, err
	}
	if err := s.installArchiveWithLogContext(c.Request.Context(), id, archive, log); err != nil {
		return AppInfo{}, err
	}
	return AppInfo{Directory: id, Manifest: publicManifest(archive.Manifest)}, nil
}

func (s *Service) Download(c *gin.Context) {
	id := c.Param("id")
	dir, manifest, err := s.resolveApp(id)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.zip"`, id))
	if err := writeArchive(c.Writer, dir, id, manifest); err != nil {
		c.Error(err)
	}
}

func (s *Service) Remove(c *gin.Context) {
	id := c.Param("id")
	if !appIDPattern.MatchString(id) {
		respond(c, nil, errors.New("invalid App id"))
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	dir, _, err := s.resolveApp(id)
	if err == nil {
		err = os.RemoveAll(dir)
	}
	respond(c, nil, err)
}

func (s *Service) UpdateEnv(c *gin.Context) {
	id := c.Param("id")
	var values map[string]string
	if err := c.ShouldBindJSON(&values); err != nil {
		respond(c, nil, errors.New("env must be a JSON string map"))
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	dir, manifest, err := s.resolveApp(id)
	if err == nil {
		err = updateEnv(&manifest, values)
	}
	if err == nil {
		data, marshalErr := json.MarshalIndent(manifest, "", "  ")
		if marshalErr != nil {
			err = marshalErr
		} else {
			err = writeAtomic(filepath.Join(dir, "app.json"), append(data, '\n'), 0o600)
		}
	}
	if err != nil {
		respond(c, nil, err)
		return
	}
	respond(c, AppInfo{Directory: id, Manifest: publicManifest(manifest)}, nil)
}

func (s *Service) GetServers(c *gin.Context) {
	servers, err := s.readServers()
	respond(c, servers, err)
}

func (s *Service) PutServers(c *gin.Context) {
	var servers []Server
	if err := c.ShouldBindJSON(&servers); err != nil {
		respond(c, nil, errors.New("invalid server list"))
		return
	}
	servers = customServers(servers)
	if err := validateServers(servers); err != nil {
		respond(c, nil, err)
		return
	}
	data, _ := json.MarshalIndent(servers, "", "  ")
	if err := os.MkdirAll(s.root, 0o755); err == nil {
		err = writeAtomic(filepath.Join(s.root, "_servers.json"), append(data, '\n'), 0o600)
		respond(c, append([]Server{builtinSipeedServer}, servers...), err)
	} else {
		respond(c, nil, err)
	}
}

func (s *Service) Catalog(c *gin.Context) {
	server, err := s.findServer(c.Param("id"))
	if err != nil {
		respond(c, nil, err)
		return
	}
	catalog, err := s.fetchCatalog(server)
	respond(c, catalog, err)
}

func (s *Service) InstallRemote(c *gin.Context) {
	app, err := s.installRemote(c, io.Discard)
	respond(c, app, err)
}

func (s *Service) InstallRemoteStream(c *gin.Context) {
	streamInstall(c, func(_ context.Context, log io.Writer) (interface{}, error) {
		return s.installRemote(c, log)
	})
}

func (s *Service) installRemote(c *gin.Context, log io.Writer) (AppInfo, error) {
	var req struct {
		ServerID string            `json:"server_id"`
		AppID    string            `json:"app_id"`
		Env      map[string]string `json:"env"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return AppInfo{}, errors.New("invalid request")
	}
	fmt.Fprintln(log, "Loading App server catalog...")
	server, err := s.findServer(req.ServerID)
	if err != nil {
		return AppInfo{}, err
	}
	catalog, archives, err := s.fetchCatalogWithArchivesContext(c.Request.Context(), server)
	if err != nil {
		return AppInfo{}, err
	}
	var item *CatalogApp
	for i := range catalog.Apps {
		if catalog.Apps[i].Manifest.AppID == req.AppID {
			item = &catalog.Apps[i]
			break
		}
	}
	if item == nil {
		return AppInfo{}, errors.New("App not found in server catalog")
	}
	fmt.Fprintf(log, "Preparing %s %s...\n", item.Manifest.Name, item.Manifest.Version)
	var archive *inspectedArchive
	if archives != nil {
		archive = archives[item.Directory]
		if archive == nil {
			err = errors.New("App not found in GitHub repository archive")
		}
	} else {
		var archiveURL string
		archiveURL, err = resolveServerURL(server.URL, item.Archive)
		var data []byte
		if err == nil {
			data, err = s.fetchContext(c.Request.Context(), archiveURL, maxArchiveBytes)
		}
		if err == nil {
			archive, err = inspectArchive(data)
		}
	}
	if err == nil {
		err = applyEnv(&archive.Manifest, req.Env)
	}
	if err == nil {
		err = s.installArchiveWithLogContext(c.Request.Context(), item.Directory, archive, log)
	}
	if err != nil {
		return AppInfo{}, err
	}
	return AppInfo{Directory: item.Directory, Manifest: publicManifest(archive.Manifest)}, nil
}

func (s *Service) list() ([]AppInfo, error) {
	entries, err := os.ReadDir(s.root)
	if os.IsNotExist(err) {
		return []AppInfo{}, nil
	}
	if err != nil {
		return nil, err
	}
	apps := make([]AppInfo, 0)
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), "_") || !appIDPattern.MatchString(entry.Name()) {
			continue
		}
		manifest, err := readManifest(filepath.Join(s.root, entry.Name(), "app.json"))
		if err != nil {
			continue
		}
		if err := validateAppIDForDirectory(manifest, entry.Name()); err != nil {
			continue
		}
		if info, err := os.Stat(filepath.Join(s.root, entry.Name(), "main.py")); err != nil || !info.Mode().IsRegular() {
			continue
		}
		apps = append(apps, AppInfo{Directory: entry.Name(), Manifest: publicManifest(manifest)})
	}
	sort.Slice(apps, func(i, j int) bool { return apps[i].Manifest.Name < apps[j].Manifest.Name })
	return apps, nil
}

func (s *Service) installArchive(id string, archive *inspectedArchive) error {
	return s.installArchiveWithLog(id, archive, io.Discard)
}

func (s *Service) installArchiveWithLog(id string, archive *inspectedArchive, log io.Writer) error {
	return s.installArchiveWithLogContext(context.Background(), id, archive, log)
}

func (s *Service) installArchiveWithLogContext(ctx context.Context, id string, archive *inspectedArchive, log io.Writer) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := installContextError(ctx); err != nil {
		return err
	}
	if log == nil {
		log = io.Discard
	}
	fmt.Fprintf(log, "Installing %s into %s...\n", archive.Manifest.AppID, filepath.Join(s.root, id))
	if err := os.MkdirAll(s.root, 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(s.root, id)); err == nil {
		return errors.New("App is already installed; remove it before reinstalling")
	}
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), "_") {
			continue
		}
		manifest, err := readManifest(filepath.Join(s.root, entry.Name(), "app.json"))
		if err == nil && manifest.AppID == archive.Manifest.AppID {
			return fmt.Errorf("App package %q is already installed", archive.Manifest.AppID)
		}
	}
	random := make([]byte, 8)
	if _, err := readAppInstallRandom(random); err != nil {
		return fmt.Errorf("create App installation identifier: %w", err)
	}
	temp := filepath.Join(s.root, "_install-"+hex.EncodeToString(random))
	defer os.RemoveAll(temp)
	fmt.Fprintln(log, "Extracting App archive...")
	if err := extractArchive(archive, temp, archive.Manifest); err != nil {
		return err
	}
	if err := installContextError(ctx); err != nil {
		return err
	}
	if archive.Manifest.PreScript != "" {
		fmt.Fprintf(log, "Running pre_script: %s\n", archive.Manifest.PreScript)
	}
	if err := runInstallScript(ctx, temp, archive.Manifest.PreScript, "pre_script", archive.Manifest, log); err != nil {
		return err
	}
	if err := installContextError(ctx); err != nil {
		return err
	}
	fmt.Fprintln(log, "Activating App files...")
	destination := filepath.Join(s.root, id)
	if err := os.Rename(temp, destination); err != nil {
		return err
	}
	if archive.Manifest.PostScript != "" {
		fmt.Fprintf(log, "Running post_script: %s\n", archive.Manifest.PostScript)
	}
	if err := runInstallScript(ctx, destination, archive.Manifest.PostScript, "post_script", archive.Manifest, log); err != nil {
		_ = os.RemoveAll(destination)
		return err
	}
	if err := installContextError(ctx); err != nil {
		_ = os.RemoveAll(destination)
		return err
	}
	fmt.Fprintln(log, "Installation completed successfully.")
	return nil
}

func installContextError(ctx context.Context) error {
	if ctx.Err() != nil {
		return errors.New("installation cancelled")
	}
	return nil
}

func (s *Service) resolveApp(id string) (string, Manifest, error) {
	if !appIDPattern.MatchString(id) {
		return "", Manifest{}, errors.New("invalid App id")
	}
	dir := filepath.Join(s.root, id)
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", Manifest{}, errors.New("App not found")
	}
	manifest, err := readManifest(filepath.Join(dir, "app.json"))
	return dir, manifest, err
}

func uploadedZIP(c *gin.Context) ([]byte, error) {
	file, err := c.FormFile("file")
	if err != nil || file.Size <= 0 || file.Size > maxArchiveBytes {
		return nil, errors.New("a ZIP file up to 64 MiB is required")
	}
	r, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(io.LimitReader(r, maxArchiveBytes+1))
}

func archiveID(archive *inspectedArchive, fallback string) string {
	if archive.Prefix != "" {
		return strings.TrimSuffix(archive.Prefix, "/")
	}
	base := strings.TrimSuffix(filepath.Base(fallback), filepath.Ext(fallback))
	if appIDPattern.MatchString(base) {
		return base
	}
	return slug(archive.Manifest.Name)
}

func slug(value string) string {
	value = strings.ToLower(value)
	var out strings.Builder
	dash := false
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			out.WriteRune(r)
			dash = false
		} else if out.Len() > 0 && !dash {
			out.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(out.String(), "-")
}

func respond(c *gin.Context, data interface{}, err error) {
	var rsp proto.Response
	if err != nil {
		rsp.ErrRsp(c, -1, err.Error())
	} else {
		rsp.OkRspWithData(c, data)
	}
}

func (s *Service) readServers() ([]Server, error) {
	data, err := os.ReadFile(filepath.Join(s.root, "_servers.json"))
	if os.IsNotExist(err) {
		return []Server{builtinSipeedServer}, nil
	}
	if err != nil {
		return []Server{builtinSipeedServer}, nil
	}
	var servers []Server
	if err := json.Unmarshal(data, &servers); err != nil {
		return []Server{builtinSipeedServer}, nil
	}
	servers = customServers(servers)
	if err := validateServers(servers); err != nil {
		return []Server{builtinSipeedServer}, nil
	}
	return append([]Server{builtinSipeedServer}, servers...), nil
}

func customServers(servers []Server) []Server {
	custom := make([]Server, 0, len(servers))
	builtinURL := strings.TrimRight(strings.ToLower(builtinSipeedServer.URL), "/")
	for _, server := range servers {
		if server.ID == builtinSipeedServer.ID ||
			strings.TrimRight(strings.ToLower(server.URL), "/") == builtinURL {
			continue
		}
		server.Builtin = false
		custom = append(custom, server)
	}
	return custom
}

func validateServers(servers []Server) error {
	ids := map[string]bool{}
	for _, server := range servers {
		if !appIDPattern.MatchString(server.ID) || strings.TrimSpace(server.Name) == "" || ids[server.ID] {
			return errors.New("server id must be unique kebab-case and name must not be empty")
		}
		ids[server.ID] = true
		u, err := url.Parse(server.URL)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil {
			return errors.New("server URL must be an HTTP(S) URL without credentials")
		}
	}
	return nil
}

func (s *Service) findServer(id string) (Server, error) {
	servers, err := s.readServers()
	if err == nil {
		for _, server := range servers {
			if server.ID == id {
				return server, nil
			}
		}
	}
	return Server{}, errors.New("App server not found")
}

func (s *Service) fetchCatalog(server Server) (Catalog, error) {
	catalog, _, err := s.fetchCatalogWithArchives(server)
	return catalog, err
}

func (s *Service) fetchCatalogWithArchives(server Server) (Catalog, map[string]*inspectedArchive, error) {
	return s.fetchCatalogWithArchivesContext(context.Background(), server)
}

func (s *Service) fetchCatalogWithArchivesContext(ctx context.Context, server Server) (Catalog, map[string]*inspectedArchive, error) {
	if archiveURL, ok := githubArchiveURL(server.URL); ok {
		data, err := s.fetchContext(ctx, archiveURL, maxArchiveBytes)
		if err != nil {
			return Catalog{}, nil, err
		}
		return inspectRepositoryArchive(data)
	}
	rawURL, err := resolveServerURL(server.URL, "index.json")
	if err != nil {
		return Catalog{}, nil, err
	}
	data, err := s.fetchContext(ctx, rawURL, 4<<20)
	if err != nil {
		return Catalog{}, nil, err
	}
	var catalog Catalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return Catalog{}, nil, errors.New("invalid App server index.json")
	}
	packages := make(map[string]bool)
	for i, app := range catalog.Apps {
		if !appIDPattern.MatchString(app.Directory) || app.Archive == "" {
			return Catalog{}, nil, errors.New("invalid App server catalog entry")
		}
		if _, err := parseManifest(mustJSON(app.Manifest)); err != nil {
			return Catalog{}, nil, err
		}
		if err := validateAppIDForDirectory(app.Manifest, app.Directory); err != nil {
			return Catalog{}, nil, err
		}
		if packages[app.Manifest.AppID] {
			return Catalog{}, nil, fmt.Errorf("duplicate app_id %q in App server catalog", app.Manifest.AppID)
		}
		packages[app.Manifest.AppID] = true
		catalog.Apps[i].Manifest = publicManifest(app.Manifest)
	}
	return catalog, nil, nil
}

func githubArchiveURL(rawURL string) (string, bool) {
	u, err := url.Parse(strings.TrimRight(strings.TrimSpace(rawURL), "/"))
	if err != nil || u.Scheme != "https" || !strings.EqualFold(u.Host, "github.com") || u.RawQuery != "" || u.Fragment != "" {
		return "", false
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", false
	}
	repository := strings.TrimSuffix(parts[1], ".git")
	if repository == "" {
		return "", false
	}
	return fmt.Sprintf("https://codeload.github.com/%s/%s/zip/HEAD", parts[0], repository), true
}

func resolveServerURL(base, ref string) (string, error) {
	b, err := url.Parse(strings.TrimRight(base, "/") + "/")
	if err != nil {
		return "", err
	}
	r, err := url.Parse(ref)
	if err != nil {
		return "", err
	}
	u := b.ResolveReference(r)
	if u.Scheme != b.Scheme || u.Host != b.Host {
		return "", errors.New("App archive must use the configured server origin")
	}
	return u.String(), nil
}

func (s *Service) fetch(rawURL string, limit int64) ([]byte, error) {
	return s.fetchContext(context.Background(), rawURL, limit)
}

func (s *Service) fetchContext(ctx context.Context, rawURL string, limit int64) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("App server returned HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil || int64(len(data)) > limit {
		return nil, errors.New("App server response is too large")
	}
	return data, nil
}

func writeAtomic(path string, data []byte, mode os.FileMode) error {
	temp := path + ".tmp"
	if err := os.WriteFile(temp, data, mode); err != nil {
		return err
	}
	return os.Rename(temp, path)
}

func mustJSON(value interface{}) []byte {
	data, _ := json.Marshal(value)
	return data
}
