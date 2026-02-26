package routing

import "sync"

var (
	globalSmartRouter     *SmartRouter
	globalSmartRouterOnce sync.Once
)

func GetGlobalSmartRouter() *SmartRouter {
	globalSmartRouterOnce.Do(func() {
		globalSmartRouter = NewSmartRouter()
	})
	return globalSmartRouter
}
