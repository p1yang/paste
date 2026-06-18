package clipboard

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/atotto/clipboard"

	"paste/backend/internal/logger"
	"paste/backend/internal/security"
	"paste/backend/internal/storage"
	"paste/backend/pkg/models"
)

type Monitor struct {
	storage       *storage.Storage
	security      *security.Manager
	lastTextHash  string
	lastImageHash string
	interval      time.Duration
	stopCh        chan struct{}
	mu            sync.Mutex
	running       bool
}

func NewMonitor(storage *storage.Storage, securityMgr *security.Manager) *Monitor {
	return &Monitor{
		storage:  storage,
		security: securityMgr,
		interval: 500 * time.Millisecond,
		stopCh:   make(chan struct{}),
	}
}

func (m *Monitor) Start() error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return fmt.Errorf("monitor already running")
	}
	m.running = true
	m.stopCh = make(chan struct{})
	m.mu.Unlock()

	go m.run()
	logger.Sugar.Info("clipboard monitor started")
	return nil
}

func (m *Monitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}
	m.running = false
	close(m.stopCh)
	logger.Sugar.Info("clipboard monitor stopped")
}

func (m *Monitor) run() {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.checkClipboard()
		}
	}
}

func (m *Monitor) checkClipboard() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.checkText(); err != nil {
		logger.Sugar.Debugw("text check failed", "error", err)
	}

	if err := m.checkImage(); err != nil {
		logger.Sugar.Debugw("image check failed", "error", err)
	}
}

func (m *Monitor) checkText() error {
	text, err := clipboard.ReadAll()
	if err != nil {
		return err
	}

	if text == "" {
		return nil
	}

	hash := computeQuickHash(text)
	if hash == m.lastTextHash {
		return nil
	}
	m.lastTextHash = hash

	appName := getFrontmostApp()
	if m.security.ShouldIgnore(appName, text) {
		logger.Sugar.Debugw("ignoring clipboard content from sensitive app", "app", appName)
		return nil
	}

	if len(text) > 5*1024*1024 {
		logger.Sugar.Debugw("clipboard content too large, skipping", "size", len(text))
		return nil
	}

	item := &models.ClipboardItem{
		Type:    models.TypeText,
		Content: text,
		AppName: appName,
	}

	_, err = m.storage.AddItem(item)
	if err != nil {
		return fmt.Errorf("failed to add text item: %w", err)
	}

	return nil
}

func (m *Monitor) checkImage() error {
	imgData, err := getClipboardImage()
	if err != nil {
		return nil
	}

	if len(imgData) == 0 {
		return nil
	}

	hash := computeQuickHash(string(imgData))
	if hash == m.lastImageHash {
		return nil
	}
	m.lastImageHash = hash

	if len(imgData) > 20*1024*1024 {
		logger.Sugar.Debugw("clipboard image too large, skipping", "size", len(imgData))
		return nil
	}

	appName := getFrontmostApp()
	if m.security.IsSensitiveApp(appName) {
		logger.Sugar.Debugw("ignoring clipboard image from sensitive app", "app", appName)
		return nil
	}

	item := &models.ClipboardItem{
		Type:      models.TypeImage,
		ImageData: imgData,
		AppName:   appName,
	}

	_, err = m.storage.AddItem(item)
	if err != nil {
		return fmt.Errorf("failed to add image item: %w", err)
	}

	return nil
}

func (m *Monitor) SetText(text string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := clipboard.WriteAll(text); err != nil {
		return fmt.Errorf("failed to write text to clipboard: %w", err)
	}
	m.lastTextHash = computeQuickHash(text)
	return nil
}

func (m *Monitor) SetImage(imgData []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := setClipboardImage(imgData); err != nil {
		return fmt.Errorf("failed to write image to clipboard: %w", err)
	}
	m.lastImageHash = computeQuickHash(string(imgData))
	return nil
}

func getClipboardImage() ([]byte, error) {
	tmpFile, err := os.CreateTemp("", "paste-clipboard-*.png")
	if err != nil {
		return nil, err
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	script := fmt.Sprintf(`
		tell application "System Events"
			set the clipboardData to «class PNGf» of (the clipboard as record)
		end tell
		set theFile to open for access POSIX file "%s" with write permission
		write the clipboardData to theFile
		close access theFile
	`, tmpPath)

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "error") {
			return nil, nil
		}
		return nil, err
	}

	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, nil
	}

	return data, nil
}

func setClipboardImage(imgData []byte) error {
	tmpFile, err := os.CreateTemp("", "paste-write-*.png")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()

	img, _, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return err
	}

	err = png.Encode(tmpFile, img)
	tmpFile.Close()
	if err != nil {
		os.Remove(tmpPath)
		return err
	}
	defer os.Remove(tmpPath)

	script := fmt.Sprintf(`
		set the clipboard to (read (POSIX file "%s") as JPEG picture)
	`, tmpPath)

	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}

func getFrontmostApp() string {
	cmd := exec.Command("osascript", "-e", `tell application "System Events" to name of first application process whose frontmost is true`)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func computeQuickHash(s string) string {
	if len(s) == 0 {
		return ""
	}
	h := uint64(1469598103934665603)
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= 1099511628211
	}
	return fmt.Sprintf("%016x", h)
}
