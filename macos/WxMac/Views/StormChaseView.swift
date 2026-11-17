import SwiftUI
import AppKit
import Foundation

/// Remote Storm Chasing and Active Severe Weather Alert Cluster Console.
struct StormChaseView: View {
    @EnvironmentObject var store: WeatherStore
    @State private var expandedClusterIDs: Set<Int> = []

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 14) {
                // Top Header & Controls
                HStack(alignment: .center, spacing: 8) {
                    VStack(alignment: .leading, spacing: 2) {
                        Text("REMOTE STORM CHASING · NATIONAL SEVERE CLUSTERS")
                            .font(.system(size: 11, weight: .bold, design: .rounded))
                            .foregroundStyle(WxGlassTheme.cyan)

                        if let payload = store.chasePayload {
                            Text("\(payload.totalClusters) ACTIVE CLUSTERS · \(payload.totalAlerts) SEVERE CELLS DETECTED")
                                .font(.system(size: 9.5, weight: .semibold, design: .monospaced))
                                .foregroundStyle(payload.totalClusters > 0 ? WxTheme.snwAmber : WxTheme.snwGreen)
                        } else {
                            Text("SCANNING CONUS RADAR & ALERT NETWORK…")
                                .font(.system(size: 9.5, weight: .semibold, design: .monospaced))
                                .foregroundStyle(WxTheme.textSecondary)
                        }
                    }

                    Spacer()

                    if store.isChaseLoading || store.isSPCLoading {
                        ProgressView().controlSize(.small).tint(WxTheme.snwCyan)
                    }

                    Button {
                        Task {
                            await store.refreshChase()
                            await store.refreshSPC()
                        }
                    } label: {
                        HStack(spacing: 4) {
                            Image(systemName: "arrow.triangle.2.circlepath")
                            Text("Scan for Storms")
                        }
                        .font(.system(size: 10.5, weight: .semibold, design: .rounded))
                        .foregroundStyle(WxGlassTheme.cyan)
                        .padding(.horizontal, 12)
                        .padding(.vertical, 5.5)
                        .wxFrostedCapsule(accent: WxGlassTheme.cyan, active: true)
                    }
                    .buttonStyle(.plain)
                    .disabled(store.isChaseLoading)
                }
                .padding(.horizontal, 4)

                // Error message banner if any
                if let err = store.chaseErrorMessage ?? store.spcErrorMessage {
                    HStack(spacing: 8) {
                        Image(systemName: "exclamationmark.triangle.fill")
                            .foregroundStyle(WxGlassTheme.conditionRed)
                        Text(err)
                            .font(.system(size: 11.5, weight: .medium))
                            .foregroundStyle(WxGlassTheme.conditionRed)
                    }
                    .padding(10)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .background(WxGlassTheme.conditionRed.opacity(0.12), in: WxGlassTheme.nestedShape())
                    .overlay(WxGlassTheme.nestedShape().strokeBorder(WxGlassTheme.conditionRed.opacity(0.35), lineWidth: 0.8))
                }

                // ── SPC Convective Outlook & Active Mesoscale Discussions ───────────
                if let spc = store.spcPayload ?? store.chasePayload?.spc {
                    spcSection(spc: spc)
                }

                // ── Upper-Air Sounding & Convective Profile ─────────────────────────
                if let sounding = store.soundingPayload?.sounding {
                    SoundingConsoleCard(sounding: sounding) {
                        store.soundingPayload = nil
                    }
                } else if store.isSoundingLoading {
                    HStack(spacing: 8) {
                        ProgressView().controlSize(.small).tint(WxGlassTheme.amber)
                        Text("Retrieving radiosonde telemetry & NSHARP profile…")
                            .font(.system(size: 11, weight: .semibold, design: .rounded))
                            .foregroundStyle(WxGlassTheme.amber)
                    }
                    .padding(10)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .background(WxGlassTheme.amber.opacity(0.10), in: WxGlassTheme.nestedShape())
                    .overlay(WxGlassTheme.nestedShape().strokeBorder(WxGlassTheme.amber.opacity(0.30), lineWidth: 0.8))
                }

                // Cluster cards list
                if let clusters = store.chasePayload?.clusters, !clusters.isEmpty {
                    VStack(alignment: .leading, spacing: 14) {
                        ForEach(Array(clusters.enumerated()), id: \.element.id) { index, cluster in
                            clusterCard(cluster: cluster, rank: index + 1)
                        }
                    }
                } else if store.isChaseLoading {
                    VStack(spacing: 8) {
                        ProgressView().controlSize(.regular).tint(WxTheme.snwCyan)
                        Text("SCANNING NWS HAZARDS & GEOGRAPHIC PROXIMITY GRAPHS…")
                            .font(.system(size: 10, weight: .bold, design: .monospaced))
                            .foregroundStyle(WxTheme.snwCyan)
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 32)
                } else {
                    // Empty quiet state
                    WxGlassCard(title: "CONUS Hazard Intercept Status", tag: "NWS.QUIET", accent: WxGlassTheme.nominalGreen) {
                        HStack(spacing: 12) {
                            Image(systemName: "shield.lefthalf.filled.badge.checkmark")
                                .font(.system(size: 28))
                                .foregroundStyle(WxGlassTheme.nominalGreen)
                            VStack(alignment: .leading, spacing: 4) {
                                Text("CONUS Sensor Network Quiet")
                                    .font(.system(size: 14, weight: .bold, design: .rounded))
                                    .foregroundStyle(WxGlassTheme.textPrimary)
                                Text("No severe weather clusters currently detected across the contiguous United States. Check back during active convective or winter storm cycles.")
                                    .font(.system(size: 12))
                                    .foregroundStyle(WxGlassTheme.textSecondary)
                            }
                        }
                        .padding(.vertical, 6)
                    }
                }

                // Footer Citation
                HStack {
                    Text("NOAA SPC CONVECTIVE INTELLIGENCE & NWS CLUSTER INTERCEPT ENGINE")
                        .font(.system(size: 8, weight: .medium, design: .monospaced))
                        .foregroundStyle(WxTheme.snwSilver.opacity(0.6))
                    Spacer()
                    if let gen = store.chasePayload?.generatedAt {
                        Text("UPDATED · \(gen.prefix(19))")
                            .font(.system(size: 8, weight: .medium, design: .monospaced))
                            .foregroundStyle(WxTheme.snwSilver.opacity(0.6))
                    }
                }
                .padding(.top, 4)
                .padding(.horizontal, 4)
            }
            .padding(14)
        }
    }

    // ── SPC Convective Outlook & Active Mesoscale Discussions Card ────────────────

    @ViewBuilder
    private func spcSection(spc: SPCPayloadDTO) -> some View {
        let ceilingCode = spc.maxNationalRisk?.code ?? "NONE"
        let statusCol = spcRiskColor(ceilingCode)

        SNWConsoleCard(
            title: "SPC Convective Outlook & Mesoscale Intelligence",
            tag: "CEILING · \(ceilingCode)",
            statusColor: statusCol
        ) {
            VStack(alignment: .leading, spacing: 10) {
                // Convective Risk Pills for Day 1, Day 2, Day 3
                HStack(spacing: 8) {
                    if let d1 = spc.day1 {
                        dayRiskPill(day: "DAY 1", item: d1)
                    }
                    if let d2 = spc.day2 {
                        dayRiskPill(day: "DAY 2", item: d2)
                    }
                    if let d3 = spc.day3 {
                        dayRiskPill(day: "DAY 3", item: d3)
                    }
                    Spacer()
                }

                // Convective summary if available
                if let summary = spc.convectiveSummary, !summary.isEmpty {
                    Text(summary)
                        .font(.system(size: 11.5))
                        .foregroundStyle(WxGlassTheme.textSecondary)
                }

                // Active Watches alert banner if any
                if let watches = spc.activeWatches, !watches.isEmpty {
                    VStack(alignment: .leading, spacing: 6) {
                        ForEach(watches) { watch in
                            HStack(spacing: 6) {
                                Image(systemName: "exclamationmark.triangle.fill")
                                    .foregroundStyle(WxTheme.snwRed)
                                Text("ACTIVE WATCH #\(watch.watchNumber) — \(watch.type.uppercased())")
                                    .font(.system(size: 9, weight: .bold, design: .monospaced))
                                    .foregroundStyle(WxTheme.snwRed)
                                if let area = watch.areaDesc {
                                    Text("· \(area)")
                                        .font(.system(size: 8.5, design: .monospaced))
                                        .foregroundStyle(WxTheme.textSecondary)
                                        .lineLimit(1)
                                }
                                Spacer()
                                if let u = watch.url, let url = URL(string: u) {
                                    Button {
                                        NSWorkspace.shared.open(url)
                                    } label: {
                                        Text("Details")
                                            .font(.system(size: 9.5, weight: .bold, design: .rounded))
                                            .foregroundStyle(WxGlassTheme.conditionRed)
                                            .padding(.horizontal, 8)
                                            .padding(.vertical, 3)
                                            .background(WxGlassTheme.conditionRed.opacity(0.16), in: Capsule())
                                            .overlay(Capsule().strokeBorder(WxGlassTheme.conditionRed.opacity(0.4), lineWidth: 0.8))
                                    }
                                    .buttonStyle(.plain)
                                }
                            }
                            .padding(8)
                            .background(WxGlassTheme.conditionRed.opacity(0.10), in: WxGlassTheme.nestedShape())
                            .overlay(WxGlassTheme.nestedShape().strokeBorder(WxGlassTheme.conditionRed.opacity(0.32), lineWidth: 0.8))
                        }
                    }
                }

                // Active Mesoscale Discussions (MCD) if any
                if let mcds = spc.activeMCDs, !mcds.isEmpty {
                    VStack(alignment: .leading, spacing: 6) {
                        ForEach(mcds) { mcd in
                            VStack(alignment: .leading, spacing: 4) {
                                HStack {
                                    HStack(spacing: 4) {
                                        Image(systemName: "bolt.fill")
                                            .font(.system(size: 9))
                                            .foregroundStyle(WxTheme.snwAmber)
                                        Text("\(mcd.name.uppercased()) · \(mcd.concerning?.uppercased() ?? "MESOSCALE DISCUSSION")")
                                            .font(.system(size: 9, weight: .bold, design: .monospaced))
                                            .foregroundStyle(WxTheme.snwAmber)
                                    }

                                    if let prob = mcd.watchProbability, !prob.isEmpty {
                                        Text("· WATCH PROB: \(prob.uppercased())")
                                            .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                                            .foregroundStyle(WxTheme.snwGold)
                                    }

                                    Spacer()

                                    if let u = mcd.url, let url = URL(string: u) {
                                        Button {
                                            NSWorkspace.shared.open(url)
                                        } label: {
                                            HStack(spacing: 3) {
                                                Text("Open MCD")
                                                Image(systemName: "arrow.up.right.square")
                                            }
                                            .font(.system(size: 9.5, weight: .semibold, design: .rounded))
                                            .foregroundStyle(WxGlassTheme.cyan)
                                            .padding(.horizontal, 8)
                                            .padding(.vertical, 3)
                                            .background(WxGlassTheme.cyan.opacity(0.14), in: Capsule())
                                            .overlay(Capsule().strokeBorder(WxGlassTheme.cyan.opacity(0.35), lineWidth: 0.8))
                                        }
                                        .buttonStyle(.plain)
                                    }
                                }

                                if let areas = mcd.areasAffected, !areas.isEmpty {
                                    Text("AREAS: \(areas)")
                                        .font(.system(size: 8.5, weight: .semibold, design: .monospaced))
                                        .foregroundStyle(WxTheme.textSecondary)
                                }

                                if let summary = mcd.summary, !summary.isEmpty {
                                    Text(summary)
                                        .font(.system(size: 11.5))
                                        .foregroundStyle(WxGlassTheme.textPrimary)
                                        .lineLimit(3)
                                }
                            }
                            .padding(10)
                            .background(WxGlassTheme.amber.opacity(0.08), in: WxGlassTheme.nestedShape())
                            .overlay(WxGlassTheme.nestedShape().strokeBorder(WxGlassTheme.amber.opacity(0.35), lineWidth: 0.8))
                        }
                    }
                }
            }
        }
    }

    @ViewBuilder
    private func dayRiskPill(day: String, item: SPCOutlookItemDTO) -> some View {
        let color = spcRiskColor(item.category.code)
        HStack(spacing: 6) {
            Text(day)
                .font(.system(size: 9, weight: .bold, design: .rounded))
                .foregroundStyle(WxGlassTheme.textSecondary)
            Text(item.category.code)
                .font(.system(size: 9.5, weight: .bold, design: .monospaced))
                .foregroundStyle(Color(red: 0.03, green: 0.07, blue: 0.14))
                .padding(.horizontal, 6)
                .padding(.vertical, 2)
                .background(color, in: Capsule())
            Text(item.category.name)
                .font(.system(size: 10, weight: .semibold, design: .rounded))
                .foregroundStyle(WxGlassTheme.textPrimary)
                .lineLimit(1)
        }
        .padding(.horizontal, 10)
        .padding(.vertical, 5.5)
        .background(Color.white.opacity(0.06), in: Capsule())
        .overlay(Capsule().strokeBorder(color.opacity(0.55), lineWidth: 0.9))
    }

    // ── Cluster Card ─────────────────────────────────────────────────────────────

    @ViewBuilder
    private func clusterCard(cluster: StormClusterDTO, rank: Int) -> some View {
        let isExpanded = expandedClusterIDs.contains(cluster.id)
        let cardColor = clusterColor(score: cluster.score)

        SNWConsoleCard(
            title: "#\(rank)  \(cluster.name.uppercased())",
            tag: "SCORE · \(cluster.score)",
            statusColor: cardColor
        ) {
            VStack(alignment: .leading, spacing: 10) {
                // Key metadata & Quick Jump Action
                HStack(alignment: .top) {
                    VStack(alignment: .leading, spacing: 6) {
                        // States list
                        HStack(spacing: 4) {
                            Text("AFFECTED:")
                                .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                                .foregroundStyle(WxTheme.snwSilver)
                            ForEach(cluster.states, id: \.self) { st in
                                Text(st)
                                    .font(.system(size: 9, weight: .bold, design: .monospaced))
                                    .foregroundStyle(WxGlassTheme.textPrimary)
                                    .padding(.horizontal, 7)
                                    .padding(.vertical, 2.5)
                                    .background(Color.white.opacity(0.08), in: Capsule())
                                    .overlay(Capsule().strokeBorder(WxGlassTheme.cyan.opacity(0.35), lineWidth: 0.7))
                            }

                            if let spcR = cluster.spcRisk, !spcR.isEmpty && spcR != "NONE" {
                                Text("SPC · \(spcR)")
                                    .font(.system(size: 9, weight: .bold, design: .rounded))
                                    .foregroundStyle(spcRiskColor(spcR))
                                    .padding(.horizontal, 7)
                                    .padding(.vertical, 2.5)
                                    .background(spcRiskColor(spcR).opacity(0.15), in: Capsule())
                                    .overlay(Capsule().strokeBorder(spcRiskColor(spcR).opacity(0.45), lineWidth: 0.7))
                            }
                        }

                        // Primary Hazard & Count
                        HStack(spacing: 8) {
                            HStack(spacing: 4) {
                                Image(systemName: "exclamationmark.triangle.fill")
                                    .font(.system(size: 9))
                                    .foregroundStyle(cardColor)
                                Text(cluster.primaryHazard.uppercased())
                                    .font(.system(size: 9, weight: .bold, design: .monospaced))
                                    .foregroundStyle(cardColor)
                            }

                            Text("·  \(cluster.totalAlerts) ALERT\(cluster.totalAlerts == 1 ? "" : "S")")
                                .font(.system(size: 8.5, weight: .medium, design: .monospaced))
                                .foregroundStyle(WxTheme.snwSilver)

                            if !cluster.nearestRadar.isEmpty {
                                Text("·  RADAR \(cluster.nearestRadar)")
                                    .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                                    .foregroundStyle(WxTheme.snwCyan)
                            }

                            if let stn = cluster.soundingStation, !stn.isEmpty {
                                Text("·  UPPER-AIR \(stn)")
                                    .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                                    .foregroundStyle(WxTheme.snwAmber)
                            }
                        }

                        // Linked MCD or Watch Context if present
                        if let mcdWatch = cluster.mcdWatch, !mcdWatch.isEmpty {
                            HStack(spacing: 4) {
                                Image(systemName: "bolt.badge.clock.fill")
                                    .font(.system(size: 8))
                                    .foregroundStyle(WxTheme.snwAmber)
                                Text("SPC CONTEXT:")
                                    .font(.system(size: 8, weight: .bold, design: .monospaced))
                                    .foregroundStyle(WxTheme.snwSilver)
                                Text(mcdWatch)
                                    .font(.system(size: 8, weight: .semibold, design: .monospaced))
                                    .foregroundStyle(WxTheme.snwAmber)
                                    .lineLimit(1)
                            }
                        }
                    }

                    Spacer()

                    VStack(alignment: .trailing, spacing: 6) {
                        Button {
                            store.chaseCluster(cluster)
                        } label: {
                            HStack(spacing: 5) {
                                Image(systemName: "scope")
                                    .font(.system(size: 11, weight: .bold))
                                Text("Intercept Radar")
                                    .font(.system(size: 10.5, weight: .bold, design: .rounded))
                            }
                            .padding(.horizontal, 12)
                            .padding(.vertical, 6)
                            .background(
                                LinearGradient(
                                    colors: [cardColor.opacity(0.28), cardColor.opacity(0.12)],
                                    startPoint: .topLeading,
                                    endPoint: .bottomTrailing
                                ),
                                in: Capsule()
                            )
                            .overlay(Capsule().strokeBorder(cardColor.opacity(0.85), lineWidth: 1))
                            .foregroundStyle(WxGlassTheme.textPrimary)
                            .shadow(color: cardColor.opacity(0.35), radius: 4)
                        }
                        .buttonStyle(.plain)
                        .help("Jump to this storm cluster and load live Doppler Radar array")

                        if let stn = cluster.soundingStation, !stn.isEmpty {
                            Button {
                                Task {
                                    await store.refreshSounding(station: stn)
                                }
                            } label: {
                                HStack(spacing: 4) {
                                    Image(systemName: "wind")
                                        .font(.system(size: 9.5, weight: .bold))
                                    Text("Load Sounding (\(stn))")
                                        .font(.system(size: 10, weight: .semibold, design: .rounded))
                                }
                                .padding(.horizontal, 10)
                                .padding(.vertical, 5)
                                .background(WxGlassTheme.amber.opacity(0.14), in: Capsule())
                                .overlay(Capsule().strokeBorder(WxGlassTheme.amber.opacity(0.45), lineWidth: 0.8))
                                .foregroundStyle(WxGlassTheme.amber)
                            }
                            .buttonStyle(.plain)
                            .help("Retrieve Upper-Air Sounding and Convective Profile for \(stn)")
                        }
                    }
                }

                // Hazard Types Badges
                if !cluster.hazardsCount.isEmpty {
                    HStack(spacing: 6) {
                        ForEach(cluster.hazardsCount.sorted(by: { $0.value > $1.value }), id: \.key) { hazard, count in
                            Text("\(hazard) (\(count))")
                                .font(.system(size: 8, weight: .semibold, design: .monospaced))
                                .foregroundStyle(WxTheme.snwSilver)
                                .padding(.horizontal, 6)
                                .padding(.vertical, 2)
                                .background(WxTheme.snwChassis.opacity(0.7), in: Capsule())
                        }
                    }
                }

                // Expandable Alerts Drawer Toggle
                Button {
                    if isExpanded {
                        expandedClusterIDs.remove(cluster.id)
                    } else {
                        expandedClusterIDs.insert(cluster.id)
                    }
                } label: {
                    HStack(spacing: 4) {
                        Image(systemName: isExpanded ? "chevron.down" : "chevron.right")
                            .font(.system(size: 8.5, weight: .bold))
                        Text(isExpanded ? "HIDE ACTIVE WARNING CELLS" : "VIEW \(cluster.cells.count) WARNING CELLS IN CLUSTER")
                            .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                    }
                    .foregroundStyle(WxTheme.snwCyan)
                    .padding(.top, 2)
                }
                .buttonStyle(.plain)

                // Expanded Cells List
                if isExpanded {
                    VStack(alignment: .leading, spacing: 8) {
                        Divider().overlay(WxTheme.border.opacity(0.3))
                        ForEach(cluster.cells) { cell in
                            cellRow(cell: cell)
                        }
                    }
                    .padding(.top, 4)
                }
            }
        }
    }

    @ViewBuilder
    private func cellRow(cell: AlertCellDTO) -> some View {
        VStack(alignment: .leading, spacing: 3) {
            HStack {
                Text(cell.event.uppercased())
                    .font(.system(size: 10.5, weight: .bold, design: .rounded))
                    .foregroundStyle(WxGlassTheme.textPrimary)
                Spacer()
                if let area = cell.areaDesc {
                    Text(area)
                        .font(.system(size: 9.5, design: .monospaced))
                        .foregroundStyle(WxGlassTheme.textMuted)
                        .lineLimit(1)
                }
            }

            if let headline = cell.headline {
                Text(headline)
                    .font(.system(size: 11))
                    .foregroundStyle(WxGlassTheme.textSecondary)
                    .lineLimit(2)
            }
        }
        .padding(9)
        .background(Color.white.opacity(0.05), in: WxGlassTheme.nestedShape())
        .overlay(WxGlassTheme.nestedShape().strokeBorder(Color.white.opacity(0.10), lineWidth: 0.7))
    }

    private func clusterColor(score: Int) -> Color {
        if score >= 100 {
            return WxTheme.snwRed
        } else if score >= 40 {
            return WxTheme.snwAmber
        } else {
            return WxTheme.snwGold
        }
    }

    private func spcRiskColor(_ code: String?) -> Color {
        switch (code ?? "").uppercased() {
        case "HIGH": return Color(red: 1.0, green: 0.0, blue: 1.0) // Magenta
        case "MDT": return WxTheme.snwRed
        case "ENH": return WxTheme.snwAmber
        case "SLGT": return WxTheme.snwGold
        case "MRGL": return Color(red: 0.45, green: 0.75, blue: 0.45) // Dark Green
        case "TSTM": return WxTheme.snwGreen
        default: return WxTheme.snwSilver
        }
    }

}

struct SoundingConsoleCard: View {
    let sounding: SoundingReportDTO
    let onClose: () -> Void

    private var cardTitle: String {
        let stn = sounding.stationId ?? sounding.location ?? "PROFILE"
        return "UPPER-AIR SOUNDING · \(stn)"
    }

    private var cardTag: String {
        sounding.indices?.convectiveRisk ?? "SOUNDING"
    }

    private var stationTitle: String {
        sounding.stationName ?? sounding.stationId ?? "Station"
    }

    private var distanceText: String? {
        guard let d = sounding.distanceMiles else { return nil }
        return "· \(Int(d)) mi away"
    }

    private var providerText: String? {
        guard let p = sounding.provider else { return nil }
        return "· \(p.uppercased())"
    }

    private var riskColor: Color {
        switch (sounding.indices?.convectiveRisk ?? "").uppercased() {
        case "SEVERE": return WxTheme.snwRed
        case "ELEVATED": return WxTheme.snwAmber
        case "MODERATE": return WxTheme.snwGold
        case "MARGINAL": return WxTheme.snwGreen
        default: return WxTheme.snwCyan
        }
    }

    var body: some View {
        SNWConsoleCard(
            title: cardTitle,
            tag: cardTag,
            statusColor: riskColor
        ) {
            VStack(alignment: .leading, spacing: 10) {
                headerView
                if let idx = sounding.indices {
                    SoundingIndicesGrid(indices: idx)
                }
            }
        }
    }

    @ViewBuilder
    private var headerView: some View {
        HStack(alignment: .center) {
            VStack(alignment: .leading, spacing: 3) {
                HStack(spacing: 6) {
                    Text(stationTitle)
                        .font(.system(size: 10, weight: .bold, design: .monospaced))
                        .foregroundStyle(WxTheme.text)
                    if let dist = distanceText {
                        Text(dist)
                            .font(.system(size: 8.5, design: .monospaced))
                            .foregroundStyle(WxTheme.textSecondary)
                    }
                    if let prov = providerText {
                        Text(prov)
                            .font(.system(size: 8, weight: .bold, design: .monospaced))
                            .foregroundStyle(WxTheme.snwCyan)
                    }
                }
                if let time = sounding.timestamp {
                    Text("OBS CYCLE: \(time)")
                        .font(.system(size: 8, design: .monospaced))
                        .foregroundStyle(WxTheme.textTertiary)
                }
            }

            Spacer()

            HStack(spacing: 8) {
                if sounding.skewtImageUrl != nil {
                    Button {
                        openSkewT()
                    } label: {
                        HStack(spacing: 4) {
                            Image(systemName: "chart.line.uptrend.xyaxis")
                            Text("Open Skew-T")
                        }
                        .font(.system(size: 9.5, weight: .semibold, design: .rounded))
                        .foregroundStyle(WxGlassTheme.cyan)
                        .padding(.horizontal, 10)
                        .padding(.vertical, 4.5)
                        .wxFrostedCapsule(accent: WxGlassTheme.cyan)
                    }
                    .buttonStyle(.plain)
                }

                Button {
                    onClose()
                } label: {
                    Image(systemName: "xmark")
                        .font(.system(size: 9, weight: .bold))
                        .foregroundStyle(WxTheme.textTertiary)
                        .padding(4)
                }
                .buttonStyle(.plain)
            }
        }
    }

    private func openSkewT() {
        guard let urlStr = sounding.skewtImageUrl, let url = URL(string: urlStr) else { return }
        NSWorkspace.shared.open(url)
    }
}

struct SoundingIndicesGrid: View {
    let indices: ConvectiveIndicesDTO

    private var sbcapeText: String {
        if let v = indices.sbcapeJkg { return "\(Int(v)) J/kg" }
        return "—"
    }
    private var mlcapeText: String {
        if let v = indices.mlcapeJkg { return "\(Int(v)) J/kg" }
        return "—"
    }
    private var cinText: String {
        if let v = indices.sbcinJkg { return "\(Int(v)) J/kg" }
        return "—"
    }
    private var liText: String {
        if let v = indices.sbliC { return String(format: "%.1f°C", v) }
        return "—"
    }
    private var shear06Text: String {
        if let v = indices.bulkShear06Kt { return "\(Int(v)) kt" }
        return "—"
    }
    private var shear01Text: String {
        if let v = indices.bulkShear01Kt { return "\(Int(v)) kt" }
        return "—"
    }
    private var srhText: String {
        if let v = indices.srh01M2s2 { return "\(Int(v)) m²/s²" }
        return "—"
    }
    private var pwatText: String {
        if let v = indices.pwatIn { return String(format: "%.2f in", v) }
        if let v = indices.pwatMm { return String(format: "%.1f mm", v) }
        return "—"
    }

    var body: some View {
        LazyVGrid(columns: [
            GridItem(.flexible()),
            GridItem(.flexible()),
            GridItem(.flexible()),
            GridItem(.flexible())
        ], spacing: 8) {
            SoundingMetricCell(label: "SBCAPE", val: sbcapeText, highlight: (indices.sbcapeJkg ?? 0) >= 1500)
            SoundingMetricCell(label: "MLCAPE", val: mlcapeText, highlight: (indices.mlcapeJkg ?? 0) >= 1000)
            SoundingMetricCell(label: "CIN", val: cinText, highlight: false)
            SoundingMetricCell(label: "LIFTED IDX", val: liText, highlight: (indices.sbliC ?? 0) <= -3)
            SoundingMetricCell(label: "0-6KM SHEAR", val: shear06Text, highlight: (indices.bulkShear06Kt ?? 0) >= 40)
            SoundingMetricCell(label: "0-1KM SHEAR", val: shear01Text, highlight: (indices.bulkShear01Kt ?? 0) >= 20)
            SoundingMetricCell(label: "0-1KM SRH", val: srhText, highlight: (indices.srh01M2s2 ?? 0) >= 150)
            SoundingMetricCell(label: "PWAT", val: pwatText, highlight: false)
        }
    }
}

struct SoundingMetricCell: View {
    let label: String
    let val: String
    let highlight: Bool

    var body: some View {
        WxSensorCapsule(
            label: label,
            value: val,
            accent: highlight ? WxGlassTheme.amber : WxGlassTheme.cyan,
            compact: true
        )
    }
}
