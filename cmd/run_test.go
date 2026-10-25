package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRunCaptured_Help(t *testing.T) {
	out, err := RunCaptured(context.Background(), []string{"wx", "--help"})
	if err != nil {
		t.Fatalf("help: %v", err)
	}
	if !strings.Contains(out, "--forecast") {
		t.Fatalf("help output missing --forecast:\n%s", out)
	}
	if !strings.Contains(out, "--json") {
		t.Fatalf("help output missing --json:\n%s", out)
	}
}

func TestRunCaptured_UnknownFlag(t *testing.T) {
	out, err := RunCaptured(context.Background(), []string{"wx", "--not-a-real-flag"})
	if err == nil {
		t.Fatal("expected an error for an unknown flag")
	}
	if !strings.Contains(out, "Incorrect Usage") && !strings.Contains(err.Error(), "flag") {
		t.Fatalf("stdout %q err %v", out, err)
	}
}

func TestRunCaptured_BlockedCallTimesOutWithoutLeak(t *testing.T) {
	prev := TestCaptureBody
	started := make(chan struct{})
	release := make(chan struct{})
	var stopOnce sync.Once
	stop := func() { stopOnce.Do(func() { close(release) }) }
	var calls int
	var mu sync.Mutex
	TestCaptureBody = func(ctx context.Context) error {
		mu.Lock()
		calls++
		n := calls
		mu.Unlock()
		if n == 1 {
			close(started)
			<-release
			fmt.Fprintln(os.Stdout, "SECRET-LATE")
			return nil
		}
		fmt.Fprintln(os.Stdout, "CALL-TWO")
		return nil
	}
	firstDone := make(chan struct{})
	var firstOut string
	var firstErr error
	t.Cleanup(func() {
		stop()
		<-firstDone
		TestCaptureBody = prev
	})
	go func() {
		defer close(firstDone)
		firstOut, firstErr = RunCaptured(context.Background(), []string{"wx", "first"})
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("first capture did not start")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	type captured struct {
		out     string
		err     error
		elapsed time.Duration
	}
	secondCh := make(chan captured, 1)
	go func() {
		start := time.Now()
		out, err := RunCaptured(ctx, []string{"wx", "second"})
		secondCh <- captured{out: out, err: err, elapsed: time.Since(start)}
	}()
	var second captured
	select {
	case second = <-secondCh:
	case <-time.After(2 * time.Second):
		stop()
		<-firstDone
		t.Fatal("second call did not return while the first capture was held")
	}
	secondOut, secondErr, secondElapsed := second.out, second.err, second.elapsed

	stop()
	select {
	case <-firstDone:
	case <-time.After(2 * time.Second):
		t.Fatal("first capture did not finish")
	}

	thirdCtx, thirdCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer thirdCancel()
	thirdOut, thirdErr := RunCaptured(thirdCtx, []string{"wx", "third"})

	if secondElapsed > time.Second {
		t.Fatalf("second call took %s while the first capture was held", secondElapsed)
	}
	if !errors.Is(secondErr, context.DeadlineExceeded) {
		t.Fatalf("second err = %v", secondErr)
	}
	if secondOut != "" {
		t.Fatalf("second stdout = %q", secondOut)
	}
	if firstErr != nil {
		t.Fatalf("first err = %v", firstErr)
	}
	if !strings.Contains(firstOut, "SECRET-LATE") {
		t.Fatalf("first stdout = %q", firstOut)
	}
	if thirdErr != nil {
		t.Fatalf("third err = %v", thirdErr)
	}
	if strings.Contains(thirdOut, "SECRET") {
		t.Fatalf("late stdout leaked: %q", thirdOut)
	}
	if !strings.Contains(thirdOut, "CALL-TWO") {
		t.Fatalf("third stdout = %q", thirdOut)
	}
}

func TestCommandContext_BackgroundWhenMissing(t *testing.T) {
	ctx := commandContext(nil)
	if ctx == nil {
		t.Fatal("nil context")
	}
	if err := ctx.Err(); err != nil {
		t.Fatalf("background context is done: %v", err)
	}
}
