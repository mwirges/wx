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
            VStack(alignment: .leading, spacing: 2) {
                HStack(spacing: 6) {
                    Image(systemName: "square.grid.2x2.fill")
                        .font(.system(size: 12, weight: .semibold))
                        .foregroundStyle(WxGlassTheme.cyan)
                    Text("Command Grid · Sensor Matrix")
                        .font(.system(size: 13, weight: .bold, design: .rounded))
                        .foregroundStyle(WxGlassTheme.textPrimary)
                }
                Text("Real-time multi-station synoptic telemetry")
                    .font(.system(size: 11, weight: .medium, design: .rounded))
                    .foregroundStyle(WxGlassTheme.textSecondary)
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
                    Text(String(format: "MAX %.0f°", w.temp))
                        .font(.system(size: 9.5, weight: .bold, design: .rounded))
                        .foregroundStyle(WxGlassTheme.conditionRed)
                    Text("·")
                        .foregroundStyle(WxGlassTheme.textMuted)
                    Text(String(format: "MIN %.0f°", c.temp))
                        .font(.system(size: 9.5, weight: .bold, design: .rounded))
                        .foregroundStyle(WxGlassTheme.cyan)
                }
                .padding(.horizontal, 10)
                .padding(.vertical, 4.5)
                .background(Color.white.opacity(0.06), in: Capsule())
                .overlay(Capsule().strokeBorder(Color.white.opacity(0.14), lineWidth: 0.8))
            }

            // Sync All Button
            Button {
                Task { await store.refreshGrid() }
            } label: {
                HStack(spacing: 4) {
                    if store.isGridLoading {
                        ProgressView().controlSize(.small).tint(WxGlassTheme.cyan)
                    } else {
                        Image(systemName: "arrow.triangle.2.circlepath")
                    }
                    Text("Sync All")
                }
                .font(.system(size: 10.5, weight: .semibold, design: .rounded))
                .foregroundStyle(WxGlassTheme.cyan)
                .padding(.horizontal, 12)
                .padding(.vertical, 5.5)
                .wxFrostedCapsule(accent: WxGlassTheme.cyan, active: true)
            }
            .buttonStyle(.plain)
            .disabled(store.isGridLoading)
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 10)
        .background {
            ZStack {
                RoundedRectangle(cornerRadius: 18, style: .continuous).fill(.ultraThinMaterial)
                RoundedRectangle(cornerRadius: 18, style: .continuous).fill(WxGlassTheme.cardTintOverlay())
            }
        }
        .overlay(RoundedRectangle(cornerRadius: 18, style: .continuous).strokeBorder(Color.white.opacity(0.16), lineWidth: 0.9))
    }

    // ── Add Station Bar ────────────────────────────────────────────────────────────

    @ViewBuilder
    private var addStationBar: some View {
        HStack(spacing: 8) {
            Image(systemName: "pin.fill")
                .font(.system(size: 11))
                .foregroundStyle(WxGlassTheme.cyan)

            TextField("Pin station to grid (e.g. Fort Wayne, IN · 46802 · KORD)…", text: $newLocationText)
                .textFieldStyle(.plain)
                .font(.system(size: 12, weight: .medium, design: .rounded))
                .foregroundStyle(WxGlassTheme.textPrimary)
                .focused($isInputFocused)
                .onSubmit {
                    addCurrentInput()
                }

            if !newLocationText.isEmpty {
                Button {
                    newLocationText = ""
                } label: {
                    Image(systemName: "xmark.circle.fill")
                        .font(.system(size: 11))
                        .foregroundStyle(WxGlassTheme.textMuted)
                }
                .buttonStyle(.plain)
            }

            Button {
                addCurrentInput()
            } label: {
                HStack(spacing: 4) {
                    Image(systemName: "plus")
                        .font(.system(size: 9.5, weight: .bold))
                    Text("Pin to Grid")
                        .font(.system(size: 10.5, weight: .bold, design: .rounded))
                }
                .foregroundStyle(Color(red: 0.03, green: 0.07, blue: 0.14))
                .padding(.horizontal, 12)
                .padding(.vertical, 5.5)
                .background(
                    LinearGradient(colors: [WxGlassTheme.cyanSoft, WxGlassTheme.cyan], startPoint: .topLeading, endPoint: .bottomTrailing),
                    in: Capsule()
                )
            }
            .buttonStyle(.plain)
            .disabled(newLocationText.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 8)
        .wxFrostedCapsule(accent: WxGlassTheme.cyan)
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
        let stationTag = card.payload?.conditions?.station ?? "STN.LIVE"
        WxGlassCard(title: card.displayName, tag: stationTag) {
            VStack(alignment: .leading, spacing: 10) {
                if card.isLoading || card.lastUpdated != nil {
                    HStack {
                        if let loc = card.payload?.conditions?.location,
                           loc.caseInsensitiveCompare(card.displayName) != .orderedSame {
                            Text(loc)
                                .font(.system(size: 11, weight: .medium, design: .rounded))
                                .foregroundStyle(WxGlassTheme.textMuted)
                                .lineLimit(1)
                        }
                        Spacer()
                        if card.isLoading {
                            ProgressView().controlSize(.small).tint(WxGlassTheme.cyan)
                        } else if let updated = card.lastUpdated {
                            Text("Updated \(updated.formatted(date: .omitted, time: .shortened))")
                                .font(.system(size: 9.5, weight: .medium, design: .monospaced))
                                .foregroundStyle(WxGlassTheme.textMuted)
                        }
                    }
                }

                if let err = card.errorMessage {
                    HStack(spacing: 6) {
                        Image(systemName: "exclamationmark.triangle.fill")
                            .foregroundStyle(WxGlassTheme.conditionRed)
                        Text(err)
                            .font(.system(size: 11.5, weight: .medium))
                            .foregroundStyle(WxGlassTheme.conditionRed)
                            .lineLimit(2)
                    }
                    .padding(.vertical, 6)
                } else if let p = card.payload, let cond = p.conditions {
                    // Temperature & Condition Row
                    HStack(alignment: .center) {
                        VStack(alignment: .leading, spacing: 2) {
                            let tempVal = isMetric ? cond.temperatureC : cond.temperatureF
                            Text(tempVal.map { String(format: "%.0f°", $0) } ?? "--")
                                .font(.system(size: 36, weight: .bold, design: .rounded))
                                .foregroundStyle(WxGlassTheme.textPrimary)

                            let feelsVal = isMetric ? cond.feelsLikeC : cond.feelsLikeF
                            if let fl = feelsVal {
                                Text(String(format: "Feels %.0f°", fl))
                                    .font(.system(size: 10.5, weight: .semibold, design: .rounded))
                                    .foregroundStyle(WxGlassTheme.textSecondary)
                            }
                        }

                        Spacer()

                        VStack(alignment: .trailing, spacing: 4) {
                            let sym = ConditionSymbol.resolvedSystemName(for: p)
                            let cardNight = ConditionSymbol.isNight(for: p)
                            Image(systemName: sym)
                                .font(.system(size: 28, weight: .medium))
                                .symbolRenderingMode(.hierarchical)
                                .foregroundStyle(ConditionSymbol.tint(for: sym, isNight: cardNight))

                            Text(ConditionSymbol.resolvedDescription(for: p) ?? "Observed")
                                .font(.system(size: 11, weight: .semibold, design: .rounded))
                                .foregroundStyle(WxGlassTheme.textSecondary)
                                .lineLimit(1)
                        }
                    }

                    Divider().overlay(Color.white.opacity(0.10))

                    // Secondary Metrics (Concentric Capsules)
                    HStack(spacing: 6) {
                        if let h = cond.humidityPct {
                            metricPill(label: "HUM", value: String(format: "%.0f%%", h))
                        }
                        if let w = (isMetric ? cond.windKph : cond.windMph) {
                            let dir = cond.windDirection ?? ""
                            metricPill(label: "WIND", value: String(format: "%@ %.0f %@", dir, w, isMetric ? "km/h" : "mph").trimmingCharacters(in: .whitespaces))
                        }
                        if let pr = (isMetric ? cond.pressureHpa : cond.pressureInhg) {
                            metricPill(label: "BARO", value: String(format: isMetric ? "%.0f hPa" : "%.2f\"", pr))
                        }
                    }

                    // Alerts inside card
                    if !p.alerts.isEmpty {
                        VStack(alignment: .leading, spacing: 4) {
                            ForEach(p.alerts.prefix(2)) { a in
                                HStack(spacing: 5) {
                                    Image(systemName: "exclamationmark.triangle.fill")
                                        .font(.system(size: 9))
                                    Text(a.event)
                                        .font(.system(size: 10, weight: .bold, design: .rounded))
                                        .lineLimit(1)
                                }
                                .foregroundStyle(WxGlassTheme.conditionRed)
                                .padding(.horizontal, 9)
                                .padding(.vertical, 3.5)
                                .background(WxGlassTheme.conditionRed.opacity(0.14), in: Capsule())
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
                    Button {
                        store.selectLocation(card.locationKey)
                        store.selectedDeskTab = .dual
                    } label: {
                        HStack(spacing: 4) {
                            Image(systemName: "rectangle.split.2x1.fill")
                                .font(.system(size: 8.5))
                            Text("Tactical")
                        }
                        .font(.system(size: 10, weight: .semibold, design: .rounded))
                        .foregroundStyle(WxGlassTheme.cyan)
                        .padding(.horizontal, 10)
                        .padding(.vertical, 4.5)
                        .background(WxGlassTheme.cyan.opacity(0.14), in: Capsule())
                        .overlay(Capsule().strokeBorder(WxGlassTheme.cyan.opacity(0.38), lineWidth: 0.8))
                    }
                    .buttonStyle(.plain)

                    Button {
                        store.selectLocation(card.locationKey)
                        store.selectedDeskTab = .radar
                        Task { await store.refreshRadar() }
                    } label: {
                        HStack(spacing: 4) {
                            Image(systemName: "dot.radiowaves.left.and.right")
                                .font(.system(size: 8.5))
                            Text("Radar")
                        }
                        .font(.system(size: 10, weight: .semibold, design: .rounded))
                        .foregroundStyle(WxGlassTheme.textSecondary)
                        .padding(.horizontal, 10)
                        .padding(.vertical, 4.5)
                        .background(Color.white.opacity(0.06), in: Capsule())
                        .overlay(Capsule().strokeBorder(Color.white.opacity(0.14), lineWidth: 0.8))
                    }
                    .buttonStyle(.plain)

                    Spacer()

                    Button {
                        Task { await store.refreshSingleGridCard(card.locationKey) }
                    } label: {
                        Image(systemName: "arrow.triangle.2.circlepath")
                            .font(.system(size: 10))
                            .foregroundStyle(WxGlassTheme.textSecondary)
                            .padding(5)
                    }
                    .buttonStyle(.plain)

                    Button {
                        store.removeFavorite(card.locationKey)
                        Task { await store.refreshGrid() }
                    } label: {
                        Image(systemName: "pin.slash")
                            .font(.system(size: 10))
                            .foregroundStyle(WxGlassTheme.textMuted)
                            .padding(5)
                    }
                    .buttonStyle(.plain)
                }
            }
        }
    }

    private func metricPill(label: String, value: String) -> some View {
        WxSensorCapsule(label: label, value: value, compact: true)
    }

    // ── Empty State View ──────────────────────────────────────────────────────────

    @ViewBuilder
    private var emptyStateView: some View {
        VStack(spacing: 12) {
            Image(systemName: "square.grid.2x2")
                .font(.system(size: 36))
                .foregroundStyle(WxTheme.snwCyan.opacity(0.6))

            Text("No Stations Pinned to Command Grid")
                .font(.system(size: 15, weight: .bold, design: .rounded))
                .foregroundStyle(WxGlassTheme.textPrimary)

            Text("Pin locations to monitor live telemetry, temperatures, and warnings simultaneously across multiple cities.")
                .font(.system(size: 12.5))
                .foregroundStyle(WxGlassTheme.textSecondary)
                .multilineTextAlignment(.center)
                .frame(maxWidth: 420)

            VStack(spacing: 8) {
                Text("QUICK-PIN REGIONAL PRESETS")
                    .font(.system(size: 9.5, weight: .bold, design: .rounded))
                    .tracking(0.5)
                    .foregroundStyle(WxGlassTheme.textMuted)

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
            HStack(spacing: 4) {
                Image(systemName: "plus.circle.fill")
                    .font(.system(size: 9))
                Text(loc)
                    .font(.system(size: 10.5, weight: .semibold, design: .rounded))
            }
            .foregroundStyle(WxGlassTheme.cyan)
            .padding(.horizontal, 10)
            .padding(.vertical, 5.5)
            .wxFrostedCapsule(accent: WxGlassTheme.cyan)
        }
        .buttonStyle(.plain)
    }
}
