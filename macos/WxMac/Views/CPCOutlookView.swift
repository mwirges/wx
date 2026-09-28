import SwiftUI

/// Climate Prediction Center (CPC) Long-Range Outlooks and Synoptic Pattern Shift Console.
struct CPCOutlookView: View {
    @EnvironmentObject var store: WeatherStore

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 14) {
                // Top Action & Status Bar
                HStack(alignment: .center, spacing: 8) {
                    VStack(alignment: .leading, spacing: 2) {
                        Text("NOAA CLIMATE PREDICTION CENTER (CPC)")
                            .font(.system(size: 11, weight: .bold, design: .monospaced))
                            .foregroundStyle(WxTheme.snwCyan)
                        if let loc = store.cpcPayload?.location ?? store.payload?.conditions?.location {
                            Text(loc.uppercased())
                                .font(.system(size: 9.5, weight: .semibold, design: .monospaced))
                                .foregroundStyle(WxTheme.textSecondary)
                        }
                    }

                    Spacer()

                    if store.isCPCLoading {
                        ProgressView().controlSize(.small).tint(WxTheme.snwCyan)
                    }

                    Button {
                        Task { await store.refreshCPC() }
                    } label: {
                        HStack(spacing: 4) {
                            Image(systemName: "arrow.triangle.2.circlepath")
                            Text("REFRESH OUTLOOKS")
                        }
                        .font(.system(size: 9, weight: .bold, design: .monospaced))
                        .foregroundStyle(WxTheme.snwCyan)
                        .padding(.horizontal, 9)
                        .padding(.vertical, 5)
                        .background(WxTheme.snwCyan.opacity(0.12), in: RoundedRectangle(cornerRadius: 4))
                        .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(WxTheme.snwCyan.opacity(0.4), lineWidth: 0.8))
                    }
                    .buttonStyle(.plain)
                    .disabled(store.isCPCLoading)
                }
                .padding(.horizontal, 4)

                // Error Banner if any
                if let err = store.cpcErrorMessage {
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

                // 1. Synoptic Pattern Shift Heads-Up Card
                patternShiftCard

                // 2. Outlooks Grid (6-10 Day and 8-14 Day Horizons)
                if let outlooks = store.cpcPayload?.outlooks, !outlooks.isEmpty {
                    VStack(alignment: .leading, spacing: 12) {
                        ForEach(outlooks) { item in
                            horizonCard(item: item)
                        }
                    }
                } else if store.isCPCLoading {
                    VStack(spacing: 8) {
                        ProgressView().controlSize(.regular).tint(WxTheme.snwCyan)
                        Text("ACQUIRING NOAA CPC 6-10D & 8-14D VECTOR TELEMETRY…")
                            .font(.system(size: 10, weight: .bold, design: .monospaced))
                            .foregroundStyle(WxTheme.snwCyan)
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 32)
                }

                // 3. Drought Outlook Card
                droughtCard

                // Footer Citation
                HStack {
                    Text("NOAA CPC · 6–10 DAY & 8–14 DAY NUMERICAL OUTLOOKS · CLIMATE TEST BED")
                        .font(.system(size: 8, weight: .medium, design: .monospaced))
                        .foregroundStyle(WxTheme.snwSilver.opacity(0.6))
                    Spacer()
                    if let fetched = store.cpcPayload?.fetchedAt {
                        Text("FETCHED // \(fetched.prefix(19))")
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

    // ── 1. Synoptic Pattern Shift Heads-Up Card ───────────────────────────────────

    @ViewBuilder
    private var patternShiftCard: some View {
        let shift = store.cpcPayload?.patternShift
        let hasShift = shift?.hasShift ?? false

        SNWConsoleCard(
            title: "Synoptic Pattern Shift Heads-Up",
            tag: "NOAA.CPC.SHIFT",
            statusColor: hasShift ? WxTheme.snwAmber : WxTheme.snwCyan
        ) {
            VStack(alignment: .leading, spacing: 10) {
                if let shift, hasShift {
                    HStack(alignment: .top, spacing: 10) {
                        Image(systemName: "exclamationmark.bubble.fill")
                            .font(.system(size: 22))
                            .foregroundStyle(WxTheme.snwAmber)
                            .shadow(color: WxTheme.snwAmber.opacity(0.6), radius: 5)

                        VStack(alignment: .leading, spacing: 6) {
                            Text(shift.summary)
                                .font(.system(size: 12.5, weight: .semibold, design: .monospaced))
                                .foregroundStyle(WxTheme.text)
                                .fixedSize(horizontal: false, vertical: true)

                            HStack(spacing: 8) {
                                if let t = shift.tempShift, !t.isEmpty {
                                    shiftTag(
                                        label: "TEMP SHIFT",
                                        value: t.uppercased(),
                                        icon: t.lowercased().contains("cool") ? "snowflake" : "flame.fill",
                                        color: t.lowercased().contains("cool") ? WxTheme.snwCyan : WxTheme.snwAmber
                                    )
                                }
                                if let p = shift.precipShift, !p.isEmpty {
                                    shiftTag(
                                        label: "PRECIP SHIFT",
                                        value: p.uppercased(),
                                        icon: p.lowercased().contains("wet") ? "cloud.heavyrain.fill" : "sun.max.fill",
                                        color: p.lowercased().contains("wet") ? WxTheme.snwGreen : WxTheme.snwGold
                                    )
                                }
                                if let conf = shift.confidence, !conf.isEmpty {
                                    shiftTag(
                                        label: "CONFIDENCE",
                                        value: conf.uppercased(),
                                        icon: "gauge.with.needle.fill",
                                        color: WxTheme.snwSilver
                                    )
                                }
                            }
                        }
                    }
                } else {
                    HStack(spacing: 8) {
                        Image(systemName: "checkmark.shield.fill")
                            .font(.system(size: 16))
                            .foregroundStyle(WxTheme.snwGreen)
                        Text("No major regime transition detected. Persistent baseline climatology prevailing across the 6–14 day horizon.")
                            .font(.system(size: 10.5, weight: .medium, design: .monospaced))
                            .foregroundStyle(WxTheme.textSecondary)
                    }
                }
            }
        }
    }

    private func shiftTag(label: String, value: String, icon: String, color: Color) -> some View {
        HStack(spacing: 4) {
            Image(systemName: icon)
                .font(.system(size: 8.5))
                .foregroundStyle(color)
            Text("\(label): \(value)")
                .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                .foregroundStyle(color)
        }
        .padding(.horizontal, 6)
        .padding(.vertical, 3)
        .background(color.opacity(0.12), in: RoundedRectangle(cornerRadius: 4))
        .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(color.opacity(0.4), lineWidth: 0.8))
    }

    // ── 2. Horizon Outlook Card (6-10D / 8-14D) ───────────────────────────────────

    @ViewBuilder
    private func horizonCard(item: CPCOutlookItemDTO) -> some View {
        SNWConsoleCard(
            title: "\(item.horizon.uppercased()) OUTLOOK",
            tag: dateRangeString(start: item.startDate, end: item.endDate),
            statusColor: WxTheme.snwCyan
        ) {
            VStack(alignment: .leading, spacing: 12) {
                // Temperature Row
                outlookRow(
                    metric: "TEMPERATURE PROBABILITY",
                    category: item.tempCategory,
                    probability: item.tempProbability,
                    isTemperature: true
                )

                Divider().overlay(WxTheme.border.opacity(0.3))

                // Precipitation Row
                outlookRow(
                    metric: "PRECIPITATION PROBABILITY",
                    category: item.precipCategory,
                    probability: item.precipProbability,
                    isTemperature: false
                )
            }
        }
    }

    @ViewBuilder
    private func outlookRow(metric: String, category: String, probability: Double, isTemperature: Bool) -> some View {
        let (color, icon) = categoryStyle(category: category, isTemperature: isTemperature)

        VStack(alignment: .leading, spacing: 5) {
            HStack {
                HStack(spacing: 5) {
                    Image(systemName: icon)
                        .font(.system(size: 11))
                        .foregroundStyle(color)
                    Text(metric)
                        .font(.system(size: 9, weight: .bold, design: .monospaced))
                        .foregroundStyle(WxTheme.snwSilver)
                }

                Spacer()

                HStack(spacing: 6) {
                    Text(category.uppercased())
                        .font(.system(size: 10, weight: .bold, design: .monospaced))
                        .foregroundStyle(color)

                    if probability > 0 {
                        Text(String(format: "%.0f%% PROB", probability))
                            .font(.system(size: 9.5, weight: .semibold, design: .monospaced))
                            .foregroundStyle(WxTheme.text)
                            .padding(.horizontal, 6)
                            .padding(.vertical, 2)
                            .background(color.opacity(0.18), in: RoundedRectangle(cornerRadius: 3))
                    }
                }
            }

            // Visual probability meter bar (0% - 100%)
            GeometryReader { geo in
                ZStack(alignment: .leading) {
                    Capsule()
                        .fill(WxTheme.snwChassis.opacity(0.9))
                        .frame(height: 6)

                    let pct = min(1.0, max(0.0, probability / 100.0))
                    Capsule()
                        .fill(
                            LinearGradient(
                                colors: [color.opacity(0.6), color],
                                startPoint: .leading,
                                endPoint: .trailing
                            )
                        )
                        .frame(width: max(8, geo.size.width * pct), height: 6)
                        .shadow(color: color.opacity(0.6), radius: 2)
                }
            }
            .frame(height: 6)
        }
    }

    private func categoryStyle(category: String, isTemperature: Bool) -> (Color, String) {
        let c = category.lowercased()
        if isTemperature {
            if c.contains("above") {
                return (WxTheme.snwAmber, "thermometer.sun.fill")
            } else if c.contains("below") {
                return (WxTheme.snwCyan, "snowflake")
            } else {
                return (WxTheme.snwSilver, "equal")
            }
        } else {
            if c.contains("above") {
                return (WxTheme.snwGreen, "cloud.heavyrain.fill")
            } else if c.contains("below") {
                return (WxTheme.snwGold, "sun.max.fill")
            } else {
                return (WxTheme.snwSilver, "equal")
            }
        }
    }

    private func dateRangeString(start: String, end: String) -> String {
        let s = formatDate(start)
        let e = formatDate(end)
        if !s.isEmpty && !e.isEmpty {
            return "\(s) — \(e)"
        }
        return "HORIZON"
    }

    private func formatDate(_ iso: String) -> String {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime]
        if let d = formatter.date(from: iso) {
            let df = DateFormatter()
            df.dateFormat = "MMM d"
            return df.string(from: d).uppercased()
        }
        return String(iso.prefix(10))
    }

    // ── 3. Drought Outlook Card ───────────────────────────────────────────────────

    @ViewBuilder
    private var droughtCard: some View {
        let drought = store.cpcPayload?.drought
        let status = drought?.status ?? "No Drought"
        let isDrought = !status.lowercased().contains("no")

        SNWConsoleCard(
            title: "U.S. Monthly Drought Outlook",
            tag: drought?.target ?? "CPC.DROUGHT",
            statusColor: isDrought ? WxTheme.snwAmber : WxTheme.snwGreen
        ) {
            HStack(spacing: 10) {
                Image(systemName: isDrought ? "exclamationmark.triangle.fill" : "drop.fill")
                    .font(.system(size: 20))
                    .foregroundStyle(isDrought ? WxTheme.snwAmber : WxTheme.snwGreen)

                VStack(alignment: .leading, spacing: 3) {
                    Text(status.uppercased())
                        .font(.system(size: 11, weight: .bold, design: .monospaced))
                        .foregroundStyle(isDrought ? WxTheme.snwAmber : WxTheme.snwGreen)

                    Text("Official monthly assessment synthesized with soil moisture anomalies and Palmer Drought Severity Index (PDSI).")
                        .font(.system(size: 8.5, weight: .medium, design: .monospaced))
                        .foregroundStyle(WxTheme.snwSilver.opacity(0.8))
                }
            }
        }
    }
}
