package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mwirges/wx/cmd"
)

// runTimeout is the deadline for one WxRunJSON call.
// Tests set a shorter value. The phone build keeps 15s.
var runTimeout = 15 * time.Second

// testDispatch, when set, replaces the CLI call. Production leaves it nil.
var testDispatch func(ctx context.Context, argv []string) (string, error)

// allowed mobile commands. forecast and alerts are root flags; the rest are
// subcommands. Anything else is rejected before the CLI runs.
var allowedCommands = map[string]struct{}{
	"hourly":  {},
	"nowcast": {},
	"radar":   {},
	"spc":     {},
	"climate": {},
	"tropics": {},
	"aqi":     {},
	"astro":   {},
}

// RunJSON runs one allow-listed command and always returns JSON.
// argsJSON is a JSON array of strings, for example ["forecast","New Haven, IN"].
// The location stays one argument. It is never split on whitespace.
func RunJSON(argsJSON string) (out string) {
	defer func() {
		if rec := recover(); rec != nil {
			out = errorJSON("panic", fmt.Sprint(rec))
		}
	}()
	return runJSON(argsJSON, runTimeout)
}

func runJSON(argsJSON string, timeout time.Duration) string {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	args, code, err := parseArgs(argsJSON)
	if err != nil {
		return errorJSON(code, err.Error())
	}
	argv, err := buildArgv(args)
	if err != nil {
		return errorJSON("unknown_command", err.Error())
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	type result struct {
		stdout   string
		err      error
		panicVal any
	}
	ch := make(chan result, 1)
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				ch <- result{panicVal: rec}
			}
		}()
		stdout, err := dispatch(ctx, argv)
		ch <- result{stdout: stdout, err: err}
	}()

	select {
	case res := <-ch:
		return finish(res.stdout, res.err, res.panicVal, ctx.Err(), timeout)
	case <-ctx.Done():
		res := <-ch
		if res.panicVal != nil {
			return errorJSON("panic", fmt.Sprint(res.panicVal))
		}
		return errorJSON("timeout", timeoutMessage(timeout))
	}
}

func dispatch(ctx context.Context, argv []string) (string, error) {
	if testDispatch != nil {
		return testDispatch(ctx, argv)
	}
	return cmd.RunCaptured(ctx, argv)
}

func finish(stdout string, err error, panicVal any, ctxErr error, timeout time.Duration) string {
	if panicVal != nil {
		return errorJSON("panic", fmt.Sprint(panicVal))
	}
	if ctxErr == context.DeadlineExceeded {
		return errorJSON("timeout", timeoutMessage(timeout))
	}
	if err != nil {
		msg := strings.TrimSpace(err.Error())
		if msg == "" {
			msg = "command failed"
		}
		return errorJSON("command_error", msg)
	}
	if strings.TrimSpace(stdout) == "" {
		return errorJSON("command_error", "command produced no output")
	}
	return stdout
}

func timeoutMessage(timeout time.Duration) string {
	return fmt.Sprintf("timed out after %s", timeout)
}

// parseArgs decodes a JSON array of strings.
// The code return is invalid_json or invalid_args.
func parseArgs(argsJSON string) ([]string, string, error) {
	trimmed := strings.TrimSpace(argsJSON)
	if trimmed == "" {
		return nil, "invalid_json", fmt.Errorf("expected a JSON array of strings")
	}
	var raw []json.RawMessage
	if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
		return nil, "invalid_json", fmt.Errorf("expected a JSON array of strings")
	}
	if len(raw) == 0 {
		return nil, "invalid_args", fmt.Errorf("expected a non-empty JSON array of strings")
	}
	args := make([]string, len(raw))
	for i, item := range raw {
		var s string
		if err := json.Unmarshal(item, &s); err != nil {
			return nil, "invalid_args", fmt.Errorf("argument %d is not a string", i)
		}
		args[i] = s
	}
	return args, "", nil
}

// buildArgv maps a mobile command onto the CLI argv that prints JSON.
// forecast and alerts are flags on the root command. The other names are
// subcommands. --json is added when the caller did not pass it.
func buildArgv(args []string) ([]string, error) {
	name := args[0]
	rest := args[1:]
	switch name {
	case "forecast":
		return appendJSON([]string{"wx", "--forecast"}, rest), nil
	case "alerts":
		return appendJSON([]string{"wx", "--alerts"}, rest), nil
	default:
		if _, ok := allowedCommands[name]; !ok {
			return nil, fmt.Errorf("unknown command %q", name)
		}
		return appendJSON([]string{"wx", name}, rest), nil
	}
}

func appendJSON(prefix, rest []string) []string {
	hasJSON := false
	for _, arg := range rest {
		if arg == "--json" || arg == "-j" {
			hasJSON = true
			break
		}
	}
	out := append([]string{}, prefix...)
	if !hasJSON {
		out = append(out, "--json")
	}
	return append(out, rest...)
}

func errorJSON(code, message string) string {
	payload := struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}{}
	payload.Error.Code = code
	payload.Error.Message = message
	b, err := json.Marshal(payload)
	if err != nil {
		return `{"error":{"code":"command_error","message":"failed to encode error"}}`
	}
	return string(b)
}
