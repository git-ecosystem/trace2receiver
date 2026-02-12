package trace2receiver

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// Test Validate with PII settings from file path
func Test_Config_Validate_WithPiiFilePath(t *testing.T) {
	tmpDir := t.TempDir()
	piiPath := filepath.Join(tmpDir, "pii.yml")
	piiContent := `
include:
  hostname: true
  username: false
`
	err := os.WriteFile(piiPath, []byte(piiContent), 0644)
	require.NoError(t, err)

	cfg := createMinimalValidConfig()
	cfg.rawPii = piiPath

	err = cfg.Validate()
	assert.NoError(t, err)
	assert.NotNil(t, cfg.Pii)
	assert.True(t, cfg.Pii.Include.Hostname)
	assert.False(t, cfg.Pii.Include.Username)
}

// Test Validate with invalid PII file path
func Test_Config_Validate_WithInvalidPiiFilePath(t *testing.T) {
	cfg := createMinimalValidConfig()
	cfg.rawPii = "/nonexistent/pii.yml"

	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pii:")
}

// Test Validate with filter settings from file path
func Test_Config_Validate_WithFilterFilePath(t *testing.T) {
	tmpDir := t.TempDir()
	filterPath := filepath.Join(tmpDir, "filter.yml")
	filterContent := `
defaults:
  ruleset: "dl:verbose"
`
	err := os.WriteFile(filterPath, []byte(filterContent), 0644)
	require.NoError(t, err)

	cfg := createMinimalValidConfig()
	cfg.rawFilter = filterPath

	err = cfg.Validate()
	assert.NoError(t, err)
	assert.NotNil(t, cfg.Filter)
}

// Test Validate with invalid filter file path
func Test_Config_Validate_WithInvalidFilterFilePath(t *testing.T) {
	cfg := createMinimalValidConfig()
	cfg.rawFilter = "/nonexistent/filter.yml"

	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "filter:")
}

// Test Validate with summary settings from file path
func Test_Config_Validate_WithSummaryFilePath(t *testing.T) {
	tmpDir := t.TempDir()
	summaryPath := filepath.Join(tmpDir, "summary.yml")
	summaryContent := `
message_patterns:
  - prefix: "error:"
    field_name: "error_count"
  - prefix: "warning:"
    field_name: "warning_count"

region_timers:
  - category: "index"
    label: "do_read_index"
    count_field: "index_read_count"
    time_field: "index_read_time"
`
	err := os.WriteFile(summaryPath, []byte(summaryContent), 0644)
	require.NoError(t, err)

	cfg := createMinimalValidConfig()
	cfg.rawSummary = summaryPath

	err = cfg.Validate()
	assert.NoError(t, err)
	assert.NotNil(t, cfg.Summary)
	assert.Equal(t, 2, len(cfg.Summary.MessagePatterns))
	assert.Equal(t, 1, len(cfg.Summary.RegionTimers))
}

// Test Validate with invalid summary file path
func Test_Config_Validate_WithInvalidSummaryFilePath(t *testing.T) {
	cfg := createMinimalValidConfig()
	cfg.rawSummary = "/nonexistent/summary.yml"

	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "summary:")
}

// Test Validate with malformed summary from file path
func Test_Config_Validate_WithMalformedSummaryFilePath(t *testing.T) {
	tmpDir := t.TempDir()
	summaryPath := filepath.Join(tmpDir, "summary.yml")
	summaryContent := `
message_patterns:
  - prefix: "error:"
    field_name: ""
`
	err := os.WriteFile(summaryPath, []byte(summaryContent), 0644)
	require.NoError(t, err)

	cfg := createMinimalValidConfig()
	cfg.rawSummary = summaryPath

	err = cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "field_name cannot be empty")
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
