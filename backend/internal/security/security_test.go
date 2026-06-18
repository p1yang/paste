package security

import (
	"testing"

	"paste/backend/pkg/models"
)

func setupTestManager() *Manager {
	config := &models.AppConfig{
		EnableSensitive:   true,
		SensitiveApps:     []string{},
		BlacklistPatterns: []string{},
		IgnoredApps:       []string{"MyCustomApp"},
	}
	return NewManager(config)
}

func TestIsSensitiveApp(t *testing.T) {
	m := setupTestManager()

	tests := []struct {
		name     string
		appName  string
		expected bool
	}{
		{"1Password", "1Password", true},
		{"Bitwarden", "Bitwarden", true},
		{"招商银行", "招商银行", true},
		{"Custom ignored app", "MyCustomApp", true},
		{"Normal app", "VS Code", false},
		{"Case insensitive", "1password", true},
		{"Whitespace trimmed", "  1Password  ", true},
		{"Empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := m.IsSensitiveApp(tt.appName)
			if result != tt.expected {
				t.Errorf("IsSensitiveApp(%q) = %v, want %v", tt.appName, result, tt.expected)
			}
		})
	}
}

func TestSensitiveProtectionDisabled(t *testing.T) {
	config := &models.AppConfig{
		EnableSensitive: false,
	}
	m := NewManager(config)

	if m.IsSensitiveApp("1Password") {
		t.Error("expected sensitive app detection to be disabled")
	}

	if m.IsBlacklistedContent("password=secret123") {
		t.Error("expected blacklist detection to be disabled")
	}
}

func TestIsBlacklistedContent(t *testing.T) {
	m := setupTestManager()

	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{"Password pattern", "password=mysecretpass", true},
		{"Password pattern case insensitive", "PASSWORD=test123", true},
		{"API key pattern", "api_key=sk-12345678901234567890", true},
		{"OpenAI API key", "sk-proj-abcdefghijklmnopqrstuvwxyz1234567890", true},
		{"Credit card number", "4111111111111111", true},
		{"CVV", "123", true},
		{"Normal text", "Hello, World!", false},
		{"Code snippet", "func main() { fmt.Println(\"hello\") }", false},
		{"URL", "https://example.com/path?query=value", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := m.IsBlacklistedContent(tt.content)
			if result != tt.expected {
				t.Errorf("IsBlacklistedContent(%q) = %v, want %v", tt.content, result, tt.expected)
			}
		})
	}
}

func TestShouldIgnore(t *testing.T) {
	m := setupTestManager()

	if !m.ShouldIgnore("1Password", "my password") {
		t.Error("should ignore content from 1Password")
	}

	if !m.ShouldIgnore("VS Code", "password=secret123") {
		t.Error("should ignore blacklisted content")
	}

	if m.ShouldIgnore("VS Code", "Normal content") {
		t.Error("should not ignore normal content from normal app")
	}
}

func TestAddAndRemoveSensitiveApp(t *testing.T) {
	m := setupTestManager()

	if m.IsSensitiveApp("MyNewApp") {
		t.Error("MyNewApp should not be sensitive initially")
	}

	m.AddSensitiveApp("MyNewApp")
	if !m.IsSensitiveApp("MyNewApp") {
		t.Error("MyNewApp should be sensitive after adding")
	}

	m.RemoveSensitiveApp("MyNewApp")
	if m.IsSensitiveApp("MyNewApp") {
		t.Error("MyNewApp should not be sensitive after removing")
	}
}

func TestGetSensitiveApps(t *testing.T) {
	m := setupTestManager()

	apps := m.GetSensitiveApps()
	if len(apps) == 0 {
		t.Error("expected at least default sensitive apps")
	}

	found := false
	for _, app := range apps {
		if app == "mycustomapp" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected MyCustomApp to be in the list")
	}
}

func TestAddBlacklistPattern(t *testing.T) {
	m := setupTestManager()

	testPattern := `test_pattern_\d+`
	if err := m.AddBlacklistPattern(testPattern); err != nil {
		t.Fatalf("failed to add pattern: %v", err)
	}

	if !m.IsBlacklistedContent("test_pattern_123") {
		t.Error("should match newly added pattern")
	}

	if err := m.AddBlacklistPattern("[invalid"); err == nil {
		t.Error("expected error for invalid regex pattern")
	}
}

func TestUpdateConfig(t *testing.T) {
	m := setupTestManager()

	newConfig := &models.AppConfig{
		EnableSensitive:   true,
		IgnoredApps:       []string{"AnotherApp"},
		BlacklistPatterns: []string{`custom_\w+`},
	}

	m.UpdateConfig(newConfig)

	if !m.IsSensitiveApp("AnotherApp") {
		t.Error("AnotherApp should be sensitive after config update")
	}

	if m.IsSensitiveApp("MyCustomApp") {
		t.Error("old ignored apps should be replaced after config update")
	}

	if !m.IsBlacklistedContent("custom_test") {
		t.Error("custom pattern should match after config update")
	}
}
