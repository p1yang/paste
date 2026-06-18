package security

import (
	"regexp"
	"strings"
	"sync"

	"paste/backend/pkg/models"
)

type Manager struct {
	sensitiveApps     map[string]bool
	blacklistPatterns []*regexp.Regexp
	enableSensitive   bool
	mu                sync.RWMutex
}

var defaultSensitiveApps = []string{
	"1Password",
	"1Password 7",
	"Bitwarden",
	"LastPass",
	"KeePassXC",
	"KeePass",
	"Dashlane",
	"Enpass",
	"RoboForm",
	"NordPass",
	"Keychain Access",
	"Authy",
	"Google Authenticator",
	"Microsoft Authenticator",
	"RSA SecurID",
	"Duo Mobile",
	"招商银行",
	"中国工商银行",
	"中国建设银行",
	"中国农业银行",
	"中国银行",
	"交通银行",
	"平安银行",
	"浦发银行",
	"中信银行",
	"兴业银行",
	"光大银行",
	"民生银行",
	"支付宝",
	"微信支付",
	"PayPal",
	"Chrome",
	"Safari",
	"Firefox",
	"Edge",
}

var defaultBlacklistPatterns = []string{
	`(?i)password\s*[:=]\s*\S+`,
	`(?i)passwd\s*[:=]\s*\S+`,
	`(?i)pwd\s*[:=]\s*\S+`,
	`(?i)secret\s*[:=]\s*\S+`,
	`(?i)token\s*[:=]\s*\S+`,
	`(?i)api[_-]?key\s*[:=]\s*\S+`,
	`(?i)access[_-]?key\s*[:=]\s*\S+`,
	`(?i)private[_-]?key\s*[:=]\s*\S+`,
	`(?i)aws[_-]?(access|secret)[_-]?key`,
	`(?i)sk-[A-Za-z0-9_-]{20,}`,
	`(?i)pk-[A-Za-z0-9_-]{20,}`,
	`(?i)xox[baprs]-[A-Za-z0-9-]{10,}`,
	`^[A-Za-z0-9+/]{40,}={0,2}$`,
	`^\d{16,19}$`,
	`^\d{3,4}$`,
}

func NewManager(config *models.AppConfig) *Manager {
	m := &Manager{
		sensitiveApps:     make(map[string]bool),
		blacklistPatterns: make([]*regexp.Regexp, 0),
		enableSensitive:   config.EnableSensitive,
	}

	apps := append(defaultSensitiveApps, config.SensitiveApps...)
	apps = append(apps, config.IgnoredApps...)
	for _, app := range apps {
		if app != "" {
			m.sensitiveApps[strings.ToLower(strings.TrimSpace(app))] = true
		}
	}

	patterns := append(defaultBlacklistPatterns, config.BlacklistPatterns...)
	for _, p := range patterns {
		if p == "" {
			continue
		}
		if compiled, err := regexp.Compile(p); err == nil {
			m.blacklistPatterns = append(m.blacklistPatterns, compiled)
		}
	}

	return m
}

func (m *Manager) UpdateConfig(config *models.AppConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sensitiveApps = make(map[string]bool)
	m.enableSensitive = config.EnableSensitive

	apps := append(defaultSensitiveApps, config.SensitiveApps...)
	apps = append(apps, config.IgnoredApps...)
	for _, app := range apps {
		if app != "" {
			m.sensitiveApps[strings.ToLower(strings.TrimSpace(app))] = true
		}
	}

	m.blacklistPatterns = make([]*regexp.Regexp, 0)
	patterns := append(defaultBlacklistPatterns, config.BlacklistPatterns...)
	for _, p := range patterns {
		if p == "" {
			continue
		}
		if compiled, err := regexp.Compile(p); err == nil {
			m.blacklistPatterns = append(m.blacklistPatterns, compiled)
		}
	}
}

func (m *Manager) IsSensitiveApp(appName string) bool {
	if !m.enableSensitive {
		return false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.sensitiveApps[strings.ToLower(strings.TrimSpace(appName))]
}

func (m *Manager) IsBlacklistedContent(content string) bool {
	if !m.enableSensitive {
		return false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, pattern := range m.blacklistPatterns {
		if pattern.MatchString(content) {
			return true
		}
	}
	return false
}

func (m *Manager) ShouldIgnore(appName string, content string) bool {
	if m.IsSensitiveApp(appName) {
		return true
	}
	if m.IsBlacklistedContent(content) {
		return true
	}
	return false
}

func (m *Manager) AddSensitiveApp(appName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sensitiveApps[strings.ToLower(strings.TrimSpace(appName))] = true
}

func (m *Manager) RemoveSensitiveApp(appName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sensitiveApps, strings.ToLower(strings.TrimSpace(appName)))
}

func (m *Manager) GetSensitiveApps() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	apps := make([]string, 0, len(m.sensitiveApps))
	for app := range m.sensitiveApps {
		apps = append(apps, app)
	}
	return apps
}

func (m *Manager) AddBlacklistPattern(pattern string) error {
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.blacklistPatterns = append(m.blacklistPatterns, compiled)
	return nil
}
