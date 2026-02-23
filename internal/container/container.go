package container

import (
	"ai-gateway/internal/cache"
	"ai-gateway/internal/config"
	"ai-gateway/internal/provider"
	"ai-gateway/internal/routing"
	"ai-gateway/internal/storage"
	"ai-gateway/pkg/security"
	"sync"

	"github.com/sirupsen/logrus"
)

var containerLogger = logrus.New()

type ServiceContainer struct {
	mu sync.RWMutex

	config       *config.Config
	registry     *provider.Registry
	cacheManager *cache.Manager
	smartRouter  *routing.SmartRouter
	storage      *storage.MemoryStorage
	security     *security.SecurityConfig

	initialized bool
}

var (
	globalContainer     *ServiceContainer
	globalContainerOnce sync.Once
)

func GetContainer() *ServiceContainer {
	globalContainerOnce.Do(func() {
		globalContainer = &ServiceContainer{}
	})
	return globalContainer
}

func (c *ServiceContainer) Initialize(cfg *config.Config) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.initialized {
		return nil
	}

	c.config = cfg

	c.security = security.GetSecurityConfig()
	if err := c.security.Validate(); err != nil {
		containerLogger.WithError(err).Warn("Security validation warning")
	}

	c.registry = provider.GetRegistry()

	c.smartRouter = routing.NewSmartRouter()

	cacheMgr, err := cache.NewManager(cache.DefaultManagerConfig())
	if err != nil {
		containerLogger.WithError(err).Warn("Failed to create cache manager, using default")
		cacheMgr = cache.NewManagerWithCache(cache.NewMemoryCache())
	}
	c.cacheManager = cacheMgr

	c.storage = storage.GetSQLite()

	c.initialized = true
	containerLogger.Info("Service container initialized")

	return nil
}

func (c *ServiceContainer) Config() *config.Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config
}

func (c *ServiceContainer) Registry() *provider.Registry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.registry
}

func (c *ServiceContainer) CacheManager() *cache.Manager {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cacheManager
}

func (c *ServiceContainer) SmartRouter() *routing.SmartRouter {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.smartRouter
}

func (c *ServiceContainer) Storage() *storage.MemoryStorage {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.storage
}

func (c *ServiceContainer) Security() *security.SecurityConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.security
}

func (c *ServiceContainer) IsInitialized() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.initialized
}

func (c *ServiceContainer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.storage != nil {
		c.storage.Close()
	}

	return nil
}

func (c *ServiceContainer) GetService(name string) interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	switch name {
	case "config":
		return c.config
	case "registry":
		return c.registry
	case "cacheManager":
		return c.cacheManager
	case "smartRouter":
		return c.smartRouter
	case "storage":
		return c.storage
	case "security":
		return c.security
	default:
		return nil
	}
}
