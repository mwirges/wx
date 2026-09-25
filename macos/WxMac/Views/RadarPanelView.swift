import SwiftUI
import AppKit

struct RadarProductOption: Identifiable {
    let id: String
    let label: String
    let hint: String
}

private let radarProducts: [RadarProductOption] = [
    RadarProductOption(id: "composite-reflectivity", label: "Composite", hint: "Max reflectivity across all tilts (conus mosaic)"),
    RadarProductOption(id: "base-reflectivity", label: "Base", hint: "Lowest 0.5° scan tilt (single-station)"),
    RadarProductOption(id: "echo-tops", label: "Echo Tops", hint: "Storm cloud top heights (MRMS mosaic)"),
    RadarProductOption(id: "storm-relative-velocity", label: "Velocity", hint: "Storm-relative wind motion (rotation)")
]

struct RadarPanelView: View {
    @EnvironmentObject var store: WeatherStore

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
        VStack(alignment: .leading, spacing: 12) {
            // Product selector
            VStack(alignment: .leading, spacing: 6) {
                Picker("Radar Product", selection: $store.selectedRadarProduct) {
                    ForEach(radarProducts) { prod in
                        Text(prod.label).tag(prod.id)
                    }
                }
                .pickerStyle(.segmented)
                .labelsHidden()
                .onChange(of: store.selectedRadarProduct) { _, _ in
                    Task { await store.refreshRadar() }
                }

                HStack {
                    if let sel = radarProducts.first(where: { $0.id == store.selectedRadarProduct }) {
                        Text(sel.hint)
                            .font(.caption2)
                            .foregroundStyle(WxTheme.textSecondary)
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
                                Image(systemName: "arrow.clockwise")
                            }
                            Text("Refresh")
                        }
                        .font(.caption)
                        .foregroundStyle(WxTheme.accent)
                    }
                    .buttonStyle(.plain)
                    .disabled(store.isRadarLoading)
                }
            }

            // Radar Image Display Area
            ZStack {
                RoundedRectangle(cornerRadius: 10, style: .continuous)
                    .fill(Color.black.opacity(0.6))
                    .aspectRatio(1, contentMode: .fit)

                if let img = store.radarImage {
                    Image(nsImage: img)
                        .resizable()
                        .aspectRatio(1, contentMode: .fit)
                        .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))

                    if store.isRadarLoading {
                        RoundedRectangle(cornerRadius: 10, style: .continuous)
                            .fill(Color.black.opacity(0.45))
                        ProgressView("Updating radar…")
                            .tint(WxTheme.accent)
                            .foregroundStyle(WxTheme.text)
                    }
                } else if store.isRadarLoading {
                    VStack(spacing: 8) {
                        ProgressView()
                            .controlSize(.regular)
                            .tint(WxTheme.accent)
                        Text("Fetching radar imagery…")
                            .font(.callout)
                            .foregroundStyle(WxTheme.textSecondary)
                    }
                } else if let err = store.radarErrorMessage {
                    VStack(spacing: 8) {
                        Image(systemName: "exclamationmark.triangle")
                            .font(.title2)
                            .foregroundStyle(WxTheme.warn)
                        Text(err)
                            .font(.caption)
                            .foregroundStyle(WxTheme.textSecondary)
                            .multilineTextAlignment(.center)
                            .padding(.horizontal, 16)
                        Button("Retry") {
                            Task { await store.refreshRadar() }
                        }
                        .foregroundStyle(WxTheme.accent)
                    }
                } else {
                    VStack(spacing: 8) {
                        Image(systemName: "dot.radiowaves.left.and.right")
                            .font(.title)
                            .foregroundStyle(WxTheme.accent.opacity(0.6))
                        Text("No radar image loaded")
                            .font(.callout)
                            .foregroundStyle(WxTheme.textSecondary)
                        Button("Load Radar") {
                            Task { await store.refreshRadar() }
                        }
                        .foregroundStyle(WxTheme.accent)
                    }
                }
            }
            .overlay(
                RoundedRectangle(cornerRadius: 10, style: .continuous)
                    .strokeBorder(WxTheme.border, lineWidth: 1)
            )

            // Metadata footer & legend
            VStack(alignment: .leading, spacing: 6) {
                HStack {
                    if let loc = store.radarPayload?.location {
                        Text(loc)
                            .font(.caption.weight(.medium))
                            .foregroundStyle(WxTheme.text)
                    }
                    if let st = store.radarPayload?.station, !st.isEmpty {
                        Text("· \(st)")
                            .font(.caption)
                            .foregroundStyle(WxTheme.textSecondary)
                    }
                    Spacer()
                    if let vt = formattedValidTime {
                        Text("Valid: \(vt)")
                            .font(.caption2)
                            .foregroundStyle(WxTheme.textSecondary)
                    }
                }

                // dBZ Reflectivity Scale Bar
                VStack(alignment: .leading, spacing: 3) {
                    HStack(spacing: 2) {
                        Text("15").font(.system(size: 8)).foregroundStyle(WxTheme.textSecondary)
                        Spacer()
                        Text("dBZ Intensity").font(.system(size: 8, weight: .semibold)).foregroundStyle(WxTheme.textSecondary)
                        Spacer()
                        Text("70+").font(.system(size: 8)).foregroundStyle(WxTheme.textSecondary)
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
                    .frame(height: 6)
                    .clipShape(Capsule())
                    .overlay(Capsule().strokeBorder(WxTheme.border.opacity(0.5), lineWidth: 0.5))

                    HStack {
                        Text("Light")
                            .font(.system(size: 9))
                            .foregroundStyle(WxTheme.textSecondary)
                        Spacer()
                        Text("Moderate")
                            .font(.system(size: 9))
                            .foregroundStyle(WxTheme.textSecondary)
                        Spacer()
                        Text("Heavy / Severe")
                            .font(.system(size: 9))
                            .foregroundStyle(WxTheme.textSecondary)
                    }
                }
                .padding(8)
                .background(
                    RoundedRectangle(cornerRadius: 8, style: .continuous)
                        .fill(WxTheme.accent.opacity(0.06))
                        .overlay(
                            RoundedRectangle(cornerRadius: 8, style: .continuous)
                                .strokeBorder(WxTheme.border.opacity(0.3), lineWidth: 1)
                        )
                )

                Text("200 km radius · NWS MRMS & IEM NEXRAD")
                    .font(.caption2)
                    .foregroundStyle(WxTheme.textSecondary.opacity(0.7))
            }
        }
        .onAppear {
            if store.radarImage == nil && !store.isRadarLoading {
                Task { await store.refreshRadar() }
            }
        }
    }
}
