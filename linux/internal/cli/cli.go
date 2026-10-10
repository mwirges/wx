// Package cli runs the wx binary and returns its JSON.
// It does not interpret weather.
package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	weatherTimeout = 35 * time.Second
	radarTimeout   = 45 * time.Second
	extraTimeout   = 25 * time.Second
)

// CLIError is a failed wx invocation.
type CLIError struct{ Msg string }

func (e CLIError) Error() string { return e.Msg }

// Cancelled means the radar subprocess was stopped.
type Cancelled struct{}

func (Cancelled) Error() string { return "cancelled" }

// WxCLI runs one wx binary. Weather stays in that process.
type WxCLI struct {
	binary string
	procs  map[*exec.Cmd]struct{}
	epoch  int
	mu     sync.Mutex
}

// New uses binary when it is non-empty. Otherwise it searches WX_BINARY,
// the repo build, and PATH.
func New(binary string) *WxCLI {
	binary = strings.TrimSpace(binary)
	if binary == "" {
		binary = LocateBinary("")
	}
	return &WxCLI{binary: binary, procs: map[*exec.Cmd]struct{}{}}
}

func (c *WxCLI) lock()   { c.mu.Lock() }
func (c *WxCLI) unlock() { c.mu.Unlock() }

// Available reports whether a binary path was chosen.
func (c *WxCLI) Available() bool { return c.binary != "" }

// Binary is the path New resolved.
func (c *WxCLI) Binary() string { return c.binary }

// CancelRadar stops every tracked radar subprocess.
func (c *WxCLI) CancelRadar() {
	c.lock()
	c.epoch++
	procs := make([]*exec.Cmd, 0, len(c.procs))
	for proc := range c.procs {
		procs = append(procs, proc)
	}
	c.unlock()
	for _, proc := range procs {
		killGroup(proc)
	}
}

func (c *WxCLI) inflight() int {
	c.lock()
	defer c.unlock()
	return len(c.procs)
}

// FetchWeather runs wx --json --forecast --alerts.
func (c *WxCLI) FetchWeather(location, units string, hourly bool) (map[string]any, error) {
	out, errText, code, err := c.run(WeatherArgs(location, units, hourly), weatherTimeout, false, nil)
	if err != nil {
		return nil, err
	}
	return decode(out, errText, code, true)
}

// FetchRadar runs wx radar --json --loop. Raw frames are transparent and the
// desk has no map under them, so the CLI returns the composited picture.
func (c *WxCLI) FetchRadar(location, product string, radius float64, bbox string, frames int) (map[string]any, error) {
	c.CancelRadar()
	c.lock()
	epoch := c.epoch
	c.unlock()
	out, errText, code, err := c.run(RadarArgs(location, product, radius, bbox, false, true, frames), radarTimeout, true, &epoch)
	if err != nil {
		return nil, err
	}
	return decode(out, errText, code, false)
}

// ExportPNG runs wx radar --save. The CLI writes the file.
func (c *WxCLI) ExportPNG(path, location, product string, radius float64) (string, error) {
	_, errText, code, err := c.run(ExportPNGArgs(path, location, product, radius), radarTimeout, true, nil)
	if err != nil {
		return "", err
	}
	if code != 0 {
		msg := strings.TrimSpace(errText)
		if msg == "" {
			msg = fmt.Sprintf("wx exited with status %d.", code)
		}
		return "", CLIError{msg}
	}
	return strings.TrimSpace(errText), nil
}

// ExportGIF runs wx radar --save-gif. The CLI writes the animation.
func (c *WxCLI) ExportGIF(path, location, product string, radius float64, frames int) (string, error) {
	_, errText, code, err := c.run(ExportGIFArgs(path, location, product, radius, frames, 500), radarTimeout, true, nil)
	if err != nil {
		return "", err
	}
	if code != 0 {
		msg := strings.TrimSpace(errText)
		if msg == "" {
			msg = fmt.Sprintf("wx exited with status %d.", code)
		}
		return "", CLIError{msg}
	}
	return strings.TrimSpace(errText), nil
}

// FetchOutlook runs wx outlook --json.
func (c *WxCLI) FetchOutlook(location string) (map[string]any, error) {
	return c.extra(OutlookArgs(location))
}

// FetchChase runs wx chase --list --json.
func (c *WxCLI) FetchChase() (map[string]any, error) { return c.extra(ChaseArgs()) }

// FetchClimate runs wx climate --json.
func (c *WxCLI) FetchClimate(location, units string) (map[string]any, error) {
	return c.extra(ClimateArgs(location, units))
}

// FetchHistory runs wx history --json.
func (c *WxCLI) FetchHistory(location, units string, days int) (map[string]any, error) {
	return c.extra(HistoryArgs(location, units, days))
}

// FetchTropics runs wx tropics --json.
func (c *WxCLI) FetchTropics(location, units, storm string) (map[string]any, error) {
	return c.extra(TropicsArgs(location, units, storm))
}

// FetchNowcast runs wx nowcast --json.
func (c *WxCLI) FetchNowcast(location, units string) (map[string]any, error) {
	return c.extra(NowcastArgs(location, units))
}

func (c *WxCLI) extra(args []string) (map[string]any, error) {
	out, errText, code, err := c.run(args, extraTimeout, false, nil)
	if err != nil {
		return nil, err
	}
	return decode(out, errText, code, false)
}

func (c *WxCLI) run(args []string, timeout time.Duration, tracked bool, epoch *int) ([]byte, string, int, error) {
	if c.binary == "" {
		return nil, "", 1, CLIError{"wx CLI binary not found. Build the project (`make build`) or set WX_BINARY."}
	}
	cmd := exec.Command(c.binary, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return nil, "", 1, CLIError{err.Error()}
	}
	if tracked {
		c.lock()
		if epoch != nil && c.epoch != *epoch {
			c.unlock()
			killGroup(cmd)
			waitCmd(cmd)
			return nil, "", -1, Cancelled{}
		}
		c.procs[cmd] = struct{}{}
		c.unlock()
		defer func() {
			c.lock()
			delete(c.procs, cmd)
			c.unlock()
		}()
	}
	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(timeout):
		killGroup(cmd)
		<-done
		return nil, "", 1, CLIError{fmt.Sprintf("wx timed out after %ds.", int(timeout.Seconds()))}
	}
	exit := 1
	if cmd.ProcessState != nil {
		exit = cmd.ProcessState.ExitCode()
	}
	if epoch != nil {
		c.lock()
		current := c.epoch
		c.unlock()
		if current != *epoch || exit < 0 {
			return nil, "", exit, Cancelled{}
		}
	}
	if tracked && exit < 0 {
		return nil, "", exit, Cancelled{}
	}
	return stdout.Bytes(), stderr.String(), exit, nil
}

func waitCmd(cmd *exec.Cmd) {
	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}
}

func killGroup(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	pid := cmd.Process.Pid
	_ = syscall.Kill(-pid, syscall.SIGTERM)
	go func() {
		time.Sleep(600 * time.Millisecond)
		_ = syscall.Kill(-pid, syscall.SIGKILL)
	}()
}

func decode(stdout []byte, stderr string, code int, requireConditions bool) (map[string]any, error) {
	text := strings.TrimSpace(string(stdout))
	if code != 0 && text == "" {
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = fmt.Sprintf("wx exited with status %d.", code)
		}
		return nil, CLIError{msg}
	}
	payload := map[string]any{}
	if text != "" {
		if err := json.Unmarshal([]byte(text), &payload); err != nil {
			return nil, CLIError{fmt.Sprintf("Failed to decode wx JSON: %v", err)}
		}
	}
	if requireConditions && payload["conditions"] == nil && code != 0 {
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = fmt.Sprintf("wx exited with status %d.", code)
		}
		return nil, CLIError{msg}
	}
	if requireConditions && payload["warning"] == nil {
		if warn := strings.TrimSpace(stderr); warn != "" {
			payload["warning"] = warn
		}
	}
	return payload, nil
}

// LocateBinary finds the wx executable. explicit wins, then WX_BINARY,
// then a sibling build/wx, then PATH, then the usual install paths.
func LocateBinary(explicit string) string {
	var candidates []string
	env := explicit
	if strings.TrimSpace(env) == "" {
		env = os.Getenv("WX_BINARY")
	}
	if strings.TrimSpace(env) != "" {
		candidates = append(candidates, strings.TrimSpace(env))
	}
	if exe, err := os.Executable(); err == nil {
		dir := exeDir(exe)
		candidates = append(candidates,
			dir+"/wx",
			dir+"/build/wx",
			dir+"/../build/wx",
		)
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, cwd+"/build/wx", cwd+"/../build/wx")
	}
	for _, dir := range strings.Split(os.Getenv("PATH"), ":") {
		if dir != "" {
			candidates = append(candidates, dir+"/wx")
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			home+"/bin/wx",
			home+"/go/bin/wx",
			home+"/.local/bin/wx",
		)
	}
	candidates = append(candidates, "/usr/local/bin/wx", "/usr/bin/wx")
	seen := map[string]bool{}
	for _, candidate := range candidates {
		if candidate == "" || seen[candidate] {
			continue
		}
		seen[candidate] = true
		if executableFile(candidate) {
			return candidate
		}
	}
	return ""
}

func exeDir(exe string) string {
	for i := len(exe) - 1; i >= 0; i-- {
		if exe[i] == '/' {
			return exe[:i]
		}
	}
	return "."
}

func executableFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Mode()&0o111 != 0
}

// WeatherArgs matches the Mac shell's wx invocation.
func WeatherArgs(location, units string, hourly bool) []string {
	args := []string{"--json", "--forecast", "--alerts"}
	if hourly {
		args = append(args, "--hourly", "--hours", "24")
	}
	if strings.TrimSpace(location) != "" {
		args = append(args, "--location", strings.TrimSpace(location))
	}
	if units != "" {
		args = append(args, "--units", units)
	}
	return args
}

// RadarArgs builds wx radar --json. The desk leaves raw off; pass raw only
// when another surface draws a map under the image.
func RadarArgs(location, product string, radius float64, bbox string, raw, loop bool, frames int) []string {
	args := []string{"radar", "--json"}
	if raw {
		args = append(args, "--raw")
	}
	if loop {
		if frames < 2 {
			frames = 2
		}
		args = append(args, "--loop", "--frames", fmt.Sprintf("%d", frames))
	}
	if bbox != "" {
		args = append(args, "--bbox", bbox)
	}
	if strings.TrimSpace(location) != "" {
		args = append(args, "--location", strings.TrimSpace(location))
	}
	if product != "" {
		args = append(args, "--product", product)
	}
	if bbox == "" && radius > 0 {
		args = append(args, "--radius", fmt.Sprintf("%.0f", radius))
	}
	return args
}

// ExportPNGArgs is wx radar --save. The shell does not encode the PNG.
func ExportPNGArgs(path, location, product string, radius float64) []string {
	args := []string{"radar", "--save", path}
	appendRadarTarget(&args, location, product, radius)
	return args
}

// ExportGIFArgs is wx radar --save-gif. The shell does not composite frames.
func ExportGIFArgs(path, location, product string, radius float64, frames, intervalMS int) []string {
	if frames < 2 {
		frames = 2
	}
	args := []string{"radar", "--save-gif", path, "--frames", fmt.Sprintf("%d", frames), "--interval", fmt.Sprintf("%d", intervalMS)}
	appendRadarTarget(&args, location, product, radius)
	return args
}

// OutlookArgs is wx outlook --json.
func OutlookArgs(location string) []string {
	args := []string{"outlook", "--json"}
	if strings.TrimSpace(location) != "" {
		args = append(args, "--location", strings.TrimSpace(location))
	}
	return args
}

// ChaseArgs is wx chase --list --json.
func ChaseArgs() []string { return []string{"chase", "--list", "--json"} }

// ClimateArgs is wx climate --json.
func ClimateArgs(location, units string) []string {
	args := []string{"climate", "--json"}
	if strings.TrimSpace(location) != "" {
		args = append(args, "--location", strings.TrimSpace(location))
	}
	if units != "" {
		args = append(args, "--units", units)
	}
	return args
}

// HistoryArgs is wx history --json.
func HistoryArgs(location, units string, days int) []string {
	args := []string{"history", "--json", "--days", fmt.Sprintf("%d", days)}
	if strings.TrimSpace(location) != "" {
		args = append(args, "--location", strings.TrimSpace(location))
	}
	if units != "" {
		args = append(args, "--units", units)
	}
	return args
}

// TropicsArgs is wx tropics --json.
func TropicsArgs(location, units, storm string) []string {
	args := []string{"tropics", "--json"}
	if strings.TrimSpace(location) != "" {
		args = append(args, "--location", strings.TrimSpace(location))
	}
	if strings.TrimSpace(storm) != "" {
		args = append(args, "--storm", strings.TrimSpace(storm))
	}
	if units != "" {
		args = append(args, "--units", units)
	}
	return args
}

// NowcastArgs is wx nowcast --json.
func NowcastArgs(location, units string) []string {
	args := []string{"nowcast", "--json"}
	if strings.TrimSpace(location) != "" {
		args = append(args, "--location", strings.TrimSpace(location))
	}
	if units != "" {
		args = append(args, "--units", units)
	}
	return args
}

func appendRadarTarget(args *[]string, location, product string, radius float64) {
	if strings.TrimSpace(location) != "" {
		*args = append(*args, "--location", strings.TrimSpace(location))
	}
	if product != "" {
		*args = append(*args, "--product", product)
	}
	if radius > 0 {
		*args = append(*args, "--radius", fmt.Sprintf("%.0f", radius))
	}
}

// IsCancel reports a cancelled radar fetch.
func IsCancel(err error) bool {
	if err == nil {
		return false
	}
	var cancelled Cancelled
	if errors.As(err, &cancelled) {
		return true
	}
	return err.Error() == "cancelled"
}
