import SwiftUI
import AppKit

struct RadarProductOption: Identifiable {
    let id: String
    let label: String
    let hint: String
}

private let radarProducts: [RadarProductOption] = [
    RadarProductOption(id: "composite-reflectivity", label: "Composite", hint: "Max reflectivity across all tilts (MRMS mosaic)"),
    RadarProductOption(id: "base-reflectivity", label: "Base", hint: "Lowest 0.5° scan tilt (MRMS mosaic)"),
    RadarProductOption(id: "echo-tops", label: "Echo Tops", hint: "Storm cloud top heights (MRMS mosaic)")
]

struct RadarPanelView: View {
    @EnvironmentObject var store: WeatherStore
    var fullScreen: Bool = false
    @State private var recenterID: Int = 0

    private var formattedValidTime: String? {
        guard let iso = store.radarPayload?.validTime else { return nil }
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
                RadarMapView(payload: payload, image: img, recenterID: recenterID)
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
                    // Location Input
                    HStack(spacing: 6) {
                        Image(systemName: "scope")
                            .font(.system(size: 11))
                            .foregroundStyle(WxTheme.snwCyan.opacity(0.8))
                        TextField("Zip or City, ST", text: $store.locationInput)
                            .textFieldStyle(.plain)
                            .font(.system(size: 11.5, weight: .medium, design: .monospaced))
                            .foregroundStyle(WxTheme.text)
                            .onSubmit {
                                Task {
                                    await store.applyLocationAndUnits()
                                    await store.refreshRadar()
                                }
                            }
                    }
                    .frame(width: 175)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 5)
                    .background(WxTheme.snwChassis.opacity(0.92), in: RoundedRectangle(cornerRadius: 6))
                    .overlay(RoundedRectangle(cornerRadius: 6).strokeBorder(WxTheme.border.opacity(0.4), lineWidth: 0.8))

                    // Product Selector
                    Picker("Radar Product", selection: $store.selectedRadarProduct) {
                        ForEach(radarProducts) { prod in
                            Text(prod.label.uppercased()).tag(prod.id)
                        }
                    }
                    .pickerStyle(.segmented)
                    .labelsHidden()
                    .frame(width: 250)
                    .background(WxTheme.snwChassis.opacity(0.92), in: RoundedRectangle(cornerRadius: 6))
                    .overlay(RoundedRectangle(cornerRadius: 6).strokeBorder(WxTheme.border.opacity(0.4), lineWidth: 0.8))
                    .onChange(of: store.selectedRadarProduct) { _, _ in
                        Task { await store.refreshRadar() }
                    }

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

                    // Scan button
                    Button {
                        Task { await store.refreshRadar() }
                    } label: {
                        HStack(spacing: 4) {
                            if store.isRadarLoading {
                                ProgressView().controlSize(.small)
                            } else {
                                Image(systemName: "arrow.triangle.2.circlepath")
                            }
                            Text("SCAN")
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

                    Spacer()

                    // Live badge
                    HStack(spacing: 5) {
                        Circle()
                            .fill(WxTheme.snwGreen)
                            .frame(width: 5, height: 5)
                            .shadow(color: WxTheme.snwGreen.opacity(0.8), radius: 3)
                        Text("NOAA MRMS 1KM // LIVE")
                            .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                            .foregroundStyle(WxTheme.text)
                    }
                    .padding(.horizontal, 8)
                    .padding(.vertical, 5)
                    .background(WxTheme.snwChassis.opacity(0.92), in: RoundedRectangle(cornerRadius: 5))
                    .overlay(RoundedRectangle(cornerRadius: 5).strokeBorder(WxTheme.snwCyan.opacity(0.45), lineWidth: 0.8))
                }
                .padding(.horizontal, 14)
                .padding(.top, 10)

                Spacer()

                // 3. Floating Bottom HUD: Scale Bar & Metadata
                HStack(alignment: .bottom) {
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

                        Text("NOAA MRMS SENSOR ARRAY · 200 KM SCAN RADIUS · WGS84 VECTOR OVERLAY")
                            .font(.system(size: 8, weight: .medium, design: .monospaced))
                            .foregroundStyle(WxTheme.snwSilver.opacity(0.7))
                    }
                    .padding(10)
                    .background(WxTheme.snwChassis.opacity(0.92))
                    .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
                    .overlay(RoundedRectangle(cornerRadius: 8).strokeBorder(WxTheme.border.opacity(0.45), lineWidth: 0.8))
                    .overlay(SNWCornerBrackets(color: WxTheme.snwCyan.opacity(0.7), length: 8, thickness: 1))
                    .frame(maxWidth: 480)

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
            // Product selector
            VStack(alignment: .leading, spacing: 5) {
                Picker("Radar Product", selection: $store.selectedRadarProduct) {
                    ForEach(radarProducts) { prod in
                        Text(prod.label.uppercased()).tag(prod.id)
                    }
                }
                .pickerStyle(.segmented)
                .labelsHidden()
                .onChange(of: store.selectedRadarProduct) { _, _ in
                    Task { await store.refreshRadar() }
                }

                HStack {
                    if let sel = radarProducts.first(where: { $0.id == store.selectedRadarProduct }) {
                        Text("// \(sel.hint.uppercased())")
                            .font(.system(size: 8.5, weight: .medium, design: .monospaced))
                            .foregroundStyle(WxTheme.snwSilver.opacity(0.8))
                    }
                    Spacer()
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
                            Text("SCAN ARRAY")
                        }
                        .font(.system(size: 9, weight: .bold, design: .monospaced))
                        .foregroundStyle(WxTheme.snwCyan)
                        .padding(.horizontal, 6)
                        .padding(.vertical, 3)
                        .background(WxTheme.snwCyan.opacity(0.12), in: RoundedRectangle(cornerRadius: 4))
                        .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(WxTheme.snwCyan.opacity(0.35), lineWidth: 0.8))
                    }
                    .buttonStyle(.plain)
                    .disabled(store.isRadarLoading)
                }
            }

            // Radar Map Display Area
            ZStack {
                RoundedRectangle(cornerRadius: 8, style: .continuous)
                    .fill(Color.black.opacity(0.7))

                if let img = store.radarImage, let payload = store.radarPayload, payload.bbox != nil, payload.center != nil {
                    ZStack(alignment: .top) {
                        RadarMapView(payload: payload, image: img, recenterID: recenterID)
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

                            HStack(spacing: 5) {
                                Circle()
                                    .fill(WxTheme.snwGreen)
                                    .frame(width: 5, height: 5)
                                    .shadow(color: WxTheme.snwGreen.opacity(0.8), radius: 3)
                                Text("NOAA MRMS 1KM // LIVE")
                                    .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                                    .foregroundStyle(WxTheme.text)
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
                    .aspectRatio(1, contentMode: .fit)
                } else if let img = store.radarImage {
                    ZStack {
                        Image(nsImage: img)
                            .resizable()
                            .aspectRatio(1, contentMode: .fit)
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
                    .aspectRatio(1, contentMode: .fit)
                } else if store.isRadarLoading {
                    VStack(spacing: 8) {
                        ProgressView()
                            .controlSize(.regular)
                            .tint(WxTheme.snwCyan)
                        Text("ACQUIRING NOAA MRMS RADAR ARRAY…")
                            .font(.system(size: 10, weight: .bold, design: .monospaced))
                            .foregroundStyle(WxTheme.snwCyan)
                    }
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
                }
            }
            .aspectRatio(1, contentMode: .fit)
            .frame(maxHeight: 380)
            .overlay(
                RoundedRectangle(cornerRadius: 8, style: .continuous)
                    .strokeBorder(WxTheme.border.opacity(0.4), lineWidth: 1)
            )
            .overlay(SNWCornerBrackets(color: WxTheme.snwCyan.opacity(0.8), length: 12, thickness: 1.5))

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
        }
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
}
