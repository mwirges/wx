package cmd

import (
	"context"
	"strings"
	"testing"
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

func TestCommandContext_BackgroundWhenMissing(t *testing.T) {
	ctx := commandContext(nil)
	if ctx == nil {
		t.Fatal("nil context")
	}
	if err := ctx.Err(); err != nil {
		t.Fatalf("background context is done: %v", err)
	}
}
