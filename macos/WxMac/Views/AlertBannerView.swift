import SwiftUI

/// Display classification for the pinned desk banner.
/// Event text only: "Warning" / "Emergency" outrank "Watch".
/// Advisories and statements stay in `AlertsListView`. This does not feed the menu-bar pip.
private enum AlertBannerKind {
    case warning
    case watch

    var rank: Int {
        switch self {
        case .warning: return 0
        case .watch: return 1
        }
    }

    var accessibilityID: String {
        switch self {
        case .warning: return "alert-banner-warning"
        case .watch: return "alert-banner-watch"
        }
    }

    var symbolName: String {
        switch self {
        case .warning: return "exclamationmark.triangle.fill"
        case .watch: return "eye.fill"
        }
    }
}

private func alertBannerKind(_ alert: Alert) -> AlertBannerKind? {
    let event = alert.event
    if event.localizedCaseInsensitiveContains("warning") || event.localizedCaseInsensitiveContains("emergency") {
        return .warning
    }
    if event.localizedCaseInsensitiveContains("watch") {
        return .watch
    }
    return nil
}

/// NWS severity order used only to pick which banner item leads.
private func alertSeverityRank(_ severity: String?) -> Int {
    switch (severity ?? "").lowercased() {
    case "extreme": return 0
    case "severe": return 1
    case "moderate": return 2
    case "minor": return 3
    default: return 4
    }
}

private struct RankedBannerAlert {
    var alert: Alert
    var kind: AlertBannerKind
    var index: Int
}

private func rankedBannerAlerts(_ alerts: [Alert]) -> [RankedBannerAlert] {
    alerts.enumerated().compactMap { item in
        guard let kind = alertBannerKind(item.element) else { return nil }
        return RankedBannerAlert(alert: item.element, kind: kind, index: item.offset)
    }
    .sorted { lhs, rhs in
        if lhs.kind.rank != rhs.kind.rank { return lhs.kind.rank < rhs.kind.rank }
        let leftSeverity = alertSeverityRank(lhs.alert.severity)
        let rightSeverity = alertSeverityRank(rhs.alert.severity)
        if leftSeverity != rightSeverity { return leftSeverity < rightSeverity }
        return lhs.index < rhs.index
    }
}

private func expiryText(_ raw: String?) -> String? {
    guard let raw = raw?.trimmingCharacters(in: .whitespacesAndNewlines), !raw.isEmpty else { return nil }
    let parser = ISO8601DateFormatter()
    parser.formatOptions = [.withInternetDateTime]
    if let date = parser.date(from: raw) {
        return date.formatted(.dateTime.month(.abbreviated).day().hour().minute())
    }
    parser.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
    if let date = parser.date(from: raw) {
        return date.formatted(.dateTime.month(.abbreviated).day().hour().minute())
    }
    return raw
}

/// Sculpted Liquid Glass Alert Ribbon pinned immediately below the Unified Bridge Toolbar on every desk tab.
struct AlertBannerView: View {
    @EnvironmentObject private var store: WeatherStore
    @State private var showDetails = false

    var body: some View {
        let ranked = rankedBannerAlerts(store.activeAlerts)
        if let lead = ranked.first {
            banner(lead: lead, extra: ranked.count - 1)
                .padding(.horizontal, 14)
                .padding(.top, 6)
                .padding(.bottom, 4)
        }
    }

    private func banner(lead: RankedBannerAlert, extra: Int) -> some View {
        let kind = lead.kind
        return Button {
            showDetails = true
        } label: {
            bannerLabel(alert: lead.alert, kind: kind, extra: extra)
        }
        .buttonStyle(.plain)
        .accessibilityIdentifier("alert-banner")
        .overlay(alignment: .topLeading) {
            kindToken(kind)
        }
        .accessibilityElement(children: .contain)
        .popover(isPresented: $showDetails, attachmentAnchor: .rect(.bounds), arrowEdge: .bottom) {
            AlertBannerDetailView()
        }
    }

    private func bannerLabel(alert: Alert, kind: AlertBannerKind, extra: Int) -> some View {
        let shape = RoundedRectangle(cornerRadius: 14, style: .continuous)
        let isWarning = (kind == .warning)

        return HStack(spacing: 10) {
            Image(systemName: kind.symbolName)
                .font(.system(size: 14, weight: .bold))
                .accessibilityHidden(true)

            Text(alert.event.uppercased())
                .font(.system(size: 12.5, weight: .bold, design: .rounded))
                .tracking(0.3)
                .lineLimit(1)
                .layoutPriority(1)

            if let expiry = expiryText(alert.expires) {
                Text("EXPIRES \(expiry.uppercased())")
                    .font(.system(size: 10.5, weight: .semibold, design: .monospaced))
                    .opacity(0.92)
                    .lineLimit(1)
            }

            Spacer(minLength: 8)

            if extra > 0 {
                Text("+\(extra) more")
                    .font(.system(size: 10.5, weight: .bold, design: .rounded))
                    .padding(.horizontal, 8)
                    .padding(.vertical, 2.5)
                    .background(Color.black.opacity(0.22), in: Capsule())
                    .lineLimit(1)
            }

            HStack(spacing: 3) {
                Text("View Advisory")
                    .font(.system(size: 11, weight: .semibold, design: .rounded))
                Image(systemName: "chevron.right")
                    .font(.system(size: 9.5, weight: .bold))
            }
            .padding(.horizontal, 9)
            .padding(.vertical, 3.5)
            .background(Color.white.opacity(isWarning ? 0.18 : 0.08), in: Capsule())
        }
        .foregroundStyle(isWarning ? Color.white : WxGlassTheme.amber)
        .padding(.horizontal, 14)
        .padding(.vertical, 8.5)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background {
            ZStack {
                shape.fill(.ultraThinMaterial)
                if isWarning {
                    shape.fill(
                        LinearGradient(
                            colors: [WxGlassTheme.conditionRed.opacity(0.92), WxGlassTheme.conditionRed.opacity(0.72)],
                            startPoint: .topLeading,
                            endPoint: .bottomTrailing
                        )
                    )
                } else {
                    shape.fill(WxGlassTheme.amber.opacity(0.16))
                }
            }
        }
        .overlay {
            shape.strokeBorder(
                isWarning ? Color.white.opacity(0.34) : WxGlassTheme.amber.opacity(0.75),
                lineWidth: isWarning ? 1.0 : 1.5
            )
        }
        .clipShape(shape)
        .shadow(
            color: (isWarning ? WxGlassTheme.conditionRed : WxGlassTheme.amber).opacity(0.28),
            radius: 12,
            y: 4
        )
        .contentShape(shape)
    }

    /// Child element so XCUITest can query the kind without scrolling the banner away.
    private func kindToken(_ kind: AlertBannerKind) -> some View {
        Text(kind.accessibilityID)
            .font(.system(size: 1))
            .frame(width: 12, height: 12)
            .clipped()
            .accessibilityElement(children: .ignore)
            .accessibilityIdentifier(kind.accessibilityID)
            .accessibilityLabel(kind.accessibilityID)
            .allowsHitTesting(false)
    }
}

/// Every active alert, including advisories the banner itself does not promote.
struct AlertBannerDetailView: View {
    @EnvironmentObject private var store: WeatherStore

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack(spacing: 6) {
                Circle()
                    .fill(WxGlassTheme.conditionRed)
                    .frame(width: 6, height: 6)
                Text("ACTIVE ALERTS")
                    .font(.system(size: 11, weight: .bold, design: .rounded))
                    .tracking(0.5)
                    .foregroundStyle(WxGlassTheme.textSecondary)
                    .accessibilityIdentifier("alert-banner-details")
            }
            .padding(.bottom, 10)

            ScrollView {
                VStack(alignment: .leading, spacing: 10) {
                    ForEach(Array(store.activeAlerts.enumerated()), id: \.offset) { item in
                        detailRow(item.element)
                    }
                }
                .frame(maxWidth: .infinity, alignment: .leading)
            }
        }
        .padding(14)
        .frame(width: 450, height: 430)
        .background(WarpGlassBackground(regime: .severe))
        .preferredColorScheme(.dark)
    }

    private func detailRow(_ alert: Alert) -> some View {
        let tint = rowTint(alert)
        let shape = RoundedRectangle(cornerRadius: 12, style: .continuous)
        return VStack(alignment: .leading, spacing: 6) {
            Text(alert.event.uppercased())
                .font(.system(size: 12.5, weight: .bold, design: .rounded))
                .foregroundStyle(tint)
            detailField("HEADLINE", alert.headline)
            detailField("AREA", alert.area)
            detailField("EXPIRES", expiryText(alert.expires) ?? alert.expires)
            detailField("DESCRIPTION", alert.description)
            detailField("INSTRUCTION", alert.instruction)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(12)
        .background(tint.opacity(0.10), in: shape)
        .overlay(shape.strokeBorder(tint.opacity(0.42), lineWidth: 0.9))
    }

    @ViewBuilder
    private func detailField(_ label: String, _ value: String?) -> some View {
        if let value = value?.trimmingCharacters(in: .whitespacesAndNewlines), !value.isEmpty {
            VStack(alignment: .leading, spacing: 2) {
                Text(label)
                    .font(.system(size: 9, weight: .bold, design: .rounded))
                    .foregroundStyle(WxGlassTheme.textMuted)
                Text(value)
                    .font(.system(size: 11.5))
                    .foregroundStyle(WxGlassTheme.textPrimary)
                    .fixedSize(horizontal: false, vertical: true)
                    .textSelection(.enabled)
            }
        }
    }

    private func rowTint(_ alert: Alert) -> Color {
        switch alertBannerKind(alert) {
        case .warning: return WxGlassTheme.conditionRed
        case .watch: return WxGlassTheme.amber
        case nil: return WxGlassTheme.textSecondary
        }
    }
}
