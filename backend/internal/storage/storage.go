package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"

	"paste/backend/internal/logger"
	"paste/backend/pkg/models"
	"paste/backend/pkg/utils"
)

type Storage struct {
	db         *sql.DB
	dataDir    string
	maxHistory int
}

var (
	ErrItemNotFound = errors.New("item not found")
	ErrDuplicate    = errors.New("duplicate item")
)

func New(dataDir string, maxHistory int) (*Storage, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "paste.db")
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	s := &Storage{
		db:         db,
		dataDir:    dataDir,
		maxHistory: maxHistory,
	}

	if err := s.initDB(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	if err := s.backupIfCorrupted(); err != nil {
		logger.Sugar.Warnw("backup check failed", "error", err)
	}

	return s, nil
}

func (s *Storage) initDB() error {
	schema := `
	CREATE TABLE IF NOT EXISTS clipboard_items (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		content TEXT,
		image_path TEXT,
		hash TEXT NOT NULL UNIQUE,
		is_favorite INTEGER NOT NULL DEFAULT 0,
		app_name TEXT,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		paste_count INTEGER NOT NULL DEFAULT 0,
		size_bytes INTEGER NOT NULL DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_items_created_at ON clipboard_items(created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_items_type ON clipboard_items(type);
	CREATE INDEX IF NOT EXISTS idx_items_favorite ON clipboard_items(is_favorite);
	CREATE INDEX IF NOT EXISTS idx_items_content ON clipboard_items(content);
	CREATE INDEX IF NOT EXISTS idx_items_hash ON clipboard_items(hash);

	CREATE TABLE IF NOT EXISTS config (
		key TEXT PRIMARY KEY,
		value TEXT
	);

	CREATE TABLE IF NOT EXISTS blacklist_patterns (
		id TEXT PRIMARY KEY,
		pattern TEXT NOT NULL,
		created_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS ignored_apps (
		id TEXT PRIMARY KEY,
		app_name TEXT NOT NULL UNIQUE,
		created_at DATETIME NOT NULL
	);
	`

	_, err := s.db.Exec(schema)
	return err
}

func (s *Storage) backupIfCorrupted() error {
	var integrity string
	err := s.db.QueryRow("PRAGMA integrity_check").Scan(&integrity)
	if err != nil {
		return err
	}

	if integrity != "ok" {
		logger.Sugar.Warnw("database integrity check failed, creating backup", "result", integrity)
		backupPath := filepath.Join(s.dataDir, fmt.Sprintf("paste-corrupted-%s.db", time.Now().Format("20060102-150405")))
		dbPath := filepath.Join(s.dataDir, "paste.db")
		if copyErr := copyFile(dbPath, backupPath); copyErr != nil {
			logger.Sugar.Errorw("failed to backup corrupted db", "error", copyErr)
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) AddItem(item *models.ClipboardItem) (*models.ClipboardItem, error) {
	now := time.Now()

	var hash string
	if item.Type == models.TypeImage {
		hash = utils.ComputeHash(item.ImageData)
	} else {
		hash = utils.ComputeTextHash(item.Content)
	}
	item.Hash = hash

	existingID, _, err := s.findExisting(hash)
	if err == nil && existingID != "" {
		if err := s.touchItem(existingID); err != nil {
			return nil, err
		}
		return s.GetItem(existingID)
	}

	if item.ID == "" {
		item.ID = uuid.New().String()
	}
	item.CreatedAt = now
	item.UpdatedAt = now

	var imagePath string
	if item.Type == models.TypeImage && len(item.ImageData) > 0 {
		imageDir := filepath.Join(s.dataDir, "images")
		if err := os.MkdirAll(imageDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create image directory: %w", err)
		}
		imagePath = filepath.Join(imageDir, fmt.Sprintf("%s.png", item.ID))
		if err := os.WriteFile(imagePath, item.ImageData, 0644); err != nil {
			return nil, fmt.Errorf("failed to save image: %w", err)
		}
		item.SizeBytes = int64(len(item.ImageData))
	} else {
		item.SizeBytes = int64(len(item.Content))
	}

	_, err = s.db.Exec(`
		INSERT INTO clipboard_items (id, type, content, image_path, hash, is_favorite, app_name, created_at, updated_at, paste_count, size_bytes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, item.ID, item.Type, item.Content, imagePath, item.Hash, boolToInt(item.IsFavorite), item.AppName, item.CreatedAt, item.UpdatedAt, item.PasteCount, item.SizeBytes)

	if err != nil {
		return nil, fmt.Errorf("failed to insert item: %w", err)
	}

	if err := s.enforceMaxHistory(); err != nil {
		logger.Sugar.Warnw("failed to enforce max history", "error", err)
	}

	logger.Sugar.Debugw("added clipboard item", "id", item.ID, "type", item.Type)
	return s.GetItem(item.ID)
}

func (s *Storage) findExisting(hash string) (string, string, error) {
	var id, itemType string
	err := s.db.QueryRow("SELECT id, type FROM clipboard_items WHERE hash = ? LIMIT 1", hash).Scan(&id, &itemType)
	if err == sql.ErrNoRows {
		return "", "", ErrItemNotFound
	}
	return id, itemType, err
}

func (s *Storage) touchItem(id string) error {
	_, err := s.db.Exec(`
		UPDATE clipboard_items SET updated_at = ?, paste_count = paste_count + 1 WHERE id = ?
	`, time.Now(), id)
	return err
}

func (s *Storage) enforceMaxHistory() error {
	var count int64
	if err := s.db.QueryRow("SELECT COUNT(*) FROM clipboard_items WHERE is_favorite = 0").Scan(&count); err != nil {
		return err
	}

	overLimit := count - int64(s.maxHistory)
	if overLimit <= 0 {
		return nil
	}

	rows, err := s.db.Query(`
		SELECT id, type, image_path FROM clipboard_items 
		WHERE is_favorite = 0 
		ORDER BY updated_at ASC 
		LIMIT ?
	`, overLimit)
	if err != nil {
		return err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id, itemType, imagePath string
		if err := rows.Scan(&id, &itemType, &imagePath); err != nil {
			continue
		}
		ids = append(ids, id)
		if models.ClipboardType(itemType) == models.TypeImage && imagePath != "" {
			_ = os.Remove(imagePath)
		}
	}

	if len(ids) > 0 {
		tx, err := s.db.Begin()
		if err != nil {
			return err
		}
		stmt, err := tx.Prepare("DELETE FROM clipboard_items WHERE id = ?")
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		defer stmt.Close()

		for _, id := range ids {
			if _, err := stmt.Exec(id); err != nil {
				continue
			}
		}
		return tx.Commit()
	}

	return nil
}

func (s *Storage) GetItem(id string) (*models.ClipboardItem, error) {
	row := s.db.QueryRow(`
		SELECT id, type, content, image_path, hash, is_favorite, app_name, created_at, updated_at, paste_count, size_bytes
		FROM clipboard_items WHERE id = ?
	`, id)

	return s.scanItem(row)
}

func (s *Storage) GetImageData(id string) ([]byte, error) {
	var imagePath string
	err := s.db.QueryRow("SELECT image_path FROM clipboard_items WHERE id = ?", id).Scan(&imagePath)
	if err != nil {
		return nil, err
	}
	if imagePath == "" {
		return nil, ErrItemNotFound
	}
	return os.ReadFile(imagePath)
}

func (s *Storage) ListItems(offset, limit int, onlyFavorites bool, itemType models.ClipboardType) ([]*models.ClipboardItem, int64, error) {
	var total int64
	countQuery := "SELECT COUNT(*) FROM clipboard_items WHERE 1=1"
	listQuery := `
		SELECT id, type, content, image_path, hash, is_favorite, app_name, created_at, updated_at, paste_count, size_bytes
		FROM clipboard_items WHERE 1=1
	`
	args := []interface{}{}
	countArgs := []interface{}{}

	if onlyFavorites {
		countQuery += " AND is_favorite = ?"
		listQuery += " AND is_favorite = ?"
		args = append(args, 1)
		countArgs = append(countArgs, 1)
	}

	if itemType != "" {
		countQuery += " AND type = ?"
		listQuery += " AND type = ?"
		args = append(args, itemType)
		countArgs = append(countArgs, itemType)
	}

	listQuery += " ORDER BY updated_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	if err := s.db.QueryRow(countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.db.Query(listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]*models.ClipboardItem, 0, limit)
	for rows.Next() {
		item, err := s.scanItem(rows)
		if err != nil {
			continue
		}
		items = append(items, item)
	}

	return items, total, nil
}

func (s *Storage) Search(query string, offset, limit int, onlyFavorites bool) ([]*models.ClipboardItem, int64, error) {
	if query == "" {
		return s.ListItems(offset, limit, onlyFavorites, "")
	}

	searchPattern := "%" + query + "%"
	var total int64

	countQuery := `
		SELECT COUNT(*) FROM clipboard_items 
		WHERE type = 'text' AND content LIKE ?
	`
	listQuery := `
		SELECT id, type, content, image_path, hash, is_favorite, app_name, created_at, updated_at, paste_count, size_bytes
		FROM clipboard_items 
		WHERE type = 'text' AND content LIKE ?
	`

	args := []interface{}{searchPattern}
	countArgs := []interface{}{searchPattern}

	if onlyFavorites {
		countQuery += " AND is_favorite = ?"
		listQuery += " AND is_favorite = ?"
		args = append(args, 1)
		countArgs = append(countArgs, 1)
	}

	listQuery += " ORDER BY updated_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	if err := s.db.QueryRow(countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.db.Query(listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]*models.ClipboardItem, 0, limit)
	for rows.Next() {
		item, err := s.scanItem(rows)
		if err != nil {
			continue
		}
		items = append(items, item)
	}

	return items, total, nil
}

func (s *Storage) scanItem(scanner interface{ Scan(dest ...interface{}) error }) (*models.ClipboardItem, error) {
	item := &models.ClipboardItem{}
	var imagePath sql.NullString
	var isFavorite int

	err := scanner.Scan(
		&item.ID, &item.Type, &item.Content, &imagePath, &item.Hash,
		&isFavorite, &item.AppName, &item.CreatedAt, &item.UpdatedAt,
		&item.PasteCount, &item.SizeBytes,
	)
	if err == sql.ErrNoRows {
		return nil, ErrItemNotFound
	}
	if err != nil {
		return nil, err
	}

	item.IsFavorite = intToBool(isFavorite)
	if item.Type == models.TypeImage && imagePath.Valid {
		item.ImageURL = "/api/v1/images/" + item.ID
	}

	return item, nil
}

func (s *Storage) SetFavorite(id string, favorite bool) error {
	result, err := s.db.Exec(`
		UPDATE clipboard_items SET is_favorite = ?, updated_at = ? WHERE id = ?
	`, boolToInt(favorite), time.Now(), id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrItemNotFound
	}
	return nil
}

func (s *Storage) IncrementPasteCount(id string) error {
	_, err := s.db.Exec(`
		UPDATE clipboard_items SET paste_count = paste_count + 1, updated_at = ? WHERE id = ?
	`, time.Now(), id)
	return err
}

func (s *Storage) DeleteItem(id string) error {
	var imagePath sql.NullString
	var itemType string
	err := s.db.QueryRow("SELECT type, image_path FROM clipboard_items WHERE id = ?", id).Scan(&itemType, &imagePath)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrItemNotFound
		}
		return err
	}

	if models.ClipboardType(itemType) == models.TypeImage && imagePath.Valid {
		_ = os.Remove(imagePath.String)
	}

	result, err := s.db.Exec("DELETE FROM clipboard_items WHERE id = ?", id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrItemNotFound
	}
	return nil
}

func (s *Storage) ClearAll(keepFavorites bool) error {
	if keepFavorites {
		rows, err := s.db.Query(`
			SELECT id, type, image_path FROM clipboard_items WHERE is_favorite = 0
		`)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var id, itemType string
			var imagePath sql.NullString
			if err := rows.Scan(&id, &itemType, &imagePath); err != nil {
				continue
			}
			if models.ClipboardType(itemType) == models.TypeImage && imagePath.Valid {
				_ = os.Remove(imagePath.String)
			}
		}

		_, err = s.db.Exec("DELETE FROM clipboard_items WHERE is_favorite = 0")
		return err
	}

	imageDir := filepath.Join(s.dataDir, "images")
	_ = os.RemoveAll(imageDir)
	_, err := s.db.Exec("DELETE FROM clipboard_items")
	return err
}

func (s *Storage) GetStats() (*models.Stats, error) {
	stats := &models.Stats{}
	var totalSize sql.NullInt64

	err := s.db.QueryRow("SELECT COUNT(*) FROM clipboard_items").Scan(&stats.TotalItems)
	if err != nil {
		return nil, err
	}
	err = s.db.QueryRow("SELECT COUNT(*) FROM clipboard_items WHERE type = 'text'").Scan(&stats.TextItems)
	if err != nil {
		return nil, err
	}
	err = s.db.QueryRow("SELECT COUNT(*) FROM clipboard_items WHERE type = 'image'").Scan(&stats.ImageItems)
	if err != nil {
		return nil, err
	}
	err = s.db.QueryRow("SELECT COUNT(*) FROM clipboard_items WHERE is_favorite = 1").Scan(&stats.FavoriteItems)
	if err != nil {
		return nil, err
	}
	err = s.db.QueryRow("SELECT COALESCE(SUM(size_bytes), 0) FROM clipboard_items").Scan(&totalSize)
	if err != nil {
		return nil, err
	}
	if totalSize.Valid {
		stats.TotalSize = totalSize.Int64
	}

	today := time.Now().Format("2006-01-02")
	err = s.db.QueryRow("SELECT COUNT(*) FROM clipboard_items WHERE DATE(created_at) = ?", today).Scan(&stats.TodayCount)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

func (s *Storage) GetAllHashes() (map[string]bool, error) {
	rows, err := s.db.Query("SELECT hash FROM clipboard_items")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	hashes := make(map[string]bool)
	for rows.Next() {
		var hash string
		if err := rows.Scan(&hash); err != nil {
			continue
		}
		hashes[hash] = true
	}
	return hashes, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func intToBool(i int) bool {
	return i == 1
}
