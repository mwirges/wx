import SwiftUI

/// Original SF Symbol alert markers — severity-driven animation only.
/// Extreme/Severe → red octagon (fast pulse). Moderate/Minor → yellow triangle (slow pulse).
/// No third-party IP, fonts, or sound.
struct AlertSeverityGlyph: View {
    let severity: String?
    var size: CGFloat = 18

    private enum Band {
        case red
        case yellow
        case muted
    }

    private var band: Band {
        switch (severity ?? "").lowercased() {
        case "extreme", "severe": return .red
        case "moderate", "minor": return .yellow
        default: return .muted
        }
    }

    private var symbolName: String {
        switch band {
        case .red: return "exclamationmark.octagon.fill"
        case .yellow: return "exclamationmark.triangle.fill"
        case .muted: return "exclamationmark.circle.fill"
        }
    }

    private var color: Color {
        switch band {
        case .red: return WxTheme.alert
        case .yellow: return WxTheme.warn
        case .muted: return WxTheme.textSecondary
        }
    }

    private var pulse: Animation {
        switch band {
        case .red:
            return .easeInOut(duration: 0.5).repeatForever(autoreverses: true)
        case .yellow:
            return .easeInOut(duration: 1.0).repeatForever(autoreverses: true)
        case .muted:
            return .default
        }
    }

    @State private var lit = false

    var body: some View {
        ZStack {
            if band != .muted {
                Circle()
                    .strokeBorder(color.opacity(lit ? 0.9 : 0.25), lineWidth: 2)
                    .frame(width: size + 10, height: size + 10)
                    .scaleEffect(lit ? 1.25 : 0.9)
                    .opacity(lit ? 0.9 : 0.35)
            }
            Image(systemName: symbolName)
                .font(.system(size: size, weight: .bold))
                .foregroundStyle(color)
                .symbolRenderingMode(.palette)
                .foregroundStyle(color, color.opacity(0.35))
                .scaleEffect(band == .muted ? 1.0 : (lit ? 1.12 : 0.95))
                .shadow(color: band == .muted ? .clear : color.opacity(lit ? 0.95 : 0.35), radius: lit ? 8 : 3)
        }
        .frame(width: size + 14, height: size + 14)
        .accessibilityLabel(accessibilityText)
        .onAppear {
            guard band != .muted else { return }
            withAnimation(pulse) { lit = true }
        }
    }

    private var accessibilityText: String {
        "\((severity ?? "Unknown")) weather alert"
    }
}
