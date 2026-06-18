package models

import "time"

type ClipboardType string

const (
	TypeText  ClipboardType = "text"
	TypeImage ClipboardType = "image"
)

type ClipboardItem struct {
	ID          string        `json:"id"`
	Type        ClipboardType `json:"type"`
	Content     string        `json:"content,omitempty"`
	ImageData   []byte        `json:"-"`
	ImageURL    string        `json:"imageUrl,omitempty"`
	Hash        string        `json:"-"`
	IsFavorite  bool          `json:"isFavorite"`
	AppName     string        `json:"appName,omitempty"`
	CreatedAt   time.Time     `json:"createdAt"`
	UpdatedAt   time.Time     `json:"updatedAt"`
	PasteCount  int           `json:"pasteCount"`
	SizeBytes   int64         `json:"sizeBytes"`
}

type SearchResult struct {
	Items []*ClipboardItem `json:"items"`
	Total int64            `json:"total"`
}

type AppConfig struct {
	Hotkey            string   `json:"hotkey"`
	AutoStart         bool     `json:"autoStart"`
	Theme             string   `json:"theme"`
	AutoPaste         bool     `json:"autoPaste"`
	MaxHistory        int      `json:"maxHistory"`
	SensitiveApps     []string `json:"sensitiveApps"`
	BlacklistPatterns []string `json:"blacklistPatterns"`
	EnableSensitive   bool     `json:"enableSensitive"`
	IgnoredApps       []string `json:"ignoredApps"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type Stats struct {
	TotalItems    int64 `json:"totalItems"`
	TextItems     int64 `json:"textItems"`
	ImageItems    int64 `json:"imageItems"`
	FavoriteItems int64 `json:"favoriteItems"`
	TotalSize     int64 `json:"totalSize"`
	TodayCount    int64 `json:"todayCount"`
}
