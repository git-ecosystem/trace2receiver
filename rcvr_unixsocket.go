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
	"time"

	"go.opentelemetry.io/collector/component"
	"golang.org/x/sys/unix"
)

// `Rcvr_UnixSocket` implements the `component.TracesReceiver` (aka `component.Receiver`
// (aka `component.Component`)) interface.
type Rcvr_UnixSocket struct {
	// These fields should be set in ctor()
	Base       *Rcvr_Base
	SocketPath string

	// Unix socket properties
	listener   *net.UnixListener
	inode      uint64
	mutex      sync.Mutex
	isShutdown bool
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
		rcvr.Base.Settings.TelemetrySettings.ReportStatus(component.NewFatalErrorEvent(err))
		return err
	}

	go rcvr.listenLoop()
	return nil
}

// Stop accepting new connections from Trace2 clients.
//
// This is part of the `component.Component` interface.
func (rcvr *Rcvr_UnixSocket) Shutdown(context.Context) error {
	rcvr.mutex.Lock()
	rcvr.isShutdown = true

	if rcvr.inode != 0 {
		// Only unlink the socket if we think we still own it.
		os.Remove(rcvr.SocketPath)
		rcvr.inode = 0
	}

	rcvr.listener.Close()
	rcvr.Base.cancel()

	rcvr.mutex.Unlock()
	return nil
}

type SocketPathnameStolenError struct {
	Pathname string
	SubErr   error
}

func NewSocketPathnameStolenError(pathname string, err error) error {
	return &SocketPathnameStolenError{
		Pathname: pathname,
		SubErr:   err,
	}
}

func (e *SocketPathnameStolenError) Error() string {
	if e.SubErr != nil {
		return fmt.Sprintf("Socket pathname stolen: '%v' error: '%s'",
			e.Pathname, e.SubErr.Error())
	} else {
		return fmt.Sprintf("Socket pathname stolen: '%v'", e.Pathname)
	}
}

type SocketInodeChangedError struct {
	InodeExpected uint64
	InodeObserved uint64
}

func NewSocketInodeChangedError(ie uint64, io uint64) error {
	return &SocketInodeChangedError{
		InodeExpected: ie,
		InodeObserved: io,
	}
}

func (e *SocketInodeChangedError) Error() string {
	return fmt.Sprintf("Inode changed: expected %v observed %v", e.InodeExpected, e.InodeObserved)
}

func get_inode(path string) (uint64, error) {
	var stat unix.Stat_t
	err := unix.Lstat(path, &stat)
	if err != nil {
		return 0, err
	}

	return stat.Ino, nil
}

// Open the server-side of a Unix domain socket.
func (rcvr *Rcvr_UnixSocket) openSocketForListening() error {
	var err error

	rcvr.mutex = sync.Mutex{}

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
	// NOTE On Unix (in addition to deleting a dead socket) we can
	// accidentally delete an active socket (currently being serviced
	// by another process) by mistake.  We cannot tell the difference
	// (without a race-prone client-connection).  Our unlink() WILL NOT
	// notify the other process; it just decrements the link-count in
	// the file system, but does not invalidate the fd in the other
	// process (just like you can unlink() a file that someone is still
	// writing and it won't actually be deleted until they close their
	// file descriptor).  In this case, the other daemon process will
	// be effectively orphaned -- listening on a socket that no one can
	// connect to.  This is a basic Unix problem and not specific to
	// OTEL Collectors or our receiver component.
	//
	// We will capture the inode of the socket that we create here and
	// add periodically verify in the listener loop's that the socket
	// still exists and has the same inode.
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

	// By default the unixsock_posix code unlinks the socket
	// so that we don't have dead sockets in the file system.
	// This works when GO completely controls the environment,
	// but can cause problems if/when another process steals
	// the socket pathname from us -- we don't want our cleanup
	// to delete their socket.
	rcvr.listener.SetUnlinkOnClose(false)

	rcvr.inode, err = get_inode(rcvr.SocketPath)
	if err != nil {
		rcvr.Base.Logger.Error(fmt.Sprintf("could not lstat created socket: %v", err))
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

	rcvr.Base.Logger.Info(fmt.Sprintf("listening on socket '%s' at '%v'", rcvr.SocketPath, rcvr.inode))
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
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		defer wg.Done()
	LOOP:
		for {
			select {
			case <-rcvr.Base.ctx.Done():
				// The main collector is requesting that we shutdown.
				// Force close the socket so that current and
				// futuer calls to `AcceptUnix()` will fail.
				rcvr.Base.Logger.Info("ctx.Done signalled")
				rcvr.listener.Close()
				break LOOP
			case <-doneListening:
				break LOOP
			case <-ticker.C:
				rcvr.mutex.Lock()
				if rcvr.isShutdown || rcvr.inode == 0 {
					// If Shutdown has already been called, then we don't
					// want our timeout handler to touch anything and
					// especially to not signal a new socket-stolen error
					// (because the shutdown code just deleted it). We only
					// want to throw a socket-stolen error if something
					// happened in the filesystem behind our back.
					ticker.Stop()
					rcvr.mutex.Unlock()
					break LOOP
				}
				// See if the socket inode was changed by external events.
				inode, err := get_inode(rcvr.SocketPath)
				if err != nil {
					// We could not lstat() our socket, assume it
					// has been deleted and/or stolen and give up.
					// (We could check the error code to be more
					// precise, but we'll probably do the same thing
					// in all cases anyway.)
					errStolen := NewSocketPathnameStolenError(rcvr.SocketPath, err)
					rcvr.Base.Logger.Error(errStolen.Error())

					rcvr.inode = 0

					rcvr.Base.Settings.TelemetrySettings.ReportStatus(component.NewFatalErrorEvent(errStolen))
					rcvr.mutex.Unlock()
					break LOOP
				}
				if inode != rcvr.inode {
					// Someone stole the pathname to the socket and
					// created a different file/socket on the path.
					// So we will never see another connection on our
					// (still functional) socket.  We should give up
					// and shutdown (without deleting the new socket
					// instance; ours should magically go away when
					// we close our file descriptor).
					errChanged := NewSocketInodeChangedError(rcvr.inode, inode)
					errStolen := NewSocketPathnameStolenError(rcvr.SocketPath, errChanged)
					rcvr.Base.Logger.Error(errStolen.Error())

					rcvr.inode = 0

					rcvr.Base.Settings.TelemetrySettings.ReportStatus(component.NewFatalErrorEvent(errStolen))
					rcvr.mutex.Unlock()
					break LOOP
				}
				rcvr.mutex.Unlock()
			}
		}
	}()

	for {
		conn, err := rcvr.listener.AcceptUnix()
		if err == nil {
			worker_id++
			go rcvr.worker(conn, worker_id)
			continue
		}

		rcvr.mutex.Lock()
		if rcvr.isShutdown || rcvr.inode == 0 {
			// We already know why the accept() failed because we closed
			// the socket, so don't bother with any error messages.
			rcvr.mutex.Unlock()
			break
		}
		if errors.Is(err, net.ErrClosed) {
			// (This may not be possible any now because of refactorings.)
			// Another thread closed our socket fd before Shutdown() was
			// called.  (Or ctx.Done() was signalled and our helper thread
			// closed the socket.) Either way, we don't want to throw up
			// a socket-stolen error because whatever did the close may
			// still be in-progress and we don't want to get a second call
			// to ReportComponentStatus() going.
			rcvr.Base.Logger.Error(fmt.Sprintf("XXX: %v", err))
			rcvr.mutex.Unlock()
			break
		}
		// Normal accept() errors do happen from time to time. Perhaps
		// the client hung up before we could service this connection.
		rcvr.Base.Logger.Error(err.Error())
		rcvr.mutex.Unlock()
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
