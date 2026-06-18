package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"paste/backend/pkg/models"
)

type Manager struct {
	config  *models.AppConfig
	path    string
	mu      sync.RWMutex
}

var defaultConfig = &models.AppConfig{
	Hotkey:          "Command+Shift+V",
	AutoStart:       false,
	Theme:           "system",
	AutoPaste:       true,
	MaxHistory:      5000,
	EnableSensitive: true,
	SensitiveApps:   []string{},
	BlacklistPatterns: []string{},
	IgnoredApps:     []string{},
}

func NewManager(dataDir string) (*Manager, error) {
	configPath := filepath.Join(dataDir, "config.json")

	m := &Manager{
		config: &models.AppConfig{},
		path:   configPath,
	}

	if err := m.load(); err != nil {
		if os.IsNotExist(err) {
			m.config = defaultConfig
			if err := m.save(); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	return m, nil
}

func (m *Manager) load() error {
	data, err := os.ReadFile(m.path)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	*m.config = *defaultConfig
	return json.Unmarshal(data, m.config)
}

func (m *Manager) save() error {
	dir := filepath.Dir(m.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.path, data, 0644)
}

func (m *Manager) Get() *models.AppConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cfg := *m.config
	cfg.SensitiveApps = append([]string{}, m.config.SensitiveApps...)
	cfg.BlacklistPatterns = append([]string{}, m.config.BlacklistPatterns...)
	cfg.IgnoredApps = append([]string{}, m.config.IgnoredApps...)
	return &cfg
}

func (m *Manager) Update(newConfig *models.AppConfig) error {
	m.mu.Lock()
	m.config = newConfig
	m.mu.Unlock()
	return m.save()
}

func (m *Manager) Path() string {
	return m.path
}
