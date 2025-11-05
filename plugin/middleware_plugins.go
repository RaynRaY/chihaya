package plugin

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"plugin"
	"sync"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/middleware"
	"github.com/chihaya/chihaya/pkg/log"
)

var (
	pluginManager = &middlewareManager{
		middlewareHook: make(map[string]middleware.Hook),
	}
)

type middlewareManager struct {
	sync.RWMutex
	middlewareHook map[string]middleware.Hook
}

type hook struct {
	pluginManager *middlewareManager
	pluginDir     string
	closing       chan struct{}
}

// HandleAnnounce implements middleware.Hook.
func (h *hook) HandleAnnounce(ctx context.Context, req *bittorrent.AnnounceRequest, resp *bittorrent.AnnounceResponse) (context.Context, error) {
	for _, v := range h.pluginManager.middlewareHook {
		if newCcontext, err := v.HandleAnnounce(ctx, req, resp); err == nil {
			ctx = newCcontext
		}
	}
	return ctx, nil
}

// HandleScrape implements middleware.Hook.
func (h *hook) HandleScrape(ctx context.Context, req *bittorrent.ScrapeRequest, resp *bittorrent.ScrapeResponse) (context.Context, error) {
	for _, v := range h.pluginManager.middlewareHook {
		if newCcontext, err := v.HandleScrape(ctx, req, resp); err == nil {
			ctx = newCcontext
		}
	}
	return ctx, nil

}

func NewMiddlewareHook(pluginDir string) (middleware.Hook, error) {
	log.Debug("loading middleware hook plugin", log.Fields{"middleware_plugin_path": pluginDir})
	h := &hook{
		pluginManager: pluginManager,
		pluginDir:     pluginDir,
		closing:       make(chan struct{}),
	}
	if _, err := os.Stat(pluginDir); os.IsNotExist(err) {
		return nil, errors.New("middleware plugin path does not exist: " + pluginDir)
	}

	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		return nil, err
	}

	pluginManager.Lock()
	defer pluginManager.Unlock()
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginSubDir := filepath.Join(pluginDir, entry.Name())
		if name, hook, err := loadPluginFromSubdir(pluginSubDir); err == nil {
			pluginManager.middlewareHook[name] = hook
		}
	}

	return h, nil
}

func loadPluginFromSubdir(pluginDir string) (string, middleware.Hook, error) {
	// 固定使用 plugin.so 作为插件文件名
	soPath := filepath.Join(pluginDir, "plugin.so")
	if _, err := os.Stat(soPath); os.IsNotExist(err) {
		return "", nil, errors.New("plugin.so not found in plugin dir " + pluginDir)
	}

	// 固定使用 config.yaml 作为配置文件名并读取内容
	configPath := filepath.Join(pluginDir, "config.yaml")
	var optionBytes []byte
	if _, err := os.Stat(configPath); err == nil {
		optionBytes, err = os.ReadFile(configPath)
		if err != nil {
			return "", nil, err
		}
	}

	p, err := plugin.Open(soPath)
	if err != nil {
		return "", nil, err
	}

	sys, err := p.Lookup(ExportedSymbol)
	if err != nil {
		return "", nil, err
	}

	pluginInstance, ok := sys.(HookPlugin)
	if !ok {
		return "", nil, errors.New("plugin symbol " + ExportedSymbol + " is not of type Plugin")
	}

	hook, err := pluginInstance.NewHook(optionBytes)
	if err != nil {
		log.Warn("failed to create hook from plugin: ", pluginInstance)
		return "", nil, err
	}

	return pluginInstance.Name, hook, nil
}
