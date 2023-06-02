//go:build !windows
// +build !windows

package trace2receiver

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"

	"go.opentelemetry.io/collector/component"
)

// `Rcvr_UnixSocket` implements the `component.TracesReceiver` (aka `component.Receiver`
// (aka `component.Component`)) interface.
type Rcvr_UnixSocket struct {
	// These fields should be set in ctor()
	Base       *Rcvr_Base
	SocketPath string

	// Unix socket properties
	listener *net.UnixListener
}

// Start receiving connections from Trace2 clients.
//
// This is part of the `component.Component` interface.
func (rcvr *Rcvr_UnixSocket) Start(unused_ctx context.Context, host component.Host) error {
	var err error

	err = rcvr.Base.Start(unused_ctx, host)
	if err != nil {
		return err
	}

	err = rcvr.openSocketForListening()
	if err != nil {
		host.ReportFatalError(err)
		return err
	}

	go rcvr.listenLoop()
	return nil
}

// Stop accepting new connections from Trace2 clients.
//
// This is part of the `component.Component` interface.
func (rcvr *Rcvr_UnixSocket) Shutdown(context.Context) error {
	rcvr.listener.Close()
	os.Remove(rcvr.SocketPath)
	rcvr.Base.cancel()
	return nil
}

// Open the server-side of a Unix domain socket.
func (rcvr *Rcvr_UnixSocket) openSocketForListening() error {
	var err error

	// The `listen(2)` system call must create the unix domain socket
	// in the file system.  If the pathname already exists on disk,
	// the listen() call will fail.
	//
	// However, we do not know whether it is a dead socket or another
	// process is currently servicing it.
	//
	// Force delete it under the assumption that the socket is dead
	// and was not properly cleaned up by the previous daemon.
	//
	// TODO On Unix we can delete an active socket (being serviced
	// by another process) and it may not notify the other process
	// (because we just called `unlink(2)`, but that just decrements
	// the link-count, but doesn't invalidate the `fd` in the other
	// process).  So that process may be effectively orphaned --
	// listening on a socket that no one can connect to.  We should
	// add code in the listener loop's subordinate thread to watch
	// for the path to disappear or for the inode associated with
	// the path to change and automatically shut down.
	_ = os.Remove(rcvr.SocketPath)

	// There are 3 types of Unix Domain Sockets: SOCK_STREAM, SOCK_DGRAM,
	// and SOCK_SEQPACKET.  Git Trace2 supports the first two.  However,
	// We're only going to support the first.  This corresponds to the
	// "af_unix:<path>" or "af_unix:stream:<path>" values for `GIT_TRACE2_EVENT`
	// environment variable or the `trace2.eventtarget` config value.
	//
	// Note: In the C# .Net Core class libraries on Unix, the NamedPipe
	// classes are implemented using SOCK_STREAM Unix Domain Sockets
	// under the hood.
	//
	// So limiting ourselves here to SOCK_STREAM is fine.
	//
	rcvr.listener, err = net.ListenUnix("unix",
		&net.UnixAddr{Name: rcvr.SocketPath,
			Net: "unix"})
	if err != nil {
		rcvr.Base.Logger.Error(fmt.Sprintf("could not create socket: %v", err))
		return err
	}

	// The UserId of the service process might be controlled by
	// the installer, /bin/launchctl, or an OS service manager.
	// We need the socket to be world writable in case the service
	// gets started as a privileged user so that ordinary Git
	// commands can write to it.  (Git silently fails if it gets
	// a permission error and just turns off telemetry in its
	// proceess.)
	os.Chmod(rcvr.SocketPath, 0666)

	rcvr.Base.Logger.Info(fmt.Sprintf("listening on '%s'", rcvr.SocketPath))
	return nil
}

// Listen for incoming connections from Trace2 clients.
// Dispatch each to a worker thread.
func (rcvr *Rcvr_UnixSocket) listenLoop() {
	var wg sync.WaitGroup
	var worker_id uint64

	doneListening := make(chan bool, 1)

	// Create a subordinate thread to watch for `context.cancelFunc`
	// being called by another thread.  We need to interrupt our
	// (blocking) call to `AcceptUnix()` in this thread and start
	// shutting down.
	//
	// However, we don't want to leak this subordinate thread if
	// our loop terminates for other reasons.
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-rcvr.Base.ctx.Done():
			// Force close the socket so that current and
			// futuer calls to `AcceptUnix()` will fail.
			rcvr.listener.Close()
		case <-doneListening:
		}
	}()

	for {
		conn, err := rcvr.listener.AcceptUnix()
		if errors.Is(err, net.ErrClosed) {
			break
		} else if err != nil {
			// Perhaps the client hung up before
			// we could service this connection.
			rcvr.Base.Logger.Error(err.Error())
		} else {
			worker_id++
			go rcvr.worker(conn, worker_id)
		}
	}

	// Tell the subordinate thread that we are finished accepting
	// connections so it can go away now.  This must not block
	// (because the subordinate may already be one (which is the
	// case if the `context.cancelFunc` was called)).
	doneListening <- true

	wg.Wait()
}

func (rcvr *Rcvr_UnixSocket) worker(conn *net.UnixConn, worker_id uint64) {
	var haveError = false
	var wg sync.WaitGroup
	defer conn.Close()

	doneReading := make(chan bool, 1)

	// Create a subordinate thread to watch for `context.cancelFunc`
	// being called by another thread.  We need to interrupt our
	// (blocking) call to `ReadBytes()` in this worker and (maybe)
	// let it emit partial results (if it can do so quickly).
	//
	// However, we don't want to leak this subordinate thread if this
	// worker normally finishes reading all the data from the client
	// Git command.
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-rcvr.Base.ctx.Done():
			// Force close the connection from the client to
			// help keep the Git command from getting stuck.
			// That is, let it get a clean write-error rather
			// than blocking on a buffer that we'll never
			// read.  (It might not actually matter, but it
			// doesn't hurt.)
			//
			// This will also cause the worker's `ReadBytes()`
			// to return an error, so that the worker can
			// terminate the loop.
			conn.Close()
		case <-doneReading:
		}
	}()

	// We assume that a `worker` represents the server side of a connection
	// from a single Git client.  That is, all events that we receive over
	// this connection are from the same process (and will therefore have
	// the same Trace2 SID).  That is, we don't have to maintain a SID to
	// Dataset mapping.
	tr2 := NewTrace2Dataset(rcvr.Base)

	tr2.pii_gather(rcvr.Base.RcvrConfig, conn)

	r := bufio.NewReader(conn)
	for {
		rawLine, err := r.ReadBytes('\n')
		if err == io.EOF {
			break
		}
		if errors.Is(err, net.ErrClosed) {
			break
		}
		if err != nil {
			rcvr.Base.Logger.Error(err.Error())
			haveError = true
			break
		}

		if processRawLine(rawLine, tr2, rcvr.Base.Logger,
			rcvr.Base.RcvrConfig.AllowCommandControlVerbs) != nil {
			haveError = true
			break
		}
	}

	// Tell the subordinate thread that we are finished reading from
	// the client so it can go away now.  This must not block (because
	// the subordinate may already be gone (which is the case if the
	// `context.cancelFunc` was called)).
	doneReading <- true

	conn.Close()

	if !haveError {
		tr2.exportTraces()
	}

	// Wait for our subordinate thread to exit
	wg.Wait()
}
