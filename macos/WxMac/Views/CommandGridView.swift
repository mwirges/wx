import SwiftUI

struct CommandGridView: View {
    @EnvironmentObject var store: WeatherStore
    @State private var newLocationText: String = ""
    @FocusState private var isInputFocused: Bool

    private var isMetric: Bool {
        store.units == "metric"
    }

    private var totalAlertsCount: Int {
        store.gridCards.reduce(0) { total, card in
            total + (card.payload?.alerts.count ?? 0)
        }
    }

    private var warmestStation: (name: String, temp: Double)? {
        var best: (String, Double)?
        for card in store.gridCards {
            guard let c = card.payload?.conditions else { continue }
            let t = isMetric ? c.temperatureC : c.temperatureF
            if let t {
                if best == nil || t > best!.1 {
                    best = (card.displayName, t)
                }
            }
        }
        return best
    }

    private var coolestStation: (name: String, temp: Double)? {
        var best: (String, Double)?
        for card in store.gridCards {
            guard let c = card.payload?.conditions else { continue }
            let t = isMetric ? c.temperatureC : c.temperatureF
            if let t {
                if best == nil || t < best!.1 {
                    best = (card.displayName, t)
                }
            }
        }
        return best
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 14) {
                // Header Bar & Matrix Stats
                headerBar

                // Add Station Quick Bar
                addStationBar

                // Station Matrix
                if store.gridCards.isEmpty && !store.isGridLoading {
                    emptyStateView
                } else {
                    matrixGrid
                }
            }
            .padding(14)
        }
        .background(WxTheme.bg)
        .onAppear {
            if store.gridCards.isEmpty {
                Task { await store.refreshGrid() }
            }
        }
    }

    // ── Header & Status Bar ────────────────────────────────────────────────────────

    @ViewBuilder
    private var headerBar: some View {
        HStack(spacing: 8) {
            VStack(alignment: .leading, spacing: 3) {
                HStack(spacing: 6) {
                    Image(systemName: "square.grid.2x2.fill")
                        .font(.system(size: 11, weight: .bold))
                        .foregroundStyle(WxTheme.snwCyan)
                    Text("COMMAND GRID // SENSOR MATRIX")
                        .font(.system(size: 10.5, weight: .bold, design: .monospaced))
                        .tracking(0.5)
                        .foregroundStyle(WxTheme.text)
                }
                Text("REAL-TIME MULTI-STATION SYNOPTIC TELEMETRY")
                    .font(.system(size: 9, design: .monospaced))
                    .foregroundStyle(WxTheme.snwSilver)
            }

            Spacer()

            // Alerts summary indicator
            if totalAlertsCount > 0 {
                HStack(spacing: 4) {
                    Image(systemName: "exclamationmark.triangle.fill")
                        .font(.system(size: 9))
                    Text("\(totalAlertsCount) ACTIVE WARNINGS")
                        .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                }
                .foregroundStyle(WxTheme.snwRed)
                .padding(.horizontal, 8)
                .padding(.vertical, 4)
                .background(WxTheme.snwRed.opacity(0.15), in: Capsule())
                .overlay(Capsule().strokeBorder(WxTheme.snwRed.opacity(0.4), lineWidth: 0.8))
            } else {
                HStack(spacing: 4) {
                    Circle().fill(WxTheme.snwGreen).frame(width: 5, height: 5)
                    Text("ALL STATIONS NOMINAL")
                        .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                }
                .foregroundStyle(WxTheme.snwGreen)
                .padding(.horizontal, 8)
                .padding(.vertical, 4)
                .background(WxTheme.snwGreen.opacity(0.12), in: Capsule())
                .overlay(Capsule().strokeBorder(WxTheme.snwGreen.opacity(0.35), lineWidth: 0.8))
            }

            // Temperature extremes
            if let w = warmestStation, let c = coolestStation, store.gridCards.count > 1 {
                HStack(spacing: 6) {
                    Text(String(format: "MAX: %.0f°", w.temp))
                        .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                        .foregroundStyle(WxTheme.snwRed)
                    Text("·")
                        .foregroundStyle(WxTheme.border)
                    Text(String(format: "MIN: %.0f°", c.temp))
                        .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                        .foregroundStyle(WxTheme.snwCyan)
                }
                .padding(.horizontal, 8)
                .padding(.vertical, 4)
                .background(WxTheme.snwChassis, in: RoundedRectangle(cornerRadius: 5))
                .overlay(RoundedRectangle(cornerRadius: 5).strokeBorder(WxTheme.border.opacity(0.35), lineWidth: 0.8))
            }

            // Sync All Button
            Button {
                Task { await store.refreshGrid() }
            } label: {
                HStack(spacing: 4) {
                    if store.isGridLoading {
                        ProgressView().controlSize(.small).tint(WxTheme.snwCyan)
                    } else {
                        Image(systemName: "arrow.triangle.2.circlepath")
                    }
                    Text("SYNC ALL")
                }
                .font(.system(size: 9, weight: .bold, design: .monospaced))
                .foregroundStyle(WxTheme.snwCyan)
                .padding(.horizontal, 9)
                .padding(.vertical, 5)
                .background(WxTheme.snwCyan.opacity(0.12), in: RoundedRectangle(cornerRadius: 5))
                .overlay(RoundedRectangle(cornerRadius: 5).strokeBorder(WxTheme.snwCyan.opacity(0.35), lineWidth: 0.8))
            }
            .buttonStyle(.plain)
            .disabled(store.isGridLoading)
        }
        .padding(10)
        .background(WxTheme.snwPanel, in: RoundedRectangle(cornerRadius: 8))
        .overlay(RoundedRectangle(cornerRadius: 8).strokeBorder(WxTheme.border.opacity(0.4), lineWidth: 0.8))
    }

    // ── Add Station Bar ────────────────────────────────────────────────────────────

    @ViewBuilder
    private var addStationBar: some View {
        HStack(spacing: 8) {
            Image(systemName: "pin.fill")
                .font(.system(size: 10))
                .foregroundStyle(WxTheme.snwCyan)

            TextField("PIN STATION TO GRID (e.g. Fort Wayne, IN / 46802 / KORD)…", text: $newLocationText)
                .textFieldStyle(.plain)
                .font(.system(size: 10, design: .monospaced))
                .foregroundStyle(WxTheme.text)
                .focused($isInputFocused)
                .onSubmit {
                    addCurrentInput()
                }

            if !newLocationText.isEmpty {
                Button {
                    newLocationText = ""
                } label: {
                    Image(systemName: "xmark.circle.fill")
                        .font(.system(size: 10))
                        .foregroundStyle(WxTheme.textSecondary)
                }
                .buttonStyle(.plain)
            }

            Button {
                addCurrentInput()
            } label: {
                HStack(spacing: 3) {
                    Image(systemName: "plus")
                        .font(.system(size: 8.5, weight: .bold))
                    Text("PIN TO GRID")
                        .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                }
                .foregroundStyle(WxTheme.snwCyan)
                .padding(.horizontal, 8)
                .padding(.vertical, 4)
                .background(WxTheme.snwCyan.opacity(0.15), in: RoundedRectangle(cornerRadius: 4))
                .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(WxTheme.snwCyan.opacity(0.4), lineWidth: 0.8))
            }
            .buttonStyle(.plain)
            .disabled(newLocationText.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
        }
        .padding(.horizontal, 10)
        .padding(.vertical, 7)
        .background(WxTheme.snwChassis.opacity(0.8), in: RoundedRectangle(cornerRadius: 6))
        .overlay(RoundedRectangle(cornerRadius: 6).strokeBorder(WxTheme.border.opacity(0.35), lineWidth: 0.8))
    }

    private func addCurrentInput() {
        let clean = newLocationText.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !clean.isEmpty else { return }
        store.addFavorite(name: clean, value: clean)
        newLocationText = ""
        isInputFocused = false
        Task { await store.refreshGrid() }
    }

    // ── Station Matrix Grid ────────────────────────────────────────────────────────

    @ViewBuilder
    private var matrixGrid: some View {
        LazyVGrid(columns: [GridItem(.adaptive(minimum: 340), spacing: 12)], spacing: 12) {
            ForEach(store.gridCards) { card in
                stationCard(card)
            }
        }
    }

    @ViewBuilder
    private func stationCard(_ card: LocationGridCardData) -> some View {
        SNWConsoleCard(title: card.displayName, tag: "STN.LIVE") {
            VStack(alignment: .leading, spacing: 10) {
                // Card Header Info & Quick Status
                HStack(alignment: .top) {
                    VStack(alignment: .leading, spacing: 2) {
                        Text(card.displayName.uppercased())
                            .font(.system(size: 11, weight: .bold, design: .monospaced))
                            .foregroundStyle(WxTheme.text)
                            .lineLimit(1)
                        if let st = card.payload?.conditions?.station, !st.isEmpty {
                            Text("STATION // \(st)")
                                .font(.system(size: 8, design: .monospaced))
                                .foregroundStyle(WxTheme.snwSilver.opacity(0.8))
                        }
                    }

                    Spacer()

                    if card.isLoading {
                        ProgressView().controlSize(.small).tint(WxTheme.snwCyan)
                    } else if let updated = card.lastUpdated {
                        Text(updated.formatted(date: .omitted, time: .shortened))
                            .font(.system(size: 8, design: .monospaced))
                            .foregroundStyle(WxTheme.snwSilver.opacity(0.6))
                    }
                }

                if let err = card.errorMessage {
                    VStack(spacing: 4) {
                        HStack(spacing: 4) {
                            Image(systemName: "exclamationmark.triangle")
                                .foregroundStyle(WxTheme.snwRed)
                            Text("TELEMETRY ERROR: \(err)")
                                .font(.system(size: 8.5, design: .monospaced))
                                .foregroundStyle(WxTheme.snwRed)
                                .lineLimit(2)
                        }
                    }
                    .padding(.vertical, 6)
                } else if let p = card.payload, let cond = p.conditions {
                    // Temperature & Condition Row
                    HStack(alignment: .center) {
                        VStack(alignment: .leading, spacing: 2) {
                            let tempVal = isMetric ? cond.temperatureC : cond.temperatureF
                            Text(tempVal.map { String(format: "%.0f°", $0) } ?? "--")
                                .font(.system(size: 32, weight: .heavy, design: .monospaced))
                                .foregroundStyle(WxTheme.text)

                            let feelsVal = isMetric ? cond.feelsLikeC : cond.feelsLikeF
                            if let fl = feelsVal {
                                Text(String(format: "FEELS LIKE %.0f°", fl))
                                    .font(.system(size: 8.5, weight: .semibold, design: .monospaced))
                                    .foregroundStyle(WxTheme.snwSilver)
                            }
                        }

                        Spacer()

                        VStack(alignment: .trailing, spacing: 4) {
                            Image(systemName: ConditionSymbol.systemName(for: cond.conditionCode))
                                .font(.system(size: 26))
                                .foregroundStyle(WxTheme.snwCyan)

                            Text(cond.description?.uppercased() ?? "OBSERVED")
                                .font(.system(size: 9, weight: .bold, design: .monospaced))
                                .foregroundStyle(WxTheme.textSecondary)
                                .lineLimit(1)
                        }
                    }

                    Divider().background(WxTheme.border.opacity(0.3))

                    // Secondary Metrics (Humidity, Wind, Pressure)
                    HStack(spacing: 12) {
                        if let h = cond.humidityPct {
                            metricPill(label: "HUM", value: String(format: "%.0f%%", h))
                        }
                        if let w = (isMetric ? cond.windKph : cond.windMph) {
                            let dir = cond.windDirection ?? ""
                            metricPill(label: "WIND", value: String(format: "%@ %.0f %@", dir, w, isMetric ? "km/h" : "mph"))
                        }
                        if let pr = (isMetric ? cond.pressureHpa : cond.pressureInhg) {
                            metricPill(label: "BARO", value: String(format: isMetric ? "%.0f hPa" : "%.2f\"", pr))
                        }
                    }

                    // Alerts inside card
                    if !p.alerts.isEmpty {
                        VStack(alignment: .leading, spacing: 4) {
                            ForEach(p.alerts.prefix(2)) { a in
                                HStack(spacing: 4) {
                                    Image(systemName: "exclamationmark.triangle.fill")
                                        .font(.system(size: 8))
                                    Text(a.event.uppercased())
                                        .font(.system(size: 8, weight: .bold, design: .monospaced))
                                        .lineLimit(1)
                                }
                                .foregroundStyle(WxTheme.snwRed)
                                .padding(.horizontal, 6)
                                .padding(.vertical, 2.5)
                                .background(WxTheme.snwRed.opacity(0.12), in: RoundedRectangle(cornerRadius: 3))
                            }
                        }
                    }
                } else if card.isLoading {
                    VStack(spacing: 6) {
                        ProgressView().controlSize(.small).tint(WxTheme.snwCyan)
                        Text("DOWNLINKING STATION TELEMETRY…")
                            .font(.system(size: 8.5, design: .monospaced))
                            .foregroundStyle(WxTheme.snwCyan)
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 20)
                }

                Divider().background(WxTheme.border.opacity(0.2))

                // Card Footer Quick Actions
                HStack(spacing: 6) {
                    // Tactical Jump
                    Button {
                        store.selectLocation(card.locationKey)
                        store.selectedDeskTab = .dual
                    } label: {
                        HStack(spacing: 3) {
                            Image(systemName: "rectangle.split.2x1.fill")
                                .font(.system(size: 8))
                            Text("TACTICAL")
                        }
                        .font(.system(size: 8, weight: .bold, design: .monospaced))
                        .foregroundStyle(WxTheme.snwCyan)
                        .padding(.horizontal, 6)
                        .padding(.vertical, 3.5)
                        .background(WxTheme.snwCyan.opacity(0.12), in: RoundedRectangle(cornerRadius: 4))
                    }
                    .buttonStyle(.plain)

                    // Radar Jump
                    Button {
                        store.selectLocation(card.locationKey)
                        store.selectedDeskTab = .radar
                        Task { await store.refreshRadar() }
                    } label: {
                        HStack(spacing: 3) {
                            Image(systemName: "dot.radiowaves.left.and.right")
                                .font(.system(size: 8))
                            Text("RADAR")
                        }
                        .font(.system(size: 8, weight: .bold, design: .monospaced))
                        .foregroundStyle(WxTheme.snwSilver)
                        .padding(.horizontal, 6)
                        .padding(.vertical, 3.5)
                        .background(WxTheme.snwChassis, in: RoundedRectangle(cornerRadius: 4))
                    }
                    .buttonStyle(.plain)

                    Spacer()

                    // Single Refresh
                    Button {
                        Task { await store.refreshSingleGridCard(card.locationKey) }
                    } label: {
                        Image(systemName: "arrow.triangle.2.circlepath")
                            .font(.system(size: 8.5))
                            .foregroundStyle(WxTheme.snwSilver)
                            .padding(4)
                    }
                    .buttonStyle(.plain)

                    // Unpin
                    Button {
                        store.removeFavorite(card.locationKey)
                        Task { await store.refreshGrid() }
                    } label: {
                        Image(systemName: "pin.slash")
                            .font(.system(size: 8.5))
                            .foregroundStyle(WxTheme.snwSilver.opacity(0.6))
                            .padding(4)
                    }
                    .buttonStyle(.plain)
                }
            }
        }
    }

    private func metricPill(label: String, value: String) -> some View {
        VStack(alignment: .leading, spacing: 1) {
            Text(label)
                .font(.system(size: 7, weight: .bold, design: .monospaced))
                .foregroundStyle(WxTheme.snwSilver.opacity(0.7))
            Text(value)
                .font(.system(size: 9, weight: .semibold, design: .monospaced))
                .foregroundStyle(WxTheme.text)
        }
    }

    // ── Empty State View ──────────────────────────────────────────────────────────

    @ViewBuilder
    private var emptyStateView: some View {
        VStack(spacing: 12) {
            Image(systemName: "square.grid.2x2")
                .font(.system(size: 36))
                .foregroundStyle(WxTheme.snwCyan.opacity(0.6))

            Text("NO STATIONS PINNED TO COMMAND GRID")
                .font(.system(size: 11, weight: .bold, design: .monospaced))
                .foregroundStyle(WxTheme.text)

            Text("Pin locations to monitor live telemetry, temperatures, and warnings simultaneously across multiple cities.")
                .font(.system(size: 9.5, design: .monospaced))
                .foregroundStyle(WxTheme.textSecondary)
                .multilineTextAlignment(.center)
                .frame(maxWidth: 420)

            VStack(spacing: 6) {
                Text("QUICK-PIN REGIONAL PRESETS:")
                    .font(.system(size: 8, weight: .bold, design: .monospaced))
                    .foregroundStyle(WxTheme.snwSilver)

                HStack(spacing: 6) {
                    presetButton("Fort Wayne, IN")
                    presetButton("New Haven, IN")
                    presetButton("Milwaukee, WI")
                    presetButton("Chicago, IL")
                }
            }
            .padding(.top, 8)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 50)
    }

    private func presetButton(_ loc: String) -> some View {
        Button {
            store.addFavorite(name: loc, value: loc)
            Task { await store.refreshGrid() }
        } label: {
            HStack(spacing: 3) {
                Image(systemName: "plus.circle.fill")
                    .font(.system(size: 8))
                Text(loc)
                    .font(.system(size: 8.5, weight: .medium, design: .monospaced))
            }
            .foregroundStyle(WxTheme.snwCyan)
            .padding(.horizontal, 8)
            .padding(.vertical, 4.5)
            .background(WxTheme.snwChassis, in: RoundedRectangle(cornerRadius: 4))
            .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(WxTheme.snwCyan.opacity(0.35), lineWidth: 0.8))
        }
        .buttonStyle(.plain)
    }
}
