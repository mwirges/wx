import SwiftUI

struct PopoverView: View {
    @EnvironmentObject var store: WeatherStore

    private var hasConditionAlerts: Bool {
        (store.payload?.alerts ?? []).contains { ConditionBand.from(severity: $0.severity) != nil }
    }

    /// Fit: badge (~88) steals period budget → 4 periods when alerts present; else 6.
    private var periodLimit: Int { hasConditionAlerts ? 4 : 6 }

    private var updatedSubtitle: String? {
        store.lastRefreshed.map { $0.formatted(date: .omitted, time: .shortened) }
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            PillHeader(title: "wx", subtitle: updatedSubtitle.map { "Updated \($0)" })
            ControlsBar(showOpenWindow: true, compact: true)
            Rectangle()
                .fill(WxTheme.border)
                .frame(height: 1)
            NowBlockView(compact: true, popoverMetrics: true)
            AlertsListView(popoverMode: true)
            PeriodsListView(limit: periodLimit, compactRows: true)
            Spacer(minLength: 0)
        }
        .padding(14)
        .frame(width: WxTheme.popoverSize.width, height: WxTheme.popoverSize.height)
        .background(WarpGlassBackground())
        .clipShape(RoundedRectangle(cornerRadius: WxTheme.corner, style: .continuous))
        .preferredColorScheme(.dark)
    }
}
