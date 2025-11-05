package main

import (
	"context"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/middleware"
	"github.com/chihaya/chihaya/pkg/log"
	"github.com/chihaya/chihaya/plugin"
	"gopkg.in/yaml.v2"
)

type Config struct {
	LogLevel string `yaml:"log_level"`
}

type logHook struct {
	cfg Config
}

// HandleAnnounce implements middleware.Hook.
func (h *logHook) HandleAnnounce(ctx context.Context, req *bittorrent.AnnounceRequest, resp *bittorrent.AnnounceResponse) (context.Context, error) {
	log.Info("Handle Announce: ", req, resp)
	return ctx, nil
}

// HandleScrape implements middleware.Hook.
func (h *logHook) HandleScrape(ctx context.Context, req *bittorrent.ScrapeRequest, resp *bittorrent.ScrapeResponse) (context.Context, error) {
	log.Info("Handle Scrape: ", req, resp)
	return ctx, nil
}

type logPlugin struct{}

func (p *logPlugin) NewHook(optionBytes []byte) (middleware.Hook, error) {
	var cfg Config
	if err := yaml.Unmarshal(optionBytes, &cfg); err != nil {
		return nil, err
	}
	return &logHook{cfg: cfg}, nil
}

var Plugin = plugin.HookPlugin{
	Driver: &logPlugin{},
	Name:   "logmiddleware",
}
