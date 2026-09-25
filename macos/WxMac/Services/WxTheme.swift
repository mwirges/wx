import SwiftUI

/// Warp Glass & Star Trek: Strange New Worlds (SNW) Bridge Console visual tokens.
enum WxTheme {
    // Core Backgrounds
    static let bg = Color(red: 0x06 / 255, green: 0x0A / 255, blue: 0x14 / 255).opacity(0.96)
    static let snwChassis = Color(red: 0x0A / 255, green: 0x12 / 255, blue: 0x22 / 255)
    static let snwPanel = Color(red: 0x0E / 255, green: 0x1A / 255, blue: 0x2D / 255).opacity(0.88)
    
    // SNW Bridge Console Signature Palette
    static let accent = Color(red: 0x38 / 255, green: 0xE1 / 255, blue: 0xFF / 255) // Phaser / Sensor Cyan
    static let snwCyan = Color(red: 0x38 / 255, green: 0xE1 / 255, blue: 0xFF / 255)
    static let snwGold = Color(red: 0xF5 / 255, green: 0xA6 / 255, blue: 0x23 / 255) // Command Gold
    static let snwAmber = Color(red: 0xFF / 255, green: 0x9F / 255, blue: 0x0A / 255) // Nav Amber
    static let snwRed = Color(red: 0xFF / 255, green: 0x3B / 255, blue: 0x56 / 255) // Condition Red Alert
    static let snwGreen = Color(red: 0x30 / 255, green: 0xD1 / 255, blue: 0x58 / 255) // Sensors Nominal Green
    static let snwSilver = Color(red: 0x94 / 255, green: 0xA7 / 255, blue: 0xC5 / 255) // Telemetry Silver
    
    static let accentSecondary = Color(red: 0x7B / 255, green: 0x8C / 255, blue: 0xFF / 255)
    static let text = Color(red: 0xF0 / 255, green: 0xF6 / 255, blue: 0xFF / 255)
    static let textSecondary = Color(red: 0x94 / 255, green: 0xA7 / 255, blue: 0xC5 / 255)
    static let warn = snwGold
    static let alert = snwRed
    static let conditionRed = snwRed
    static let conditionYellow = snwGold
    static let border = snwCyan.opacity(0.35)
    static let corner: CGFloat = 8
    static let popoverSize = CGSize(width: 420, height: 620)
    static let periodRowHeight: CGFloat = 44

    static func severityColor(_ s: String?) -> Color {
        switch (s ?? "").lowercased() {
        case "extreme": return snwRed
        case "severe": return snwRed.opacity(0.85)
        case "moderate": return snwGold
        case "minor": return snwCyan
        default: return textSecondary
        }
    }
}

/// Starfleet bridge tactical corner brackets framing panels.
struct SNWCornerBrackets: View {
    var color: Color = WxTheme.snwCyan.opacity(0.55)
    var length: CGFloat = 7
    var thickness: CGFloat = 1.0

    var body: some View {
        GeometryReader { geo in
            Path { p in
                let w = geo.size.width
                let h = geo.size.height

                // Top-Left
                p.move(to: CGPoint(x: 0, y: length))
                p.addLine(to: CGPoint(x: 0, y: 0))
                p.addLine(to: CGPoint(x: length, y: 0))

                // Top-Right
                p.move(to: CGPoint(x: w - length, y: 0))
                p.addLine(to: CGPoint(x: w, y: 0))
                p.addLine(to: CGPoint(x: w, y: length))

                // Bottom-Left
                p.move(to: CGPoint(x: 0, y: h - length))
                p.addLine(to: CGPoint(x: 0, y: h))
                p.addLine(to: CGPoint(x: length, y: h))

                // Bottom-Right
                p.move(to: CGPoint(x: w - length, y: h))
                p.addLine(to: CGPoint(x: w, y: h))
                p.addLine(to: CGPoint(x: w, y: h - length))
            }
            .stroke(color, lineWidth: thickness)
        }
        .allowsHitTesting(false)
    }
}

/// Bridge console section card with technical header and corner brackets.
struct SNWConsoleCard<Content: View>: View {
    var title: String?
    var tag: String?
    var statusColor: Color = WxTheme.snwCyan
    @ViewBuilder var content: Content

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            if title != nil || tag != nil {
                HStack(spacing: 6) {
                    Circle()
                        .fill(statusColor)
                        .frame(width: 5, height: 5)
                        .shadow(color: statusColor.opacity(0.8), radius: 3)
                    if let title {
                        Text(title.uppercased())
                            .font(.system(size: 9.5, weight: .bold, design: .monospaced))
                            .tracking(0.8)
                            .foregroundStyle(statusColor)
                    }
                    Spacer()
                    if let tag {
                        Text(tag.uppercased())
                            .font(.system(size: 8.5, weight: .semibold, design: .monospaced))
                            .foregroundStyle(WxTheme.snwSilver.opacity(0.75))
                    }
                }
                .padding(.horizontal, 10)
                .padding(.top, 8)

                Rectangle()
                    .fill(
                        LinearGradient(
                            colors: [statusColor.opacity(0.4), statusColor.opacity(0.08), .clear],
                            startPoint: .leading,
                            endPoint: .trailing
                        )
                    )
                    .frame(height: 1)
            }

            content
                .padding(10)
        }
        .background(
            RoundedRectangle(cornerRadius: 8, style: .continuous)
                .fill(WxTheme.snwPanel)
        )
        .overlay(
            RoundedRectangle(cornerRadius: 8, style: .continuous)
                .strokeBorder(WxTheme.border.opacity(0.35), lineWidth: 0.8)
        )
        .overlay(SNWCornerBrackets())
    }
}

/// Starship Bridge Telemetry Header Pill
struct SNWHeaderPill: View {
    let title: String
    let subtitle: String?

    var body: some View {
        HStack(spacing: 8) {
            HStack(spacing: 4) {
                Circle()
                    .fill(WxTheme.snwGreen)
                    .frame(width: 5, height: 5)
                    .shadow(color: WxTheme.snwGreen.opacity(0.8), radius: 3)
                Text("NCC-1701")
                    .font(.system(size: 9, weight: .black, design: .monospaced))
                    .foregroundStyle(WxTheme.snwGold)
            }
            .padding(.horizontal, 6)
            .padding(.vertical, 3)
            .background(WxTheme.snwGold.opacity(0.12), in: RoundedRectangle(cornerRadius: 4))
            .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(WxTheme.snwGold.opacity(0.35), lineWidth: 0.5))

            Text(title.uppercased())
                .font(.system(size: 10.5, weight: .bold, design: .monospaced))
                .tracking(1.0)
                .foregroundStyle(WxTheme.text)

            Spacer()

            if let subtitle {
                Text(subtitle.uppercased())
                    .font(.system(size: 9, weight: .medium, design: .monospaced))
                    .foregroundStyle(WxTheme.snwCyan.opacity(0.85))
            }
        }
        .padding(.horizontal, 10)
        .padding(.vertical, 6)
        .background(
            RoundedRectangle(cornerRadius: 6, style: .continuous)
                .fill(WxTheme.snwPanel)
                .overlay(RoundedRectangle(cornerRadius: 6).strokeBorder(WxTheme.snwCyan.opacity(0.25), lineWidth: 0.8))
        )
        .overlay(SNWCornerBrackets(color: WxTheme.snwCyan.opacity(0.6), length: 6, thickness: 1))
    }
}

/// Starfleet Sensor Readout Tile
struct SNWMetricTile: View {
    let label: String
    let value: String
    var accent: Color = WxTheme.snwCyan

    var body: some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(label.uppercased())
                .font(.system(size: 8, weight: .bold, design: .monospaced))
                .foregroundStyle(WxTheme.snwSilver)
                .lineLimit(1)
            Text(value)
                .font(.system(size: 12, weight: .semibold, design: .rounded))
                .foregroundStyle(WxTheme.text)
                .shadow(color: accent.opacity(0.3), radius: 2)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.horizontal, 8)
        .padding(.vertical, 5)
        .background(
            RoundedRectangle(cornerRadius: 5, style: .continuous)
                .fill(WxTheme.snwChassis.opacity(0.85))
                .overlay(RoundedRectangle(cornerRadius: 5).strokeBorder(accent.opacity(0.22), lineWidth: 0.6))
        )
        .overlay(SNWCornerBrackets(color: accent.opacity(0.45), length: 5, thickness: 0.8))
    }
}

struct WarpGlassBackground: View {
    var body: some View {
        ZStack {
            WxTheme.bg
            Rectangle().fill(.ultraThinMaterial)
        }
        .overlay(
            RoundedRectangle(cornerRadius: WxTheme.corner, style: .continuous)
                .strokeBorder(WxTheme.border, lineWidth: 1)
        )
    }
}

struct PillHeader: View {
    let title: String
    let subtitle: String?

    var body: some View {
        SNWHeaderPill(title: title, subtitle: subtitle)
    }
}
