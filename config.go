package trace2receiver

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v2"
)

// `Config` represents the complete configuration settings for
// an individual receiver declaration from the `config.yaml`.
//
// These fields must be public (start with capital letter) so
// that the generic code in the collector can find them.
//
// We have different types of OS-specific paths where we listen
// for Trace2 telemetry.  We allow both types in a single config
// file, so that we can share it between clients; only the
// correct one for the platform will actually be used.
type Config struct {
	// On Windows, this is a named pipe.  The canonical form is
	// (the backslash spelling of) `//./pipe/<pipename>`.
	//
	// `CreateNamedPipeW()` documents that named pipes can only be
	// created on the local NPFS and must use `.` rather than a
	// general UNC hostname.  (Subsequent clients can connect to
	// a remote pipe, but a server can only CREATE a local one.
	//
	// Therefore, we allow the pipename to be abbreviated in the
	// `config.yaml` as just `<pipename>` and assume the prefix.
	//
	// This config file field is ignored on non-Windows platforms.
	NamedPipePath string `mapstructure:"pipe"`

	// On Unix, this is a Unix domain socket.  This is an absolute
	// or relative pathname on the local file system.  To avoid
	// confusion with the existing Git Trace2 setup, we allow this
	// to be of the form `af_unix:[<mode>:]<pathname>` and strip
	// off the prefix.
	//
	// This config file field is ignored on Windows platforms.
	UnixSocketPath string `mapstructure:"socket"`

	// Allow command and control verbs to be embedded in the Trace2
	// data stream.
	AllowCommandControlVerbs bool `mapstructure:"enable_commands"`

	// Pathname to YML file containing PII settings.
	PiiSettingsPath string `mapstructure:"pii_settings"`
	PiiSettings     *PiiSettings
}

// `Validate()` checks if the receiver configuration is valid.
//
// This function is called once for each `trace2receiver[/<qualifier>]:`
// declaration (in the top-level `receivers:` section).
//
// The file format and the customer collector framework
// allows more than one instance of a `trace2receiver` to be
// defined (presumably with different source types, pathnames,
// or verbosity) and run concurrently within this process.
// See: https://opentelemetry.io/docs/collector/configuration/
//
// A receiver declaration does not imply that it will actually
// be instantiated (realized) in the factory.  The receiver
// declaration causes a `cfg *Config` to be instantiated and
// that's it.  (The instantiation in the factory is controlled
// by the `service.pipelines.traces.receivers:` array.)
func (cfg *Config) Validate() error {

	var path string
	var err error

	if runtime.GOOS == "windows" {
		if len(cfg.NamedPipePath) == 0 {
			return fmt.Errorf("receivers.trace2receiver.pipe not defined")
		}
		path, err = normalize_named_pipe_path(cfg.NamedPipePath)
		if err != nil {
			return fmt.Errorf("receivers.trace2receiver.pipe invalid: '%s'",
				err.Error())
		}
		cfg.NamedPipePath = path
	} else {
		if len(cfg.UnixSocketPath) == 0 {
			return fmt.Errorf("receivers.trace2receiver.socket not defined")
		}
		path, err = normalize_uds_path(cfg.UnixSocketPath)
		if err != nil {
			return fmt.Errorf("receivers.trace2receiver.socket invalid: '%s'",
				err.Error())
		}
		cfg.UnixSocketPath = path
	}

	if len(cfg.PiiSettingsPath) > 0 {
		data, err := os.ReadFile(cfg.PiiSettingsPath)
		if err != nil {
			return fmt.Errorf("receivers.pii_settings could not read '%s': '%s'",
				cfg.PiiSettingsPath, err.Error())
		}
		cfg.PiiSettings = new(PiiSettings)
		err = yaml.Unmarshal(data, cfg.PiiSettings)
		if err != nil {
			return fmt.Errorf("receivers.pii_settings could not parse '%s': '%s'",
				cfg.PiiSettingsPath, err.Error())
		}
	}

	return nil
}

// Require (the backslash spelling of) `//./pipe/<pipename>` but allow
// `<pipename>` as an alias for the full spelling.  Complain if given a
// regular UNC or drive letter pathname.
func normalize_named_pipe_path(in string) (string, error) {

	in_lower := strings.ToLower(in)      // normalize to lowercase
	in_slash := filepath.Clean(in_lower) // normalize to backslashes
	if strings.HasPrefix(in_slash, `\\.\pipe\`) {
		// We were given a NPFS path.  Use the original as is.
		return in, nil
	}

	if strings.HasPrefix(in_slash, `\\`) {
		// We were given a general UNC path.  Reject it.
		return "", fmt.Errorf(`expect '[\\.\pipe\]<pipename>'`)
	}

	if len(in) > 2 && in[1] == ':' {
		// We have a drive letter. Reject it.
		return "", fmt.Errorf(`expect '[\\.\pipe\]<pipename>'`)
	}

	// We cannot use `filepath.VolumeName()` or `filepath.Abs()`
	// because they will be interpreted relative to the CWD
	// which is not on the NPFS.
	//
	// So assume that this relative path is a shortcut and join it
	// with our required prefix.

	out := filepath.Join(`\\.\pipe`, in)
	return out, nil
}

// Pathnames for Unix domain sockets are just normal Unix
// pathnames.  However, we do allow an optional `af_unix:`
// or `af_unix:stream:` prefix.  (This helps if they set it
// to the value of the GIT_TRACE2_EVENT string, which does
// require the prefix.)
func normalize_uds_path(in string) (string, error) {

	p, found := strings.CutPrefix(in, "af_unix:stream:")
	if found {
		return p, nil
	}

	_, found = strings.CutPrefix(in, "af_unix:dgram:")
	if found {
		return "", fmt.Errorf("SOCK_DGRAM sockets are not supported")
	}

	p, found = strings.CutPrefix(in, "af_unix:")
	if found {
		return p, nil
	}

	return in, nil
}
