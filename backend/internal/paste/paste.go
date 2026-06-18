package paste

import (
	"fmt"
	"os/exec"
	"time"

	"paste/backend/internal/clipboard"
	"paste/backend/internal/logger"
	"paste/backend/internal/storage"
	"paste/backend/pkg/models"
)

type Manager struct {
	storage   *storage.Storage
	clipboard *clipboard.Monitor
}

func NewManager(storage *storage.Storage, cb *clipboard.Monitor) *Manager {
	return &Manager{
		storage:   storage,
		clipboard: cb,
	}
}

func (m *Manager) PasteItem(item *models.ClipboardItem) error {
	if item == nil {
		return fmt.Errorf("item is nil")
	}

	if err := m.storage.IncrementPasteCount(item.ID); err != nil {
		logger.Sugar.Warnw("failed to increment paste count", "error", err, "id", item.ID)
	}

	if item.Type == models.TypeText {
		if err := m.clipboard.SetText(item.Content); err != nil {
			return fmt.Errorf("failed to set clipboard text: %w", err)
		}
	} else if item.Type == models.TypeImage {
		imgData, err := m.storage.GetImageData(item.ID)
		if err != nil {
			return fmt.Errorf("failed to get image data: %w", err)
		}
		if err := m.clipboard.SetImage(imgData); err != nil {
			return fmt.Errorf("failed to set clipboard image: %w", err)
		}
	}

	time.Sleep(50 * time.Millisecond)

	if err := simulatePaste(); err != nil {
		return fmt.Errorf("failed to simulate paste: %w", err)
	}

	logger.Sugar.Debugw("pasted item", "id", item.ID, "type", item.Type)
	return nil
}

func (m *Manager) CopyItem(item *models.ClipboardItem) error {
	if item == nil {
		return fmt.Errorf("item is nil")
	}

	if err := m.storage.IncrementPasteCount(item.ID); err != nil {
		logger.Sugar.Warnw("failed to increment paste count", "error", err, "id", item.ID)
	}

	if item.Type == models.TypeText {
		if err := m.clipboard.SetText(item.Content); err != nil {
			return fmt.Errorf("failed to set clipboard text: %w", err)
		}
	} else if item.Type == models.TypeImage {
		imgData, err := m.storage.GetImageData(item.ID)
		if err != nil {
			return fmt.Errorf("failed to get image data: %w", err)
		}
		if err := m.clipboard.SetImage(imgData); err != nil {
			return fmt.Errorf("failed to set clipboard image: %w", err)
		}
	}

	logger.Sugar.Debugw("copied item to clipboard", "id", item.ID, "type", item.Type)
	return nil
}

func simulatePaste() error {
	script := `
		tell application "System Events"
			keystroke "v" using command down
		end tell
	`
	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("osascript error: %v, output: %s", err, string(output))
	}
	return nil
}
