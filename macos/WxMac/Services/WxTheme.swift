import SwiftUI

/// Warp Glass visual tokens — presentation only (no weather logic).
enum WxTheme {
    static let bg = Color(red: 0x0B / 255, green: 0x12 / 255, blue: 0x20 / 255).opacity(0.92)
    static let accent = Color(red: 0x3D / 255, green: 0xE1 / 255, blue: 0xFF / 255)
    static let accentSecondary = Color(red: 0x7B / 255, green: 0x8C / 255, blue: 0xFF / 255)
    static let text = Color(red: 0xE8 / 255, green: 0xEE / 255, blue: 0xF8 / 255)
    static let textSecondary = Color(red: 0x9A / 255, green: 0xA8 / 255, blue: 0xC2 / 255)
    static let warn = Color(red: 0xE6 / 255, green: 0xC3 / 255, blue: 0x5C / 255)
    static let alert = Color(red: 0xFF / 255, green: 0x5C / 255, blue: 0x7A / 255)
    static let border = accent.opacity(0.30)
    static let corner: CGFloat = 12
    static let popoverSize = CGSize(width: 420, height: 620)
    static let periodRowHeight: CGFloat = 44

    static func severityColor(_ s: String?) -> Color {
        switch (s ?? "").lowercased() {
        case "extreme": return alert
        case "severe": return alert.opacity(0.85)
        case "moderate": return warn
        case "minor": return accentSecondary
        default: return textSecondary
        }
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
        HStack(spacing: 8) {
            Circle()
                .fill(WxTheme.accent)
                .frame(width: 6, height: 6)
            Text(title)
                .font(.system(.subheadline, design: .rounded).weight(.semibold))
                .foregroundStyle(WxTheme.text)
            Spacer()
            if let subtitle {
                Text(subtitle)
                    .font(.caption2)
                    .foregroundStyle(WxTheme.textSecondary)
            }
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 8)
        .background(
            Capsule(style: .continuous)
                .fill(WxTheme.accent.opacity(0.12))
                .overlay(Capsule(style: .continuous).strokeBorder(WxTheme.border, lineWidth: 1))
        )
    }
}
