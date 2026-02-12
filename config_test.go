package trace2receiver

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test Validate with minimal valid config on Windows
func Test_Config_Validate_MinimalWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test")
	}

	cfg := &Config{
		NamedPipePath: "test-pipe",
	}

	err := cfg.Validate()
	assert.NoError(t, err)
	assert.Equal(t, `\\.\pipe\test-pipe`, cfg.NamedPipePath)
}

// Test Validate with minimal valid config on Unix
func Test_Config_Validate_MinimalUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix-specific test")
	}

	cfg := &Config{
		UnixSocketPath: "/tmp/test.socket",
	}

	err := cfg.Validate()
	assert.NoError(t, err)
	assert.Equal(t, "/tmp/test.socket", cfg.UnixSocketPath)
}

// Test Validate fails when pipe is missing on Windows
func Test_Config_Validate_MissingPipeWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test")
	}

	cfg := &Config{}

	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pipe not defined")
}

// Test Validate fails when socket is missing on Unix
func Test_Config_Validate_MissingSocketUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix-specific test")
	}

	cfg := &Config{}

	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "socket not defined")
}

// Test Validate with full pipe path on Windows
func Test_Config_Validate_FullPipePathWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test")
	}

	cfg := &Config{
		NamedPipePath: `\\.\pipe\my-test-pipe`,
	}

	err := cfg.Validate()
	assert.NoError(t, err)
	assert.Equal(t, `\\.\pipe\my-test-pipe`, cfg.NamedPipePath)
}

// Test Validate with af_unix prefix on Unix
func Test_Config_Validate_AfUnixPrefixUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix-specific test")
	}

	cfg := &Config{
		UnixSocketPath: "af_unix:/tmp/test.socket",
	}

	err := cfg.Validate()
	assert.NoError(t, err)
	assert.Equal(t, "/tmp/test.socket", cfg.UnixSocketPath)
}

// Test Validate with af_unix:stream prefix on Unix
func Test_Config_Validate_AfUnixStreamPrefixUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix-specific test")
	}

	cfg := &Config{
		UnixSocketPath: "af_unix:stream:/tmp/test.socket",
	}

	err := cfg.Validate()
	assert.NoError(t, err)
	assert.Equal(t, "/tmp/test.socket", cfg.UnixSocketPath)
}

// Test Validate rejects UNC paths on Windows
func Test_Config_Validate_RejectUNCWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test")
	}

	cfg := &Config{
		NamedPipePath: `\\server\share\pipe`,
	}

	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

// Test Validate rejects drive letter paths on Windows
func Test_Config_Validate_RejectDriveLetterWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test")
	}

	cfg := &Config{
		NamedPipePath: `C:\temp\pipe`,
	}

	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

// Test Validate rejects SOCK_DGRAM on Unix
func Test_Config_Validate_RejectDgramUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix-specific test")
	}

	cfg := &Config{
		UnixSocketPath: "af_unix:dgram:/tmp/test.socket",
	}

	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SOCK_DGRAM sockets are not supported")
}

// Test Validate with valid PII settings (inline)
func Test_Config_Validate_WithValidPiiSettings(t *testing.T) {
	cfg := createMinimalValidConfig()
	cfg.Pii = &PiiSettings{
		Include: PiiInclude{
			Hostname: true,
			Username: false,
		},
	}

	err := cfg.Validate()
	assert.NoError(t, err)
	assert.NotNil(t, cfg.Pii)
}

// Test Validate with valid filter settings (inline)
func Test_Config_Validate_WithValidFilterSettings(t *testing.T) {
	cfg := createMinimalValidConfig()
	cfg.Filter = &FilterSettings{
		Defaults: FilterDefaults{
			RulesetName: "dl:verbose",
		},
	}

	err := cfg.Validate()
	assert.NoError(t, err)
	assert.NotNil(t, cfg.Filter)
}

// Test Validate with valid summary settings (inline)
func Test_Config_Validate_WithValidSummary(t *testing.T) {
	cfg := createMinimalValidConfig()
	cfg.Summary = &SummarySettings{
		MessagePatterns: []MessagePatternRule{
			{Prefix: "error:", FieldName: "error_count"},
			{Prefix: "warning:", FieldName: "warning_count"},
		},
		RegionTimers: []RegionTimerRule{
			{Category: "index", Label: "do_read_index", CountField: "index_read_count", TimeField: "index_read_time"},
		},
	}

	err := cfg.Validate()
	assert.NoError(t, err)
	assert.NotNil(t, cfg.Summary)
	assert.Equal(t, 2, len(cfg.Summary.MessagePatterns))
	assert.Equal(t, 1, len(cfg.Summary.RegionTimers))
}

// Test Validate with invalid summary settings (empty field_name)
func Test_Config_Validate_WithMalformedSummary(t *testing.T) {
	cfg := createMinimalValidConfig()
	cfg.Summary = &SummarySettings{
		MessagePatterns: []MessagePatternRule{
			{Prefix: "error:", FieldName: ""},
		},
	}

	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "field_name cannot be empty")
}

// Test Validate with summary settings with duplicate field names
func Test_Config_Validate_WithDuplicateSummaryFields(t *testing.T) {
	cfg := createMinimalValidConfig()
	cfg.Summary = &SummarySettings{
		MessagePatterns: []MessagePatternRule{
			{Prefix: "error:", FieldName: "count"},
			{Prefix: "warning:", FieldName: "count"},
		},
	}

	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate field_name")
}

// Test Validate with all optional settings valid (inline)
func Test_Config_Validate_WithAllOptionalSettings(t *testing.T) {
	cfg := createMinimalValidConfig()
	cfg.Pii = &PiiSettings{
		Include: PiiInclude{
			Hostname: true,
		},
	}
	cfg.Filter = &FilterSettings{
		Defaults: FilterDefaults{
			RulesetName: "dl:summary",
		},
	}
	cfg.Summary = &SummarySettings{
		MessagePatterns: []MessagePatternRule{
			{Prefix: "error:", FieldName: "error_count"},
		},
		RegionTimers: []RegionTimerRule{
			{Category: "index", Label: "do_read_index", TimeField: "index_read_time"},
		},
	}

	err := cfg.Validate()
	assert.NoError(t, err)
	assert.NotNil(t, cfg.Pii)
	assert.NotNil(t, cfg.Filter)
	assert.NotNil(t, cfg.Summary)
}

// Test Validate with command control enabled
func Test_Config_Validate_WithCommandControlEnabled(t *testing.T) {
	cfg := createMinimalValidConfig()
	cfg.AllowCommandControlVerbs = true

	err := cfg.Validate()
	assert.NoError(t, err)
}

// Helper function to create a minimal valid config for the current platform
func createMinimalValidConfig() *Config {
	if runtime.GOOS == "windows" {
		return &Config{
			NamedPipePath: "test-pipe",
		}
	}
	return &Config{
		UnixSocketPath: "/tmp/test.socket",
	}
}
