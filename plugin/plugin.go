package plugin

import (
	"fmt"

	"github.com/chihaya/chihaya/middleware"
	"github.com/chihaya/chihaya/pkg/log"
	"github.com/chihaya/chihaya/storage"
	yaml "gopkg.in/yaml.v2"
)

const Name = "plugin"

func init() {
	storage.RegisterDriver(Name, driver{})
	middleware.RegisterDriver(Name, driver{})
}

type driver struct{}

// NewPeerStore implements storage.Driver.
func (d driver) NewPeerStore(icfg interface{}) (storage.PeerStore, error) {
	bytes, err := yaml.Marshal(icfg)
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = yaml.Unmarshal(bytes, &cfg)
	if err != nil {
		return nil, err
	}
	return NewStoragePeerStore(cfg.StoragePluginPath)
}

// NewHook implements middleware.Driver.
func (d driver) NewHook(options []byte) (middleware.Hook, error) {
	var cfg Config
	if err := yaml.Unmarshal(options, &cfg); err != nil {
		return nil, fmt.Errorf("invalid options for middleware %s: %w", Name, err)
	}
	return NewMiddlewareHook(cfg.MiddlewarePluginPath)
}

type Config struct {
	MiddlewarePluginPath string `yaml:"middleware_plugin_path"`
	StoragePluginPath    string `yaml:"storage_plugin_path"`
}

// LogFields implements log.Fielder.
func (c Config) LogFields() log.Fields {
	return log.Fields{
		"middleware_plugin_path": c.MiddlewarePluginPath,
		"storage_plugin_path":    c.StoragePluginPath,
	}
}

const ExportedSymbol = "Plugin"

type HookPlugin struct {
	middleware.Driver
	Name string
}

func (p HookPlugin) LogFields() log.Fields {
	return log.Fields{
		"name": p.Name,
	}
}

type PeerStorePlugin struct {
	storage.Driver
	Name string
}

func (p PeerStorePlugin) LogFields() log.Fields {
	return log.Fields{
		"name": p.Name,
	}
}
