package application

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
)

func TestApplicationRegionStoreDefaultsToGlobal(t *testing.T) {
	store := newApplicationRegionStore(filepath.Join(t.TempDir(), "missing", "region"))
	if got := store.load(); got != proto.ApplicationRegionGlobal {
		t.Fatalf("missing region = %q, want global", got)
	}

	if err := os.MkdirAll(filepath.Dir(store.path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.path, []byte("invalid\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := store.load(); got != proto.ApplicationRegionGlobal {
		t.Fatalf("invalid region = %q, want global", got)
	}

	if err := os.WriteFile(store.path, []byte(strings.Repeat("x", applicationRegionMaxBytes+1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := store.load(); got != proto.ApplicationRegionGlobal {
		t.Fatalf("oversized region = %q, want global", got)
	}
}

func TestApplicationRegionStoreSavesAtomically(t *testing.T) {
	directory := t.TempDir()
	store := newApplicationRegionStore(filepath.Join(directory, "nested", "region"))
	if err := store.save(proto.ApplicationRegionCN); err != nil {
		t.Fatal(err)
	}
	if got := store.load(); got != proto.ApplicationRegionCN {
		t.Fatalf("saved region = %q, want cn", got)
	}
	content, err := os.ReadFile(store.path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "cn\n" {
		t.Fatalf("saved content = %q, want %q", content, "cn\\n")
	}
	info, err := os.Stat(store.path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0o644); got != want {
		t.Fatalf("saved mode = %o, want %o", got, want)
	}
	entries, err := os.ReadDir(filepath.Dir(store.path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "region" {
		t.Fatalf("temporary files remain: %v", entries)
	}
}

func TestApplicationRegionStoreRejectsSymlink(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "target")
	path := filepath.Join(directory, "region")
	if err := os.WriteFile(target, []byte("global\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	store := newApplicationRegionStore(path)
	if got := store.load(); got != proto.ApplicationRegionGlobal {
		t.Fatalf("symlink load = %q, want global", got)
	}
	if err := store.save(proto.ApplicationRegionCN); err == nil {
		t.Fatal("save through symlink unexpectedly succeeded")
	}
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "global\n" {
		t.Fatalf("symlink target changed to %q", content)
	}
}

type fakeApplicationRegionStore struct {
	mu             sync.Mutex
	region         proto.ApplicationRegion
	saves          []proto.ApplicationRegion
	firstSaveError error
	saveEntered    chan struct{}
	continueSave   chan struct{}
}

type fakeRegionCoordinator struct {
	regions            []proto.ApplicationRegion
	cloudCleanupFailed bool
	err                error
	releases           int
}

func (coordinator *fakeRegionCoordinator) PrepareRegionChange(
	_ context.Context,
	region proto.ApplicationRegion,
) (bool, func(), error) {
	coordinator.regions = append(coordinator.regions, region)
	if coordinator.err != nil {
		return coordinator.cloudCleanupFailed, nil, coordinator.err
	}
	return coordinator.cloudCleanupFailed, func() { coordinator.releases++ }, nil
}

func (store *fakeApplicationRegionStore) load() proto.ApplicationRegion {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.region
}

func (store *fakeApplicationRegionStore) save(region proto.ApplicationRegion) error {
	store.mu.Lock()
	store.saves = append(store.saves, region)
	first := len(store.saves) == 1
	if first && store.saveEntered != nil {
		close(store.saveEntered)
	}
	if first && store.continueSave != nil {
		continueChannel := store.continueSave
		store.mu.Unlock()
		<-continueChannel
		store.mu.Lock()
	}
	err := store.firstSaveError
	store.firstSaveError = nil
	if err == nil {
		store.region = region
	}
	store.mu.Unlock()
	return err
}

func TestLoadRegionDelegatesToDefaultStore(t *testing.T) {
	stub := &loadRegionStub{region: proto.ApplicationRegionCN}
	original := defaultApplicationRegionStore
	defaultApplicationRegionStore = stub
	t.Cleanup(func() { defaultApplicationRegionStore = original })

	if got := LoadRegion(); got != proto.ApplicationRegionCN {
		t.Fatalf("LoadRegion() = %q, want %q", got, proto.ApplicationRegionCN)
	}
	if stub.loads != 1 {
		t.Fatalf("default store load calls = %d, want 1", stub.loads)
	}
}

type loadRegionStub struct {
	loads  int
	region proto.ApplicationRegion
}

func (stub *loadRegionStub) load() proto.ApplicationRegion {
	stub.loads++
	return stub.region
}

func (stub *loadRegionStub) save(proto.ApplicationRegion) error { return nil }

func newRegionContext(body string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/application/region", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	context, _ := gin.CreateTestContext(recorder)
	context.Request = request
	return context, recorder
}

// noopFeatureManifestSync keeps a region-switch test hermetic: the production
// fallback would reach the CDN and rewrite the device manifest.
func noopFeatureManifestSync(context.Context, proto.ApplicationRegion) error { return nil }

func TestGetRegionReturnsPersistedValue(t *testing.T) {
	service := &Service{regionStore: &fakeApplicationRegionStore{region: proto.ApplicationRegionCN}}
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/application/region", nil)
	service.GetRegion(context)
	response, data := decodeRegionResponse(t, recorder)
	if response.Code != 0 || data.Region != proto.ApplicationRegionCN {
		t.Fatalf("get response = %+v / %+v", response, data)
	}
}

func decodeRegionResponse(t *testing.T, recorder *httptest.ResponseRecorder) (proto.Response, proto.GetApplicationRegionRsp) {
	t.Helper()
	var envelope struct {
		Code int                           `json:"code"`
		Msg  string                        `json:"msg"`
		Data proto.GetApplicationRegionRsp `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, recorder.Body.String())
	}
	return proto.Response{Code: envelope.Code, Msg: envelope.Msg}, envelope.Data
}

func TestSetRegionSameValueDoesNotRestart(t *testing.T) {
	store := &fakeApplicationRegionStore{region: proto.ApplicationRegionGlobal}
	coordinator := &fakeRegionCoordinator{}
	restarts := 0
	service := &Service{
		regionStore:       store,
		restartKVMComm:    func() error { restarts++; return nil },
		regionCoordinator: coordinator,
	}
	context, recorder := newRegionContext(`{"region":"global"}`)
	service.SetRegion(context)
	response, data := decodeRegionResponse(t, recorder)
	if response.Code != 0 || data.Region != proto.ApplicationRegionGlobal {
		t.Fatalf("same-region response = %+v / %+v", response, data)
	}
	if restarts != 0 {
		t.Fatalf("same-region restarts = %d, want 0", restarts)
	}
	if len(coordinator.regions) != 0 || coordinator.releases != 0 {
		t.Fatalf("same-region coordinator calls = %v, releases=%d", coordinator.regions, coordinator.releases)
	}
}

func TestSetRegionCoordinatesManagedCleanupThroughApplicationEndpoint(t *testing.T) {
	store := &fakeApplicationRegionStore{region: proto.ApplicationRegionCN}
	coordinator := &fakeRegionCoordinator{cloudCleanupFailed: true}
	restarts := 0
	service := &Service{
		regionStore:         store,
		restartKVMComm:      func() error { restarts++; return nil },
		regionCoordinator:   coordinator,
		syncFeatureManifest: noopFeatureManifestSync,
	}
	context, recorder := newRegionContext(`{"region":"global"}`)
	service.SetRegion(context)
	response, data := decodeRegionResponse(t, recorder)
	if response.Code != 0 || data.Region != proto.ApplicationRegionGlobal || !data.CloudCleanupFailed {
		t.Fatalf("coordinated response = %+v / %+v", response, data)
	}
	if !reflect.DeepEqual(coordinator.regions, []proto.ApplicationRegion{proto.ApplicationRegionGlobal}) {
		t.Fatalf("coordinated regions = %v", coordinator.regions)
	}
	if coordinator.releases != 1 {
		t.Fatalf("coordinator releases = %d, want 1", coordinator.releases)
	}
	if store.load() != proto.ApplicationRegionGlobal || restarts != 1 {
		t.Fatalf("region transition = region %q, restarts %d", store.load(), restarts)
	}
}

func TestSetRegionCoordinatorFailureBlocksPersistence(t *testing.T) {
	store := &fakeApplicationRegionStore{region: proto.ApplicationRegionCN}
	coordinator := &fakeRegionCoordinator{err: errors.New("local managed logout failed")}
	restarts := 0
	service := &Service{
		regionStore:       store,
		restartKVMComm:    func() error { restarts++; return nil },
		regionCoordinator: coordinator,
	}
	context, recorder := newRegionContext(`{"region":"global"}`)
	service.SetRegion(context)
	response, data := decodeRegionResponse(t, recorder)
	if response.Code == 0 || data.Region != proto.ApplicationRegionCN {
		t.Fatalf("coordinator failure response = %+v / %+v", response, data)
	}
	if len(store.saves) != 0 || restarts != 0 || coordinator.releases != 0 {
		t.Fatalf(
			"coordinator failure mutated state: saves=%v restarts=%d releases=%d",
			store.saves,
			restarts,
			coordinator.releases,
		)
	}
}

func TestSetRegionChangesAndRestarts(t *testing.T) {
	store := &fakeApplicationRegionStore{region: proto.ApplicationRegionGlobal}
	restarts := 0
	service := &Service{
		regionStore:         store,
		syncFeatureManifest: noopFeatureManifestSync,
		restartKVMComm: func() error {
			restarts++
			return nil
		},
	}
	context, recorder := newRegionContext(`{"region":"cn"}`)
	service.SetRegion(context)
	response, data := decodeRegionResponse(t, recorder)
	if response.Code != 0 || data.Region != proto.ApplicationRegionCN {
		t.Fatalf("changed response = %+v / %+v", response, data)
	}
	if got := store.load(); got != proto.ApplicationRegionCN {
		t.Fatalf("stored region = %q, want cn", got)
	}
	if restarts != 1 {
		t.Fatalf("restarts = %d, want 1", restarts)
	}
}

func TestSetRegionSaveFailureRestoresPreviousValue(t *testing.T) {
	store := &fakeApplicationRegionStore{
		region:         proto.ApplicationRegionGlobal,
		firstSaveError: errors.New("disk full"),
	}
	restarts := 0
	service := &Service{
		regionStore:         store,
		syncFeatureManifest: noopFeatureManifestSync,
		restartKVMComm:      func() error { restarts++; return nil },
	}
	context, recorder := newRegionContext(`{"region":"cn"}`)
	service.SetRegion(context)
	response, data := decodeRegionResponse(t, recorder)
	if response.Code == 0 || data.Region != proto.ApplicationRegionGlobal {
		t.Fatalf("save failure response = %+v / %+v", response, data)
	}
	if got := store.load(); got != proto.ApplicationRegionGlobal {
		t.Fatalf("stored region after save failure = %q", got)
	}
	if len(store.saves) != 2 || store.saves[0] != proto.ApplicationRegionCN || store.saves[1] != proto.ApplicationRegionGlobal {
		t.Fatalf("save sequence = %v", store.saves)
	}
	if restarts != 0 {
		t.Fatalf("save failure restarts = %d, want 0", restarts)
	}
}

func TestSetRegionRestartFailureRollsBackAndRestarts(t *testing.T) {
	store := &fakeApplicationRegionStore{region: proto.ApplicationRegionGlobal}
	restarts := 0
	service := &Service{
		regionStore:         store,
		syncFeatureManifest: noopFeatureManifestSync,
		restartKVMComm: func() error {
			restarts++
			if restarts == 1 {
				return errors.New("systemd unavailable")
			}
			return nil
		},
	}
	context, recorder := newRegionContext(`{"region":"cn"}`)
	service.SetRegion(context)
	response, data := decodeRegionResponse(t, recorder)
	if response.Code == 0 || data.Region != proto.ApplicationRegionGlobal {
		t.Fatalf("restart failure response = %+v / %+v", response, data)
	}
	if got := store.load(); got != proto.ApplicationRegionGlobal {
		t.Fatalf("stored region after restart failure = %q", got)
	}
	if restarts != 2 {
		t.Fatalf("restart calls = %d, want 2", restarts)
	}
}

func TestSetRegionRejectsInvalidJSONWithoutRestart(t *testing.T) {
	store := &fakeApplicationRegionStore{region: proto.ApplicationRegionGlobal}
	restarts := 0
	service := &Service{regionStore: store, restartKVMComm: func() error { restarts++; return nil }}
	for _, body := range []string{
		`{"region":"us"}`,
		`{"region":"cn","extra":true}`,
		`{"region":"cn"} {"region":"global"}`,
		`{}`,
	} {
		context, recorder := newRegionContext(body)
		service.SetRegion(context)
		response, _ := decodeRegionResponse(t, recorder)
		if response.Code == 0 {
			t.Fatalf("body %q unexpectedly succeeded", body)
		}
	}
	if restarts != 0 || store.load() != proto.ApplicationRegionGlobal {
		t.Fatalf("invalid requests changed state: restarts=%d region=%q", restarts, store.load())
	}
}

func TestSetRegionSerializesMutations(t *testing.T) {
	store := &fakeApplicationRegionStore{
		region:       proto.ApplicationRegionGlobal,
		saveEntered:  make(chan struct{}),
		continueSave: make(chan struct{}),
	}
	service := &Service{
		regionStore:         store,
		syncFeatureManifest: noopFeatureManifestSync,
		restartKVMComm:      func() error { return nil },
	}
	firstContext, firstRecorder := newRegionContext(`{"region":"cn"}`)
	secondContext, secondRecorder := newRegionContext(`{"region":"global"}`)
	firstDone := make(chan struct{})
	go func() {
		service.SetRegion(firstContext)
		close(firstDone)
	}()
	select {
	case <-store.saveEntered:
	case <-time.After(time.Second):
		t.Fatal("first region mutation did not enter save")
	}
	secondDone := make(chan struct{})
	go func() {
		service.SetRegion(secondContext)
		close(secondDone)
	}()
	select {
	case <-secondDone:
		t.Fatal("second region mutation bypassed region mutex")
	case <-time.After(50 * time.Millisecond):
	}
	close(store.continueSave)
	select {
	case <-firstDone:
	case <-time.After(time.Second):
		t.Fatal("first region mutation did not finish")
	}
	select {
	case <-secondDone:
	case <-time.After(time.Second):
		t.Fatal("second region mutation did not finish")
	}
	firstResponse, _ := decodeRegionResponse(t, firstRecorder)
	secondResponse, _ := decodeRegionResponse(t, secondRecorder)
	if firstResponse.Code != 0 || secondResponse.Code != 0 {
		t.Fatalf("serialized responses = %+v, %+v", firstResponse, secondResponse)
	}
}

// orderedRegionStore records region saves into a test-owned order log while
// delegating to the existing fake so load and save semantics are unchanged.
type orderedRegionStore struct {
	*fakeApplicationRegionStore
	onSave func()
}

func (store *orderedRegionStore) save(region proto.ApplicationRegion) error {
	if store.onSave != nil {
		store.onSave()
	}
	return store.fakeApplicationRegionStore.save(region)
}

func TestSetRegionSyncsFeatureManifestBeforePublishingRegion(t *testing.T) {
	var (
		mu           sync.Mutex
		order        []string
		regionAtSync proto.ApplicationRegion
	)
	record := func(step string) {
		mu.Lock()
		order = append(order, step)
		mu.Unlock()
	}

	fake := &fakeApplicationRegionStore{region: proto.ApplicationRegionGlobal}
	store := &orderedRegionStore{
		fakeApplicationRegionStore: fake,
		onSave:                     func() { record("save") },
	}
	service := &Service{
		regionStore: store,
		restartKVMComm: func() error {
			record("restart")
			return nil
		},
		syncFeatureManifest: func(_ context.Context, region proto.ApplicationRegion) error {
			if region != proto.ApplicationRegionCN {
				t.Errorf("sync region = %q, want cn", region)
			}
			regionAtSync = fake.load()
			record("sync")
			return nil
		},
	}

	context, recorder := newRegionContext(`{"region":"cn"}`)
	service.SetRegion(context)

	response, data := decodeRegionResponse(t, recorder)
	if response.Code != 0 || data.Region != proto.ApplicationRegionCN {
		t.Fatalf("response = %+v / %+v", response, data)
	}
	mu.Lock()
	gotOrder := append([]string(nil), order...)
	mu.Unlock()
	if !reflect.DeepEqual(gotOrder, []string{"sync", "save", "restart"}) {
		t.Fatalf("SetRegion order = %v, want [sync save restart]", gotOrder)
	}
	if regionAtSync != proto.ApplicationRegionGlobal {
		t.Fatalf("region observed by sync = %q, want pre-switch global", regionAtSync)
	}
}

func TestSetRegionFeatureManifestSyncFailureStillSwitchesRegion(t *testing.T) {
	manifestPath := filepath.Join(t.TempDir(), "server-feature.json")
	writeServerFeatureManifest(t, manifestPath, validDisabledFeatureManifest)
	before, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}

	syncCalls := 0
	fake := &fakeApplicationRegionStore{region: proto.ApplicationRegionGlobal}
	restarts := 0
	service := &Service{
		regionStore:    fake,
		restartKVMComm: func() error { restarts++; return nil },
		syncFeatureManifest: func(_ context.Context, _ proto.ApplicationRegion) error {
			syncCalls++
			return errors.New("manifest source unavailable")
		},
	}

	context, recorder := newRegionContext(`{"region":"cn"}`)
	service.SetRegion(context)

	response, data := decodeRegionResponse(t, recorder)
	if response.Code != 0 || data.Region != proto.ApplicationRegionCN {
		t.Fatalf("sync failure response = %+v / %+v", response, data)
	}
	if syncCalls != 1 {
		t.Fatalf("sync calls = %d, want 1", syncCalls)
	}
	if len(fake.saves) != 1 || fake.saves[0] != proto.ApplicationRegionCN {
		t.Fatalf("sync failure saves = %v, want [cn]", fake.saves)
	}
	if restarts != 1 {
		t.Fatalf("sync failure restarts = %d, want 1", restarts)
	}
	after, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("manifest fixture changed from %q to %q", before, after)
	}
}

func TestSetRegionGlobalSyncsSharedFeatureManifest(t *testing.T) {
	store := &fakeApplicationRegionStore{region: proto.ApplicationRegionCN}
	syncCalls := 0
	service := &Service{
		regionStore:    store,
		restartKVMComm: func() error { return nil },
		syncFeatureManifest: func(_ context.Context, _ proto.ApplicationRegion) error {
			syncCalls++
			return nil
		},
	}

	context, recorder := newRegionContext(`{"region":"global"}`)
	service.SetRegion(context)

	response, data := decodeRegionResponse(t, recorder)
	if response.Code != 0 || data.Region != proto.ApplicationRegionGlobal {
		t.Fatalf("global switch response = %+v / %+v", response, data)
	}
	if syncCalls != 1 {
		t.Fatalf("global switch sync calls = %d, want 1", syncCalls)
	}
}

func TestSetRegionSameValueDoesNotSyncFeatureManifest(t *testing.T) {
	store := &fakeApplicationRegionStore{region: proto.ApplicationRegionCN}
	syncCalls := 0
	service := &Service{
		regionStore:    store,
		restartKVMComm: func() error { return nil },
		syncFeatureManifest: func(_ context.Context, _ proto.ApplicationRegion) error {
			syncCalls++
			return nil
		},
	}

	context, recorder := newRegionContext(`{"region":"cn"}`)
	service.SetRegion(context)

	response, data := decodeRegionResponse(t, recorder)
	if response.Code != 0 || data.Region != proto.ApplicationRegionCN {
		t.Fatalf("same-value response = %+v / %+v", response, data)
	}
	if syncCalls != 0 {
		t.Fatalf("same-value sync calls = %d, want 0", syncCalls)
	}
}

func TestSetRegionWaitsForFeatureManifestSyncBeforeSaving(t *testing.T) {
	store := &fakeApplicationRegionStore{
		region:      proto.ApplicationRegionGlobal,
		saveEntered: make(chan struct{}),
	}
	syncStarted := make(chan struct{})
	releaseSync := make(chan struct{})
	service := &Service{
		regionStore:    store,
		restartKVMComm: func() error { return nil },
		syncFeatureManifest: func(_ context.Context, _ proto.ApplicationRegion) error {
			close(syncStarted)
			<-releaseSync
			return nil
		},
	}

	context, recorder := newRegionContext(`{"region":"cn"}`)
	done := make(chan struct{})
	go func() {
		service.SetRegion(context)
		close(done)
	}()

	select {
	case <-syncStarted:
	case <-time.After(time.Second):
		t.Fatal("feature manifest sync was not invoked")
	}
	select {
	case <-store.saveEntered:
		t.Fatal("region save ran before the feature manifest sync returned")
	case <-time.After(50 * time.Millisecond):
	}

	close(releaseSync)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("SetRegion did not finish after the sync was released")
	}

	response, data := decodeRegionResponse(t, recorder)
	if response.Code != 0 || data.Region != proto.ApplicationRegionCN {
		t.Fatalf("blocking sync response = %+v / %+v", response, data)
	}
	if got := store.load(); got != proto.ApplicationRegionCN {
		t.Fatalf("stored region = %q, want cn", got)
	}
}

func TestSyncRegionFeatureManifestFallsBackToPackageSync(t *testing.T) {
	server := newFeatureManifestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/config/cn/features.json" {
			t.Errorf("unexpected manifest path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(validEnabledFeatureManifest))
	})
	manifestPath := filepath.Join(t.TempDir(), "server-feature.json")
	useTestServerFeatureStore(t, manifestPath)
	useTestServerFeatureSource(t, server.URL, server.Client())
	service := &Service{}
	if err := service.syncRegionFeatureManifest(context.Background(), proto.ApplicationRegionGlobal); err != nil {
		t.Fatalf("global fallback error = %v", err)
	}
	assertManifestBytes(t, manifestPath, validEnabledFeatureManifest)
}

// TestSetRegionSkipsManifestSyncWhenToggleDisabled proves the region-change
// trigger inherits the pull toggle through the shared choke point: with
// serverFeatureSync.enabled false and a local manifest present, the region
// switch still completes (marker saved, kvmcomm restarted) but no CDN request
// is issued and the local manifest stays byte-identical. syncFeatureManifest is
// deliberately left nil so the change reaches the production sync, which the
// toggle then gates.
func TestSetRegionSkipsManifestSyncWhenToggleDisabled(t *testing.T) {
	counter := &featureManifestRequestCounter{}
	server := newFeatureManifestServer(t, counter.handler(validEnabledFeatureManifest))
	directory := t.TempDir()
	path := filepath.Join(directory, "server-feature.json")
	writeServerFeatureManifest(t, path, validDisabledFeatureManifest)
	useTestServerFeatureStore(t, path)
	useTestServerFeatureSource(t, server.URL, server.Client())
	useServerFeatureSyncEnabled(t, false)
	before := manifestInode(t, path)

	store := &fakeApplicationRegionStore{region: proto.ApplicationRegionGlobal}
	restarts := 0
	service := &Service{
		regionStore:    store,
		restartKVMComm: func() error { restarts++; return nil },
	}

	context, recorder := newRegionContext(`{"region":"cn"}`)
	service.SetRegion(context)

	response, data := decodeRegionResponse(t, recorder)
	if response.Code != 0 || data.Region != proto.ApplicationRegionCN {
		t.Fatalf("region change response = %+v / %+v, want success on cn", response, data)
	}
	if count := counter.count(); count != 0 {
		t.Fatalf("disabled sync issued %d CDN requests, want 0", count)
	}
	assertManifestBytes(t, path, validDisabledFeatureManifest)
	if after := manifestInode(t, path); after != before {
		t.Fatalf("manifest inode changed from %d to %d on a skipped sync", before, after)
	}
	if got := store.load(); got != proto.ApplicationRegionCN {
		t.Fatalf("stored region = %q, want cn", got)
	}
	if restarts != 1 {
		t.Fatalf("kvmcomm restarts = %d, want 1", restarts)
	}
}
