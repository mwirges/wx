import SwiftUI

struct PopoverView: View {
    @EnvironmentObject var store: WeatherStore

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 12) {
                ControlsBar(showOpenWindow: true)
                Divider()
                NowBlockView(compact: true)
                AlertsListView()
                PeriodsListView(limit: 6)
            }
            .padding(12)
        }
        .frame(width: 360, height: 480)
    }
}
