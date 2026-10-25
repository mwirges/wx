import SwiftUI
import WidgetKit

@main
struct WxWidgetBundle: WidgetBundle {
    var body: some Widget {
        WxConditionsWidget()
        NowcastLiveActivityWidget()
    }
}
