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

                Picker("Desk Mode", selection: $store.selectedDeskTab) {
                    ForEach(DeskTab.allCases) { tab in
                        Text(tab.rawValue).tag(tab)
                    }
                }
                .pickerStyle(.segmented)
                .labelsHidden()
                .onChange(of: store.selectedDeskTab) { _, newTab in
                    if newTab == .radar && store.radarImage == nil && !store.isRadarLoading {
                        Task { await store.refreshRadar() }
                    }
                }

                Rectangle()
                    .fill(WxTheme.border)
                    .frame(height: 1)

                if store.selectedDeskTab == .weather {
                    NowBlockView(compact: false, popoverMetrics: false)
                    AlertsListView(popoverMode: false)
                    PeriodsListView(limit: nil, compactRows: false)
                } else {
                    RadarPanelView()
                }
            }
            .padding(16)
        }
        .frame(minWidth: 420, minHeight: 620)
        .background(WarpGlassBackground())
        .preferredColorScheme(.dark)
    }
}
