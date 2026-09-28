import SwiftUI
import AppKit
import Foundation

/// NOAA National Hurricane Center (NHC) Tropical Cyclone & Invest Tracker Console.
struct TropicsView: View {
    @EnvironmentObject var store: WeatherStore

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 14) {
                // Header & Refresh Action
                HStack(alignment: .center, spacing: 8) {
                    VStack(alignment: .leading, spacing: 2) {
                        Text("NOAA NATIONAL HURRICANE CENTER // TROPICAL TRACKER")
                            .font(.system(size: 11, weight: .bold, design: .monospaced))
                            .foregroundStyle(WxTheme.snwCyan)

                        if let payload = store.tropicsPayload?.tropics {
                            let count = payload.totalActive
                            let text = count > 0 ? "\(count) ACTIVE TROPICAL CYCLONE\(count == 1 ? "" : "S") DETECTED" : "ALL BASINS QUIET // NO ACTIVE CYCLONES"
                            Text(text)
                                .font(.system(size: 9.5, weight: .semibold, design: .monospaced))
                                .foregroundStyle(count > 0 ? WxTheme.snwAmber : WxTheme.snwGreen)
                        } else {
                            Text("SCANNING ATLANTIC & PACIFIC BASIN FEEDS…")
                                .font(.system(size: 9.5, weight: .semibold, design: .monospaced))
                                .foregroundStyle(WxTheme.textSecondary)
                        }
                    }

                    Spacer()

                    if store.isTropicsLoading {
                        ProgressView().controlSize(.small).tint(WxTheme.snwCyan)
                    }

                    Button {
                        Task {
                            await store.refreshTropics()
                        }
                    } label: {
                        HStack(spacing: 4) {
                            Image(systemName: "arrow.triangle.2.circlepath")
                            Text("SCAN BASINS")
                        }
                        .font(.system(size: 9, weight: .bold, design: .monospaced))
                        .foregroundStyle(WxTheme.snwCyan)
                        .padding(.horizontal, 9)
                        .padding(.vertical, 5)
                        .background(WxTheme.snwCyan.opacity(0.12), in: RoundedRectangle(cornerRadius: 4))
                        .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(WxTheme.snwCyan.opacity(0.4), lineWidth: 0.8))
                    }
                    .buttonStyle(.plain)
                    .disabled(store.isTropicsLoading)
                }
                .padding(.horizontal, 4)

                // Error banner
                if let err = store.tropicsErrorMessage {
                    HStack(spacing: 8) {
                        Image(systemName: "exclamationmark.triangle.fill")
                            .foregroundStyle(WxTheme.snwRed)
                        Text("// NHC TELEMETRY FAULT: \(err)")
                            .font(.system(size: 10, weight: .medium, design: .monospaced))
                            .foregroundStyle(WxTheme.snwRed)
                    }
                    .padding(8)
                    .background(WxTheme.snwRed.opacity(0.12), in: RoundedRectangle(cornerRadius: 6))
                    .overlay(RoundedRectangle(cornerRadius: 6).strokeBorder(WxTheme.snwRed.opacity(0.3), lineWidth: 0.8))
                }

                // Active Storms List
                if let storms = store.tropicsPayload?.tropics?.storms, !storms.isEmpty {
                    VStack(alignment: .leading, spacing: 14) {
                        ForEach(Array(storms.enumerated()), id: \.element.id) { index, storm in
                            TropicalStormCard(storm: storm, rank: index + 1)
                        }
                    }
                } else if store.isTropicsLoading {
                    VStack(spacing: 8) {
                        ProgressView().controlSize(.regular).tint(WxTheme.snwCyan)
                        Text("SCANNING NOAA NHC ADVISORIES & TRACK FORECASTS…")
                            .font(.system(size: 10, weight: .bold, design: .monospaced))
                            .foregroundStyle(WxTheme.snwCyan)
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 32)
                } else {
                    // Empty quiet state
                    SNWConsoleCard(title: "National Hurricane Center Status", tag: "NHC.QUIET", statusColor: WxTheme.snwGreen) {
                        HStack(spacing: 12) {
                            Image(systemName: "shield.lefthalf.filled.badge.checkmark")
                                .font(.system(size: 28))
                                .foregroundStyle(WxTheme.snwGreen)
                            VStack(alignment: .leading, spacing: 4) {
                                Text("ATLANTIC & PACIFIC BASINS QUIET")
                                    .font(.system(size: 10.5, weight: .bold, design: .monospaced))
                                    .foregroundStyle(WxTheme.snwGreen)
                                Text("No active tropical depressions, storms, or hurricanes currently monitored.")
                                    .font(.system(size: 9, design: .monospaced))
                                    .foregroundStyle(WxTheme.textSecondary)
                            }
                        }
                        .padding(10)
                    }
                }

                // Disturbances Section (Invest areas from TWO)
                if let disturbances = store.tropicsPayload?.tropics?.disturbances, !disturbances.isEmpty {
                    disturbancesSection(disturbances: disturbances)
                }

                // Basin 7-Day Graphical Maps Links
                basinMapsFooter
            }
            .padding(14)
        }
        .background(WxTheme.bg)
        .task {
            if store.tropicsPayload == nil && !store.isTropicsLoading {
                await store.refreshTropics()
            }
        }
    }

    @ViewBuilder
    private func disturbancesSection(disturbances: [TropicalDisturbanceDTO]) -> some View {
        SNWConsoleCard(
            title: "TROPICAL WEATHER OUTLOOK // INVEST DISTURBANCES",
            tag: "\(disturbances.count) ACTIVE INVESTS",
            statusColor: WxTheme.snwAmber
        ) {
            VStack(alignment: .leading, spacing: 10) {
                ForEach(disturbances) { d in
                    VStack(alignment: .leading, spacing: 5) {
                        HStack(spacing: 6) {
                            Text("[\(d.id)]")
                                .font(.system(size: 9.5, weight: .bold, design: .monospaced))
                                .foregroundStyle(WxTheme.snwAmber)
                            Text(d.name)
                                .font(.system(size: 9.5, weight: .bold, design: .monospaced))
                                .foregroundStyle(WxTheme.text)
                            Text("· \(d.basin) Basin")
                                .font(.system(size: 8.5, design: .monospaced))
                                .foregroundStyle(WxTheme.textSecondary)
                            Spacer()
                        }

                        HStack(spacing: 8) {
                            formationPill(label: "48-HOUR", chance: d.chance48h, cat: d.category48h)
                            formationPill(label: "7-DAY", chance: d.chance7d, cat: d.category7d)
                        }

                        if let summary = d.summary, !summary.isEmpty {
                            Text(summary)
                                .font(.system(size: 8.5, design: .monospaced))
                                .foregroundStyle(WxTheme.snwSilver)
                                .lineLimit(3)
                        }
                    }
                    .padding(8)
                    .background(WxTheme.snwChassis.opacity(0.7), in: RoundedRectangle(cornerRadius: 4))
                }
            }
        }
    }

    @ViewBuilder
    private func formationPill(label: String, chance: Int, cat: String) -> some View {
        let color = chanceColor(chance)
        HStack(spacing: 4) {
            Text(label)
                .font(.system(size: 7.5, weight: .bold, design: .monospaced))
                .foregroundStyle(WxTheme.snwSilver)
            Text("\(chance)% [\(cat.uppercased())]")
                .font(.system(size: 8, weight: .bold, design: .monospaced))
                .foregroundStyle(color)
        }
        .padding(.horizontal, 6)
        .padding(.vertical, 3)
        .background(color.opacity(0.12), in: RoundedRectangle(cornerRadius: 3))
        .overlay(RoundedRectangle(cornerRadius: 3).strokeBorder(color.opacity(0.4), lineWidth: 0.6))
    }

    private func chanceColor(_ chance: Int) -> Color {
        if chance >= 60 { return WxTheme.snwRed }
        if chance >= 40 { return WxTheme.snwAmber }
        if chance >= 20 { return WxTheme.snwGold }
        return WxTheme.snwGreen
    }

    @ViewBuilder
    private var basinMapsFooter: some View {
        if let tropics = store.tropicsPayload?.tropics {
            HStack(spacing: 10) {
                if let urlStr = tropics.atlanticOutlookUrl, let url = URL(string: urlStr) {
                    Button {
                        NSWorkspace.shared.open(url)
                    } label: {
                        HStack(spacing: 4) {
                            Image(systemName: "map.fill")
                            Text("ATLANTIC 7-DAY OUTLOOK")
                        }
                        .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                        .foregroundStyle(WxTheme.snwCyan)
                        .padding(.horizontal, 10)
                        .padding(.vertical, 5)
                        .background(WxTheme.snwCyan.opacity(0.12), in: RoundedRectangle(cornerRadius: 4))
                        .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(WxTheme.snwCyan.opacity(0.4), lineWidth: 0.8))
                    }
                    .buttonStyle(.plain)
                }

                if let urlStr = tropics.pacificOutlookUrl, let url = URL(string: urlStr) {
                    Button {
                        NSWorkspace.shared.open(url)
                    } label: {
                        HStack(spacing: 4) {
                            Image(systemName: "map.fill")
                            Text("EAST PACIFIC 7-DAY OUTLOOK")
                        }
                        .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                        .foregroundStyle(WxTheme.snwCyan)
                        .padding(.horizontal, 10)
                        .padding(.vertical, 5)
                        .background(WxTheme.snwCyan.opacity(0.12), in: RoundedRectangle(cornerRadius: 4))
                        .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(WxTheme.snwCyan.opacity(0.4), lineWidth: 0.8))
                    }
                    .buttonStyle(.plain)
                }

                Spacer()
            }
            .padding(.top, 4)
        }
    }
}

/// Storm telemetry card for individual tropical cyclones.
struct TropicalStormCard: View {
    @EnvironmentObject var store: WeatherStore
    let storm: TropicalStormDTO
    let rank: Int

    private var statusColor: Color {
        switch storm.category {
        case 5: return Color(red: 1.0, green: 0.0, blue: 1.0) // Magenta
        case 4: return WxTheme.snwRed
        case 3: return WxTheme.snwAmber
        case 1, 2: return WxTheme.snwGold
        default:
            if storm.classification.uppercased() == "TS" {
                return WxTheme.snwCyan
            }
            return WxTheme.snwGreen
        }
    }

    private var cardTitle: String {
        var t = "#\(rank)  \(storm.classificationName ?? "CYCLONE") \(storm.name.uppercased())"
        if let adv = storm.advisoryNumber {
            t += " #\(adv)"
        }
        return t
    }

    private var cardTag: String {
        storm.categoryLabel?.uppercased() ?? storm.classification.uppercased()
    }

    var body: some View {
        SNWConsoleCard(title: cardTitle, tag: cardTag, statusColor: statusColor) {
            VStack(alignment: .leading, spacing: 10) {
                // Top row with Position & External Graphics Action
                HStack(alignment: .top) {
                    VStack(alignment: .leading, spacing: 4) {
                        HStack(spacing: 6) {
                            Text("POSITION: \(storm.locationText)")
                                .font(.system(size: 9, weight: .bold, design: .monospaced))
                                .foregroundStyle(WxTheme.text)
                            if let dist = storm.distanceMiles {
                                Text("· \(Int(dist)) mi away")
                                    .font(.system(size: 8.5, weight: .semibold, design: .monospaced))
                                    .foregroundStyle(WxTheme.snwCyan)
                            }
                        }

                        if let prox = storm.proximityText, !prox.isEmpty {
                            Text(prox)
                                .font(.system(size: 8.5, design: .monospaced))
                                .foregroundStyle(WxTheme.textSecondary)
                        }
                    }

                    Spacer()

                    HStack(spacing: 6) {
                        if let gUrl = storm.graphicsUrl, let url = URL(string: gUrl) {
                            Button {
                                NSWorkspace.shared.open(url)
                            } label: {
                                HStack(spacing: 4) {
                                    Image(systemName: "tornado")
                                        .font(.system(size: 9, weight: .bold))
                                    Text("TRACK & CONE")
                                        .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                                }
                                .padding(.horizontal, 8)
                                .padding(.vertical, 5)
                                .background(statusColor.opacity(0.15), in: RoundedRectangle(cornerRadius: 4))
                                .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(statusColor.opacity(0.5), lineWidth: 0.8))
                                .foregroundStyle(statusColor)
                            }
                            .buttonStyle(.plain)
                            .help("Open NOAA NHC track and cone of uncertainty graphics")
                        }

                        Button {
                            // Intercept coordinates and jump to radar
                            let coord = String(format: "%.4f,%.4f", storm.latitude, storm.longitude)
                            store.selectLocation(coord)
                            store.selectedDeskTab = .radar
                            store.selectedRadarRadius = 250
                            Task { await store.refreshRadar() }
                        } label: {
                            HStack(spacing: 4) {
                                Image(systemName: "scope")
                                    .font(.system(size: 9, weight: .bold))
                                Text("RADAR")
                                    .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                            }
                            .padding(.horizontal, 8)
                            .padding(.vertical, 5)
                            .background(WxTheme.snwCyan.opacity(0.12), in: RoundedRectangle(cornerRadius: 4))
                            .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(WxTheme.snwCyan.opacity(0.4), lineWidth: 0.8))
                            .foregroundStyle(WxTheme.snwCyan)
                        }
                        .buttonStyle(.plain)
                        .help("Focus Doppler Radar on cyclone coordinates")
                    }
                }

                // 4-cell Metric Grid
                LazyVGrid(columns: [
                    GridItem(.flexible()),
                    GridItem(.flexible()),
                    GridItem(.flexible()),
                    GridItem(.flexible())
                ], spacing: 8) {
                    tropicsMetricCell(label: "MAX WINDS", val: "\(storm.windSpeedMph) mph (\(storm.intensityKt) kt)", highlight: storm.intensityKt >= 64)
                    tropicsMetricCell(label: "PRESSURE", val: storm.pressureMb > 0 ? "\(storm.pressureMb) mb" : "N/A", highlight: storm.pressureMb > 0 && storm.pressureMb <= 980)
                    tropicsMetricCell(label: "MOVEMENT", val: "\(storm.movementCompass) @ \(storm.movementSpeedMph) mph", highlight: false)
                    tropicsMetricCell(label: "HEADING", val: "\(storm.movementDir)°", highlight: false)
                }

                // Headline if present
                if let headline = storm.headline, !headline.isEmpty {
                    HStack(spacing: 5) {
                        Image(systemName: "bolt.fill")
                            .font(.system(size: 8.5))
                            .foregroundStyle(WxTheme.snwAmber)
                        Text(headline)
                            .font(.system(size: 8.5, weight: .medium, design: .monospaced))
                            .foregroundStyle(WxTheme.snwAmber)
                            .lineLimit(2)
                    }
                    .padding(6)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .background(WxTheme.snwAmber.opacity(0.08), in: RoundedRectangle(cornerRadius: 4))
                }

                // Active Watches & Warnings if present
                if let watches = storm.watchesWarnings, !watches.isEmpty {
                    VStack(alignment: .leading, spacing: 3) {
                        HStack(spacing: 4) {
                            Image(systemName: "exclamationmark.triangle.fill")
                                .font(.system(size: 8.5))
                                .foregroundStyle(WxTheme.snwRed)
                            Text("ACTIVE COASTAL WATCHES / WARNINGS:")
                                .font(.system(size: 8, weight: .bold, design: .monospaced))
                                .foregroundStyle(WxTheme.snwRed)
                        }
                        ForEach(watches, id: \.self) { w in
                            Text("• \(w)")
                                .font(.system(size: 8, design: .monospaced))
                                .foregroundStyle(WxTheme.text)
                        }
                    }
                    .padding(6)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .background(WxTheme.snwRed.opacity(0.08), in: RoundedRectangle(cornerRadius: 4))
                }

                // Bulletins Links
                HStack(spacing: 12) {
                    if let pUrl = storm.publicAdvisoryUrl, let url = URL(string: pUrl) {
                        Button {
                            NSWorkspace.shared.open(url)
                        } label: {
                            HStack(spacing: 3) {
                                Text("PUBLIC ADVISORY")
                                Image(systemName: "arrow.up.right.square")
                            }
                            .font(.system(size: 8, weight: .bold, design: .monospaced))
                            .foregroundStyle(WxTheme.snwSilver)
                        }
                        .buttonStyle(.plain)
                    }

                    if let dUrl = storm.forecastDiscussionUrl, let url = URL(string: dUrl) {
                        Button {
                            NSWorkspace.shared.open(url)
                        } label: {
                            HStack(spacing: 3) {
                                Text("FORECAST DISCUSSION")
                                Image(systemName: "arrow.up.right.square")
                            }
                            .font(.system(size: 8, weight: .bold, design: .monospaced))
                            .foregroundStyle(WxTheme.snwSilver)
                        }
                        .buttonStyle(.plain)
                    }

                    Spacer()
                }
            }
        }
    }

    @ViewBuilder
    private func tropicsMetricCell(label: String, val: String, highlight: Bool) -> some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(label)
                .font(.system(size: 7.5, weight: .bold, design: .monospaced))
                .foregroundStyle(WxTheme.snwSilver)
            Text(val)
                .font(.system(size: 9.5, weight: .bold, design: .monospaced))
                .foregroundStyle(highlight ? statusColor : WxTheme.text)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(5)
        .background(WxTheme.snwChassis.opacity(0.8), in: RoundedRectangle(cornerRadius: 3))
    }
}
