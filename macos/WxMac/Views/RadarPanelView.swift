import SwiftUI
import AppKit

struct RadarProductOption: Identifiable {
    let id: String
    let label: String
    let fullName: String
    let hint: String
}

private let radarProducts: [RadarProductOption] = [
    RadarProductOption(id: "composite-reflectivity", label: "Composite", fullName: "Composite Reflectivity", hint: "Max reflectivity across all tilts (MRMS mosaic)"),
    RadarProductOption(id: "base-reflectivity", label: "Base", fullName: "Base Reflectivity", hint: "Lowest 0.5° scan tilt (MRMS mosaic)"),
    RadarProductOption(id: "storm-relative-velocity", label: "Velocity", fullName: "Storm-Relative Velocity", hint: "Storm-relative velocity & rotation (RIDGE)"),
    RadarProductOption(id: "echo-tops", label: "Echo Tops", fullName: "Echo Tops", hint: "Storm cloud top heights (MRMS mosaic)"),
    RadarProductOption(id: "precip-type", label: "Precip Type", fullName: "Precipitation Type", hint: "Surface precipitation classification (MRMS)"),
    RadarProductOption(id: "one-hour-precip", label: "1-Hr Precip", fullName: "1-Hour Precipitation", hint: "1-hour precipitation accumulation (QPE)"),
    RadarProductOption(id: "storm-total-precip", label: "Storm Total", fullName: "Storm Total Precip", hint: "Storm total precipitation accumulation (RIDGE)")
]

struct RadarProductMenu: View {
    @EnvironmentObject var store: WeatherStore
    var isHUD: Bool = false

    private var selectedOption: RadarProductOption {
        radarProducts.first(where: { $0.id == store.selectedRadarProduct }) ?? radarProducts[0]
    }

    var body: some View {
        Menu {
            Picker("Radar Product", selection: $store.selectedRadarProduct) {
                ForEach(radarProducts) { prod in
                    Text("\(prod.fullName.uppercased())  —  \(prod.hint)").tag(prod.id)
                }
            }
            .pickerStyle(.inline)
        } label: {
            HStack(spacing: 5) {
                Image(systemName: "antenna.radiowaves.left.and.right")
                    .font(.system(size: 10, weight: .bold))
                    .foregroundStyle(WxGlassTheme.cyan)
                Text(selectedOption.label)
                    .font(.system(size: 10.5, weight: .semibold, design: .rounded))
                    .foregroundStyle(WxGlassTheme.textPrimary)
                Image(systemName: "chevron.up.chevron.down")
                    .font(.system(size: 7.5, weight: .bold))
                    .foregroundStyle(WxGlassTheme.cyan.opacity(0.85))
            }
            .padding(.horizontal, 10)
            .padding(.vertical, 5.5)
            .wxFrostedCapsule(accent: WxGlassTheme.cyan)
        }
        .menuStyle(.borderlessButton)
        .menuIndicator(.hidden)
        .fixedSize()
        .onChange(of: store.selectedRadarProduct) { _, _ in
            Task { await store.refreshRadar() }
        }
    }
}

struct RadarCoverageOption: Identifiable {
    let id: Double
    let label: String
    let fullName: String
    let hint: String
}

private let radarCoverages: [RadarCoverageOption] = [
    RadarCoverageOption(id: 0, label: "Auto", fullName: "Auto (Viewport)", hint: "Dynamically fit to map window & zoom"),
    RadarCoverageOption(id: 150, label: "Local", fullName: "Local (150 km)", hint: "Single-station high-resolution NEXRAD scan"),
    RadarCoverageOption(id: 250, label: "Metro", fullName: "Metro (250 km)", hint: "Near-field dual-pol composite"),
    RadarCoverageOption(id: 500, label: "Regional", fullName: "Regional (500 km)", hint: "Multi-radar composite mosaic (MRMS)"),
    RadarCoverageOption(id: 1000, label: "Synoptic", fullName: "Synoptic (1000 km)", hint: "Multi-state sector mosaic"),
    RadarCoverageOption(id: 2000, label: "CONUS", fullName: "CONUS (2000 km)", hint: "Seamless continental US national mosaic"),
]

struct RadarCoverageMenu: View {
    @EnvironmentObject var store: WeatherStore
    var isHUD: Bool = false

    private var selectedOption: RadarCoverageOption {
        if store.currentRadarBBox != nil || store.selectedRadarRadius == 0 {
            return radarCoverages[0]
        }
        return radarCoverages.filter { $0.id > 0 }.min(by: { abs($0.id - store.selectedRadarRadius) < abs($1.id - store.selectedRadarRadius) }) ?? radarCoverages[0]
    }

    var body: some View {
        Menu {
            Picker("Coverage Scale", selection: $store.selectedRadarRadius) {
                ForEach(radarCoverages) { cov in
                    Text("\(cov.fullName.uppercased())  —  \(cov.hint)").tag(cov.id)
                }
            }
            .pickerStyle(.inline)
        } label: {
            HStack(spacing: 5) {
                Image(systemName: (store.currentRadarBBox != nil || store.selectedRadarRadius > 200) ? "circle.grid.cross.fill" : "circle.circle")
                    .font(.system(size: 10, weight: .bold))
                    .foregroundStyle((store.currentRadarBBox != nil || store.selectedRadarRadius > 200) ? WxGlassTheme.nominalGreen : WxGlassTheme.cyan)
                Text("Scale · \(selectedOption.label)")
                    .font(.system(size: 10.5, weight: .semibold, design: .rounded))
                    .foregroundStyle(WxGlassTheme.textPrimary)
                Image(systemName: "chevron.up.chevron.down")
                    .font(.system(size: 7.5, weight: .bold))
                    .foregroundStyle(WxGlassTheme.cyan.opacity(0.85))
            }
            .padding(.horizontal, 10)
            .padding(.vertical, 5.5)
            .wxFrostedCapsule(accent: WxGlassTheme.cyan)
        }
        .menuStyle(.borderlessButton)
        .menuIndicator(.hidden)
        .fixedSize()
        .onChange(of: store.selectedRadarRadius) { _, newRadius in
            store.setRadarRadius(newRadius)
        }
    }
}

struct RadarPanelView: View {
    @EnvironmentObject var store: WeatherStore
    var fullScreen: Bool = false
    var flexibleHeight: Bool = false
    @State private var recenterID: Int = 0

    private var formattedValidTime: String? {
        let iso: String
        if store.radarFrames.indices.contains(store.activeFrameIndex) {
            iso = store.radarFrames[store.activeFrameIndex].validTime
        } else if let payloadIso = store.radarPayload?.validTime {
            iso = payloadIso
        } else {
            return nil
        }
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        if let date = formatter.date(from: iso) {
            return date.formatted(date: .omitted, time: .shortened)
        }
        formatter.formatOptions = [.withInternetDateTime]
        if let date = formatter.date(from: iso) {
            return date.formatted(date: .omitted, time: .shortened)
        }
        return iso
    }

    var body: some View {
        Group {
            if fullScreen {
                fullScreenRadarBody
            } else {
                compactRadarBody
            }
        }
        .onAppear {
            if store.radarImage == nil && !store.isRadarLoading {
                Task { await store.refreshRadar() }
            }
        }
    }

    // ── Full Screen Edge-to-Edge Radar View ─────────────────────────────────────────

    @ViewBuilder
    private var fullScreenRadarBody: some View {
        ZStack(alignment: .top) {
            // 1. Edge-to-edge interactive MapKit radar view
            if let img = store.radarImage, let payload = store.radarPayload, payload.bbox != nil, payload.center != nil {
                RadarMapView(payload: payload, image: img, recenterID: recenterID, onVisibleRadiusChanged: { visibleKm in
                    handleVisibleRadiusChanged(visibleKm)
                }, onBBoxNeedsUpdate: { newBBox in
                    Task { await store.refreshRadar(bbox: newBBox) }
                })
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let img = store.radarImage {
                Image(nsImage: img)
                    .resizable()
                    .aspectRatio(contentMode: .fit)
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if store.isRadarLoading {
                VStack(spacing: 8) {
                    ProgressView().controlSize(.regular).tint(WxGlassTheme.cyan)
                    Text("Acquiring NOAA MRMS Radar Array…")
                        .font(.system(size: 11.5, weight: .semibold, design: .rounded))
                        .foregroundStyle(WxGlassTheme.cyanSoft)
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
                .background(Color.black.opacity(0.65))
            } else if let err = store.radarErrorMessage {
                VStack(spacing: 8) {
                    Image(systemName: "exclamationmark.triangle")
                        .font(.title2)
                        .foregroundStyle(WxGlassTheme.conditionRed)
                    Text("Telemetry Fault: \(err)")
                        .font(.system(size: 11.5, design: .rounded))
                        .foregroundStyle(WxGlassTheme.conditionRed)
                        .multilineTextAlignment(.center)
                        .padding(.horizontal, 20)
                    Button("Retry Scan") {
                        Task { await store.refreshRadar() }
                    }
                    .font(.system(size: 10.5, weight: .semibold, design: .rounded))
                    .foregroundStyle(WxGlassTheme.cyan)
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
                .background(Color.black.opacity(0.65))
            } else {
                VStack(spacing: 8) {
                    Image(systemName: "dot.radiowaves.left.and.right")
                        .font(.system(size: 32))
                        .foregroundStyle(WxGlassTheme.cyan.opacity(0.6))
                    Text("Radar Array Standby")
                        .font(.system(size: 11.5, weight: .semibold, design: .rounded))
                        .foregroundStyle(WxGlassTheme.textSecondary)
                    Button("Engage Scan") {
                        Task { await store.refreshRadar() }
                    }
                    .font(.system(size: 10.5, weight: .semibold, design: .rounded))
                    .foregroundStyle(WxGlassTheme.cyan)
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
                .background(Color.black.opacity(0.65))
            }

            // 2. Floating Top Frosted Glass HUD Bar
            VStack(spacing: 0) {
                HStack(spacing: 8) {
                    // Product Selector Menu
                    RadarProductMenu(isHUD: true)

                    // Coverage Scale Menu
                    RadarCoverageMenu(isHUD: true)

                    // Recenter button
                    Button {
                        recenterID += 1
                    } label: {
                        HStack(spacing: 4) {
                            Image(systemName: "scope")
                                .font(.system(size: 9.5, weight: .semibold))
                            Text("Recenter")
                                .font(.system(size: 10, weight: .semibold, design: .rounded))
                        }
                        .padding(.horizontal, 10)
                        .padding(.vertical, 5.5)
                        .wxFrostedCapsule(accent: WxGlassTheme.cyan)
                        .foregroundStyle(WxGlassTheme.textPrimary)
                    }
                    .buttonStyle(.plain)

                    // Refresh button
                    Button {
                        Task { await store.refreshRadar() }
                    } label: {
                        HStack(spacing: 4) {
                            if store.isRadarLoading {
                                ProgressView().controlSize(.small)
                            } else {
                                Image(systemName: "arrow.triangle.2.circlepath")
                            }
                            Text(store.isRadarLoading ? "Refreshing…" : "Refresh")
                        }
                        .font(.system(size: 10, weight: .semibold, design: .rounded))
                        .foregroundStyle(WxGlassTheme.cyan)
                        .padding(.horizontal, 10)
                        .padding(.vertical, 5.5)
                        .wxFrostedCapsule(accent: WxGlassTheme.cyan, active: true)
                    }
                    .buttonStyle(.plain)
                    .disabled(store.isRadarLoading)
                    .help("Refresh NOAA MRMS radar telemetry now")

                    Spacer()

                    // Retrieved timestamp readout
                    if let refreshed = store.lastRadarRefreshed {
                        TimelineView(.periodic(from: .now, by: 5.0)) { timeline in
                            let seconds = max(0, Int(timeline.date.timeIntervalSince(refreshed)))
                            let relText = seconds < 10 ? "just now" : (seconds < 60 ? "\(seconds)s ago" : "\(seconds/60)m ago")
                            HStack(spacing: 5) {
                                Image(systemName: "clock.arrow.circlepath")
                                    .font(.system(size: 9))
                                    .foregroundStyle(WxGlassTheme.cyan)
                                Text("\(refreshed.formatted(date: .omitted, time: .shortened)) (\(relText))")
                                    .font(.system(size: 9.5, weight: .semibold, design: .monospaced))
                                    .foregroundStyle(WxGlassTheme.textSecondary)
                            }
                            .padding(.horizontal, 10)
                            .padding(.vertical, 5.5)
                            .wxFrostedCapsule(accent: WxGlassTheme.cyan)
                        }
                    }

                    // Live badge / Multi-radar composite indicator
                    let isComp = (store.radarPayload?.isComposite ?? false) || store.selectedRadarRadius > 200 || store.currentRadarBBox != nil
                    let stationList = store.radarPayload?.stations ?? []
                    HStack(spacing: 5) {
                        Circle()
                            .fill(WxGlassTheme.nominalGreen)
                            .frame(width: 6, height: 6)
                            .shadow(color: WxGlassTheme.nominalGreen.opacity(0.8), radius: 3)
                        if isComp && stationList.count > 1 {
                            Text("MRMS Mosaic · \(stationList.count) Sites")
                                .font(.system(size: 10, weight: .semibold, design: .rounded))
                                .foregroundStyle(WxGlassTheme.cyanSoft)
                                .help("Contributing NEXRAD radars: " + stationList.joined(separator: ", "))
                        } else if isComp {
                            Text("MRMS Composite")
                                .font(.system(size: 10, weight: .semibold, design: .rounded))
                                .foregroundStyle(WxGlassTheme.cyanSoft)
                        } else {
                            Text("MRMS 1km Live")
                                .font(.system(size: 10, weight: .semibold, design: .rounded))
                                .foregroundStyle(WxGlassTheme.textPrimary)
                        }
                    }
                    .padding(.horizontal, 10)
                    .padding(.vertical, 5.5)
                    .wxFrostedCapsule(accent: WxGlassTheme.cyan)
                }
                .padding(.horizontal, 14)
                .padding(.top, 10)

                Spacer()

                // 3. Floating Bottom HUD: Scale Bar & Metadata + Radar Transport Bar
                HStack(alignment: .bottom, spacing: 10) {
                    VStack(alignment: .leading, spacing: 6) {
                        HStack {
                            if let loc = store.radarPayload?.location {
                                Text(loc)
                                    .font(.system(size: 11, weight: .bold, design: .rounded))
                                    .foregroundStyle(WxGlassTheme.textPrimary)
                            }
                            if let st = store.radarPayload?.station, !st.isEmpty {
                                Text("· \(st)")
                                    .font(.system(size: 10, weight: .semibold, design: .monospaced))
                                    .foregroundStyle(WxGlassTheme.textSecondary)
                            }
                            Spacer()
                            if let vt = formattedValidTime {
                                Text("Valid \(vt)")
                                    .font(.system(size: 9.5, weight: .bold, design: .monospaced))
                                    .foregroundStyle(WxGlassTheme.cyan)
                            }
                        }

                        scaleBarView
                    }
                    .padding(12)
                    .background {
                        ZStack {
                            RoundedRectangle(cornerRadius: 18, style: .continuous).fill(.ultraThinMaterial)
                            RoundedRectangle(cornerRadius: 18, style: .continuous).fill(WxGlassTheme.pillTintOverlay())
                        }
                    }
                    .clipShape(RoundedRectangle(cornerRadius: 18, style: .continuous))
                    .overlay(
                        RoundedRectangle(cornerRadius: 18, style: .continuous)
                            .strokeBorder(Color.white.opacity(0.18), lineWidth: 1)
                    )
                    .shadow(color: Color.black.opacity(0.32), radius: 14, y: 6)
                    .frame(maxWidth: 420)

                    RadarTransportBar(compact: false)
                        .frame(maxWidth: 460)

                    Spacer()
                }
                .padding(.horizontal, 14)
                .padding(.bottom, 14)
            }

            // Downlinking overlay indicator
            if store.isRadarLoading && store.radarImage != nil {
                VStack {
                    Spacer()
                    HStack(spacing: 8) {
                        ProgressView().controlSize(.small).tint(WxTheme.snwCyan)
                        Text("DOWNLINKING MRMS TELEMETRY…")
                            .font(.system(size: 9.5, weight: .bold, design: .monospaced))
                            .foregroundStyle(WxTheme.snwCyan)
                    }
                    .padding(.horizontal, 14)
                    .padding(.vertical, 7)
                    .background(WxTheme.snwChassis.opacity(0.9), in: Capsule())
                    .overlay(Capsule().strokeBorder(WxTheme.snwCyan.opacity(0.5), lineWidth: 0.8))
                    .padding(.bottom, 80)
                }
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    // ── Compact Radar Card View (Used in Dual Console) ──────────────────────────────

    @ViewBuilder
    private var compactRadarBody: some View {
        VStack(alignment: .leading, spacing: 10) {
            // Product selector bar
            HStack(spacing: 8) {
                RadarProductMenu(isHUD: false)
                RadarCoverageMenu(isHUD: false)

                Spacer(minLength: 4)

                if let refreshed = store.lastRadarRefreshed {
                    TimelineView(.periodic(from: .now, by: 5.0)) { timeline in
                        let seconds = max(0, Int(timeline.date.timeIntervalSince(refreshed)))
                        let relText = seconds < 10 ? "just now" : (seconds < 60 ? "\(seconds)s ago" : "\(seconds/60)m ago")
                        HStack(spacing: 4) {
                            Image(systemName: "clock.arrow.circlepath")
                                .font(.system(size: 8.5))
                                .foregroundStyle(WxGlassTheme.cyan)
                            Text("\(refreshed.formatted(date: .omitted, time: .shortened)) · \(relText)")
                                .font(.system(size: 9, weight: .semibold, design: .monospaced))
                                .foregroundStyle(WxGlassTheme.textSecondary)
                                .lineLimit(1)
                        }
                        .padding(.horizontal, 9)
                        .padding(.vertical, 5)
                        .wxFrostedCapsule(accent: WxGlassTheme.cyan)
                    }
                }

                Button {
                    Task { await store.refreshRadar() }
                } label: {
                    HStack(spacing: 4) {
                        if store.isRadarLoading {
                            ProgressView()
                                .controlSize(.small)
                        } else {
                            Image(systemName: "arrow.triangle.2.circlepath")
                        }
                        Text(store.isRadarLoading ? "Refreshing…" : "Refresh")
                    }
                    .font(.system(size: 10, weight: .semibold, design: .rounded))
                    .foregroundStyle(WxGlassTheme.cyan)
                    .padding(.horizontal, 10)
                    .padding(.vertical, 5)
                    .wxFrostedCapsule(accent: WxGlassTheme.cyan, active: true)
                }
                .buttonStyle(.plain)
                .disabled(store.isRadarLoading)
                .help("Refresh NOAA MRMS radar telemetry now")
            }

            // Radar Map Display Area
            ZStack {
                RoundedRectangle(cornerRadius: 14, style: .continuous)
                    .fill(Color.black.opacity(0.65))

                if let img = store.radarImage, let payload = store.radarPayload, payload.bbox != nil, payload.center != nil {
                    ZStack(alignment: .top) {
                        RadarMapView(payload: payload, image: img, recenterID: recenterID, onVisibleRadiusChanged: { visibleKm in
                            handleVisibleRadiusChanged(visibleKm)
                        }, onBBoxNeedsUpdate: { newBBox in
                            Task { await store.refreshRadar(bbox: newBBox) }
                        })
                        .clipShape(RoundedRectangle(cornerRadius: 14, style: .continuous))

                        // Floating frosted HUD controls
                        HStack {
                            Button {
                                recenterID += 1
                            } label: {
                                HStack(spacing: 4) {
                                    Image(systemName: "scope")
                                        .font(.system(size: 9.5))
                                    Text("Recenter")
                                        .font(.system(size: 10, weight: .semibold, design: .rounded))
                                }
                                .padding(.horizontal, 10)
                                .padding(.vertical, 5)
                                .wxFrostedCapsule(accent: WxGlassTheme.cyan)
                                .foregroundStyle(WxGlassTheme.textPrimary)
                            }
                            .buttonStyle(.plain)

                            Spacer()

                            let isComp = (store.radarPayload?.isComposite ?? false) || store.selectedRadarRadius > 200
                            let stationList = store.radarPayload?.stations ?? []
                            HStack(spacing: 5) {
                                Circle()
                                    .fill(WxGlassTheme.nominalGreen)
                                    .frame(width: 5.5, height: 5.5)
                                    .shadow(color: WxGlassTheme.nominalGreen.opacity(0.8), radius: 3)
                                if isComp && stationList.count > 1 {
                                    Text("MRMS Mosaic (\(stationList.count))")
                                        .font(.system(size: 9.5, weight: .semibold, design: .rounded))
                                        .foregroundStyle(WxGlassTheme.cyanSoft)
                                        .help("Contributing NEXRAD radars: " + stationList.joined(separator: ", "))
                                } else if isComp {
                                    Text("MRMS Composite")
                                        .font(.system(size: 9.5, weight: .semibold, design: .rounded))
                                        .foregroundStyle(WxGlassTheme.cyanSoft)
                                } else {
                                    Text("MRMS 1km Live")
                                        .font(.system(size: 9.5, weight: .semibold, design: .rounded))
                                        .foregroundStyle(WxGlassTheme.textPrimary)
                                }
                            }
                            .padding(.horizontal, 10)
                            .padding(.vertical, 5)
                            .wxFrostedCapsule(accent: WxGlassTheme.cyan)
                        }
                        .padding(10)

                        if store.isRadarLoading {
                            RoundedRectangle(cornerRadius: 14, style: .continuous)
                                .fill(Color.black.opacity(0.48))
                            VStack(spacing: 6) {
                                ProgressView()
                                    .tint(WxGlassTheme.cyan)
                                Text("Downlinking MRMS Telemetry…")
                                    .font(.system(size: 10.5, weight: .semibold, design: .rounded))
                                    .foregroundStyle(WxGlassTheme.cyanSoft)
                            }
                        }
                    }
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if let img = store.radarImage {
                    ZStack {
                        Image(nsImage: img)
                            .resizable()
                            .aspectRatio(contentMode: .fit)
                            .clipShape(RoundedRectangle(cornerRadius: 14, style: .continuous))

                        if store.isRadarLoading {
                            RoundedRectangle(cornerRadius: 14, style: .continuous)
                                .fill(Color.black.opacity(0.48))
                            ProgressView("Updating Sensors…")
                                .tint(WxGlassTheme.cyan)
                                .font(.system(size: 10.5, weight: .semibold, design: .rounded))
                                .foregroundStyle(WxGlassTheme.textPrimary)
                        }
                    }
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if store.isRadarLoading {
                    VStack(spacing: 8) {
                        ProgressView()
                            .controlSize(.regular)
                            .tint(WxGlassTheme.cyan)
                        Text("Acquiring NOAA MRMS Radar Array…")
                            .font(.system(size: 11, weight: .semibold, design: .rounded))
                            .foregroundStyle(WxGlassTheme.cyanSoft)
                    }
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if let err = store.radarErrorMessage {
                    VStack(spacing: 8) {
                        Image(systemName: "exclamationmark.triangle")
                            .font(.title2)
                            .foregroundStyle(WxGlassTheme.conditionRed)
                        Text("Telemetry Fault: \(err)")
                            .font(.system(size: 11, design: .rounded))
                            .foregroundStyle(WxGlassTheme.conditionRed)
                            .multilineTextAlignment(.center)
                            .padding(.horizontal, 16)
                        Button("Retry Scan") {
                            Task { await store.refreshRadar() }
                        }
                        .font(.system(size: 10.5, weight: .semibold, design: .rounded))
                        .foregroundStyle(WxGlassTheme.cyan)
                    }
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else {
                    VStack(spacing: 8) {
                        Image(systemName: "dot.radiowaves.left.and.right")
                            .font(.title)
                            .foregroundStyle(WxGlassTheme.cyan.opacity(0.6))
                        Text("Radar Array Standby")
                            .font(.system(size: 11, weight: .semibold, design: .rounded))
                            .foregroundStyle(WxGlassTheme.textSecondary)
                        Button("Engage Scan") {
                            Task { await store.refreshRadar() }
                        }
                        .font(.system(size: 10.5, weight: .semibold, design: .rounded))
                        .foregroundStyle(WxGlassTheme.cyan)
                    }
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                }
            }
            .frame(maxWidth: .infinity)
            .frame(height: flexibleHeight ? nil : 380)
            .frame(maxHeight: flexibleHeight ? .infinity : 380)
            .overlay(
                RoundedRectangle(cornerRadius: 14, style: .continuous)
                    .strokeBorder(Color.white.opacity(0.16), lineWidth: 1)
            )

            // Radar Transport Playback Bar
            RadarTransportBar(compact: !flexibleHeight)
                .frame(maxWidth: .infinity)

            // Metadata footer & legend
            VStack(alignment: .leading, spacing: 6) {
                HStack {
                    if let loc = store.radarPayload?.location {
                        Text(loc)
                            .font(.system(size: 10.5, weight: .bold, design: .rounded))
                            .foregroundStyle(WxGlassTheme.textPrimary)
                    }
                    if let st = store.radarPayload?.station, !st.isEmpty {
                        Text("· \(st)")
                            .font(.system(size: 9.5, weight: .semibold, design: .monospaced))
                            .foregroundStyle(WxGlassTheme.textSecondary)
                    }
                    Spacer()
                    if let vt = formattedValidTime {
                        Text("Valid \(vt)")
                            .font(.system(size: 9, weight: .bold, design: .monospaced))
                            .foregroundStyle(WxGlassTheme.cyanSoft)
                    }
                }

                scaleBarView

                Text("MRMS Sensor Array · 200 km Scan Radius · WGS84 Vector Overlay")
                    .font(.system(size: 8.5, weight: .medium, design: .rounded))
                    .foregroundStyle(WxGlassTheme.textMuted)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
        .frame(maxWidth: .infinity)
        .frame(maxHeight: flexibleHeight ? .infinity : nil)
    }

    // ── Shared Scale Bar Component ──────────────────────────────────────────────────

    @ViewBuilder
    private var scaleBarView: some View {
        let nested = WxGlassTheme.nestedShape()
        if store.selectedRadarProduct == "echo-tops" {
            // Echo Tops Scale (kft)
            VStack(alignment: .leading, spacing: 4) {
                HStack(spacing: 2) {
                    Text("10 kft").font(.system(size: 8.5, weight: .medium, design: .monospaced)).foregroundStyle(WxGlassTheme.textSecondary)
                    Spacer()
                    Text("Echo Tops · Cloud Height")
                        .font(.system(size: 9, weight: .bold, design: .rounded))
                        .foregroundStyle(WxGlassTheme.textSecondary)
                    Spacer()
                    Text("60+ kft").font(.system(size: 8.5, weight: .medium, design: .monospaced)).foregroundStyle(WxGlassTheme.textSecondary)
                }

                LinearGradient(
                    colors: [
                        Color(red: 0.1, green: 0.3, blue: 0.8), // Deep blue (10-15 kft)
                        Color(red: 0.1, green: 0.7, blue: 0.8), // Cyan (20-25 kft)
                        Color(red: 0.1, green: 0.8, blue: 0.2), // Green (30-35 kft)
                        Color(red: 0.9, green: 0.9, blue: 0.1), // Yellow (40-45 kft)
                        Color(red: 1.0, green: 0.4, blue: 0.1), // Orange (50 kft)
                        Color(red: 0.9, green: 0.1, blue: 0.1), // Red (55 kft)
                        Color(red: 0.8, green: 0.1, blue: 0.8)  // Purple (60+ kft)
                    ],
                    startPoint: .leading,
                    endPoint: .trailing
                )
                .frame(height: 5.5)
                .clipShape(Capsule())
                .overlay(Capsule().strokeBorder(Color.white.opacity(0.22), lineWidth: 0.5))

                HStack {
                    Text("Low Tops")
                    Spacer()
                    Text("Mid Level")
                    Spacer()
                    Text("Convective Core")
                }
                .font(.system(size: 8.5, weight: .medium, design: .rounded))
                .foregroundStyle(WxGlassTheme.textMuted)
            }
            .padding(.horizontal, 10)
            .padding(.vertical, 8)
            .background(Color.white.opacity(0.05), in: nested)
            .overlay(nested.strokeBorder(Color.white.opacity(0.12), lineWidth: 0.8))
        } else {
            // dBZ Reflectivity Scale Bar
            VStack(alignment: .leading, spacing: 4) {
                HStack(spacing: 2) {
                    Text("15 dBZ").font(.system(size: 8.5, weight: .medium, design: .monospaced)).foregroundStyle(WxGlassTheme.textSecondary)
                    Spacer()
                    Text("Reflectivity · dBZ Intensity")
                        .font(.system(size: 9, weight: .bold, design: .rounded))
                        .foregroundStyle(WxGlassTheme.textSecondary)
                    Spacer()
                    Text("70+ dBZ").font(.system(size: 8.5, weight: .medium, design: .monospaced)).foregroundStyle(WxGlassTheme.textSecondary)
                }

                LinearGradient(
                    colors: [
                        Color(red: 0.1, green: 0.8, blue: 0.2),  // Light green (15-25)
                        Color(red: 0.0, green: 0.6, blue: 0.1),  // Dark green (30)
                        Color(red: 0.9, green: 0.9, blue: 0.1),  // Yellow (35-40)
                        Color(red: 1.0, green: 0.5, blue: 0.0),  // Orange (45-50)
                        Color(red: 0.9, green: 0.1, blue: 0.1),  // Red (55-60)
                        Color(red: 0.8, green: 0.0, blue: 0.8),  // Purple (65)
                        Color(red: 0.9, green: 0.6, blue: 0.9)   // White/Pink (70+)
                    ],
                    startPoint: .leading,
                    endPoint: .trailing
                )
                .frame(height: 5.5)
                .clipShape(Capsule())
                .overlay(Capsule().strokeBorder(Color.white.opacity(0.22), lineWidth: 0.5))

                HStack {
                    Text("Light")
                    Spacer()
                    Text("Moderate")
                    Spacer()
                    Text("Heavy / Severe")
                }
                .font(.system(size: 8.5, weight: .medium, design: .rounded))
                .foregroundStyle(WxGlassTheme.textMuted)
            }
            .padding(.horizontal, 10)
            .padding(.vertical, 8)
            .background(Color.white.opacity(0.05), in: nested)
            .overlay(nested.strokeBorder(Color.white.opacity(0.12), lineWidth: 0.8))
        }
    }

    private func handleVisibleRadiusChanged(_ visibleKm: Double) {
        // Viewport scale tracking is handled dynamically by map overlays
    }
}

struct RadarTransportBar: View {
    @EnvironmentObject var store: WeatherStore
    var compact: Bool = false

    @State private var exportToast: String?

    private var hasFrames: Bool {
        store.radarFrames.count > 1
    }

    private var activeFrame: DecodedRadarFrame? {
        guard store.radarFrames.indices.contains(store.activeFrameIndex) else { return nil }
        return store.radarFrames[store.activeFrameIndex]
    }

    private var currentFrameImage: NSImage? {
        activeFrame?.image ?? store.radarImage
    }

    private var exportBaseName: String {
        let loc = store.locationInput.trimmingCharacters(in: .whitespacesAndNewlines)
        let name = loc.isEmpty ? "radar" : loc
        let time = activeFrame?.validTime ?? "latest"
        return "\(name)-\(time)"
    }

    private var formattedTime: String {
        guard let frame = activeFrame else {
            return store.radarPayload?.validTime ?? "--:--"
        }
        if let d = frame.date {
            return d.formatted(date: .omitted, time: .shortened)
        }
        return frame.validTime
    }

    private func triggerToast(_ msg: String) {
        exportToast = msg
        Task {
            try? await Task.sleep(nanoseconds: 1_800_000_000)
            exportToast = nil
        }
    }

    var body: some View {
        HStack(spacing: compact ? 6 : 10) {
            // Play / Pause Capsule Pill
            Button {
                store.toggleLoop()
            } label: {
                HStack(spacing: 4) {
                    Image(systemName: store.isLoopPlaying ? "pause.fill" : "play.fill")
                        .font(.system(size: compact ? 9 : 10, weight: .bold))
                    if !compact {
                        Text(store.isLoopPlaying ? "Pause" : "Loop")
                            .font(.system(size: 10, weight: .bold, design: .rounded))
                    }
                }
                .foregroundStyle(store.isLoopPlaying ? Color(red: 0.03, green: 0.07, blue: 0.14) : WxGlassTheme.cyan)
                .padding(.horizontal, compact ? 9 : 12)
                .padding(.vertical, compact ? 5 : 6)
                .background(
                    store.isLoopPlaying ? WxGlassTheme.cyan : WxGlassTheme.cyan.opacity(0.14),
                    in: Capsule()
                )
                .overlay(
                    Capsule()
                        .strokeBorder(WxGlassTheme.cyan.opacity(store.isLoopPlaying ? 0.95 : 0.60), lineWidth: 1)
                )
            }
            .buttonStyle(.plain)
            .disabled(!hasFrames)

            // Step Backward Circular Pill
            Button {
                store.stepFrameBackward()
            } label: {
                Image(systemName: "backward.frame.fill")
                    .font(.system(size: compact ? 8.5 : 9.5, weight: .semibold))
                    .foregroundStyle(WxGlassTheme.textSecondary)
                    .frame(width: compact ? 24 : 27, height: compact ? 24 : 27)
                    .background(Color.white.opacity(0.07), in: Circle())
                    .overlay(Circle().strokeBorder(Color.white.opacity(0.16), lineWidth: 0.8))
            }
            .buttonStyle(.plain)
            .disabled(!hasFrames)

            // Step Forward Circular Pill
            Button {
                store.stepFrameForward()
            } label: {
                Image(systemName: "forward.frame.fill")
                    .font(.system(size: compact ? 8.5 : 9.5, weight: .semibold))
                    .foregroundStyle(WxGlassTheme.textSecondary)
                    .frame(width: compact ? 24 : 27, height: compact ? 24 : 27)
                    .background(Color.white.opacity(0.07), in: Circle())
                    .overlay(Circle().strokeBorder(Color.white.opacity(0.16), lineWidth: 0.8))
            }
            .buttonStyle(.plain)
            .disabled(!hasFrames)

            // Interactive Frame Timeline Ticks
            if hasFrames {
                HStack(spacing: 3) {
                    ForEach(store.radarFrames) { f in
                        let isSelected = (f.id == store.activeFrameIndex)
                        Button {
                            store.seekFrame(to: f.id)
                        } label: {
                            Capsule()
                                .fill(isSelected ? WxGlassTheme.cyan : (f.isLive ? WxGlassTheme.nominalGreen.opacity(0.75) : Color.white.opacity(0.28)))
                                .frame(width: compact ? 12 : 18, height: isSelected ? 8 : 5)
                                .animation(.spring(response: 0.2, dampingFraction: 0.7), value: isSelected)
                        }
                        .buttonStyle(.plain)
                    }
                }
                .padding(.horizontal, 4)
            }

            Spacer(minLength: 4)

            // Time & Relative Badge
            HStack(spacing: 6) {
                Text(formattedTime)
                    .font(.system(size: compact ? 9 : 10, weight: .bold, design: .monospaced))
                    .foregroundStyle(WxGlassTheme.textPrimary)

                let isLive = activeFrame?.isLive ?? true
                Button {
                    store.jumpToLive()
                } label: {
                    HStack(spacing: 3) {
                        Circle()
                            .fill(isLive ? WxGlassTheme.nominalGreen : WxGlassTheme.cyan)
                            .frame(width: 4.5, height: 4.5)
                        Text(activeFrame?.label ?? "LIVE")
                            .font(.system(size: 8.5, weight: .heavy, design: .monospaced))
                            .foregroundStyle(isLive ? WxGlassTheme.nominalGreen : WxGlassTheme.cyan)
                    }
                    .padding(.horizontal, 7)
                    .padding(.vertical, 3)
                    .background(
                        (isLive ? WxGlassTheme.nominalGreen : WxGlassTheme.cyan).opacity(0.14),
                        in: Capsule()
                    )
                    .overlay(
                        Capsule().strokeBorder((isLive ? WxGlassTheme.nominalGreen : WxGlassTheme.cyan).opacity(0.42), lineWidth: 0.8)
                    )
                }
                .buttonStyle(.plain)

                // Quick Refresh Circular Pill
                Button {
                    Task { await store.refreshRadar() }
                } label: {
                    Group {
                        if store.isRadarLoading {
                            ProgressView().controlSize(.mini)
                        } else {
                            Image(systemName: "arrow.clockwise")
                                .font(.system(size: compact ? 8.5 : 9.5, weight: .bold))
                        }
                    }
                    .foregroundStyle(WxGlassTheme.cyan)
                    .frame(width: compact ? 24 : 26, height: compact ? 24 : 26)
                    .background(Color.white.opacity(0.07), in: Circle())
                    .overlay(Circle().strokeBorder(WxGlassTheme.cyan.opacity(0.38), lineWidth: 0.8))
                }
                .buttonStyle(.plain)
                .disabled(store.isRadarLoading)
                .help("Refresh NOAA MRMS radar telemetry now")
            }

            // Speed toggle (only in full HUD)
            if !compact {
                let fast = (store.loopStepMs == 190)
                Button {
                    store.loopStepMs = fast ? 380 : 190
                } label: {
                    Text(fast ? "2×" : "1×")
                        .font(.system(size: 9.5, weight: .bold, design: .rounded))
                        .foregroundStyle(fast ? WxGlassTheme.cyan : WxGlassTheme.textSecondary)
                        .padding(.horizontal, 8)
                        .padding(.vertical, 4.5)
                        .background(fast ? WxGlassTheme.cyan.opacity(0.16) : Color.white.opacity(0.07), in: Capsule())
                        .overlay(Capsule().strokeBorder(fast ? WxGlassTheme.cyan.opacity(0.55) : Color.white.opacity(0.16), lineWidth: 0.8))
                }
                .buttonStyle(.plain)
            }

            // Export Scan Menu Capsule
            Menu {
                Button {
                    if let img = currentFrameImage {
                        RadarExportService.copyImageToPasteboard(img)
                        triggerToast("Copied")
                    }
                } label: {
                    Label("Copy Frame to Clipboard", systemImage: "doc.on.doc")
                }
                .disabled(currentFrameImage == nil)

                Button {
                    if let img = currentFrameImage {
                        RadarExportService.saveImageAsPNG(img, suggestedFilename: "wx-radar-\(exportBaseName)")
                    }
                } label: {
                    Label("Save Frame (PNG)...", systemImage: "arrow.down.doc")
                }
                .disabled(currentFrameImage == nil)

                Button {
                    let images = store.radarFrames.map { $0.image }
                    if !images.isEmpty {
                        let delay = Double(store.loopStepMs) / 1000.0
                        RadarExportService.exportLoopAsGIF(
                            images: images,
                            frameDelay: delay,
                            suggestedFilename: "wx-radar-loop-\(exportBaseName)"
                        )
                    }
                } label: {
                    Label("Export Loop (\(max(1, store.radarFrames.count))-Frame GIF)...", systemImage: "film")
                }
                .disabled(store.radarFrames.isEmpty)
            } label: {
                HStack(spacing: 4) {
                    if let toast = exportToast {
                        Text(toast)
                            .font(.system(size: 9.5, weight: .bold, design: .rounded))
                            .foregroundStyle(WxGlassTheme.nominalGreen)
                    } else {
                        Image(systemName: "square.and.arrow.up")
                            .font(.system(size: compact ? 8.5 : 9.5, weight: .bold))
                        if !compact {
                            Text("Export")
                                .font(.system(size: 9.5, weight: .semibold, design: .rounded))
                        }
                    }
                }
                .foregroundStyle(exportToast != nil ? WxGlassTheme.nominalGreen : WxGlassTheme.cyan)
                .padding(.horizontal, compact ? 8 : 10)
                .padding(.vertical, compact ? 4.5 : 5)
                .background(Color.white.opacity(0.07), in: Capsule())
                .overlay(Capsule().strokeBorder(WxGlassTheme.cyan.opacity(0.42), lineWidth: 0.8))
            }
            .menuStyle(.borderlessButton)
            .fixedSize()
            .help("Export Scan: Copy PNG to clipboard, save PNG, or export animated GIF loop")
        }
        .padding(.horizontal, compact ? 10 : 12)
        .padding(.vertical, compact ? 7 : 9)
        .background {
            ZStack {
                RoundedRectangle(cornerRadius: 16, style: .continuous).fill(.ultraThinMaterial)
                RoundedRectangle(cornerRadius: 16, style: .continuous).fill(WxGlassTheme.pillTintOverlay())
            }
        }
        .clipShape(RoundedRectangle(cornerRadius: 16, style: .continuous))
        .overlay(RoundedRectangle(cornerRadius: 16, style: .continuous).strokeBorder(Color.white.opacity(0.18), lineWidth: 0.9))
    }
}

