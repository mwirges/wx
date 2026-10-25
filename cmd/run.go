package cmd

import (
	"bytes"
	"context"
	"io"
	"os"

	"github.com/urfave/cli/v2"
)

// captureGate is one token. A mutex cannot wake up when ctx ends, so a second
// WxRunJSON call would sit in Lock until the timed-out command finally returned.
// The token stays out until that command returns, which keeps its late stdout
// in its own pipe instead of the next call's.
var captureGate = make(chan struct{}, 1)

func init() {
	captureGate <- struct{}{}
}

// TestCaptureBody, when set, replaces the CLI while the capture gate and the
// stdout pipe are held. Tests use it to block a call or write captured output.
// Production leaves it nil.
var TestCaptureBody func(ctx context.Context) error

// RunCaptured runs the wx CLI with args (argv[0] is the program name) and
// returns everything the command wrote to stdout. The process does not exit.
// ctx cancels the command when the caller sets a deadline. Waiting for the
// capture gate stops when ctx ends.
func RunCaptured(ctx context.Context, args []string) (stdout string, err error) {
	if ctx == nil {
		ctx = context.Background()
	}

	select {
	case <-captureGate:
	case <-ctx.Done():
		return "", ctx.Err()
	}
	defer func() { captureGate <- struct{}{} }()

	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		return "", pipeErr
	}

	oldExit := cli.OsExiter
	cli.OsExiter = func(int) {}
	oldStdout := os.Stdout
	os.Stdout = w

	var buf bytes.Buffer
	readDone := make(chan struct{})
	go func() {
		_, _ = io.Copy(&buf, r)
		close(readDone)
	}()

	// Close the write end and restore stdout before reading buf. A deferred
	// restore still runs if RunContext panics, but the returned string is
	// filled only after the reader has drained the pipe.
	closed := false
	closePipe := func() {
		if closed {
			return
		}
		closed = true
		_ = w.Close()
		os.Stdout = oldStdout
		<-readDone
		_ = r.Close()
		cli.OsExiter = oldExit
	}
	defer closePipe()

	if TestCaptureBody != nil {
		err = TestCaptureBody(ctx)
	} else {
		app := NewApp()
		app.Writer = w
		app.ErrWriter = os.Stderr
		err = app.RunContext(ctx, args)
	}
	closePipe()
	stdout = buf.String()
	return stdout, err
}
