# trace2receiver/internal/go-winio

This directory contains a stripped down fork of
[Microsoft/go-winio](https://github.com/microsoft/go-winio.git).
It only includes support for Windows named pipes; everything else
has been removed.

The `go-winio` repository was forked so that I could use the changes
proposed in my
[pipe: add server backlog for concurrent `Accept()` PR](https://github.com/microsoft/go-winio/pull/291)
without the complications of `replace` statements in the `go.mod` files
of generated custom collectors.

The `go-winio` repository was also forked to allow me maintain
reponsibility for the maintenance and support of the named pipe
routines.  It avoids adding a support load from the OTEL community
onto the team that nicely developed and donated this code.

I've included a brief summary of the code.  Please refer to the original
[Microsoft/go-winio/README](https://github.com/microsoft/go-winio/README.md)
for complete information on this code.

## `trace2receiver/internal/go-winio` Code Summary

This repository contains utilities for efficiently performing Win32 IO operations in
Go. Currently, this is focused on accessing named pipes and other file handles, and
for using named pipes as a net transport.

This code relies on IO completion ports to avoid blocking IO on system threads, allowing Go
to reuse the thread to schedule another goroutine. This limits support to Windows Vista and
newer operating systems. This is similar to the implementation of network sockets in Go's net
package.

## `trace2receiver/internal/go-winio` License

Please see the [LICENSE](./LICENSE) file for licensing information.

## `trace2receiver/internal/go-winio` Contributing

This project welcomes contributions and suggestions.
You can contribute to the forked version of `go-winio` using the instructions
in the `trace2receiver` root directory.  To contribute to the upstream version
of `go-winio` please see their contributing guidelines.

## Special Thanks

Thanks to [natefinch][natefinch] for the inspiration for this library.
See [npipe](https://github.com/natefinch/npipe) for another named pipe implementation.

[natefinch]: https://github.com/natefinch
