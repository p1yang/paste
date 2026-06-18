package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"paste/backend/internal/autostart"
	"paste/backend/internal/clipboard"
	"paste/backend/internal/config"
	"paste/backend/internal/logger"
	"paste/backend/internal/paste"
	"paste/backend/internal/security"
	"paste/backend/internal/storage"
	"paste/backend/pkg/models"
)

type Server struct {
	engine       *gin.Engine
	httpServer   *http.Server
	storage      *storage.Storage
	clipboard    *clipboard.Monitor
	paste        *paste.Manager
	config       *config.Manager
	security     *security.Manager
	autostart    *autostart.Manager
	port         int
}

func NewServer(
	port int,
	storage *storage.Storage,
	clipboard *clipboard.Monitor,
	pasteMgr *paste.Manager,
	configMgr *config.Manager,
	securityMgr *security.Manager,
	autostartMgr *autostart.Manager,
) *Server {
	gin.SetMode(gin.ReleaseMode)

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(corsMiddleware())
	engine.Use(requestLogger())

	s := &Server{
		engine:    engine,
		storage:   storage,
		clipboard: clipboard,
		paste:     pasteMgr,
		config:    configMgr,
		security:  securityMgr,
		autostart: autostartMgr,
		port:      port,
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	api := s.engine.Group("/api/v1")
	{
		api.GET("/health", s.healthCheck)

		items := api.Group("/items")
		{
			items.GET("", s.listItems)
			items.GET("/:id", s.getItem)
			items.DELETE("/:id", s.deleteItem)
			items.PUT("/:id/favorite", s.toggleFavorite)
			items.POST("/:id/copy", s.copyItem)
			items.POST("/:id/paste", s.pasteItem)
		}

		api.GET("/search", s.searchItems)
		api.GET("/stats", s.getStats)
		api.DELETE("/items", s.clearAll)

		api.GET("/images/:id", s.getImage)

		cfg := api.Group("/config")
		{
			cfg.GET("", s.getConfig)
			cfg.PUT("", s.updateConfig)
		}

		api.GET("/autostart", s.getAutostart)
		api.PUT("/autostart", s.setAutostart)

		api.GET("/sensitive-apps", s.getSensitiveApps)
	}
}

func (s *Server) Start() error {
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("127.0.0.1:%d", s.port),
		Handler:      s.engine,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	logger.Sugar.Infow("starting API server", "port", s.port)

	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to start server: %w", err)
	}
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()

		if statusCode >= 500 {
			logger.Sugar.Errorw("request failed",
				"method", c.Request.Method,
				"path", path,
				"query", query,
				"status", statusCode,
				"latency", latency,
				"error", c.Errors.String(),
			)
		} else {
			logger.Sugar.Debugw("request completed",
				"method", c.Request.Method,
				"path", path,
				"status", statusCode,
				"latency", latency,
			)
		}
	}
}

func (s *Server) success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    data,
	})
}

func (s *Server) error(c *gin.Context, statusCode int, err error) {
	logger.Sugar.Errorw("API error", "path", c.Request.URL.Path, "error", err)
	c.JSON(statusCode, models.APIResponse{
		Success: false,
		Error:   err.Error(),
	})
}

func (s *Server) healthCheck(c *gin.Context) {
	s.success(c, gin.H{
		"status":  "ok",
		"version": "1.0.0",
		"pid":     os.Getpid(),
	})
}

func getIntQuery(c *gin.Context, key string, defaultValue int) int {
	val := c.Query(key)
	if val == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func getBoolQuery(c *gin.Context, key string, defaultValue bool) bool {
	val := c.Query(key)
	if val == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseBool(val)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func (s *Server) listItems(c *gin.Context) {
	offset := getIntQuery(c, "offset", 0)
	limit := getIntQuery(c, "limit", 50)
	favorites := getBoolQuery(c, "favorites", false)
	itemType := models.ClipboardType(c.Query("type"))

	if limit > 200 {
		limit = 200
	}

	items, total, err := s.storage.ListItems(offset, limit, favorites, itemType)
	if err != nil {
		s.error(c, http.StatusInternalServerError, err)
		return
	}

	s.success(c, &models.SearchResult{
		Items: items,
		Total: total,
	})
}

func (s *Server) getItem(c *gin.Context) {
	id := c.Param("id")
	item, err := s.storage.GetItem(id)
	if err != nil {
		if errors.Is(err, storage.ErrItemNotFound) {
			s.error(c, http.StatusNotFound, err)
			return
		}
		s.error(c, http.StatusInternalServerError, err)
		return
	}
	s.success(c, item)
}

func (s *Server) deleteItem(c *gin.Context) {
	id := c.Param("id")
	if err := s.storage.DeleteItem(id); err != nil {
		if errors.Is(err, storage.ErrItemNotFound) {
			s.error(c, http.StatusNotFound, err)
			return
		}
		s.error(c, http.StatusInternalServerError, err)
		return
	}
	s.success(c, nil)
}

func (s *Server) toggleFavorite(c *gin.Context) {
	id := c.Param("id")

	var body struct {
		Favorite bool `json:"favorite"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		s.error(c, http.StatusBadRequest, fmt.Errorf("invalid request body"))
		return
	}

	if err := s.storage.SetFavorite(id, body.Favorite); err != nil {
		if errors.Is(err, storage.ErrItemNotFound) {
			s.error(c, http.StatusNotFound, err)
			return
		}
		s.error(c, http.StatusInternalServerError, err)
		return
	}

	item, _ := s.storage.GetItem(id)
	s.success(c, item)
}

func (s *Server) copyItem(c *gin.Context) {
	id := c.Param("id")
	item, err := s.storage.GetItem(id)
	if err != nil {
		if errors.Is(err, storage.ErrItemNotFound) {
			s.error(c, http.StatusNotFound, err)
			return
		}
		s.error(c, http.StatusInternalServerError, err)
		return
	}

	if err := s.paste.CopyItem(item); err != nil {
		s.error(c, http.StatusInternalServerError, err)
		return
	}
	s.success(c, nil)
}

func (s *Server) pasteItem(c *gin.Context) {
	id := c.Param("id")
	item, err := s.storage.GetItem(id)
	if err != nil {
		if errors.Is(err, storage.ErrItemNotFound) {
			s.error(c, http.StatusNotFound, err)
			return
		}
		s.error(c, http.StatusInternalServerError, err)
		return
	}

	if err := s.paste.PasteItem(item); err != nil {
		s.error(c, http.StatusInternalServerError, err)
		return
	}
	s.success(c, nil)
}

func (s *Server) searchItems(c *gin.Context) {
	query := c.Query("q")
	offset := getIntQuery(c, "offset", 0)
	limit := getIntQuery(c, "limit", 50)
	favorites := getBoolQuery(c, "favorites", false)

	if limit > 200 {
		limit = 200
	}

	items, total, err := s.storage.Search(query, offset, limit, favorites)
	if err != nil {
		s.error(c, http.StatusInternalServerError, err)
		return
	}

	s.success(c, &models.SearchResult{
		Items: items,
		Total: total,
	})
}

func (s *Server) getStats(c *gin.Context) {
	stats, err := s.storage.GetStats()
	if err != nil {
		s.error(c, http.StatusInternalServerError, err)
		return
	}
	s.success(c, stats)
}

func (s *Server) clearAll(c *gin.Context) {
	keepFavorites := getBoolQuery(c, "keepFavorites", true)
	if err := s.storage.ClearAll(keepFavorites); err != nil {
		s.error(c, http.StatusInternalServerError, err)
		return
	}
	s.success(c, nil)
}

func (s *Server) getImage(c *gin.Context) {
	id := c.Param("id")
	data, err := s.storage.GetImageData(id)
	if err != nil {
		if errors.Is(err, storage.ErrItemNotFound) {
			s.error(c, http.StatusNotFound, err)
			return
		}
		s.error(c, http.StatusInternalServerError, err)
		return
	}

	c.Data(http.StatusOK, "image/png", data)
}

func (s *Server) getConfig(c *gin.Context) {
	s.success(c, s.config.Get())
}

func (s *Server) updateConfig(c *gin.Context) {
	var cfg models.AppConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		s.error(c, http.StatusBadRequest, fmt.Errorf("invalid config: %w", err))
		return
	}

	if err := s.config.Update(&cfg); err != nil {
		s.error(c, http.StatusInternalServerError, err)
		return
	}

	s.security.UpdateConfig(&cfg)

	if err := s.autostart.Toggle(cfg.AutoStart); err != nil {
		logger.Sugar.Warnw("failed to set autostart", "error", err)
	}

	s.success(c, s.config.Get())
}

func (s *Server) getAutostart(c *gin.Context) {
	s.success(c, gin.H{
		"enabled": s.autostart.IsEnabled(),
	})
}

func (s *Server) setAutostart(c *gin.Context) {
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		s.error(c, http.StatusBadRequest, fmt.Errorf("invalid request"))
		return
	}

	if err := s.autostart.Toggle(body.Enabled); err != nil {
		s.error(c, http.StatusInternalServerError, err)
		return
	}

	cfg := s.config.Get()
	cfg.AutoStart = body.Enabled
	if err := s.config.Update(cfg); err != nil {
		logger.Sugar.Warnw("failed to update config autostart", "error", err)
	}

	s.success(c, gin.H{"enabled": body.Enabled})
}

func (s *Server) getSensitiveApps(c *gin.Context) {
	s.success(c, gin.H{
		"apps": s.security.GetSensitiveApps(),
	})
}
