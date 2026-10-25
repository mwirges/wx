package cmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"sync"

	"github.com/urfave/cli/v2"
)

// captureMu serializes stdout swaps. Command output is written to os.Stdout,
// so two captures in one process cannot run at the same time.
var captureMu sync.Mutex

// RunCaptured runs the wx CLI with args (argv[0] is the program name) and
// returns everything the command wrote to stdout. The process does not exit.
// ctx cancels the command when the caller sets a deadline.
func RunCaptured(ctx context.Context, args []string) (stdout string, err error) {
	if ctx == nil {
		ctx = context.Background()
	}

	captureMu.Lock()
	defer captureMu.Unlock()

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

	app := NewApp()
	app.Writer = w
	app.ErrWriter = os.Stderr
	err = app.RunContext(ctx, args)
	closePipe()
	stdout = buf.String()
	return stdout, err
}
