package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

const (
	lockFileName = "mcp.lock"
	pidFileName  = "mcp.pid"
)

// errRefusePID is returned when a helper is asked to touch pid 0, -1, or 1.
// Those values mean "this process", "every process", or init. Callers must
// not pass them to a signal.
var errRefusePID = errors.New("mcp: refusing to signal pid <= 1")

// PIDRecord is the mcp.pid JSON document. It names the serve process that
// acquired the lock. It is not a license to signal an arbitrary pid.
type PIDRecord struct {
	PID   int      `json:"pid"`
	Exe   string   `json:"exe"`
	Argv0 string   `json:"argv0,omitempty"`
	Args  []string `json:"args"`
}

// ObservedProcess is what the OS says about a pid right now.
type ObservedProcess struct {
	PID     int
	Alive   bool
	ExePath string
	Command string
}

// Lock is an exclusive flock plus the pidfile written by this process.
// Release removes the pidfile only when it still names this process.
type Lock struct {
	f       *os.File
	pidPath string
	pid     int
}

// RuntimePaths returns mcp.lock and mcp.pid inside dir.
func RuntimePaths(dir string) (lockPath, pidPath string) {
	return filepath.Join(dir, lockFileName), filepath.Join(dir, pidFileName)
}

// Acquire takes the exclusive lock in dir and writes mcp.pid for this process.
// A live verified wx mcp serve already recorded here returns ErrAlreadyRunning
// and is not signaled. A stale or foreign pidfile is replaced and that pid
// is not signaled. The flock is the single-instance gate shared by the CLI
// and the Mac app.
func Acquire(dir string) (*Lock, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, errors.New("mcp: empty runtime dir")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("mcp: runtime dir: %w", err)
	}
	lockPath, pidPath := RuntimePaths(dir)
	if rec, err := ReadPIDFile(pidPath); err == nil {
		if obs, oerr := Observe(rec.PID); oerr == nil && MaySignal(rec, obs) == nil {
			return nil, fmt.Errorf("%w (pid %d)", ErrAlreadyRunning, rec.PID)
		}
	}

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("mcp: open lock: %w", err)
	}
	if err := os.Chmod(lockPath, 0o600); err != nil {
		f.Close()
		return nil, fmt.Errorf("mcp: lock mode: %w", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return nil, fmt.Errorf("%w: %s", ErrAlreadyRunning, lockPath)
	}

	// Recheck after the lock so two starters cannot both pass the pre-check.
	if rec, err := ReadPIDFile(pidPath); err == nil && rec.PID != os.Getpid() {
		if obs, oerr := Observe(rec.PID); oerr == nil && MaySignal(rec, obs) == nil {
			_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
			f.Close()
			return nil, fmt.Errorf("%w (pid %d)", ErrAlreadyRunning, rec.PID)
		}
	}

	exe, _ := os.Executable()
	if resolved, err := filepath.EvalSymlinks(exe); err == nil && resolved != "" {
		exe = resolved
	}
	rec := PIDRecord{
		PID:   os.Getpid(),
		Exe:   exe,
		Argv0: os.Args[0],
		Args:  append([]string(nil), os.Args[1:]...),
	}
	if err := writePIDFile(pidPath, rec); err != nil {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
		return nil, err
	}
	return &Lock{f: f, pidPath: pidPath, pid: rec.PID}, nil
}

// Release drops the flock and removes mcp.pid when it still names us.
func (l *Lock) Release() error {
	if l == nil || l.f == nil {
		return nil
	}
	if rec, err := ReadPIDFile(l.pidPath); err == nil && rec.PID == l.pid {
		_ = os.Remove(l.pidPath)
	}
	uerr := syscall.Flock(int(l.f.Fd()), syscall.LOCK_UN)
	cerr := l.f.Close()
	l.f = nil
	if uerr != nil {
		return uerr
	}
	return cerr
}

// ReadPIDFile parses mcp.pid. A pid <= 1 is refused.
func ReadPIDFile(path string) (PIDRecord, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return PIDRecord{}, err
	}
	var rec PIDRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return PIDRecord{}, fmt.Errorf("mcp: pidfile: %w", err)
	}
	if rec.PID <= 1 {
		return PIDRecord{}, fmt.Errorf("%w: pidfile pid %d", errRefusePID, rec.PID)
	}
	return rec, nil
}

func writePIDFile(path string, rec PIDRecord) error {
	if rec.PID <= 1 {
		return fmt.Errorf("%w: not writing pid %d", errRefusePID, rec.PID)
	}
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("mcp: pidfile temp: %w", err)
	}
	_, werr := f.Write(data)
	cerr := f.Close()
	if werr != nil || cerr != nil {
		_ = os.Remove(tmp)
		if werr != nil {
			return werr
		}
		return cerr
	}
	if err := os.Chmod(tmp, 0o600); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Chmod(path, 0o600)
}

// MaySignal reports whether it is safe to signal recorded.PID.
// It returns nil only when the pid is > 1, still alive, the executable is
// wx or wx-cli, and the command line is that binary running `mcp serve`.
// It does not send a signal.
func MaySignal(recorded PIDRecord, obs ObservedProcess) error {
	if recorded.PID <= 1 || obs.PID <= 1 {
		return errRefusePID
	}
	if recorded.PID != obs.PID {
		return fmt.Errorf("mcp: pid mismatch: recorded %d observed %d", recorded.PID, obs.PID)
	}
	if !obs.Alive {
		return fmt.Errorf("mcp: pid %d is not running", recorded.PID)
	}
	if !isWXServeBinary(recorded.Exe) {
		return fmt.Errorf("mcp: recorded executable %q is not wx or wx-cli", recorded.Exe)
	}
	if !commandIsServe(recorded, obs) {
		return fmt.Errorf("mcp: pid %d is not wx mcp serve", recorded.PID)
	}
	return nil
}

// SignalVerified sends sig to recorded.PID only when MaySignal allows it.
// It never signals pid <= 1, a negative process-group id, or a pid other
// than the one recorded. A rejected call does not call kill.
func SignalVerified(recorded PIDRecord, obs ObservedProcess, sig syscall.Signal) error {
	if recorded.PID <= 1 || obs.PID <= 1 || recorded.PID != obs.PID {
		return errRefusePID
	}
	if sig < 0 {
		return errors.New("mcp: refusing a negative signal")
	}
	if err := MaySignal(recorded, obs); err != nil {
		return err
	}
	// Re-check immediately before the syscall. kill(2) with pid 0 or -1
	// is a process-group signal; those values never reach this call.
	if recorded.PID <= 1 {
		return errRefusePID
	}
	return syscall.Kill(recorded.PID, sig)
}

// Observe reads process liveness and the command line. It does not signal.
// pid <= 1 is an error and does not call kill, including the signal-0 probe.
func Observe(pid int) (ObservedProcess, error) {
	if pid <= 1 {
		return ObservedProcess{}, errRefusePID
	}
	obs := ObservedProcess{PID: pid, Alive: processAlive(pid)}
	cmd, err := processCommand(pid)
	if err != nil && !obs.Alive {
		return obs, err
	}
	obs.Command = cmd
	obs.ExePath = processExe(pid, cmd)
	return obs, nil
}

func processAlive(pid int) bool {
	if pid <= 1 {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}

func processCommand(pid int) (string, error) {
	if pid <= 1 {
		return "", errRefusePID
	}
	if runtime.GOOS == "linux" {
		raw, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
		if err != nil {
			return "", err
		}
		parts := strings.Split(strings.TrimRight(string(raw), "\x00"), "\x00")
		return strings.Join(parts, " "), nil
	}
	out, err := exec.Command("/bin/ps", "-p", strconv.Itoa(pid), "-ww", "-o", "command=").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func processExe(pid int, command string) string {
	if runtime.GOOS == "linux" && pid > 1 {
		if exe, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid)); err == nil {
			return exe
		}
	}
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func isWXServeBinary(path string) bool {
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, ".exe")
	return base == "wx" || base == "wx-cli"
}

func commandIsServe(recorded PIDRecord, obs ObservedProcess) bool {
	cmd := strings.TrimSpace(obs.Command)
	if cmd == "" || !hasAdjacent(cmd, "mcp", "serve") {
		return false
	}
	if sameExecutable(recorded.Exe, obs.ExePath) || hasExePrefix(cmd, recorded.Exe) {
		return true
	}
	if recorded.Argv0 != "" && (sameExecutable(recorded.Argv0, obs.ExePath) || hasExePrefix(cmd, recorded.Argv0)) {
		return true
	}
	return false
}

func hasExePrefix(command, exe string) bool {
	exe = strings.TrimSpace(exe)
	if exe == "" {
		return false
	}
	return command == exe || strings.HasPrefix(command, exe+" ")
}

func sameExecutable(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" || b == "" {
		return false
	}
	if filepath.Clean(a) == filepath.Clean(b) {
		return true
	}
	ar, aerr := filepath.EvalSymlinks(a)
	br, berr := filepath.EvalSymlinks(b)
	return aerr == nil && berr == nil && ar == br
}

func hasAdjacent(command, a, b string) bool {
	fields := strings.Fields(command)
	for i := 0; i < len(fields)-1; i++ {
		if fields[i] == a && fields[i+1] == b {
			return true
		}
	}
	return false
}

// ParentPIDFromArgs reads a --parent-pid N pair. The value must be > 1.
func ParentPIDFromArgs(args []string) (int, bool) {
	for i := 0; i < len(args)-1; i++ {
		if args[i] != "--parent-pid" {
			continue
		}
		n, err := strconv.Atoi(args[i+1])
		if err != nil || n <= 1 {
			return 0, false
		}
		return n, true
	}
	return 0, false
}
