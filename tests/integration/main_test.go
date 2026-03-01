package integration

import (
	"sync"

	"ai-gateway/internal/provider"

	"github.com/gin-gonic/gin"
)

type Provider = provider.Provider

var setGinTestModeOnce sync.Once

func setGinTestMode() {
	setGinTestModeOnce.Do(func() {
		gin.SetMode(gin.TestMode)
	})
}
