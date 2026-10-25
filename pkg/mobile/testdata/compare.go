// Command compare reports whether two wx JSON documents match.
// Clock-only fields (age, age_seconds, fetched_at) are removed first.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: compare cli.json bridge.json")
		os.Exit(2)
	}
	a, err := normalizeFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "cli: %v\n", err)
		os.Exit(1)
	}
	b, err := normalizeFile(os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "bridge: %v\n", err)
		os.Exit(1)
	}
	if a != b {
		fmt.Fprintf(os.Stderr, "CLI and bridge JSON differ\n--- cli ---\n%s\n--- bridge ---\n%s\n", a, b)
		os.Exit(1)
	}
}

func normalizeFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return "", fmt.Errorf("%s is empty", path)
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return "", fmt.Errorf("%s: %w\n%s", path, err, data)
	}
	v = dropClock(v)
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func dropClock(v any) any {
	switch t := v.(type) {
	case map[string]any:
		delete(t, "age")
		delete(t, "age_seconds")
		delete(t, "fetched_at")
		for k, child := range t {
			t[k] = dropClock(child)
		}
		return t
	case []any:
		for i := range t {
			t[i] = dropClock(t[i])
		}
		return t
	default:
		return v
	}
}
