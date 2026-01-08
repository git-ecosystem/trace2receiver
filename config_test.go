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

// Test Validate with valid PII settings file
func Test_Config_Validate_WithValidPiiSettings(t *testing.T) {
	// Create a temporary PII settings file
	tmpDir := t.TempDir()
	piiPath := filepath.Join(tmpDir, "pii.yml")
	piiContent := `
pii_filter:
  domains:
    - pattern: "example.com"
      replace: "<domain>"
`
	err := os.WriteFile(piiPath, []byte(piiContent), 0644)
	require.NoError(t, err)

	cfg := createMinimalValidConfig()
	cfg.PiiSettingsPath = piiPath

	err = cfg.Validate()
	assert.NoError(t, err)
	assert.NotNil(t, cfg.piiSettings)
}

// Test Validate with invalid PII settings file
func Test_Config_Validate_WithInvalidPiiSettings(t *testing.T) {
	cfg := createMinimalValidConfig()
	cfg.PiiSettingsPath = "/nonexistent/pii.yml"

	err := cfg.Validate()
	assert.Error(t, err)
}

// Test Validate with valid filter settings file
func Test_Config_Validate_WithValidFilterSettings(t *testing.T) {
	// Create a temporary filter settings file
	tmpDir := t.TempDir()
	filterPath := filepath.Join(tmpDir, "filter.yml")
	filterContent := `
default_action: accept
`
	err := os.WriteFile(filterPath, []byte(filterContent), 0644)
	require.NoError(t, err)

	cfg := createMinimalValidConfig()
	cfg.FilterSettingsPath = filterPath

	err = cfg.Validate()
	assert.NoError(t, err)
	assert.NotNil(t, cfg.filterSettings)
}

// Test Validate with invalid filter settings file
func Test_Config_Validate_WithInvalidFilterSettings(t *testing.T) {
	cfg := createMinimalValidConfig()
	cfg.FilterSettingsPath = "/nonexistent/filter.yml"

	err := cfg.Validate()
	assert.Error(t, err)
}

// Test Validate with valid summary settings file
func Test_Config_Validate_WithValidSummary(t *testing.T) {
	// Create a temporary summary settings file
	tmpDir := t.TempDir()
	summaryPath := filepath.Join(tmpDir, "custom_summary.yml")
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
	cfg.SummaryPath = summaryPath

	err = cfg.Validate()
	assert.NoError(t, err)
	assert.NotNil(t, cfg.summary)
	assert.Equal(t, 2, len(cfg.summary.MessagePatterns))
	assert.Equal(t, 1, len(cfg.summary.RegionTimers))
}

// Test Validate with invalid summary settings file (nonexistent)
func Test_Config_Validate_WithNonexistentSummary(t *testing.T) {
	cfg := createMinimalValidConfig()
	cfg.SummaryPath = "/nonexistent/custom_summary.yml"

	err := cfg.Validate()
	assert.Error(t, err)
}

// Test Validate with invalid summary settings file (malformed YAML)
func Test_Config_Validate_WithMalformedSummary(t *testing.T) {
	// Create a temporary malformed summary settings file
	tmpDir := t.TempDir()
	summaryPath := filepath.Join(tmpDir, "custom_summary.yml")
	summaryContent := `
message_patterns:
  - prefix: "error:"
    field_name: ""
`
	err := os.WriteFile(summaryPath, []byte(summaryContent), 0644)
	require.NoError(t, err)

	cfg := createMinimalValidConfig()
	cfg.SummaryPath = summaryPath

	err = cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "field_name cannot be empty")
}

// Test Validate with summary settings with duplicate field names
func Test_Config_Validate_WithDuplicateSummaryFields(t *testing.T) {
	// Create a temporary summary settings file with duplicate fields
	tmpDir := t.TempDir()
	summaryPath := filepath.Join(tmpDir, "custom_summary.yml")
	summaryContent := `
message_patterns:
  - prefix: "error:"
    field_name: "count"
  - prefix: "warning:"
    field_name: "count"
`
	err := os.WriteFile(summaryPath, []byte(summaryContent), 0644)
	require.NoError(t, err)

	cfg := createMinimalValidConfig()
	cfg.SummaryPath = summaryPath

	err = cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate field_name")
}

// Test Validate with all optional settings valid
func Test_Config_Validate_WithAllOptionalSettings(t *testing.T) {
	// Create temporary files for all settings
	tmpDir := t.TempDir()

	piiPath := filepath.Join(tmpDir, "pii.yml")
	piiContent := `
pii_filter:
  domains:
    - pattern: "example.com"
      replace: "<domain>"
`
	err := os.WriteFile(piiPath, []byte(piiContent), 0644)
	require.NoError(t, err)

	filterPath := filepath.Join(tmpDir, "filter.yml")
	filterContent := `
default_action: accept
`
	err = os.WriteFile(filterPath, []byte(filterContent), 0644)
	require.NoError(t, err)

	summaryPath := filepath.Join(tmpDir, "custom_summary.yml")
	summaryContent := `
message_patterns:
  - prefix: "error:"
    field_name: "error_count"

region_timers:
  - category: "index"
    label: "do_read_index"
    time_field: "index_read_time"
`
	err = os.WriteFile(summaryPath, []byte(summaryContent), 0644)
	require.NoError(t, err)

	cfg := createMinimalValidConfig()
	cfg.PiiSettingsPath = piiPath
	cfg.FilterSettingsPath = filterPath
	cfg.SummaryPath = summaryPath

	err = cfg.Validate()
	assert.NoError(t, err)
	assert.NotNil(t, cfg.piiSettings)
	assert.NotNil(t, cfg.filterSettings)
	assert.NotNil(t, cfg.summary)
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
