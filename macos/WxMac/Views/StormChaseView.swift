import SwiftUI

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
                        Text("REMOTE STORM CHASING // NATIONAL SEVERE CLUSTERS")
                            .font(.system(size: 11, weight: .bold, design: .monospaced))
                            .foregroundStyle(WxTheme.snwCyan)

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

                    if store.isChaseLoading {
                        ProgressView().controlSize(.small).tint(WxTheme.snwCyan)
                    }

                    Button {
                        Task { await store.refreshChase() }
                    } label: {
                        HStack(spacing: 4) {
                            Image(systemName: "arrow.triangle.2.circlepath")
                            Text("SCAN FOR STORMS")
                        }
                        .font(.system(size: 9, weight: .bold, design: .monospaced))
                        .foregroundStyle(WxTheme.snwCyan)
                        .padding(.horizontal, 9)
                        .padding(.vertical, 5)
                        .background(WxTheme.snwCyan.opacity(0.12), in: RoundedRectangle(cornerRadius: 4))
                        .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(WxTheme.snwCyan.opacity(0.4), lineWidth: 0.8))
                    }
                    .buttonStyle(.plain)
                    .disabled(store.isChaseLoading)
                }
                .padding(.horizontal, 4)

                // Error message banner if any
                if let err = store.chaseErrorMessage {
                    HStack(spacing: 8) {
                        Image(systemName: "exclamationmark.triangle.fill")
                            .foregroundStyle(WxTheme.snwRed)
                        Text("// TELEMETRY FAULT: \(err)")
                            .font(.system(size: 10, weight: .medium, design: .monospaced))
                            .foregroundStyle(WxTheme.snwRed)
                    }
                    .padding(8)
                    .background(WxTheme.snwRed.opacity(0.12), in: RoundedRectangle(cornerRadius: 6))
                    .overlay(RoundedRectangle(cornerRadius: 6).strokeBorder(WxTheme.snwRed.opacity(0.3), lineWidth: 0.8))
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
                    SNWConsoleCard(title: "CONUS Hazard Intercept Status", tag: "NWS.QUIET", statusColor: WxTheme.snwGreen) {
                        HStack(spacing: 12) {
                            Image(systemName: "shield.lefthalf.filled.badge.checkmark")
                                .font(.system(size: 28))
                                .foregroundStyle(WxTheme.snwGreen)
                            VStack(alignment: .leading, spacing: 4) {
                                Text("CONUS SENSOR NETWORK QUIET")
                                    .font(.system(size: 12, weight: .bold, design: .monospaced))
                                    .foregroundStyle(WxTheme.text)
                                Text("No severe weather clusters currently detected across the contiguous United States. Check back during active convective or winter storm cycles.")
                                    .font(.system(size: 9.5, weight: .medium, design: .monospaced))
                                    .foregroundStyle(WxTheme.textSecondary)
                            }
                        }
                        .padding(.vertical, 8)
                    }
                }

                // Footer Citation
                HStack {
                    Text("NOAA NWS ALERTS CLUSTER INTERCEPT ENGINE · GEOGRAPHIC ADJACENCY CORRELATION")
                        .font(.system(size: 8, weight: .medium, design: .monospaced))
                        .foregroundStyle(WxTheme.snwSilver.opacity(0.6))
                    Spacer()
                    if let gen = store.chasePayload?.generatedAt {
                        Text("UPDATED // \(gen.prefix(19))")
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

    // ── Cluster Card ─────────────────────────────────────────────────────────────

    @ViewBuilder
    private func clusterCard(cluster: StormClusterDTO, rank: Int) -> some View {
        let isExpanded = expandedClusterIDs.contains(cluster.id)
        let cardColor = clusterColor(score: cluster.score)

        SNWConsoleCard(
            title: "#\(rank)  \(cluster.name.uppercased())",
            tag: "SCORE // \(cluster.score)",
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
                                    .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                                    .foregroundStyle(WxTheme.text)
                                    .padding(.horizontal, 5)
                                    .padding(.vertical, 2)
                                    .background(WxTheme.snwChassis.opacity(0.9), in: RoundedRectangle(cornerRadius: 3))
                                    .overlay(RoundedRectangle(cornerRadius: 3).strokeBorder(WxTheme.snwCyan.opacity(0.4), lineWidth: 0.6))
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
                                Text("·  RADAR // \(cluster.nearestRadar)")
                                    .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                                    .foregroundStyle(WxTheme.snwCyan)
                            }
                        }
                    }

                    Spacer()

                    // Radar Intercept Jump Button
                    Button {
                        store.chaseCluster(cluster)
                    } label: {
                        HStack(spacing: 5) {
                            Image(systemName: "scope")
                                .font(.system(size: 11, weight: .bold))
                            Text("INTERCEPT RADAR")
                                .font(.system(size: 9.5, weight: .bold, design: .monospaced))
                        }
                        .padding(.horizontal, 10)
                        .padding(.vertical, 6)
                        .background(
                            LinearGradient(
                                colors: [cardColor.opacity(0.25), cardColor.opacity(0.1)],
                                startPoint: .top,
                                endPoint: .bottom
                            ),
                            in: RoundedRectangle(cornerRadius: 5)
                        )
                        .overlay(RoundedRectangle(cornerRadius: 5).strokeBorder(cardColor, lineWidth: 1))
                        .foregroundStyle(WxTheme.text)
                        .shadow(color: cardColor.opacity(0.4), radius: 3)
                    }
                    .buttonStyle(.plain)
                    .help("Jump to this storm cluster and load live Doppler Radar array")
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
                    .font(.system(size: 9, weight: .bold, design: .monospaced))
                    .foregroundStyle(WxTheme.text)
                Spacer()
                if let area = cell.areaDesc {
                    Text(area)
                        .font(.system(size: 8, design: .monospaced))
                        .foregroundStyle(WxTheme.snwSilver.opacity(0.8))
                        .lineLimit(1)
                }
            }

            if let headline = cell.headline {
                Text(headline)
                    .font(.system(size: 8.5, design: .monospaced))
                    .foregroundStyle(WxTheme.snwSilver)
                    .lineLimit(2)
            }
        }
        .padding(6)
        .background(WxTheme.snwChassis.opacity(0.6), in: RoundedRectangle(cornerRadius: 4))
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
}
