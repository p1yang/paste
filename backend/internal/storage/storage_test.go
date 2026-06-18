package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"paste/backend/internal/logger"
	"paste/backend/pkg/models"
)

func setupTestStorage(t *testing.T) *Storage {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "paste-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	if err := logger.Init(tmpDir); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to init logger: %v", err)
	}

	s, err := New(tmpDir, 100)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create storage: %v", err)
	}

	t.Cleanup(func() {
		s.Close()
		os.RemoveAll(tmpDir)
	})

	return s
}

func TestAddAndGetTextItem(t *testing.T) {
	s := setupTestStorage(t)

	item := &models.ClipboardItem{
		Type:    models.TypeText,
		Content: "Hello, World!",
		AppName: "TestApp",
	}

	added, err := s.AddItem(item)
	if err != nil {
		t.Fatalf("failed to add item: %v", err)
	}

	if added.ID == "" {
		t.Error("expected non-empty ID")
	}
	if added.Type != models.TypeText {
		t.Errorf("expected type text, got %s", added.Type)
	}
	if added.Content != "Hello, World!" {
		t.Errorf("expected content 'Hello, World!', got '%s'", added.Content)
	}
	if added.AppName != "TestApp" {
		t.Errorf("expected app name 'TestApp', got '%s'", added.AppName)
	}
	if !added.CreatedAt.Before(time.Now().Add(time.Second)) {
		t.Error("expected created at to be in past")
	}

	got, err := s.GetItem(added.ID)
	if err != nil {
		t.Fatalf("failed to get item: %v", err)
	}
	if got.Content != added.Content {
		t.Errorf("content mismatch: got %v, want %v", got.Content, added.Content)
	}
}

func TestAddDuplicateItem(t *testing.T) {
	s := setupTestStorage(t)

	item1 := &models.ClipboardItem{
		Type:    models.TypeText,
		Content: "duplicate test",
	}

	added1, err := s.AddItem(item1)
	if err != nil {
		t.Fatalf("failed to add first item: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	item2 := &models.ClipboardItem{
		Type:    models.TypeText,
		Content: "duplicate test",
	}

	added2, err := s.AddItem(item2)
	if err != nil {
		t.Fatalf("failed to add duplicate item: %v", err)
	}

	if added2.ID != added1.ID {
		t.Error("expected duplicate to return same ID")
	}
	if added2.PasteCount != added1.PasteCount+1 {
		t.Errorf("expected paste count to increment, got %d, want %d", added2.PasteCount, added1.PasteCount+1)
	}

	items, total, err := s.ListItems(0, 10, false, "")
	if err != nil {
		t.Fatalf("failed to list items: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 item, got %d", total)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item in list, got %d", len(items))
	}
}

func TestListItemsWithPagination(t *testing.T) {
	s := setupTestStorage(t)

	for i := 0; i < 25; i++ {
		item := &models.ClipboardItem{
			Type:    models.TypeText,
			Content:  "item " + string(rune('A'+i%26)),
		}
		if _, err := s.AddItem(item); err != nil {
			t.Fatalf("failed to add item %d: %v", i, err)
		}
	}

	items, total, err := s.ListItems(0, 10, false, "")
	if err != nil {
		t.Fatalf("failed to list items: %v", err)
	}
	if total < 25 {
		t.Errorf("expected at least 25 items, got %d", total)
	}
	if len(items) != 10 {
		t.Errorf("expected 10 items per page, got %d", len(items))
	}
}

func TestToggleFavorite(t *testing.T) {
	s := setupTestStorage(t)

	item := &models.ClipboardItem{
		Type:    models.TypeText,
		Content: "favorite test",
	}
	added, err := s.AddItem(item)
	if err != nil {
		t.Fatalf("failed to add item: %v", err)
	}

	if added.IsFavorite {
		t.Error("expected item to not be favorite initially")
	}

	if err := s.SetFavorite(added.ID, true); err != nil {
		t.Fatalf("failed to set favorite: %v", err)
	}

	got, err := s.GetItem(added.ID)
	if err != nil {
		t.Fatalf("failed to get item: %v", err)
	}
	if !got.IsFavorite {
		t.Error("expected item to be favorite")
	}

	favItems, favTotal, err := s.ListItems(0, 10, true, "")
	if err != nil {
		t.Fatalf("failed to list favorites: %v", err)
	}
	if favTotal != 1 {
		t.Errorf("expected 1 favorite, got %d", favTotal)
	}
	if len(favItems) != 1 {
		t.Errorf("expected 1 favorite item, got %d", len(favItems))
	}
}

func TestDeleteItem(t *testing.T) {
	s := setupTestStorage(t)

	item := &models.ClipboardItem{
		Type:    models.TypeText,
		Content: "delete test",
	}
	added, err := s.AddItem(item)
	if err != nil {
		t.Fatalf("failed to add item: %v", err)
	}

	if err := s.DeleteItem(added.ID); err != nil {
		t.Fatalf("failed to delete item: %v", err)
	}

	_, err = s.GetItem(added.ID)
	if err != ErrItemNotFound {
		t.Errorf("expected ErrItemNotFound, got %v", err)
	}
}

func TestSearchItems(t *testing.T) {
	s := setupTestStorage(t)

	items := []*models.ClipboardItem{
		{Type: models.TypeText, Content: "Hello World"},
		{Type: models.TypeText, Content: "Hello Go"},
		{Type: models.TypeText, Content: "Goodbye World"},
		{Type: models.TypeText, Content: "Testing search"},
	}

	for _, item := range items {
		if _, err := s.AddItem(item); err != nil {
			t.Fatalf("failed to add item: %v", err)
		}
	}

	result, total, err := s.Search("Hello", 0, 10, false)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if total != 2 {
		t.Errorf("expected 2 results for 'Hello', got %d", total)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 result items, got %d", len(result))
	}
}

func TestEnforceMaxHistory(t *testing.T) {
	s := setupTestStorage(t)
	s.maxHistory = 5

	for i := 0; i < 10; i++ {
		item := &models.ClipboardItem{
			Type:    models.TypeText,
			Content:  "item " + string(rune('A'+i)),
		}
		if _, err := s.AddItem(item); err != nil {
			t.Fatalf("failed to add item %d: %v", i, err)
		}
	}

	time.Sleep(100 * time.Millisecond)

	_, total, err := s.ListItems(0, 100, false, "")
	if err != nil {
		t.Fatalf("failed to list items: %v", err)
	}
	if total > 5 {
		t.Errorf("expected max 5 non-favorite items, got %d", total)
	}
}

func TestGetStats(t *testing.T) {
	s := setupTestStorage(t)

	for i := 0; i < 3; i++ {
		item := &models.ClipboardItem{
			Type:    models.TypeText,
			Content:  "text " + string(rune('A'+i)),
		}
		added, err := s.AddItem(item)
		if err != nil {
			t.Fatalf("failed to add item: %v", err)
		}
		if i == 0 {
			if err := s.SetFavorite(added.ID, true); err != nil {
				t.Fatalf("failed to set favorite: %v", err)
			}
		}
	}

	stats, err := s.GetStats()
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}
	if stats.TotalItems < 3 {
		t.Errorf("expected at least 3 total items, got %d", stats.TotalItems)
	}
	if stats.TextItems < 3 {
		t.Errorf("expected at least 3 text items, got %d", stats.TextItems)
	}
	if stats.FavoriteItems != 1 {
		t.Errorf("expected 1 favorite item, got %d", stats.FavoriteItems)
	}
}

func TestClearAll(t *testing.T) {
	s := setupTestStorage(t)

	for i := 0; i < 5; i++ {
		item := &models.ClipboardItem{
			Type:    models.TypeText,
			Content:  "item " + string(rune('A'+i)),
		}
		added, err := s.AddItem(item)
		if err != nil {
			t.Fatalf("failed to add item: %v", err)
		}
		if i == 0 {
			if err := s.SetFavorite(added.ID, true); err != nil {
				t.Fatalf("failed to set favorite: %v", err)
			}
		}
	}

	if err := s.ClearAll(true); err != nil {
		t.Fatalf("failed to clear all: %v", err)
	}

	_, total, err := s.ListItems(0, 100, false, "")
	if err != nil {
		t.Fatalf("failed to list items: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 favorite remaining, got %d", total)
	}
}

func TestIncrementPasteCount(t *testing.T) {
	s := setupTestStorage(t)

	item := &models.ClipboardItem{
		Type:    models.TypeText,
		Content: "paste count test",
	}
	added, err := s.AddItem(item)
	if err != nil {
		t.Fatalf("failed to add item: %v", err)
	}

	initialCount := added.PasteCount

	if err := s.IncrementPasteCount(added.ID); err != nil {
		t.Fatalf("failed to increment paste count: %v", err)
	}

	got, err := s.GetItem(added.ID)
	if err != nil {
		t.Fatalf("failed to get item: %v", err)
	}
	if got.PasteCount != initialCount+1 {
		t.Errorf("expected paste count %d, got %d", initialCount+1, got.PasteCount)
	}
}

func TestImagePath(t *testing.T) {
	s := setupTestStorage(t)

	imgDir := filepath.Join(s.dataDir, "images")
	if _, err := os.Stat(imgDir); os.IsNotExist(err) {
		t.Logf("image directory does not exist yet, expected")
	}

	item := &models.ClipboardItem{
		Type:      models.TypeImage,
		ImageData: []byte{0x89, 0x50, 0x4E, 0x47},
	}
	added, err := s.AddItem(item)
	if err != nil {
		t.Fatalf("failed to add image item: %v", err)
	}

	if added.ImageURL == "" {
		t.Error("expected image URL to be set")
	}

	_, err = os.Stat(imgDir)
	if os.IsNotExist(err) {
		t.Error("expected image directory to exist after adding image")
	}
}
