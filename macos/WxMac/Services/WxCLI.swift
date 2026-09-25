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
            return "wx CLI binary not found. Build the project (`make build` or `make mac-build`) or ensure `wx` is installed on PATH."
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
        // 1. Explicit environment override
        if let path = ProcessInfo.processInfo.environment["WX_BINARY"], !path.isEmpty,
           FileManager.default.isExecutableFile(atPath: path) {
            return path
        }

        // 2. Embedded helper in the app bundle (Contents/MacOS/wx-cli)
        if let bundled = Bundle.main.url(forAuxiliaryExecutable: "wx-cli")?.path,
           FileManager.default.isExecutableFile(atPath: bundled) {
            return bundled
        }
        if let bundledAux = Bundle.main.url(forAuxiliaryExecutable: "wx")?.path,
           bundledAux != Bundle.main.executablePath,
           FileManager.default.isExecutableFile(atPath: bundledAux) {
            return bundledAux
        }

        // 3. Resources inside app bundle (Contents/Resources/wx-cli or wx)
        if let resURL = Bundle.main.resourceURL {
            let resCli = resURL.appendingPathComponent("wx-cli").path
            if FileManager.default.isExecutableFile(atPath: resCli) { return resCli }
            let resWx = resURL.appendingPathComponent("wx").path
            if FileManager.default.isExecutableFile(atPath: resWx) { return resWx }
        }

        // 4. Sibling binary in build directory during local dev
        let siblingBuild = Bundle.main.bundleURL.deletingLastPathComponent().appendingPathComponent("wx").path
        if FileManager.default.isExecutableFile(atPath: siblingBuild) {
            return siblingBuild
        }

        // 5. User PATH entries
        if let pathVar = ProcessInfo.processInfo.environment["PATH"] {
            for dir in pathVar.split(separator: ":") {
                let candidate = URL(fileURLWithPath: String(dir)).appendingPathComponent("wx").path
                if FileManager.default.isExecutableFile(atPath: candidate) {
                    return candidate
                }
            }
        }

        // 6. Common macOS install locations (Homebrew, user ~/bin, Go bin, local bin)
        let home = FileManager.default.homeDirectoryForCurrentUser
        let commonLocations: [String] = [
            home.appendingPathComponent("bin/wx").path,
            home.appendingPathComponent("go/bin/wx").path,
            home.appendingPathComponent(".local/bin/wx").path,
            "/opt/homebrew/bin/wx",
            "/usr/local/bin/wx",
            "/usr/bin/wx",
        ]
        for candidate in commonLocations {
            if FileManager.default.isExecutableFile(atPath: candidate) {
                return candidate
            }
        }

        return nil
    }

    private struct ProcessResult {
        let stdout: Data
        let stderr: String
        let exitCode: Int32
        let timedOut: Bool
    }

    private static func runProcess(
        binary: String,
        args: [String],
        timeoutSeconds: TimeInterval
    ) throws -> ProcessResult {
        let proc = Process()
        proc.executableURL = URL(fileURLWithPath: binary)
        proc.arguments = args
        let outPipe = Pipe()
        let errPipe = Pipe()
        proc.standardOutput = outPipe
        proc.standardError = errPipe

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

        var stdoutData = Data()
        var stderrData = Data()
        let readGroup = DispatchGroup()

        readGroup.enter()
        DispatchQueue.global().async {
            stdoutData = outPipe.fileHandleForReading.readDataToEndOfFile()
            readGroup.leave()
        }

        readGroup.enter()
        DispatchQueue.global().async {
            stderrData = errPipe.fileHandleForReading.readDataToEndOfFile()
            readGroup.leave()
        }

        proc.waitUntilExit()
        readGroup.wait()
        timer.cancel()

        let stderr = String(data: stderrData, encoding: .utf8) ?? ""
        return ProcessResult(
            stdout: stdoutData,
            stderr: stderr,
            exitCode: proc.terminationStatus,
            timedOut: timedOut
        )
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

        let res = try runProcess(binary: binary, args: args, timeoutSeconds: timeoutSeconds)
        if res.timedOut { throw WxCLIError.timeout }

        // Conditions hard-fail → non-zero. Forecast/alerts may still produce partial JSON with warnings on stderr.
        if res.exitCode != 0 && res.stdout.isEmpty {
            throw WxCLIError.failed(status: res.exitCode, stderr: res.stderr)
        }

        do {
            var payload = try JSONDecoder().decode(WxPayload.self, from: res.stdout)
            if payload.conditions == nil && res.exitCode != 0 {
                throw WxCLIError.failed(status: res.exitCode, stderr: res.stderr)
            }
            if payload.warning == nil {
                let warn = res.stderr.trimmingCharacters(in: .whitespacesAndNewlines)
                if !warn.isEmpty { payload.warning = warn }
            }
            return payload
        } catch let e as WxCLIError {
            throw e
        } catch {
            throw WxCLIError.decode(error.localizedDescription)
        }
    }

    static func fetchRadar(
        location: String?,
        product: String? = nil,
        radiusKm: Double? = nil,
        timeoutSeconds: TimeInterval = 45
    ) throws -> RadarPayload {
        guard let binary = locateBinary() else { throw WxCLIError.binaryMissing }

        var args = ["radar", "--json"]
        if let location, !location.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            args += ["--location", location.trimmingCharacters(in: .whitespacesAndNewlines)]
        }
        if let product, !product.isEmpty {
            args += ["--product", product]
        }
        if let radius = radiusKm, radius > 0 {
            args += ["--radius", String(format: "%.0f", radius)]
        }

        let res = try runProcess(binary: binary, args: args, timeoutSeconds: timeoutSeconds)
        if res.timedOut { throw WxCLIError.timeout }

        if res.exitCode != 0 && res.stdout.isEmpty {
            throw WxCLIError.failed(status: res.exitCode, stderr: res.stderr)
        }

        do {
            return try JSONDecoder().decode(RadarPayload.self, from: res.stdout)
        } catch let e as WxCLIError {
            throw e
        } catch {
            throw WxCLIError.decode(error.localizedDescription)
        }
    }
}
