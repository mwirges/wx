import SwiftUI

struct SurfaceView: View {
    @EnvironmentObject private var store: IOSStore

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 12) {
                if let error = store.surfaceError, !error.isEmpty {
                    Text(error)
                        .font(.system(size: 13))
                        .foregroundStyle(WxTheme.snwRed)
                        .accessibilityIdentifier("wx.error")
                }
                observation
                nowcastCard
                dayList
                hourlyStrip
            }
            .padding(12)
            .accessibilityElement(children: .contain)
        }
        .accessibilityIdentifier("surface.root")
    }

    private var observation: some View {
        let conditions = store.payload?.conditions
        return SNWConsoleCard(title: "Observation", tag: conditions?.station, statusColor: WxTheme.snwCyan) {
            HStack(alignment: .firstTextBaseline, spacing: 10) {
                Image(systemName: ConditionSymbol.systemName(for: conditions?.conditionCode))
                    .font(.system(size: 34))
                    .foregroundStyle(WxTheme.snwCyan)
                Text(IOSFormat.observation(conditions, units: store.units))
                    .font(.system(size: 42, weight: .bold, design: .rounded))
                    .foregroundStyle(WxTheme.text)
                    .accessibilityIdentifier("surface.temp")
                Spacer()
                if store.isLoading {
                    ProgressView().tint(WxTheme.snwCyan)
                }
            }
            Text(conditions?.location ?? store.place)
                .font(.headline)
                .foregroundStyle(WxTheme.text)
                .accessibilityIdentifier("surface.location")
            if let description = conditions?.description, !description.isEmpty {
                Text(description)
                    .font(.subheadline)
                    .foregroundStyle(WxTheme.textSecondary)
                    .accessibilityIdentifier("surface.condition")
            }
        }
    }

    /// Phase "none" uppercases to NONE, which reads as missing data next to a
    /// real dry headline such as "Clear". Hide that sentinel. A failed nowcast
    /// leaves the headline as an em dash.
    private var nowcastPhaseTag: String? {
        let phase = store.nowcast?.nowcast?.primaryPhase?
            .trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if phase.isEmpty || phase.caseInsensitiveCompare("none") == .orderedSame {
            return nil
        }
        return phase
    }

    private var nowcastCard: some View {
        SNWConsoleCard(title: "Nowcast", tag: nowcastPhaseTag, statusColor: WxTheme.snwGold) {
            Text(store.nowcastHeadline ?? "—")
                .font(.system(size: 16, weight: .semibold))
                .foregroundStyle(WxTheme.text)
                .accessibilityIdentifier("surface.nowcast")
            if let summary = store.nowcast?.nowcast?.summary, !summary.isEmpty {
                Text(summary)
                    .font(.system(size: 13))
                    .foregroundStyle(WxTheme.textSecondary)
            }
        }
    }

    private var dayList: some View {
        let periods = store.payload?.forecast?.periods ?? []
        return SNWConsoleCard(title: "Days", tag: nil) {
            if periods.isEmpty {
                Text("No day periods")
                    .foregroundStyle(WxTheme.textSecondary)
            } else {
                VStack(alignment: .leading, spacing: 8) {
                    ForEach(periods) { period in
                        HStack {
                            Text(period.name)
                                .font(.system(size: 14, weight: .medium))
                                .foregroundStyle(WxTheme.text)
                            Spacer()
                            Text(IOSFormat.period(period, units: store.units))
                                .font(.system(size: 14, weight: .bold, design: .monospaced))
                                .foregroundStyle(WxTheme.snwCyan)
                        }
                        if let blurb = period.shortDescription, !blurb.isEmpty {
                            Text(blurb)
                                .font(.system(size: 12))
                                .foregroundStyle(WxTheme.textSecondary)
                        }
                    }
                }
            }
        }
        .accessibilityIdentifier("surface.days")
    }

    private var hourlyStrip: some View {
        let hours = store.payload?.forecast?.hourly ?? []
        return SNWConsoleCard(title: "Hourly", tag: nil) {
            if hours.isEmpty {
                Text("No hourly periods")
                    .foregroundStyle(WxTheme.textSecondary)
            } else {
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: 10) {
                        ForEach(hours) { period in
                            VStack(spacing: 2) {
                                Text(IOSFormat.hourLabel(period))
                                    .font(.system(size: 11, weight: .medium, design: .monospaced))
                                    .foregroundStyle(WxTheme.textSecondary)
                                Text(IOSFormat.period(period, units: store.units))
                                    .font(.system(size: 13, weight: .bold, design: .monospaced))
                                    .foregroundStyle(WxTheme.snwCyan)
                            }
                            .frame(minWidth: 48)
                        }
                    }
                }
            }
        }
        .accessibilityIdentifier("surface.hourly")
    }
}
