package inputrouter

import (
	"sync"

	"NanoKVM-Server/service/hid"
)

var (
	defaultRouter *Router
	routerOnce    sync.Once
)

func GetRouter() *Router {
	routerOnce.Do(func() {
		defaultRouter = newRouter(
			hid.GetHid(),
			NewDefaultManagedTraceHub(0),
			pointerModeEventGate(hid.GetPointerModeStatus),
		)
		hid.SetInputRouterSink(defaultRouter)
	})
	return defaultRouter
}
