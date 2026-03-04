package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ChromePath     string
	IdleTimeout    time.Duration
	ActionTimeout  time.Duration
	MaxSessions    int
	ViewportWidth  int
	ViewportHeight int
}

func envOrDefault(key string, defaultVal string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultVal
	}
	return value
}

func envDuration(key string, defaultVal time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return defaultVal
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil {
		return defaultVal
	}
	return parsed
}

func envInt(key string, defaultVal int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return defaultVal
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return defaultVal
	}
	return parsed
}

func envSeconds(key string, defaultVal time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return defaultVal
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return defaultVal
	}
	return time.Duration(parsed) * time.Second
}

func envViewport(key string, defaultWidth int, defaultHeight int) (int, int) {
	raw := os.Getenv(key)
	if raw == "" {
		return defaultWidth, defaultHeight
	}
	parts := strings.SplitN(raw, "x", 2)
	if len(parts) != 2 {
		return defaultWidth, defaultHeight
	}
	width, widthErr := strconv.Atoi(parts[0])
	if widthErr != nil {
		return defaultWidth, defaultHeight
	}
	height, heightErr := strconv.Atoi(parts[1])
	if heightErr != nil {
		return defaultWidth, defaultHeight
	}
	return width, height
}

func Load() Config {
	viewportWidth, viewportHeight := envViewport("BROWSE_DEFAULT_VIEWPORT", 1280, 800)
	return Config{
		ChromePath:     envOrDefault("BROWSE_CHROME_PATH", ""),
		IdleTimeout:    envDuration("BROWSE_IDLE_TIMEOUT", 30*time.Minute),
		ActionTimeout:  envSeconds("BROWSE_ACTION_TIMEOUT", 5*time.Second),
		MaxSessions:    envInt("BROWSE_MAX_SESSIONS", 10),
		ViewportWidth:  viewportWidth,
		ViewportHeight: viewportHeight,
	}
}
