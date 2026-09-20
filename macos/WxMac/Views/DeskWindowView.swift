import SwiftUI

struct DeskWindowView: View {
    @EnvironmentObject var store: WeatherStore

    private var updatedSubtitle: String? {
        store.lastRefreshed.map { $0.formatted(date: .omitted, time: .shortened) }
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 14) {
                PillHeader(title: "wx Desk", subtitle: updatedSubtitle.map { "Updated \($0)" })
                ControlsBar(showOpenWindow: false, compact: false)
                Rectangle()
                    .fill(WxTheme.border)
                    .frame(height: 1)
                NowBlockView(compact: false, popoverMetrics: false)
                AlertsListView(popoverMode: false)
                PeriodsListView(limit: nil, compactRows: false)
            }
            .padding(16)
        }
        .frame(minWidth: 420, minHeight: 620)
        .background(WarpGlassBackground())
        .preferredColorScheme(.dark)
    }
}
