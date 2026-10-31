import AppKit
import SwiftUI

enum MCPEndpoint {
    static let defaultPort = 9742

    static func url(host: String, port: Int) -> String {
        "http://\(host):\(port)/mcp"
    }

    static func addLine(host: String, port: Int) -> String {
        "grok mcp add --transport http wx \(url(host: host, port: port))"
    }

    static func addLineWithHeader(host: String, port: Int, token: String) -> String {
        "\(addLine(host: host, port: port)) --header \"Authorization: Bearer \(token)\""
    }
}

struct MCPSettingsView: View {
    @ObservedObject private var supervisor = MCPServerSupervisor.shared
    @State private var enabled = false
    @State private var requireToken = false
    @State private var ready = false
    @State private var copyNote = ""

    var body: some View {
        Form {
            Section {
                Toggle("Local MCP server", isOn: $enabled)
                    .accessibilityIdentifier("mcp-enabled-toggle")
                LabeledContent("Status") {
                    Text(statusTitle)
                        .accessibilityIdentifier("mcp-status")
                }
                Text(supervisor.statusDetail)
                    .font(.callout)
                    .foregroundStyle(.secondary)
                    .textSelection(.enabled)
                    .accessibilityIdentifier("mcp-status-detail")
            } header: {
                Text("MCP")
            } footer: {
                Text("Off until you turn it on. The server listens on 127.0.0.1 only and stops when you turn it off or quit wx.")
            }

            Section {
                endpointRow(host: "127.0.0.1", identifier: "ipv4")
                endpointRow(host: "localhost", identifier: "localhost")
            } header: {
                Text("Endpoints")
            } footer: {
                Text("Both hosts are this loopback server. Paste either line into a terminal.")
            }

            Section {
                Toggle("Require token", isOn: $requireToken)
                    .accessibilityIdentifier("mcp-require-token")
                Button("Copy grok mcp add with Authorization header") {
                    copyHeader(host: "127.0.0.1")
                }
                .accessibilityIdentifier("mcp-copy-header")
                .disabled(!requireToken)
                Button("Copy localhost add line with header") {
                    copyHeader(host: "localhost")
                }
                .disabled(!requireToken)
                if !copyNote.isEmpty {
                    Text(copyNote)
                        .font(.callout)
                        .foregroundStyle(.secondary)
                }
            } header: {
                Text("Token")
            } footer: {
                Text("Off by default, so grok mcp add works with no header. On a shared Mac, turn this on. The token is stored in a mode 0600 file next to config and is not shown here.")
            }

            if supervisor.phase == .error {
                Button("Retry") {
                    supervisor.setEnabled(true)
                }
            }
        }
        .formStyle(.grouped)
        .frame(minWidth: 540, minHeight: 460)
        .padding(8)
        .onAppear(perform: load)
        .onChange(of: enabled) { _, newValue in
            guard ready else { return }
            WxConfig.updateMCP { $0.enabled = newValue }
            supervisor.setEnabled(newValue)
            copyNote = ""
        }
        .onChange(of: requireToken) { _, newValue in
            guard ready else { return }
            applyRequireToken(newValue)
        }
    }

    private var statusTitle: String {
        switch supervisor.phase {
        case .stopped:
            return "Stopped"
        case .running:
            return "Running"
        case .restarting:
            return "Restarting"
        case .error:
            return "Error"
        }
    }

    private var port: Int {
        supervisor.port == 0 ? MCPEndpoint.defaultPort : supervisor.port
    }

    @ViewBuilder
    private func endpointRow(host: String, identifier: String) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(MCPEndpoint.url(host: host, port: port))
                .font(.system(.body, design: .monospaced))
                .textSelection(.enabled)
                .accessibilityIdentifier("mcp-url-\(identifier)")
            Text(MCPEndpoint.addLine(host: host, port: port))
                .font(.system(.caption, design: .monospaced))
                .textSelection(.enabled)
                .accessibilityIdentifier("mcp-add-\(identifier)")
            Button("Copy \(host) add line") {
                paste(MCPEndpoint.addLine(host: host, port: port), note: "Copied the \(host) add line.")
            }
            .accessibilityIdentifier("mcp-copy-\(identifier)")
        }
        .padding(.vertical, 2)
    }

    private func load() {
        let mcp = WxConfig.load().mcp
        enabled = mcp?.enabled ?? false
        requireToken = mcp?.requireToken ?? false
        DispatchQueue.main.async {
            ready = true
        }
    }

    private func applyRequireToken(_ on: Bool) {
        if on {
            do {
                _ = try WxConfig.ensureMCPToken()
                WxConfig.updateMCP { $0.requireToken = true }
            } catch {
                copyNote = error.localizedDescription
                requireToken = false
                return
            }
        } else {
            WxConfig.updateMCP { $0.requireToken = false }
        }
        copyNote = ""
        if enabled {
            supervisor.restartForConfigChange()
        }
    }

    private func copyHeader(host: String) {
        do {
            let token = try WxConfig.readMCPToken()
            paste(
                MCPEndpoint.addLineWithHeader(host: host, port: port, token: token),
                note: "Copied the \(host) add line with an Authorization header."
            )
        } catch {
            copyNote = error.localizedDescription
        }
    }

    private func paste(_ text: String, note: String) {
        let board = NSPasteboard.general
        board.clearContents()
        board.setString(text, forType: .string)
        copyNote = note
    }
}
