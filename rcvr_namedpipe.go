//go:build windows
// +build windows

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

	"github.com/Microsoft/go-winio"
)

type Rcvr_NamedPipe struct {
	// These fields should be set in ctor()
	Base          *Rcvr_Base
	NamedPipePath string

	// Windows named pipe properties
	listener net.Listener
}

// Start receiving connections from Trace2 clients.
//
// This is part of the `component.Component` interface.
func (rcvr *Rcvr_NamedPipe) Start(unused_ctx context.Context, host component.Host) error {
	var err error

	err = rcvr.Base.Start(unused_ctx, host)
	if err != nil {
		return err
	}

	err = rcvr.openNamedPipeServer()
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
func (rcvr *Rcvr_NamedPipe) Shutdown(context.Context) error {
	rcvr.listener.Close()
	os.Remove(rcvr.NamedPipePath)
	rcvr.Base.cancel()
	return nil
}

// Open the server-side of a named pipe.
func (rcvr *Rcvr_NamedPipe) openNamedPipeServer() (err error) {
	_ = os.Remove(rcvr.NamedPipePath)

	c := winio.PipeConfig{
		SecurityDescriptor: "",
		MessageMode:        false,
		InputBufferSize:    65536,
		OutputBufferSize:   65536,
	}

	rcvr.listener, err = winio.ListenPipe(rcvr.NamedPipePath, &c)
	if err != nil || rcvr.listener == nil {
		rcvr.Base.Logger.Error(fmt.Sprintf("could not create named pipe: %v", err))
		return err
	}

	rcvr.Base.Logger.Info(fmt.Sprintf("listening on '%s'", rcvr.NamedPipePath))
	return nil
}

// Listen for incoming connections from Trace2 clients.
// Dispatch each to a worker thread.
func (rcvr *Rcvr_NamedPipe) listenLoop() {
	var wg sync.WaitGroup
	var worker_id uint64

	doneListening := make(chan bool, 1)

	// Create a subordinate thread to watch for `context.cancelFunc`
	// being called by another thread.  We need to interrupt our
	// (blocking) call to `Accept()` in this thread and start
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
			// futuer calls to `Accept()` will fail.
			rcvr.listener.Close()
		case <-doneListening:
		}
	}()

	for {
		conn, err := rcvr.listener.Accept()
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

func (rcvr *Rcvr_NamedPipe) worker(conn net.Conn, worker_id uint64) {
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
	tr2 := NewTrace2Dataset()

	var nrBytesRead int = 0

	r := bufio.NewReader(conn)
	for {
		rawLine, err := r.ReadBytes('\n')
		if err == io.EOF {
			if nrBytesRead == 0 {
				rcvr.Base.Logger.Debug(fmt.Sprintf("[dsid %06d] EOF after %d bytes", tr2.datasetId, nrBytesRead))
			}
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

		nrBytesRead += len(rawLine)

		err = rcvr.Base.processRawLine(rawLine, tr2)
		if err != nil {
			rcvr.Base.Logger.Error(err.Error())
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
		tr2.exportTraces(rcvr.Base)
	}

	// Wait for our subordinate thread to exit
	wg.Wait()
}
