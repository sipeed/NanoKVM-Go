package inputrouter

import "sort"

type RouteStatus struct {
	Kind         string `json:"kind"`
	Route        string `json:"route"`
	Dispatched   uint64 `json:"dispatched"`
	Failed       uint64 `json:"failed"`
	WriteRetries uint64 `json:"write_retries"`
	Dropped      uint64 `json:"backpressure_dropped"`
	Owner        uint64 `json:"owner_session,omitempty"`
}

type SourceStatus struct {
	Source string `json:"source"`
	Events uint64 `json:"events"`
}

type SessionStatus struct {
	ID              uint64         `json:"id"`
	Events          uint64         `json:"events"`
	FirstActivityMS int64          `json:"first_activity_ms"`
	LastActivityMS  int64          `json:"last_activity_ms"`
	LastSourceSeq   uint32         `json:"last_source_sequence"`
	Sources         []SourceStatus `json:"sources"`
	Routes          []string       `json:"routes"`
}

type RuntimeStatus struct {
	Received                uint64          `json:"received"`
	Dispatched              uint64          `json:"dispatched"`
	Rejected                uint64          `json:"rejected"`
	DecodeRejected          uint64          `json:"decode_rejected"`
	DispatchFailed          uint64          `json:"dispatch_failed"`
	BackpressureDropped     uint64          `json:"backpressure_dropped"`
	BrowserCaptured         uint64          `json:"browser_captured"`
	BrowserCaptureRejected  uint64          `json:"browser_capture_rejected"`
	LastActivityMS          int64           `json:"last_activity_ms,omitempty"`
	LastError               string          `json:"last_error,omitempty"`
	DroppedDebugRecords     uint64          `json:"dropped_debug_records"`
	DroppedStreamRecords    uint64          `json:"dropped_stream_records"`
	DeliveredDebugRecords   uint64          `json:"delivered_debug_records"`
	UnavailableDebugRecords uint64          `json:"unavailable_debug_records"`
	DebugConnected          bool            `json:"debug_connected"`
	DebugConnections        uint64          `json:"debug_connections"`
	Quiescing               bool            `json:"quiescing"`
	InputEpoch              uint64          `json:"input_epoch"`
	ActiveSessions          uint64          `json:"active_sessions"`
	Sources                 []SourceStatus  `json:"sources"`
	Sessions                []SessionStatus `json:"sessions"`
	Routes                  []RouteStatus   `json:"routes"`
}

type sessionRuntime struct {
	events          uint64
	firstActivityMS int64
	lastActivityMS  int64
	lastSourceSeq   uint32
	sources         map[Source]uint64
	routes          map[string]struct{}
}

func (router *Router) recordSessionEvent(
	session SessionID,
	event Event,
	route string,
	timestamp int64,
) {
	runtime := router.sessions[session]
	if runtime == nil {
		runtime = &sessionRuntime{
			firstActivityMS: timestamp,
			sources:         make(map[Source]uint64),
			routes:          make(map[string]struct{}),
		}
		router.sessions[session] = runtime
	}
	runtime.events++
	runtime.lastActivityMS = timestamp
	runtime.lastSourceSeq = event.Sequence
	runtime.sources[event.Source]++
	runtime.routes[route] = struct{}{}
	router.sources[event.Source]++
}

func (router *Router) Status() RuntimeStatus {
	router.dispatch.Lock()
	inputEpoch := router.routingEpoch.Load()
	owners := make(map[Kind]SessionID, len(router.owners))
	for kind, owner := range router.owners {
		owners[kind] = owner
	}
	sources := sourceStatuses(router.sources)
	sessions := make([]SessionStatus, 0, len(router.sessions))
	for id, runtime := range router.sessions {
		routes := make([]string, 0, len(runtime.routes))
		for route := range runtime.routes {
			routes = append(routes, route)
		}
		sort.Strings(routes)
		sessions = append(sessions, SessionStatus{
			ID:              uint64(id),
			Events:          runtime.events,
			FirstActivityMS: runtime.firstActivityMS,
			LastActivityMS:  runtime.lastActivityMS,
			LastSourceSeq:   runtime.lastSourceSeq,
			Sources:         sourceStatuses(runtime.sources),
			Routes:          routes,
		})
	}
	router.dispatch.Unlock()
	sort.Slice(sessions, func(left, right int) bool {
		return sessions[left].ID < sessions[right].ID
	})

	router.routeMutex.RLock()
	routes := make([]RouteStatus, 0, len(router.routes))
	for kind, route := range router.routes {
		routes = append(routes, RouteStatus{
			Kind:         kindName(kind),
			Route:        route.Name,
			Dispatched:   route.written.Load(),
			Failed:       route.failed.Load(),
			WriteRetries: route.retried.Load(),
			Dropped:      route.dropped.Load(),
			Owner:        uint64(owners[kind]),
		})
	}
	router.routeMutex.RUnlock()
	sort.Slice(routes, func(left, right int) bool {
		return routes[left].Route < routes[right].Route
	})

	router.errorMutex.RLock()
	lastError := router.lastError
	router.errorMutex.RUnlock()
	return RuntimeStatus{
		Received:                router.received.Load(),
		Dispatched:              router.dispatched.Load(),
		Rejected:                router.rejected.Load(),
		DecodeRejected:          router.decodeRejected.Load(),
		DispatchFailed:          router.dispatchFailed.Load(),
		BackpressureDropped:     router.backpressureDropped.Load(),
		BrowserCaptured:         router.browserCaptured.Load(),
		BrowserCaptureRejected:  router.browserCaptureRejected.Load(),
		LastActivityMS:          router.lastActive.Load(),
		LastError:               lastError,
		DroppedDebugRecords:     router.traces.DroppedDebugRecords(),
		DroppedStreamRecords:    router.traces.DroppedStreamRecords(),
		DeliveredDebugRecords:   router.traces.DeliveredDebugRecords(),
		UnavailableDebugRecords: router.traces.UnavailableDebugRecords(),
		DebugConnected:          router.traces.DebugConnected(),
		DebugConnections:        router.traces.DebugConnections(),
		Quiescing:               inputEpochQuiescing(inputEpoch),
		InputEpoch:              inputEpoch,
		ActiveSessions:          uint64(len(sessions)),
		Sources:                 sources,
		Sessions:                sessions,
		Routes:                  routes,
	}
}

func sourceStatuses(counts map[Source]uint64) []SourceStatus {
	statuses := make([]SourceStatus, 0, len(counts))
	for source, events := range counts {
		statuses = append(statuses, SourceStatus{Source: sourceName(source), Events: events})
	}
	sort.Slice(statuses, func(left, right int) bool {
		return statuses[left].Source < statuses[right].Source
	})
	return statuses
}
