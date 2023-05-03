package trace2receiver

import (
	"runtime/debug"
	"strings"
)

// Automatically set our version string using the (usually canonical)
// semantic version tag specified in a `require` in the `go.mod` of the
// executable into which we are linked.  https://go.dev/ref/mod#vcs-find
//
// Note that modules are consumed in source form, rather than as binary
// artifacts, so we cannot force the consumer to build with `-ldflags`
// to set `-X '<var>=<ver>` in their build scripts.
//
// Also, we don't want to force a manual update of a hard-coded constant
// when we create a tag; that is too easy to forget.
//
// Since GOLANG bakes this information into the binary, let's extract
// it and use it.  See also: `git version -m <exe>`.
//
// We should always create a tag of the form `v<i>.<j>.<k>` when we
// make a release.  And let the consumer ask for it by name.  We set
// the default here in case we are not consumed as a module, such as
// in our local unit tests.

var Trace2ReceiverVersion string = "v0.0.0-unset"

func init() {
	if bi, ok := debug.ReadBuildInfo(); ok {
		for k := range bi.Deps {
			p := bi.Deps[k].Path
			if strings.Contains(p, "trace2receiver") {
				Trace2ReceiverVersion = bi.Deps[k].Version
				return
			}
		}
	}
}
