package plugin

import (
	"errors"
	"os"
	"path/filepath"
	"plugin"
	"sync"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/pkg/log"
	"github.com/chihaya/chihaya/pkg/stop"
	"github.com/chihaya/chihaya/storage"
)

var (
	peerStorageManager = &peerStoreManager{
		peerStore: nil,
	}
)

type peerStoreManager struct {
	sync.RWMutex
	peerStore storage.PeerStore
}

type peerStore struct {
	peerStoreManager *peerStoreManager
	storageDir       string
	closing          chan struct{}
}

// AnnouncePeers implements storage.PeerStore.
func (*peerStore) AnnouncePeers(infoHash bittorrent.InfoHash, seeder bool, numWant int, p bittorrent.Peer) (peers []bittorrent.Peer, err error) {
	return peerStorageManager.peerStore.AnnouncePeers(infoHash, seeder, numWant, p)
}

// DeleteLeecher implements storage.PeerStore.
func (*peerStore) DeleteLeecher(infoHash bittorrent.InfoHash, p bittorrent.Peer) error {
	return peerStorageManager.peerStore.DeleteLeecher(infoHash, p)
}

// DeleteSeeder implements storage.PeerStore.
func (*peerStore) DeleteSeeder(infoHash bittorrent.InfoHash, p bittorrent.Peer) error {
	return peerStorageManager.peerStore.DeleteSeeder(infoHash, p)
}

// GraduateLeecher implements storage.PeerStore.
func (*peerStore) GraduateLeecher(infoHash bittorrent.InfoHash, p bittorrent.Peer) error {
	return peerStorageManager.peerStore.GraduateLeecher(infoHash, p)
}

// LogFields implements storage.PeerStore.
func (p *peerStore) LogFields() log.Fields {
	return p.peerStoreManager.peerStore.LogFields()
}

// PutLeecher implements storage.PeerStore.
func (*peerStore) PutLeecher(infoHash bittorrent.InfoHash, p bittorrent.Peer) error {
	return peerStorageManager.peerStore.PutLeecher(infoHash, p)
}

// PutSeeder implements storage.PeerStore.
func (*peerStore) PutSeeder(infoHash bittorrent.InfoHash, p bittorrent.Peer) error {
	return peerStorageManager.peerStore.PutSeeder(infoHash, p)
}

// ScrapeSwarm implements storage.PeerStore.
func (p *peerStore) ScrapeSwarm(infoHash bittorrent.InfoHash, addressFamily bittorrent.AddressFamily) bittorrent.Scrape {
	return p.peerStoreManager.peerStore.ScrapeSwarm(infoHash, addressFamily)
}

// Stop implements storage.PeerStore.
func (p *peerStore) Stop() stop.Result {
	return p.peerStoreManager.peerStore.Stop()
}

func NewStoragePeerStore(pluginDir string) (storage.PeerStore, error) {
	log.Debug("loading storage peer store plugin from %s", log.Fields{"storage_plugin_path": pluginDir})
	sp := &peerStore{
		peerStoreManager: peerStorageManager,
		storageDir:       pluginDir,
		closing:          make(chan struct{}),
	}
	if _, err := os.Stat(pluginDir); os.IsNotExist(err) {
		return nil, errors.New("middleware plugin path does not exist: " + pluginDir)
	}

	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		return nil, err
	}

	peerStorageManager.Lock()
	defer peerStorageManager.Unlock()
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginSubDir := filepath.Join(pluginDir, entry.Name())
		if _, ps, err := loadStorePeerPluginFromSubdir(pluginSubDir); err == nil {
			peerStorageManager.peerStore = ps
		}
	}

	return sp, nil
}

func loadStorePeerPluginFromSubdir(pluginDir string) (string, storage.PeerStore, error) {
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

	pluginInstance, ok := sys.(PeerStorePlugin)
	if !ok {
		return "", nil, errors.New("plugin symbol " + ExportedSymbol + " is not of type Plugin")
	}

	ps, err := pluginInstance.NewPeerStore(optionBytes)
	if err != nil {
		log.Warn("failed to create peer store from plugin: ", pluginInstance)
		return "", nil, err
	}

	return pluginInstance.Name, ps, nil
}
