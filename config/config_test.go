package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoad_ReturnsDefaultsWhenNoEnvVarsSet(t *testing.T) {
	result := Load()

	assert.Equal(t, "", result.ChromePath)
	assert.Equal(t, 30*time.Minute, result.IdleTimeout)
	assert.Equal(t, 10, result.MaxSessions)
	assert.Equal(t, 1280, result.ViewportWidth)
	assert.Equal(t, 800, result.ViewportHeight)
}

func TestLoad_ReadsChromePath(t *testing.T) {
	t.Setenv("BROWSE_CHROME_PATH", "/usr/bin/chromium")

	result := Load()

	assert.Equal(t, "/usr/bin/chromium", result.ChromePath)
}

func TestLoad_ReadsIdleTimeout(t *testing.T) {
	t.Setenv("BROWSE_IDLE_TIMEOUT", "15m")

	result := Load()

	assert.Equal(t, 15*time.Minute, result.IdleTimeout)
}

func TestLoad_ReadsMaxSessions(t *testing.T) {
	t.Setenv("BROWSE_MAX_SESSIONS", "20")

	result := Load()

	assert.Equal(t, 20, result.MaxSessions)
}

func TestLoad_ReadsViewport(t *testing.T) {
	t.Setenv("BROWSE_DEFAULT_VIEWPORT", "1920x1080")

	result := Load()

	assert.Equal(t, 1920, result.ViewportWidth)
	assert.Equal(t, 1080, result.ViewportHeight)
}

func TestLoad_FallsBackToDefaultForInvalidIdleTimeout(t *testing.T) {
	t.Setenv("BROWSE_IDLE_TIMEOUT", "notaduration")

	result := Load()

	assert.Equal(t, 30*time.Minute, result.IdleTimeout)
}

func TestLoad_FallsBackToDefaultForInvalidMaxSessions(t *testing.T) {
	t.Setenv("BROWSE_MAX_SESSIONS", "abc")

	result := Load()

	assert.Equal(t, 10, result.MaxSessions)
}

func TestLoad_FallsBackToDefaultForInvalidViewport(t *testing.T) {
	t.Setenv("BROWSE_DEFAULT_VIEWPORT", "invalid")

	result := Load()

	assert.Equal(t, 1280, result.ViewportWidth)
	assert.Equal(t, 800, result.ViewportHeight)
}

func TestLoad_ActionTimeoutDefault(t *testing.T) {
	result := Load()

	assert.Equal(t, 5*time.Second, result.ActionTimeout)
}

func TestLoad_ActionTimeoutFromEnv(t *testing.T) {
	t.Setenv("BROWSE_ACTION_TIMEOUT", "10")

	result := Load()

	assert.Equal(t, 10*time.Second, result.ActionTimeout)
}
