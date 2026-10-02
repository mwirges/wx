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
                    .font(.system(size: 9.5, weight: .bold))
                    .foregroundStyle(WxTheme.snwCyan)
                Text("PRODUCT // \(selectedOption.label.uppercased())")
                    .font(.system(size: 9.5, weight: .bold, design: .monospaced))
                    .foregroundStyle(WxTheme.text)
                Image(systemName: "chevron.up.chevron.down")
                    .font(.system(size: 7.5, weight: .bold))
                    .foregroundStyle(WxTheme.snwCyan.opacity(0.8))
            }
            .padding(.horizontal, 8)
            .padding(.vertical, 5)
            .background(isHUD ? WxTheme.snwChassis.opacity(0.92) : WxTheme.snwPanel, in: RoundedRectangle(cornerRadius: 5))
            .overlay(RoundedRectangle(cornerRadius: 5).strokeBorder(WxTheme.border.opacity(0.45), lineWidth: 0.8))
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
                    .font(.system(size: 9.5, weight: .bold))
                    .foregroundStyle((store.currentRadarBBox != nil || store.selectedRadarRadius > 200) ? WxTheme.snwGreen : WxTheme.snwCyan)
                Text("SCALE // \(selectedOption.label.uppercased())")
                    .font(.system(size: 9.5, weight: .bold, design: .monospaced))
                    .foregroundStyle(WxTheme.text)
                Image(systemName: "chevron.up.chevron.down")
                    .font(.system(size: 7.5, weight: .bold))
                    .foregroundStyle(WxTheme.snwCyan.opacity(0.8))
            }
            .padding(.horizontal, 8)
            .padding(.vertical, 5)
            .background(isHUD ? WxTheme.snwChassis.opacity(0.92) : WxTheme.snwPanel, in: RoundedRectangle(cornerRadius: 5))
            .overlay(RoundedRectangle(cornerRadius: 5).strokeBorder(WxTheme.border.opacity(0.45), lineWidth: 0.8))
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
                    ProgressView().controlSize(.regular).tint(WxTheme.snwCyan)
                    Text("ACQUIRING NOAA MRMS RADAR ARRAY…")
                        .font(.system(size: 11, weight: .bold, design: .monospaced))
                        .foregroundStyle(WxTheme.snwCyan)
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
                .background(Color.black.opacity(0.85))
            } else if let err = store.radarErrorMessage {
                VStack(spacing: 8) {
                    Image(systemName: "exclamationmark.triangle")
                        .font(.title)
                        .foregroundStyle(WxTheme.snwRed)
                    Text("// TELEMETRY FAULT: \(err)")
                        .font(.system(size: 11, design: .monospaced))
                        .foregroundStyle(WxTheme.snwRed)
                    Button("RETRY SCAN") {
                        Task { await store.refreshRadar() }
                    }
                    .font(.system(size: 10, weight: .bold, design: .monospaced))
                    .foregroundStyle(WxTheme.snwCyan)
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
                .background(Color.black.opacity(0.85))
            } else {
                VStack(spacing: 8) {
                    Image(systemName: "dot.radiowaves.left.and.right")
                        .font(.system(size: 32))
                        .foregroundStyle(WxTheme.snwCyan.opacity(0.6))
                    Text("RADAR ARRAY OFFLINE")
                        .font(.system(size: 11, weight: .bold, design: .monospaced))
                        .foregroundStyle(WxTheme.textSecondary)
                    Button("ENGAGE SCAN") {
                        Task { await store.refreshRadar() }
                    }
                    .font(.system(size: 10, weight: .bold, design: .monospaced))
                    .foregroundStyle(WxTheme.snwCyan)
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
                .background(Color.black.opacity(0.85))
            }

            // 2. Floating Top Tactical HUD Bar
            VStack(spacing: 0) {
                HStack(spacing: 8) {
                    // Location Input with GPS and Favorites
                    LocationBarView(isHUD: true) {
                        Task {
                            await store.applyLocationAndUnits()
                            await store.refreshRadar()
                        }
                    }
                    .frame(width: 210)

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
                                .font(.system(size: 9))
                            Text("RECENTER")
                                .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                        }
                        .padding(.horizontal, 8)
                        .padding(.vertical, 5)
                        .background(WxTheme.snwChassis.opacity(0.92), in: RoundedRectangle(cornerRadius: 5))
                        .overlay(RoundedRectangle(cornerRadius: 5).strokeBorder(WxTheme.snwCyan.opacity(0.45), lineWidth: 0.8))
                        .foregroundStyle(WxTheme.text)
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
                            Text(store.isRadarLoading ? "REFRESHING…" : "REFRESH")
                        }
                        .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                        .foregroundStyle(WxTheme.snwCyan)
                        .padding(.horizontal, 8)
                        .padding(.vertical, 5)
                        .background(WxTheme.snwCyan.opacity(0.15), in: RoundedRectangle(cornerRadius: 5))
                        .overlay(RoundedRectangle(cornerRadius: 5).strokeBorder(WxTheme.snwCyan.opacity(0.45), lineWidth: 0.8))
                    }
                    .buttonStyle(.plain)
                    .disabled(store.isRadarLoading)
                    .help("Refresh NOAA MRMS radar telemetry now")

                    Spacer()

                    // Retrieved timestamp readout
                    if let refreshed = store.lastRadarRefreshed {
                        TimelineView(.periodic(from: .now, by: 5.0)) { timeline in
                            let seconds = max(0, Int(timeline.date.timeIntervalSince(refreshed)))
                            let relText = seconds < 10 ? "JUST NOW" : (seconds < 60 ? "\(seconds)s AGO" : "\(seconds/60)m AGO")
                            HStack(spacing: 4) {
                                Image(systemName: "clock.arrow.circlepath")
                                    .font(.system(size: 8))
                                    .foregroundStyle(WxTheme.snwCyan)
                                Text("FETCHED // \(refreshed.formatted(date: .omitted, time: .standard)) (\(relText))")
                                    .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                                    .foregroundStyle(WxTheme.snwSilver)
                            }
                            .padding(.horizontal, 8)
                            .padding(.vertical, 5)
                            .background(WxTheme.snwChassis.opacity(0.92), in: RoundedRectangle(cornerRadius: 5))
                            .overlay(RoundedRectangle(cornerRadius: 5).strokeBorder(WxTheme.border.opacity(0.45), lineWidth: 0.8))
                        }
                    }

                    // Live badge / Multi-radar composite indicator
                    let isComp = (store.radarPayload?.isComposite ?? false) || store.selectedRadarRadius > 200 || store.currentRadarBBox != nil
                    let stationList = store.radarPayload?.stations ?? []
                    HStack(spacing: 5) {
                        Circle()
                            .fill(WxTheme.snwGreen)
                            .frame(width: 5, height: 5)
                            .shadow(color: WxTheme.snwGreen.opacity(0.8), radius: 3)
                        if isComp && stationList.count > 1 {
                            Text("NOAA MRMS // MOSAIC (\(stationList.count) SITES)")
                                .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                                .foregroundStyle(WxTheme.snwCyan)
                                .help("Contributing NEXRAD radars: " + stationList.joined(separator: ", "))
                        } else if isComp {
                            Text("NOAA MRMS // COMPOSITE MOSAIC")
                                .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                                .foregroundStyle(WxTheme.snwCyan)
                        } else {
                            Text("NOAA MRMS 1KM // LIVE")
                                .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                                .foregroundStyle(WxTheme.text)
                        }
                    }
                    .padding(.horizontal, 8)
                    .padding(.vertical, 5)
                    .background(WxTheme.snwChassis.opacity(0.92), in: RoundedRectangle(cornerRadius: 5))
                    .overlay(RoundedRectangle(cornerRadius: 5).strokeBorder(WxTheme.snwCyan.opacity(0.45), lineWidth: 0.8))
                }
                .padding(.horizontal, 14)
                .padding(.top, 10)

                Spacer()

                // 3. Floating Bottom HUD: Scale Bar & Metadata + Radar Transport Bar
                HStack(alignment: .bottom, spacing: 10) {
                    VStack(alignment: .leading, spacing: 6) {
                        HStack {
                            if let loc = store.radarPayload?.location {
                                Text(loc.uppercased())
                                    .font(.system(size: 9.5, weight: .bold, design: .monospaced))
                                    .foregroundStyle(WxTheme.text)
                            }
                            if let st = store.radarPayload?.station, !st.isEmpty {
                                Text("· RADAR // \(st)")
                                    .font(.system(size: 9, design: .monospaced))
                                    .foregroundStyle(WxTheme.snwSilver)
                            }
                            Spacer()
                            if let vt = formattedValidTime {
                                Text("OBSERVED // \(vt)")
                                    .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                                    .foregroundStyle(WxTheme.snwCyan)
                            }
                            if let refreshed = store.lastRadarRefreshed {
                                Text("· RETRIEVED // \(refreshed.formatted(date: .omitted, time: .standard))")
                                    .font(.system(size: 8.5, design: .monospaced))
                                    .foregroundStyle(WxTheme.snwSilver.opacity(0.85))
                            }
                        }

                        scaleBarView

                        let isCompFooter = (store.radarPayload?.isComposite ?? false) || store.selectedRadarRadius > 200
                        let radKm = String(format: "%.0f", store.radarPayload?.radiusKm ?? store.selectedRadarRadius)
                        let stationList = store.radarPayload?.stations ?? []
                        let footerText: String = {
                            if isCompFooter && stationList.count > 1 {
                                return "NOAA MRMS MULTI-RADAR COMPOSITE MOSAIC · \(radKm) KM RADIUS · \(stationList.count) SITES · WGS84 VECTOR OVERLAY"
                            } else if isCompFooter {
                                return "NOAA MRMS MULTI-RADAR COMPOSITE MOSAIC · \(radKm) KM RADIUS · WGS84 VECTOR OVERLAY"
                            } else {
                                return "NOAA MRMS SENSOR ARRAY · \(radKm) KM SCAN RADIUS · WGS84 VECTOR OVERLAY"
                            }
                        }()
                        Text(footerText)
                            .font(.system(size: 8, weight: .medium, design: .monospaced))
                            .foregroundStyle(WxTheme.snwSilver.opacity(0.7))
                    }
                    .padding(10)
                    .background(WxTheme.snwChassis.opacity(0.92))
                    .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
                    .overlay(RoundedRectangle(cornerRadius: 8).strokeBorder(WxTheme.border.opacity(0.45), lineWidth: 0.8))
                    .overlay(SNWCornerBrackets(color: WxTheme.snwCyan.opacity(0.7), length: 8, thickness: 1))
                    .frame(maxWidth: 440)

                    RadarTransportBar(compact: false)
                        .frame(maxWidth: 440)

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

                if let sel = radarProducts.first(where: { $0.id == store.selectedRadarProduct }) {
                    Text("// \(sel.hint.uppercased())")
                        .font(.system(size: 8.5, weight: .medium, design: .monospaced))
                        .foregroundStyle(WxTheme.snwSilver.opacity(0.75))
                        .lineLimit(1)
                        .truncationMode(.tail)
                }

                if let refreshed = store.lastRadarRefreshed {
                    TimelineView(.periodic(from: .now, by: 5.0)) { timeline in
                        let seconds = max(0, Int(timeline.date.timeIntervalSince(refreshed)))
                        let relText = seconds < 10 ? "JUST NOW" : (seconds < 60 ? "\(seconds)s AGO" : "\(seconds/60)m AGO")
                        Text("// FETCHED \(refreshed.formatted(date: .omitted, time: .standard)) (\(relText))")
                            .font(.system(size: 8, weight: .bold, design: .monospaced))
                            .foregroundStyle(WxTheme.snwSilver.opacity(0.85))
                            .lineLimit(1)
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
                        Text(store.isRadarLoading ? "REFRESHING…" : "REFRESH")
                    }
                    .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                    .foregroundStyle(WxTheme.snwCyan)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 4.5)
                    .background(WxTheme.snwCyan.opacity(0.12), in: RoundedRectangle(cornerRadius: 4))
                    .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(WxTheme.snwCyan.opacity(0.35), lineWidth: 0.8))
                }
                .buttonStyle(.plain)
                .disabled(store.isRadarLoading)
                .help("Refresh NOAA MRMS radar telemetry now")
            }

            // Radar Map Display Area
            ZStack {
                RoundedRectangle(cornerRadius: 8, style: .continuous)
                    .fill(Color.black.opacity(0.7))

                if let img = store.radarImage, let payload = store.radarPayload, payload.bbox != nil, payload.center != nil {
                    ZStack(alignment: .top) {
                        RadarMapView(payload: payload, image: img, recenterID: recenterID, onVisibleRadiusChanged: { visibleKm in
                            handleVisibleRadiusChanged(visibleKm)
                        }, onBBoxNeedsUpdate: { newBBox in
                            Task { await store.refreshRadar(bbox: newBBox) }
                        })
                        .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))

                        // Floating tactical HUD controls
                        HStack {
                            Button {
                                recenterID += 1
                            } label: {
                                HStack(spacing: 4) {
                                    Image(systemName: "scope")
                                        .font(.system(size: 9))
                                    Text("RECENTER GRID")
                                        .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                                }
                                .padding(.horizontal, 8)
                                .padding(.vertical, 4)
                                .background(WxTheme.snwPanel, in: RoundedRectangle(cornerRadius: 4))
                                .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(WxTheme.snwCyan.opacity(0.4), lineWidth: 0.8))
                                .foregroundStyle(WxTheme.text)
                            }
                            .buttonStyle(.plain)

                            Spacer()

                            let isComp = (store.radarPayload?.isComposite ?? false) || store.selectedRadarRadius > 200
                            let stationList = store.radarPayload?.stations ?? []
                            HStack(spacing: 5) {
                                Circle()
                                    .fill(WxTheme.snwGreen)
                                    .frame(width: 5, height: 5)
                                    .shadow(color: WxTheme.snwGreen.opacity(0.8), radius: 3)
                                if isComp && stationList.count > 1 {
                                    Text("NOAA MRMS // MOSAIC (\(stationList.count))")
                                        .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                                        .foregroundStyle(WxTheme.snwCyan)
                                        .help("Contributing NEXRAD radars: " + stationList.joined(separator: ", "))
                                } else if isComp {
                                    Text("NOAA MRMS // COMPOSITE")
                                        .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                                        .foregroundStyle(WxTheme.snwCyan)
                                } else {
                                    Text("NOAA MRMS 1KM // LIVE")
                                        .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                                        .foregroundStyle(WxTheme.text)
                                }
                            }
                            .padding(.horizontal, 8)
                            .padding(.vertical, 4)
                            .background(WxTheme.snwPanel, in: RoundedRectangle(cornerRadius: 4))
                            .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(WxTheme.snwCyan.opacity(0.4), lineWidth: 0.8))
                        }
                        .padding(10)

                        if store.isRadarLoading {
                            RoundedRectangle(cornerRadius: 8, style: .continuous)
                                .fill(Color.black.opacity(0.55))
                            VStack(spacing: 6) {
                                ProgressView()
                                    .tint(WxTheme.snwCyan)
                                Text("DOWNLINKING MRMS TELEMETRY…")
                                    .font(.system(size: 9.5, weight: .bold, design: .monospaced))
                                    .foregroundStyle(WxTheme.snwCyan)
                            }
                        }
                    }
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if let img = store.radarImage {
                    ZStack {
                        Image(nsImage: img)
                            .resizable()
                            .aspectRatio(contentMode: .fit)
                            .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))

                        if store.isRadarLoading {
                            RoundedRectangle(cornerRadius: 8, style: .continuous)
                                .fill(Color.black.opacity(0.55))
                            ProgressView("UPDATING SENSORS…")
                                .tint(WxTheme.snwCyan)
                                .font(.system(size: 9.5, weight: .bold, design: .monospaced))
                                .foregroundStyle(WxTheme.text)
                        }
                    }
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if store.isRadarLoading {
                    VStack(spacing: 8) {
                        ProgressView()
                            .controlSize(.regular)
                            .tint(WxTheme.snwCyan)
                        Text("ACQUIRING NOAA MRMS RADAR ARRAY…")
                            .font(.system(size: 10, weight: .bold, design: .monospaced))
                            .foregroundStyle(WxTheme.snwCyan)
                    }
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if let err = store.radarErrorMessage {
                    VStack(spacing: 8) {
                        Image(systemName: "exclamationmark.triangle")
                            .font(.title2)
                            .foregroundStyle(WxTheme.snwRed)
                        Text("// TELEMETRY FAULT: \(err)")
                            .font(.system(size: 10, design: .monospaced))
                            .foregroundStyle(WxTheme.snwRed)
                            .multilineTextAlignment(.center)
                            .padding(.horizontal, 16)
                        Button("RETRY SCAN") {
                            Task { await store.refreshRadar() }
                        }
                        .font(.system(size: 9.5, weight: .bold, design: .monospaced))
                        .foregroundStyle(WxTheme.snwCyan)
                    }
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else {
                    VStack(spacing: 8) {
                        Image(systemName: "dot.radiowaves.left.and.right")
                            .font(.title)
                            .foregroundStyle(WxTheme.snwCyan.opacity(0.6))
                        Text("RADAR ARRAY OFFLINE")
                            .font(.system(size: 10, weight: .bold, design: .monospaced))
                            .foregroundStyle(WxTheme.textSecondary)
                        Button("ENGAGE SCAN") {
                            Task { await store.refreshRadar() }
                        }
                        .font(.system(size: 9.5, weight: .bold, design: .monospaced))
                        .foregroundStyle(WxTheme.snwCyan)
                    }
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                }
            }
            .frame(maxWidth: .infinity)
            .frame(height: flexibleHeight ? nil : 380)
            .frame(maxHeight: flexibleHeight ? .infinity : 380)
            .overlay(
                RoundedRectangle(cornerRadius: 8, style: .continuous)
                    .strokeBorder(WxTheme.border.opacity(0.4), lineWidth: 1)
            )
            .overlay(SNWCornerBrackets(color: WxTheme.snwCyan.opacity(0.8), length: 12, thickness: 1.5))

            // Radar Transport Playback Bar
            RadarTransportBar(compact: !flexibleHeight)
                .frame(maxWidth: .infinity)

            // Metadata footer & legend
            VStack(alignment: .leading, spacing: 6) {
                HStack {
                    if let loc = store.radarPayload?.location {
                        Text(loc.uppercased())
                            .font(.system(size: 9.5, weight: .bold, design: .monospaced))
                            .foregroundStyle(WxTheme.text)
                    }
                    if let st = store.radarPayload?.station, !st.isEmpty {
                        Text("· RADAR // \(st)")
                            .font(.system(size: 9, design: .monospaced))
                            .foregroundStyle(WxTheme.snwSilver)
                    }
                    Spacer()
                    if let vt = formattedValidTime {
                        Text("VALID // \(vt)")
                            .font(.system(size: 8.5, design: .monospaced))
                            .foregroundStyle(WxTheme.snwSilver.opacity(0.8))
                    }
                }

                scaleBarView

                Text("MRMS SENSOR ARRAY · 200 KM SCAN RADIUS · WGS84 VECTOR OVERLAY")
                    .font(.system(size: 8, weight: .medium, design: .monospaced))
                    .foregroundStyle(WxTheme.snwSilver.opacity(0.7))
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
        .frame(maxWidth: .infinity)
        .frame(maxHeight: flexibleHeight ? .infinity : nil)
    }

    // ── Shared Scale Bar Component ──────────────────────────────────────────────────

    @ViewBuilder
    private var scaleBarView: some View {
        if store.selectedRadarProduct == "echo-tops" {
            // Echo Tops Scale (kft)
            VStack(alignment: .leading, spacing: 3) {
                HStack(spacing: 2) {
                    Text("10 kft").font(.system(size: 8, design: .monospaced)).foregroundStyle(WxTheme.textSecondary)
                    Spacer()
                    Text("ECHO TOPS // CLOUD HEIGHT").font(.system(size: 8, weight: .bold, design: .monospaced)).foregroundStyle(WxTheme.snwSilver)
                    Spacer()
                    Text("60+ kft").font(.system(size: 8, design: .monospaced)).foregroundStyle(WxTheme.textSecondary)
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
                .frame(height: 5)
                .clipShape(Capsule())
                .overlay(Capsule().strokeBorder(WxTheme.border.opacity(0.4), lineWidth: 0.5))

                HStack {
                    Text("LOW TOPS")
                        .font(.system(size: 8, design: .monospaced))
                        .foregroundStyle(WxTheme.textSecondary)
                    Spacer()
                    Text("MID LEVEL")
                        .font(.system(size: 8, design: .monospaced))
                        .foregroundStyle(WxTheme.textSecondary)
                    Spacer()
                    Text("CONVECTIVE CORE")
                        .font(.system(size: 8, design: .monospaced))
                        .foregroundStyle(WxTheme.textSecondary)
                }
            }
            .padding(8)
            .background(
                RoundedRectangle(cornerRadius: 6, style: .continuous)
                    .fill(WxTheme.snwChassis.opacity(0.8))
                    .overlay(
                        RoundedRectangle(cornerRadius: 6, style: .continuous)
                            .strokeBorder(WxTheme.border.opacity(0.25), lineWidth: 0.8)
                    )
            )
        } else {
            // dBZ Reflectivity Scale Bar
            VStack(alignment: .leading, spacing: 3) {
                HStack(spacing: 2) {
                    Text("15").font(.system(size: 8, design: .monospaced)).foregroundStyle(WxTheme.textSecondary)
                    Spacer()
                    Text("REFLECTIVITY // dBZ INTENSITY").font(.system(size: 8, weight: .bold, design: .monospaced)).foregroundStyle(WxTheme.snwSilver)
                    Spacer()
                    Text("70+").font(.system(size: 8, design: .monospaced)).foregroundStyle(WxTheme.textSecondary)
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
                .frame(height: 5)
                .clipShape(Capsule())
                .overlay(Capsule().strokeBorder(WxTheme.border.opacity(0.4), lineWidth: 0.5))

                HStack {
                    Text("LIGHT")
                        .font(.system(size: 8, design: .monospaced))
                        .foregroundStyle(WxTheme.textSecondary)
                    Spacer()
                    Text("MODERATE")
                        .font(.system(size: 8, design: .monospaced))
                        .foregroundStyle(WxTheme.textSecondary)
                    Spacer()
                    Text("HEAVY / SEVERE")
                        .font(.system(size: 8, design: .monospaced))
                        .foregroundStyle(WxTheme.textSecondary)
                }
            }
            .padding(8)
            .background(
                RoundedRectangle(cornerRadius: 6, style: .continuous)
                    .fill(WxTheme.snwChassis.opacity(0.8))
                    .overlay(
                        RoundedRectangle(cornerRadius: 6, style: .continuous)
                            .strokeBorder(WxTheme.border.opacity(0.25), lineWidth: 0.8)
                    )
            )
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
            // Play / Pause Button
            Button {
                store.toggleLoop()
            } label: {
                HStack(spacing: 4) {
                    Image(systemName: store.isLoopPlaying ? "pause.fill" : "play.fill")
                        .font(.system(size: compact ? 9 : 10.5, weight: .bold))
                    if !compact {
                        Text(store.isLoopPlaying ? "PAUSE" : "LOOP")
                            .font(.system(size: 9, weight: .bold, design: .monospaced))
                    }
                }
                .foregroundStyle(store.isLoopPlaying ? Color.black : WxTheme.snwCyan)
                .padding(.horizontal, compact ? 7 : 10)
                .padding(.vertical, compact ? 4 : 5.5)
                .background(
                    store.isLoopPlaying ? WxTheme.snwCyan : WxTheme.snwPanel,
                    in: RoundedRectangle(cornerRadius: 5)
                )
                .overlay(
                    RoundedRectangle(cornerRadius: 5)
                        .strokeBorder(WxTheme.snwCyan.opacity(0.8), lineWidth: 1)
                )
            }
            .buttonStyle(.plain)
            .disabled(!hasFrames)

            // Step Backward Button
            Button {
                store.stepFrameBackward()
            } label: {
                Image(systemName: "backward.frame.fill")
                    .font(.system(size: compact ? 8.5 : 9.5))
                    .foregroundStyle(WxTheme.snwSilver)
                    .padding(compact ? 4 : 5.5)
                    .background(WxTheme.snwPanel, in: RoundedRectangle(cornerRadius: 4))
                    .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(WxTheme.border.opacity(0.4), lineWidth: 0.8))
            }
            .buttonStyle(.plain)
            .disabled(!hasFrames)

            // Step Forward Button
            Button {
                store.stepFrameForward()
            } label: {
                Image(systemName: "forward.frame.fill")
                    .font(.system(size: compact ? 8.5 : 9.5))
                    .foregroundStyle(WxTheme.snwSilver)
                    .padding(compact ? 4 : 5.5)
                    .background(WxTheme.snwPanel, in: RoundedRectangle(cornerRadius: 4))
                    .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(WxTheme.border.opacity(0.4), lineWidth: 0.8))
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
                                .fill(isSelected ? WxTheme.snwCyan : (f.isLive ? WxTheme.snwGreen.opacity(0.7) : WxTheme.snwSilver.opacity(0.35)))
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
            HStack(spacing: 5) {
                Text(formattedTime)
                    .font(.system(size: compact ? 9 : 10, weight: .bold, design: .monospaced))
                    .foregroundStyle(WxTheme.text)

                let isLive = activeFrame?.isLive ?? true
                Button {
                    store.jumpToLive()
                } label: {
                    HStack(spacing: 3) {
                        Circle()
                            .fill(isLive ? WxTheme.snwGreen : WxTheme.snwCyan)
                            .frame(width: 4.5, height: 4.5)
                        Text(activeFrame?.label ?? "LIVE")
                            .font(.system(size: 8, weight: .heavy, design: .monospaced))
                            .foregroundStyle(isLive ? WxTheme.snwGreen : WxTheme.snwCyan)
                    }
                    .padding(.horizontal, 6)
                    .padding(.vertical, 2.5)
                    .background(
                        (isLive ? WxTheme.snwGreen : WxTheme.snwCyan).opacity(0.12),
                        in: Capsule()
                    )
                    .overlay(
                        Capsule().strokeBorder((isLive ? WxTheme.snwGreen : WxTheme.snwCyan).opacity(0.4), lineWidth: 0.8)
                    )
                }
                .buttonStyle(.plain)

                // Quick Refresh Button
                Button {
                    Task { await store.refreshRadar() }
                } label: {
                    HStack(spacing: 3) {
                        if store.isRadarLoading {
                            ProgressView().controlSize(.small)
                        } else {
                            Image(systemName: "arrow.clockwise")
                                .font(.system(size: compact ? 8 : 9, weight: .bold))
                        }
                    }
                    .foregroundStyle(WxTheme.snwCyan)
                    .padding(compact ? 4 : 5)
                    .background(WxTheme.snwPanel, in: RoundedRectangle(cornerRadius: 4))
                    .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(WxTheme.snwCyan.opacity(0.4), lineWidth: 0.8))
                }
                .buttonStyle(.plain)
                .disabled(store.isRadarLoading)
                .help("Refresh NOAA MRMS radar telemetry now")
            }

            // Speed toggle (only in full HUD)
            if !compact {
                Button {
                    store.loopStepMs = (store.loopStepMs == 380) ? 190 : 380
                } label: {
                    Text(store.loopStepMs == 380 ? "1X" : "2X")
                        .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                        .foregroundStyle(store.loopStepMs == 190 ? WxTheme.snwCyan : WxTheme.snwSilver)
                        .padding(.horizontal, 6)
                        .padding(.vertical, 4)
                        .background(WxTheme.snwPanel, in: RoundedRectangle(cornerRadius: 4))
                        .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(WxTheme.border.opacity(0.4), lineWidth: 0.8))
                }
                .buttonStyle(.plain)
            }

            // Export Scan Menu
            Menu {
                Button {
                    if let img = currentFrameImage {
                        RadarExportService.copyImageToPasteboard(img)
                        triggerToast("COPIED")
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
                HStack(spacing: 3) {
                    if let toast = exportToast {
                        Text(toast)
                            .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                            .foregroundStyle(WxTheme.snwGreen)
                    } else {
                        Image(systemName: "square.and.arrow.up")
                            .font(.system(size: compact ? 8 : 9, weight: .bold))
                        if !compact {
                            Text("EXPORT SCAN")
                                .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                        }
                    }
                }
                .foregroundStyle(exportToast != nil ? WxTheme.snwGreen : WxTheme.snwCyan)
                .padding(.horizontal, compact ? 6 : 8)
                .padding(.vertical, compact ? 3.5 : 4)
                .background(WxTheme.snwPanel, in: RoundedRectangle(cornerRadius: 4))
                .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(WxTheme.snwCyan.opacity(0.5), lineWidth: 0.8))
            }
            .menuStyle(.borderlessButton)
            .fixedSize()
            .help("Export Scan: Copy PNG to clipboard, save PNG, or export animated GIF loop")
        }
        .padding(.horizontal, compact ? 10 : 12)
        .padding(.vertical, compact ? 6 : 8)
        .background(WxTheme.snwChassis.opacity(0.92))
        .clipShape(RoundedRectangle(cornerRadius: 7, style: .continuous))
        .overlay(RoundedRectangle(cornerRadius: 7).strokeBorder(WxTheme.border.opacity(0.45), lineWidth: 0.8))
    }
}

