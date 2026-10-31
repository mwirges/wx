import AppKit
import Foundation
import Darwin

/// Supervises one embedded `wx-cli mcp serve` child.
///
/// This is not `WxCLI.runProcess`. The child is long-lived. The app keeps
/// the stdin pipe open and passes `--parent-pid` plus `--exit-on-stdin-close`,
/// so the child exits if the app is killed without `applicationWillTerminate`.
///
/// Stop signals only the pid this supervisor spawned, or a pidfile child
/// whose parent pid is already dead, and only after `verify` matches
/// wx/wx-cli plus `mcp serve`. pid 0, pid -1, and any other pid are not
/// signaled. There is no process-group kill.
@MainActor
final class MCPServerSupervisor: ObservableObject {
    static let shared = MCPServerSupervisor()

    enum Phase: String {
        case stopped
        case running
        case restarting
        case error
    }

    private enum Existing {
        case none
        case adopt(pid: Int32, exe: String)
        case recycle(pid: Int32, exe: String)
        case refuse(String)
    }

    private struct PIDFile: Decodable {
        var pid: Int
        var exe: String
        var argv0: String?
        var args: [String]
    }

    @Published private(set) var phase: Phase = .stopped
    @Published private(set) var statusDetail: String = "Stopped"
    @Published private(set) var port: Int = 9742

    private var process: Process?
    private var stdinPipe: Pipe?
    private var stdoutPipe: Pipe?
    private var stderrPipe: Pipe?
    private var spawnedPID: Int32 = 0
    private var serveBinary: String = ""
    private var wantsRun = false
    private var stopIntentional = false
    private var restartAttempt = 0
    private var childStartedAt: Date?
    private var restartTask: Task<Void, Never>?
    private var generation = 0
    private var stderrTail = ""
    private var redactionSecret = ""

    /// Matches mcp.RestartDelay: attempts 1...5 wait 0.5, 1, 2, 4, 8 seconds.
    static let maxRestarts = 5
    private let stableUptime: TimeInterval = 30

    private init() {}

    func startIfEnabled() {
        if ProcessInfo.processInfo.environment["WX_UI_TEST"] == "1" {
            phase = .stopped
            statusDetail = "Stopped"
            return
        }
        refreshPort()
        if WxConfig.load().mcp?.enabled == true {
            wantsRun = true
            stopIntentional = false
            beginStart(resetBudget: true)
        } else {
            wantsRun = false
            phase = .stopped
            statusDetail = "Stopped"
        }
    }

    func setEnabled(_ on: Bool) {
        if !on {
            wantsRun = false
            stopOwnedChild()
            phase = .stopped
            statusDetail = "Stopped"
            stopIntentional = false
            return
        }
        wantsRun = true
        stopIntentional = false
        beginStart(resetBudget: true)
    }

    /// Toggle stayed on. The child must reload require_token from config.
    func restartForConfigChange() {
        guard wantsRun else { return }
        stopOwnedChild()
        stopIntentional = false
        beginStart(resetBudget: true)
    }

    func stopForQuit() {
        wantsRun = false
        stopOwnedChild()
        phase = .stopped
        statusDetail = "Stopped"
    }

    static func restartDelay(attempt: Int) -> TimeInterval? {
        guard (1...maxRestarts).contains(attempt) else { return nil }
        return min(0.5 * pow(2.0, Double(attempt - 1)), 8)
    }

    private func beginStart(resetBudget: Bool) {
        if resetBudget {
            restartAttempt = 0
        }
        refreshPort()
        refreshSecret()
        if process?.isRunning == true {
            phase = .running
            statusDetail = "Running on 127.0.0.1:\(port)"
            return
        }
        switch existingChild() {
        case .refuse(let message):
            phase = .error
            statusDetail = message
            return
        case .recycle(let pid, let exe):
            _ = signalVerified(pid: pid, exe: exe)
            waitUntilDead(pid)
        case .adopt(let pid, let exe):
            spawnedPID = pid
            serveBinary = exe
            phase = .running
            statusDetail = "Running on 127.0.0.1:\(port)"
            return
        case .none:
            break
        }
        guard let binary = locateServeBinary() else {
            phase = .error
            statusDetail = "Embedded wx-cli was not found. Rebuild with make mac-build."
            return
        }
        generation &+= 1
        let gen = generation
        do {
            try spawn(binary: binary, generation: gen)
        } catch {
            phase = .error
            statusDetail = error.localizedDescription
        }
    }

    private func spawn(binary: String, generation gen: Int) throws {
        let proc = Process()
        proc.executableURL = URL(fileURLWithPath: binary)
        let parent = ProcessInfo.processInfo.processIdentifier
        proc.arguments = [
            "mcp", "serve",
            "--config", WxConfig.configURL.path,
            "--parent-pid", String(parent),
            "--exit-on-stdin-close",
        ]
        let stdin = Pipe()
        let stdout = Pipe()
        let stderr = Pipe()
        proc.standardInput = stdin
        proc.standardOutput = stdout
        proc.standardError = stderr
        proc.terminationHandler = { [weak self] exited in
            let code = exited.terminationStatus
            Task { @MainActor in
                self?.handleExit(generation: gen, code: code)
            }
        }
        try proc.run()
        let pid = proc.processIdentifier
        guard pid > 1 else {
            proc.terminate()
            throw MCPServeError.refused("wx-cli returned pid \(pid)")
        }
        process = proc
        stdinPipe = stdin
        stdoutPipe = stdout
        stderrPipe = stderr
        spawnedPID = pid
        serveBinary = binary
        childStartedAt = Date()
        stderrTail = ""
        drain(stdout)
        drain(stderr)
        phase = .running
        statusDetail = "Running on 127.0.0.1:\(port)"
    }

    private func handleExit(generation gen: Int, code: Int32) {
        guard gen == generation else { return }
        let started = childStartedAt
        process = nil
        spawnedPID = 0
        closeStdin()
        clearDrains()
        guard wantsRun, !stopIntentional else {
            phase = .stopped
            statusDetail = "Stopped"
            return
        }
        let tail = stderrTail
        if isTerminal(tail) {
            phase = .error
            statusDetail = tail.isEmpty ? "wx mcp serve exited \(code) and will not retry." : String(tail.suffix(500))
            return
        }
        if let started, Date().timeIntervalSince(started) >= stableUptime {
            restartAttempt = 0
        }
        restartAttempt += 1
        guard let delay = Self.restartDelay(attempt: restartAttempt) else {
            phase = .error
            statusDetail = "MCP server stopped after \(Self.maxRestarts) restarts. The toggle is still on."
            return
        }
        phase = .restarting
        statusDetail = "Restarting (attempt \(restartAttempt) of \(Self.maxRestarts))"
        restartTask?.cancel()
        restartTask = Task { @MainActor in
            try? await Task.sleep(nanoseconds: UInt64(delay * 1_000_000_000))
            guard !Task.isCancelled, wantsRun, !stopIntentional else { return }
            beginStart(resetBudget: false)
        }
    }

    private func stopOwnedChild() {
        restartTask?.cancel()
        restartTask = nil
        stopIntentional = true
        generation &+= 1
        let pid = spawnedPID
        let exe = serveBinary
        let proc = process
        spawnedPID = 0
        process = nil
        closeStdin()
        clearDrains()
        if pid > 1 {
            _ = signalVerified(pid: pid, exe: exe)
            waitUntilDead(pid)
        }
        // Stdin is already closed, so a verified child exits even when the
        // signal was refused. Do not call Process.terminate (that signals
        // without a fresh check) and do not block quit forever.
        if let proc {
            let deadline = Date().addingTimeInterval(2)
            while proc.isRunning && Date() < deadline {
                Thread.sleep(forTimeInterval: 0.05)
            }
        }
    }

    /// Signals one pid after a fresh wx-cli mcp serve check. Returns false
    /// without calling kill when the check fails. Never uses pid <= 1.
    @discardableResult
    private func signalVerified(pid: Int32, exe: String) -> Bool {
        guard pid > 1 else { return false }
        guard verify(pid: pid, expectedExe: exe) else { return false }
        guard kill(pid, SIGTERM) == 0 else { return false }
        let deadline = Date().addingTimeInterval(2)
        while Date() < deadline {
            if !processAlive(pid) {
                reap(pid)
                return true
            }
            Thread.sleep(forTimeInterval: 0.05)
        }
        guard pid > 1, verify(pid: pid, expectedExe: exe) else { return false }
        _ = kill(pid, SIGKILL)
        waitUntilDead(pid)
        reap(pid)
        return true
    }

    private func reap(_ pid: Int32) {
        guard pid > 1 else { return }
        var status: Int32 = 0
        _ = waitpid(pid, &status, WNOHANG)
    }

    private func waitUntilDead(_ pid: Int32) {
        guard pid > 1 else { return }
        let deadline = Date().addingTimeInterval(2)
        while Date() < deadline && processAlive(pid) {
            Thread.sleep(forTimeInterval: 0.05)
        }
        reap(pid)
    }

    private func verify(pid: Int32, expectedExe: String) -> Bool {
        guard pid > 1, isWXBinary(expectedExe) else { return false }
        guard let observed = procPath(pid), pathsMatch(observed, expectedExe), isWXBinary(observed) else {
            return false
        }
        guard let command = processCommand(pid), commandHasMCPServe(command) else {
            return false
        }
        return true
    }

    private func existingChild() -> Existing {
        let url = WxConfig.configURL.deletingLastPathComponent().appendingPathComponent("mcp.pid")
        guard let data = try? Data(contentsOf: url),
              let rec = try? JSONDecoder().decode(PIDFile.self, from: data),
              rec.pid > 1 else {
            return .none
        }
        let pid = Int32(rec.pid)
        let exe = rec.exe
        guard verify(pid: pid, expectedExe: exe) else {
            // Stale pidfile or a foreign holder. Do not signal it.
            // The new child fails closed if that process still owns the port.
            return .none
        }
        if let parent = parentPID(rec.args) {
            let me = ProcessInfo.processInfo.processIdentifier
            if parent == me {
                return .adopt(pid: pid, exe: exe)
            }
            if parent > 1 && processAlive(parent) {
                return .refuse("Another wx MCP server is already running (pid \(pid)). This copy will not take the port.")
            }
            return .recycle(pid: pid, exe: exe)
        }
        return .refuse("wx mcp serve is already running (pid \(pid)). Refusing to start a second listener.")
    }

    private func parentPID(_ args: [String]) -> Int32? {
        guard let index = args.firstIndex(of: "--parent-pid"), index + 1 < args.count else {
            return nil
        }
        guard let value = Int32(args[index + 1]), value > 1 else {
            return nil
        }
        return value
    }

    private func locateServeBinary() -> String? {
        if let bundled = Bundle.main.url(forAuxiliaryExecutable: "wx-cli")?.path,
           FileManager.default.isExecutableFile(atPath: bundled) {
            return bundled
        }
        return WxCLI.locateBinary()
    }

    private func refreshPort() {
        let configured = WxConfig.load().mcp?.port ?? 0
        port = configured == 0 ? 9742 : configured
    }

    private func refreshSecret() {
        redactionSecret = (try? WxConfig.readMCPToken()) ?? ""
    }

    private func closeStdin() {
        if let pipe = stdinPipe {
            try? pipe.fileHandleForWriting.close()
        }
        stdinPipe = nil
    }

    private func clearDrains() {
        stdoutPipe?.fileHandleForReading.readabilityHandler = nil
        stderrPipe?.fileHandleForReading.readabilityHandler = nil
        stdoutPipe = nil
        stderrPipe = nil
    }

    private func drain(_ pipe: Pipe) {
        pipe.fileHandleForReading.readabilityHandler = { [weak self] handle in
            let data = handle.availableData
            if data.isEmpty {
                handle.readabilityHandler = nil
                return
            }
            let text = String(data: data, encoding: .utf8) ?? ""
            Task { @MainActor in
                self?.noteOutput(text)
            }
        }
    }

    private func noteOutput(_ text: String) {
        var safe = text
        if !redactionSecret.isEmpty {
            safe = safe.replacingOccurrences(of: redactionSecret, with: "[redacted]")
        }
        if safe.localizedCaseInsensitiveContains("authorization:") {
            safe = "[redacted]\n"
        }
        stderrTail.append(safe)
        if stderrTail.count > 4000 {
            stderrTail = String(stderrTail.suffix(4000))
        }
    }

    private func isTerminal(_ text: String) -> Bool {
        let lowered = text.lowercased()
        let keys = [
            "already running",
            "in use",
            "refusing",
            "disabled",
            "require_token",
            "not mode 0600",
            "parent pid",
            "not running",
        ]
        return keys.contains { lowered.contains($0) }
    }
}

private enum MCPServeError: LocalizedError {
    case refused(String)

    var errorDescription: String? {
        switch self {
        case .refused(let message):
            return message
        }
    }
}

private func isWXBinary(_ path: String) -> Bool {
    let base = URL(fileURLWithPath: path).lastPathComponent
    return base == "wx" || base == "wx-cli"
}

private func pathsMatch(_ lhs: String, _ rhs: String) -> Bool {
    if lhs == rhs {
        return true
    }
    let left = URL(fileURLWithPath: lhs).resolvingSymlinksInPath().path
    let right = URL(fileURLWithPath: rhs).resolvingSymlinksInPath().path
    return left == right
}

private func procPath(_ pid: Int32) -> String? {
    guard pid > 1 else { return nil }
    var buf = [CChar](repeating: 0, count: Int(MAXPATHLEN))
    let n = proc_pidpath(pid, &buf, UInt32(buf.count))
    guard n > 0 else { return nil }
    return String(cString: buf)
}

private func processCommand(_ pid: Int32) -> String? {
    guard pid > 1 else { return nil }
    let proc = Process()
    proc.executableURL = URL(fileURLWithPath: "/bin/ps")
    proc.arguments = ["-p", String(pid), "-ww", "-o", "command="]
    let out = Pipe()
    proc.standardOutput = out
    proc.standardError = Pipe()
    do {
        try proc.run()
    } catch {
        return nil
    }
    proc.waitUntilExit()
    let text = String(data: out.fileHandleForReading.readDataToEndOfFile(), encoding: .utf8)?
        .trimmingCharacters(in: .whitespacesAndNewlines)
    if text?.isEmpty ?? true {
        return nil
    }
    return text
}

private func commandHasMCPServe(_ command: String) -> Bool {
    let fields = command.split(whereSeparator: { $0 == " " || $0 == "\t" }).map(String.init)
    guard fields.count >= 2 else { return false }
    for index in 0..<(fields.count - 1) where fields[index] == "mcp" && fields[index + 1] == "serve" {
        return true
    }
    return false
}

private func processAlive(_ pid: Int32) -> Bool {
    guard pid > 1 else { return false }
    return kill(pid, 0) == 0
}
