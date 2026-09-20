import Foundation

enum WxCLIError: LocalizedError {
    case binaryMissing
    case timeout
    case failed(status: Int32, stderr: String)
    case decode(String)
    case configFailed(String)

    var errorDescription: String? {
        switch self {
        case .binaryMissing:
            return "wx binary not found on PATH. Install or build the Go CLI (`make build`) and ensure `wx` is on your PATH."
        case .timeout:
            return "wx timed out after 35s."
        case .failed(let status, let stderr):
            let trimmed = stderr.trimmingCharacters(in: .whitespacesAndNewlines)
            return trimmed.isEmpty ? "wx exited with status \(status)." : trimmed
        case .decode(let msg):
            return "Failed to decode wx JSON: \(msg)"
        case .configFailed(let msg):
            return msg
        }
    }
}

enum WxCLI {
    static func locateBinary() -> String? {
        if let path = ProcessInfo.processInfo.environment["WX_BINARY"], !path.isEmpty,
           FileManager.default.isExecutableFile(atPath: path) {
            return path
        }
        let proc = Process()
        proc.executableURL = URL(fileURLWithPath: "/usr/bin/which")
        proc.arguments = ["wx"]
        let out = Pipe()
        proc.standardOutput = out
        proc.standardError = Pipe()
        do {
            try proc.run()
            proc.waitUntilExit()
        } catch {
            return nil
        }
        guard proc.terminationStatus == 0 else { return nil }
        let path = String(data: out.fileHandleForReading.readDataToEndOfFile(), encoding: .utf8)?
            .trimmingCharacters(in: .whitespacesAndNewlines)
        guard let path, !path.isEmpty, FileManager.default.isExecutableFile(atPath: path) else { return nil }
        return path
    }

    static func fetch(location: String?, units: String?, timeoutSeconds: TimeInterval = 35) throws -> WxPayload {
        guard let binary = locateBinary() else { throw WxCLIError.binaryMissing }

        var args = ["--json", "--forecast", "--alerts"]
        if let location, !location.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            args += ["--location", location.trimmingCharacters(in: .whitespacesAndNewlines)]
        }
        if let units, !units.isEmpty {
            args += ["--units", units]
        }

        let proc = Process()
        proc.executableURL = URL(fileURLWithPath: binary)
        proc.arguments = args
        let outPipe = Pipe()
        let errPipe = Pipe()
        proc.standardOutput = outPipe
        proc.standardError = errPipe

        let group = DispatchGroup()
        group.enter()
        var timedOut = false
        let timer = DispatchSource.makeTimerSource(queue: .global())
        timer.schedule(deadline: .now() + timeoutSeconds)
        timer.setEventHandler {
            if proc.isRunning {
                timedOut = true
                proc.terminate()
            }
        }
        timer.resume()

        try proc.run()
        proc.waitUntilExit()
        timer.cancel()
        group.leave()

        let stdout = outPipe.fileHandleForReading.readDataToEndOfFile()
        let stderr = String(data: errPipe.fileHandleForReading.readDataToEndOfFile(), encoding: .utf8) ?? ""

        if timedOut { throw WxCLIError.timeout }

        // Conditions hard-fail → non-zero. Forecast/alerts may still produce partial JSON with warnings on stderr.
        if proc.terminationStatus != 0 && stdout.isEmpty {
            throw WxCLIError.failed(status: proc.terminationStatus, stderr: stderr)
        }

        do {
            var payload = try JSONDecoder().decode(WxPayload.self, from: stdout)
            if payload.conditions == nil && proc.terminationStatus != 0 {
                throw WxCLIError.failed(status: proc.terminationStatus, stderr: stderr)
            }
            if payload.warning == nil {
                let warn = stderr.trimmingCharacters(in: .whitespacesAndNewlines)
                if !warn.isEmpty { payload.warning = warn }
            }
            return payload
        } catch let e as WxCLIError {
            throw e
        } catch {
            throw WxCLIError.decode(error.localizedDescription)
        }
    }
}
