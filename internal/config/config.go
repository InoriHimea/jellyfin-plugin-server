package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

type ProxyType string

const (
	ProxyNone   ProxyType = ""
	ProxyHTTP   ProxyType = "http"
	ProxyHTTPS  ProxyType = "https"
	ProxySOCKS5 ProxyType = "socks5"
)

type ProxyConfig struct {
	Type     ProxyType `json:"type"`
	Address  string    `json:"address"`
	Username string    `json:"username"`
	Password string    `json:"password"`
	NoProxy  string    `json:"no_proxy"`
}

type StorageConfig struct {
	DataDir         string `json:"data_dir"`
	MaxDiskMB       int64  `json:"max_disk_mb"`
	KeepVersions    int    `json:"keep_versions"`
	CleanupSchedule string `json:"cleanup_schedule"`
}

type CacheConfig struct {
	ManifestTTLSeconds int `json:"manifest_ttl_seconds"`
	MaxConcurrentDL    int `json:"max_concurrent_downloads"`
}

type AuthConfig struct {
	Enabled  bool   `json:"enabled"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type ServerConfig struct {
	Host      string `json:"host"`
	Port      int    `json:"port"`
	PublicURL string `json:"public_url"` // Override for manifest sourceUrl base (e.g. https://example.com:9443)
}

// CompatConfig records the Jellyfin server version this proxy's admin panel
// should check plugin versions against. Jellyfin's catalog page lets you
// install any listed version whose targetAbi merely parses, but the actual
// "Not Supported" verdict is decided later at plugin load time by comparing
// the running server's version against the plugin's targetAbi — a stricter,
// separate check. We have no way to detect which Jellyfin instance(s) are
// polling this server, so this is manually configured, purely to flag
// mismatches in our own catalog/packages UI before the user installs
// something that will fail to load.
type CompatConfig struct {
	JellyfinVersion string `json:"jellyfin_version"`
}

type Config struct {
	Server  ServerConfig  `json:"server"`
	Storage StorageConfig `json:"storage"`
	Cache   CacheConfig   `json:"cache"`
	Proxy   ProxyConfig   `json:"proxy"`
	Auth    AuthConfig    `json:"auth"`
	Compat  CompatConfig  `json:"compat"`
	LogJSON bool          `json:"log_json"`
}

var (
	mu       sync.RWMutex
	current  *Config
	cfgPath  string
)

func Defaults() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		Storage: StorageConfig{
			DataDir:         "./data",
			MaxDiskMB:       10240,
			KeepVersions:    3,
			CleanupSchedule: "0 3 * * *",
		},
		Cache: CacheConfig{
			ManifestTTLSeconds: 86400,
			MaxConcurrentDL:    4,
		},
		Proxy:   ProxyConfig{},
		Auth:    AuthConfig{Enabled: false},
		Compat:  CompatConfig{},
		LogJSON: false,
	}
}

func Load(path string) (*Config, error) {
	cfgPath = path
	cfg := Defaults()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			applyEnv(cfg)
			mu.Lock()
			current = cfg
			mu.Unlock()
			return cfg, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	applyEnv(cfg)
	mu.Lock()
	current = cfg
	mu.Unlock()
	return cfg, nil
}

// applyEnv overrides config fields with environment variables.
func applyEnv(cfg *Config) {
	if v := os.Getenv("JPSERVER_PUBLIC_URL"); v != "" {
		cfg.Server.PublicURL = v
	}
	if v := os.Getenv("JPSERVER_HOST"); v != "" {
		cfg.Server.Host = v
	}
	if v := os.Getenv("JPSERVER_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = p
		}
	}
	if v := os.Getenv("JPSERVER_DATA_DIR"); v != "" {
		cfg.Storage.DataDir = v
	}
	if v := os.Getenv("JPSERVER_LOG_JSON"); v == "true" || v == "1" {
		cfg.LogJSON = true
	}
	if v := os.Getenv("JPSERVER_MAX_DISK_MB"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			cfg.Storage.MaxDiskMB = n
		}
	}
	if v := os.Getenv("JPSERVER_MAX_CONCURRENT_DL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Cache.MaxConcurrentDL = n
		}
	}
}

func Get() *Config {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

func Update(cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(cfgPath), 0755); err != nil {
		return err
	}

	if err := os.WriteFile(cfgPath, data, 0644); err != nil {
		return err
	}

	mu.Lock()
	current = cfg
	mu.Unlock()
	return nil
}

func PackagesDir() string {
	mu.RLock()
	defer mu.RUnlock()
	return filepath.Join(current.Storage.DataDir, "packages")
}

// ImagesDir is the on-disk cache for proxied plugin images.
func ImagesDir() string {
	mu.RLock()
	defer mu.RUnlock()
	return filepath.Join(current.Storage.DataDir, "images")
}

func DBPath() string {
	mu.RLock()
	defer mu.RUnlock()
	return filepath.Join(current.Storage.DataDir, "jellyfin.db")
}
