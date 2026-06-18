package autostart

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const launchdPlistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>%s</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<false/>
</dict>
</plist>
`

type Manager struct {
	label     string
	plistPath string
	appPath   string
}

func NewManager(appName string) *Manager {
	homeDir, _ := os.UserHomeDir()
	launchAgentsDir := filepath.Join(homeDir, "Library", "LaunchAgents")
	label := fmt.Sprintf("com.%s.%s", strings.ToLower(appName), "launcher")

	return &Manager{
		label:     label,
		plistPath: filepath.Join(launchAgentsDir, label+".plist"),
		appPath:   getAppPath(),
	}
}

func getAppPath() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}

	parts := strings.Split(exe, "/")
	for i, part := range parts {
		if strings.HasSuffix(part, ".app") {
			return filepath.Join(parts[:i+1]...)
		}
	}
	return exe
}

func (m *Manager) IsEnabled() bool {
	_, err := os.Stat(m.plistPath)
	return err == nil
}

func (m *Manager) Enable() error {
	dir := filepath.Dir(m.plistPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create LaunchAgents directory: %w", err)
	}

	plistContent := fmt.Sprintf(launchdPlistTemplate, m.label, m.appPath)
	if err := os.WriteFile(m.plistPath, []byte(plistContent), 0644); err != nil {
		return fmt.Errorf("failed to write plist file: %w", err)
	}

	return nil
}

func (m *Manager) Disable() error {
	if _, err := os.Stat(m.plistPath); os.IsNotExist(err) {
		return nil
	}

	return os.Remove(m.plistPath)
}

func (m *Manager) Toggle(enable bool) error {
	if enable {
		return m.Enable()
	}
	return m.Disable()
}
