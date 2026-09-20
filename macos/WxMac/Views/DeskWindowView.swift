import SwiftUI

struct DeskWindowView: View {
    @EnvironmentObject var store: WeatherStore

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                ControlsBar(showOpenWindow: false)
                Divider()
                NowBlockView(compact: false)
                AlertsListView()
                PeriodsListView(limit: nil)
            }
            .padding(16)
        }
        .frame(minWidth: 420, minHeight: 560)
    }
}
