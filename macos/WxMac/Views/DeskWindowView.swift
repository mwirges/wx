import SwiftUI

struct DeskWindowView: View {
    @EnvironmentObject var store: WeatherStore

    private var updatedSubtitle: String? {
        store.lastRefreshed.map { $0.formatted(date: .omitted, time: .shortened) }
    }

    private var currentRegime: WxAtmosphereRegime {
        WxAtmosphereRegime.resolve(
            conditions: store.payload?.conditions,
            alerts: store.activeAlerts,
            astronomy: store.payload?.conditions?.astronomy ?? store.payload?.astronomy,
            nowcast: store.nowcastPayload?.nowcast ?? store.payload?.conditions?.nowcast ?? store.payload?.nowcast,
            forecast: store.payload?.forecast
        )
    }

    var body: some View {
        VStack(spacing: 0) {
            unifiedBridgeToolbar

            AlertBannerView()

            GeometryReader { geo in
                let isWide = geo.size.width >= 760
                if store.selectedDeskTab == .dual && isWide {
                    let leftWidth = max(380, min(470, geo.size.width * 0.40))
                    HStack(alignment: .top, spacing: 14) {
                        ScrollView {
                            VStack(alignment: .leading, spacing: 12) {
                                WxGlassCard(
                                    title: "Atmospheric Telemetry",
                                    tag: store.payload?.conditions?.station ?? "GRID.OBS"
                                ) {
                                    NowBlockView(compact: false, popoverMetrics: false)
                                }

                                AlertsListView(popoverMode: false)

                                ephemerisCard

                                airQualityCard

                                nowcastCard

                                tacticalSignalsCard

                                if !(store.payload?.forecast?.hourly ?? []).isEmpty {
                                    WxGlassCard(title: "Hourly Forecast", tag: "24H") {
                                        HourlyStripView()
                                    }
                                }

                                WxGlassCard(title: "Synoptic Forecast Log", tag: "NOAA.NWS") {
                                    PeriodsListView(limit: nil, compactRows: false)
                                }
                            }
                            .padding(.vertical, 10)
                            .padding(.leading, 14)
                            .padding(.trailing, 2)
                        }
                        .frame(width: leftWidth, height: geo.size.height)

                        VStack(alignment: .leading, spacing: 0) {
                            WxGlassCard(title: "Doppler Radar Array", tag: "NOAA.MRMS") {
                                RadarPanelView(fullScreen: false, flexibleHeight: true)
                                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                            }
                            .frame(maxWidth: .infinity, maxHeight: .infinity)
                        }
                        .padding(.vertical, 10)
                        .padding(.leading, 2)
                        .padding(.trailing, 14)
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                    }
                    .frame(width: geo.size.width, height: geo.size.height)
                } else if store.selectedDeskTab == .radar {
                    RadarPanelView(fullScreen: true)
                        .frame(width: geo.size.width, height: geo.size.height)
                } else if store.selectedDeskTab == .outlooks {
                    CPCOutlookView()
                        .frame(width: geo.size.width, height: geo.size.height)
                } else if store.selectedDeskTab == .chase {
                    StormChaseView()
                        .frame(width: geo.size.width, height: geo.size.height)
                } else if store.selectedDeskTab == .climate {
                    ClimateTrendsView()
                        .frame(width: geo.size.width, height: geo.size.height)
                } else if store.selectedDeskTab == .tropics {
                    TropicsView()
                        .frame(width: geo.size.width, height: geo.size.height)
                } else if store.selectedDeskTab == .grid {
                    CommandGridView()
                        .frame(width: geo.size.width, height: geo.size.height)
                } else {
                    ScrollView {
                        VStack(alignment: .leading, spacing: 12) {
                            WxGlassCard(
                                title: "Atmospheric Telemetry",
                                tag: store.payload?.conditions?.station ?? "GRID.OBS"
                            ) {
                                NowBlockView(compact: false, popoverMetrics: false)
                            }

                            AlertsListView(popoverMode: false)

                            ephemerisCard

                            airQualityCard

                            nowcastCard

                            tacticalSignalsCard

                            if !(store.payload?.forecast?.hourly ?? []).isEmpty {
                                WxGlassCard(title: "Hourly Forecast", tag: "24H") {
                                    HourlyStripView()
                                }
                            }

                            WxGlassCard(title: "Synoptic Forecast Log", tag: "NOAA.NWS") {
                                PeriodsListView(limit: nil, compactRows: false)
                            }

                            if store.selectedDeskTab == .dual {
                                WxGlassCard(title: "Doppler Radar Array", tag: "NOAA.MRMS") {
                                    RadarPanelView(fullScreen: false)
                                }
                            }
                        }
                        .padding(14)
                    }
                }
            }
        }
        .frame(minWidth: 520, idealWidth: 940, maxWidth: .infinity, minHeight: 600, idealHeight: 760, maxHeight: .infinity)
        .background(WarpGlassBackground(regime: currentRegime))
        .preferredColorScheme(.dark)
        .onAppear {
            if (store.selectedDeskTab == .dual || store.selectedDeskTab == .radar) && store.radarImage == nil && !store.isRadarLoading {
                Task { await store.refreshRadar() }
            }
        }
    }

    // MARK: - Single-Row 54pt Unified Bridge Glass Toolbar

    private var unifiedBridgeToolbar: some View {
        VStack(alignment: .leading, spacing: 5) {
            ViewThatFits(in: .horizontal) {
                // 1. Expanded wide layout (>= 1060pt): single ~54pt row with icons + labels on all 8 modes
                HStack(alignment: .center, spacing: 10) {
                    Color.clear.frame(width: 58, height: 28)

                    VStack(alignment: .leading, spacing: 1) {
                        Text("TACTICAL SENSOR CONTROL")
                            .font(.system(size: 7.5, weight: .bold, design: .rounded))
                            .tracking(0.6)
                            .foregroundStyle(WxGlassTheme.textMuted)
                            .lineLimit(1)
                        LocationBarView(isHUD: true)
                            .frame(minWidth: 185, idealWidth: 215, maxWidth: 260)
                    }

                    Spacer(minLength: 4)

                    modeSelectorPill(showAllIcons: true, compactPadding: false)

                    Spacer(minLength: 4)

                    toolbarRightControls
                }

                // 2. Default 940pt Desk window layout (~850pt–1059pt): single ~54pt row with active-mode icon + all 8 labels
                HStack(alignment: .center, spacing: 8) {
                    Color.clear.frame(width: 58, height: 28)

                    VStack(alignment: .leading, spacing: 1) {
                        Text("TACTICAL SENSOR CONTROL")
                            .font(.system(size: 7.5, weight: .bold, design: .rounded))
                            .tracking(0.6)
                            .foregroundStyle(WxGlassTheme.textMuted)
                            .lineLimit(1)
                        LocationBarView(isHUD: true)
                            .frame(minWidth: 165, idealWidth: 185, maxWidth: 230)
                    }

                    Spacer(minLength: 4)

                    modeSelectorPill(showAllIcons: false, compactPadding: true)

                    Spacer(minLength: 4)

                    toolbarRightControls
                }

                // 3. Compact responsive fallback when window resized below ~850pt (down to 520pt minWidth)
                VStack(alignment: .leading, spacing: 5) {
                    HStack(alignment: .center, spacing: 8) {
                        Color.clear.frame(width: 58, height: 24)

                        VStack(alignment: .leading, spacing: 1) {
                            Text("TACTICAL SENSOR CONTROL")
                                .font(.system(size: 7.5, weight: .bold, design: .rounded))
                                .tracking(0.6)
                                .foregroundStyle(WxGlassTheme.textMuted)
                                .lineLimit(1)
                            LocationBarView(isHUD: true)
                        }

                        Spacer(minLength: 4)

                        toolbarRightControls
                    }

                    ScrollView(.horizontal, showsIndicators: false) {
                        modeSelectorPill(showAllIcons: true, compactPadding: false)
                    }
                }
            }

            if !store.favorites.isEmpty {
                FavoritesQuickBarView()
                    .padding(.leading, 4)
            }

            toolbarLocationNotice
        }
        .padding(.horizontal, 12)
        .padding(.top, 7)
        .padding(.bottom, 7)
        .background {
            ZStack {
                Rectangle().fill(.ultraThinMaterial)
                Rectangle().fill(Color(red: 0.04, green: 0.09, blue: 0.18).opacity(0.62))
            }
        }
        .overlay(alignment: .bottom) {
            LinearGradient(
                colors: [.clear, WxGlassTheme.cyan.opacity(0.45), Color.white.opacity(0.28), WxGlassTheme.cyan.opacity(0.45), .clear],
                startPoint: .leading,
                endPoint: .trailing
            )
            .frame(height: 1)
        }
    }

    private var toolbarRightControls: some View {
        HStack(spacing: 6) {
            Picker("Units", selection: $store.units) {
                Text("°F").tag("imperial")
                Text("°C").tag("metric")
            }
            .pickerStyle(.segmented)
            .labelsHidden()
            .frame(width: 62)
            .onChange(of: store.units) { _, _ in
                Task { await store.applyLocationAndUnits() }
            }

            Menu {
                Text("MENU BAR & HUD MODE")
                    .font(.system(size: 9, weight: .bold, design: .monospaced))
                Divider()
                ForEach(MenuBarFormat.allCases) { fmt in
                    Button {
                        store.setMenuBarFormat(fmt)
                    } label: {
                        HStack {
                            Text(fmt.displayName)
                            if store.menuBarFormat == fmt {
                                Image(systemName: "checkmark")
                            }
                        }
                    }
                }
            } label: {
                Image(systemName: "menubar.arrow.up.rectangle")
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundStyle(WxGlassTheme.cyan)
                    .frame(width: 28, height: 26)
                    .wxFrostedCapsule(accent: WxGlassTheme.cyan)
            }
            .menuStyle(.borderlessButton)
            .menuIndicator(.hidden)
            .fixedSize()
            .help("Configure Menu Bar & HUD Popover Mode")

            Button {
                Task {
                    await store.refresh()
                    await store.refreshCPC()
                    await store.refreshChase()
                }
            } label: {
                HStack(spacing: 5) {
                    if store.isLoading {
                        ProgressView()
                            .controlSize(.mini)
                            .tint(WxGlassTheme.cyan)
                    } else {
                        Circle()
                            .fill(WxGlassTheme.nominalGreen)
                            .frame(width: 5.5, height: 5.5)
                            .shadow(color: WxGlassTheme.nominalGreen.opacity(0.8), radius: 2)
                    }
                    Text(updatedSubtitle ?? "SYNC")
                        .font(.system(size: 10, weight: .semibold, design: .monospaced))
                        .foregroundStyle(WxGlassTheme.textSecondary)
                }
                .padding(.horizontal, 9)
                .padding(.vertical, 5)
                .wxFrostedCapsule(accent: WxGlassTheme.cyan)
            }
            .buttonStyle(.plain)
            .disabled(store.isLoading)
            .help("Refresh Sensor Telemetry (⌘R)")
        }
    }

    @ViewBuilder
    private var toolbarLocationNotice: some View {
        if let err = store.errorMessage {
            if err == LocationManagerError.accessOffMessage {
                HStack(spacing: 10) {
                    Image(systemName: "location.slash.fill")
                        .font(.system(size: 11, weight: .semibold))
                        .foregroundStyle(WxGlassTheme.conditionRed)
                    Text(LocationManagerError.accessOffMessage)
                        .font(.system(size: 11, weight: .medium, design: .rounded))
                        .foregroundStyle(WxGlassTheme.conditionRed)
                        .accessibilityElement(children: .ignore)
                        .accessibilityIdentifier("wx.location-denied")
                        .accessibilityLabel(LocationManagerError.accessOffMessage)
                    Button {
                        store.openLocationSettings()
                    } label: {
                        Text("Open Location Services Settings")
                            .font(.system(size: 11, weight: .semibold, design: .rounded))
                            .underline()
                            .foregroundStyle(WxGlassTheme.cyan)
                    }
                    .buttonStyle(.plain)
                    .accessibilityIdentifier("wx.open-location-settings")
                    .accessibilityLabel("Open Location Services Settings")
                    Spacer()
                }
                .padding(.horizontal, 6)
            } else {
                Text("// ALERT: \(err)")
                    .font(.system(size: 10.5, weight: .medium, design: .monospaced))
                    .foregroundStyle(WxGlassTheme.conditionRed)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(.horizontal, 6)
                    .accessibilityElement(children: .ignore)
                    .accessibilityIdentifier("wx.location-alert")
                    .accessibilityLabel("// ALERT: \(err)")
            }
        } else if let warn = store.payload?.warning, !warn.isEmpty {
            Text(warn)
                .font(.system(size: 10.5, weight: .medium, design: .rounded))
                .foregroundStyle(WxGlassTheme.amber)
                .lineLimit(1)
                .help(warn)
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding(.horizontal, 6)
        }
    }

    private func modeSelectorPill(showAllIcons: Bool, compactPadding: Bool) -> some View {
        HStack(spacing: compactPadding ? 1.5 : 2) {
            modeButton(tab: .dual, label: "Tactical", icon: "rectangle.split.2x1.fill", showAllIcons: showAllIcons, compactPadding: compactPadding)
            modeButton(tab: .weather, label: "Surface", icon: "thermometer.sun.fill", showAllIcons: showAllIcons, compactPadding: compactPadding)
            modeButton(tab: .radar, label: "Radar", icon: "dot.radiowaves.left.and.right", showAllIcons: showAllIcons, compactPadding: compactPadding)
            modeButton(tab: .outlooks, label: "Outlooks", icon: "chart.line.uptrend.xyaxis", showAllIcons: showAllIcons, compactPadding: compactPadding)
            modeButton(tab: .chase, label: "Chase", icon: "bolt.shield.fill", showAllIcons: showAllIcons, compactPadding: compactPadding)
            modeButton(tab: .climate, label: "Climate", icon: "calendar.day.timeline.left", showAllIcons: showAllIcons, compactPadding: compactPadding)
            modeButton(tab: .tropics, label: "Tropics", icon: "hurricane", showAllIcons: showAllIcons, compactPadding: compactPadding)
            modeButton(tab: .grid, label: "Grid", icon: "square.grid.2x2.fill", showAllIcons: showAllIcons, compactPadding: compactPadding)
        }
        .padding(compactPadding ? 3 : 3.5)
        .background {
            ZStack {
                Capsule().fill(.ultraThinMaterial)
                Capsule().fill(Color.black.opacity(0.28))
            }
        }
        .overlay(Capsule().strokeBorder(Color.white.opacity(0.16), lineWidth: 0.9))
    }

    private func modeButton(tab: DeskTab, label: String, icon: String, showAllIcons: Bool, compactPadding: Bool) -> some View {
        let isSelected = (store.selectedDeskTab == tab)
        return Button {
            store.selectedDeskTab = tab
            if (tab == .dual || tab == .radar) && store.radarImage == nil && !store.isRadarLoading {
                Task { await store.refreshRadar() }
            }
            if tab == .outlooks && store.cpcPayload == nil && !store.isCPCLoading {
                Task { await store.refreshCPC() }
            }
            if tab == .chase && store.chasePayload == nil && !store.isChaseLoading {
                Task { await store.refreshChase() }
            }
            if tab == .climate && store.historyPayload == nil && !store.isHistoryLoading {
                Task { await store.refreshHistory() }
            }
            if tab == .tropics && store.tropicsPayload == nil && !store.isTropicsLoading {
                Task { await store.refreshTropics() }
            }
            if tab == .grid && store.gridCards.isEmpty && !store.isGridLoading {
                Task { await store.refreshGrid() }
            }
        } label: {
            HStack(spacing: compactPadding ? 3 : 4) {
                if showAllIcons || isSelected {
                    Image(systemName: icon)
                        .font(.system(size: compactPadding ? 9.5 : 10, weight: .semibold))
                }
                Text(label)
                    .font(.system(size: compactPadding ? 10 : 10.5, weight: .semibold, design: .rounded))
                    .lineLimit(1)
            }
            .padding(.horizontal, compactPadding ? 6.5 : 8)
            .padding(.vertical, 5)
            .background {
                if isSelected {
                    Capsule()
                        .fill(WxGlassTheme.cyan.opacity(0.24))
                        .overlay(Capsule().strokeBorder(WxGlassTheme.cyan.opacity(0.65), lineWidth: 0.9))
                }
            }
            .foregroundStyle(isSelected ? WxGlassTheme.textPrimary : WxGlassTheme.textSecondary)
        }
        .buttonStyle(.plain)
    }

    // MARK: - Sub-Cards

    @ViewBuilder
    private var tacticalSignalsCard: some View {
        let hasShift = store.cpcPayload?.patternShift.hasShift == true
        let hasChase = (store.chasePayload?.totalClusters ?? 0) > 0

        if hasShift || hasChase {
            WxGlassCard(title: "Synoptic Signals & Active Hazards", tag: "NOAA.INTEL", accent: WxGlassTheme.amber) {
                VStack(alignment: .leading, spacing: 10) {
                    if let shift = store.cpcPayload?.patternShift, shift.hasShift {
                        HStack(alignment: .top, spacing: 8) {
                            Image(systemName: "exclamationmark.bubble.fill")
                                .font(.system(size: 14))
                                .foregroundStyle(WxGlassTheme.amber)
                            VStack(alignment: .leading, spacing: 4) {
                                HStack {
                                    Text("Regime Shift")
                                        .font(.system(size: 11.5, weight: .bold, design: .rounded))
                                        .foregroundStyle(WxGlassTheme.amber)
                                    Spacer()
                                    Button {
                                        store.selectedDeskTab = .outlooks
                                    } label: {
                                        HStack(spacing: 3) {
                                            Text("Outlooks")
                                            Image(systemName: "arrow.right")
                                        }
                                        .font(.system(size: 10.5, weight: .semibold, design: .rounded))
                                        .foregroundStyle(WxGlassTheme.cyan)
                                        .padding(.horizontal, 8)
                                        .padding(.vertical, 3)
                                        .background(WxGlassTheme.cyan.opacity(0.14), in: Capsule())
                                    }
                                    .buttonStyle(.plain)
                                }
                                Text(shift.summary)
                                    .font(.system(size: 11.5))
                                    .foregroundStyle(WxGlassTheme.textPrimary)
                                    .fixedSize(horizontal: false, vertical: true)
                            }
                        }
                    }

                    if hasShift && hasChase {
                        Divider().overlay(Color.white.opacity(0.12))
                    }

                    if let chase = store.chasePayload, chase.totalClusters > 0 {
                        HStack(alignment: .top, spacing: 8) {
                            Image(systemName: "bolt.shield.fill")
                                .font(.system(size: 14))
                                .foregroundStyle(WxGlassTheme.amber)
                            VStack(alignment: .leading, spacing: 4) {
                                HStack {
                                    Text("Storm Chase · \(chase.totalClusters) Active Clusters")
                                        .font(.system(size: 11.5, weight: .bold, design: .rounded))
                                        .foregroundStyle(WxGlassTheme.amber)
                                    Spacer()
                                    Button {
                                        store.selectedDeskTab = .chase
                                    } label: {
                                        HStack(spacing: 3) {
                                            Text("Open Chase")
                                            Image(systemName: "arrow.right")
                                        }
                                        .font(.system(size: 10.5, weight: .semibold, design: .rounded))
                                        .foregroundStyle(WxGlassTheme.cyan)
                                        .padding(.horizontal, 8)
                                        .padding(.vertical, 3)
                                        .background(WxGlassTheme.cyan.opacity(0.14), in: Capsule())
                                    }
                                    .buttonStyle(.plain)
                                }
                                if let top = chase.clusters.first {
                                    Text("Top: \(top.name) (\(top.totalAlerts) warnings, score \(top.score))")
                                        .font(.system(size: 11.5))
                                        .foregroundStyle(WxGlassTheme.textSecondary)
                                        .lineLimit(1)
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    @ViewBuilder
    private var ephemerisCard: some View {
        if let astro = store.payload?.conditions?.astronomy ?? store.payload?.astronomy {
            EphemerisCardView(astronomy: astro)
        }
    }

    @ViewBuilder
    private var airQualityCard: some View {
        if let aq = store.payload?.conditions?.airQuality ?? store.payload?.airQuality {
            AirQualityCardView(airQuality: aq)
        }
    }

    @ViewBuilder
    private var nowcastCard: some View {
        if let nc = store.nowcastPayload?.nowcast ?? store.payload?.conditions?.nowcast ?? store.payload?.nowcast {
            NowcastCardView(nowcast: nc)
        }
    }
}
