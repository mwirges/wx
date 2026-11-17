import SwiftUI

struct SurfaceView: View {
    @EnvironmentObject private var store: IOSStore
    @ScaledMetric(relativeTo: .largeTitle) private var heroTempSize: CGFloat = 62

    var body: some View {
        ScrollView(showsIndicators: false) {
            VStack(alignment: .leading, spacing: 14) {
                if let error = store.surfaceError, !error.isEmpty {
                    HStack(spacing: 8) {
                        Image(systemName: "exclamationmark.triangle.fill")
                            .foregroundStyle(WxGlassTheme.conditionRed)
                        Text(error)
                            .font(.system(size: 13, weight: .medium))
                            .foregroundStyle(WxGlassTheme.conditionRed)
                            .accessibilityIdentifier("wx.error")
                    }
                    .padding(12)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .background(WxGlassTheme.conditionRed.opacity(0.14), in: RoundedRectangle(cornerRadius: 16, style: .continuous))
                    .overlay(
                        RoundedRectangle(cornerRadius: 16, style: .continuous)
                            .strokeBorder(WxGlassTheme.conditionRed.opacity(0.45), lineWidth: 1)
                    )
                }

                heroObservation
                alertRibbon
                nowcastCard
                hourlyStrip
                dayList
            }
            .padding(.horizontal, 14)
            .padding(.top, 6)
            .padding(.bottom, 8)
            .accessibilityElement(children: .contain)
        }
        .wxDockScrollClearance()
        .accessibilityIdentifier("surface.root")
    }

    // MARK: - Open-Sky Atmospheric Hero

    private var heroObservation: some View {
        let conditions = store.payload?.conditions
        let air = conditions?.airQuality ?? store.payload?.airQuality
        let hiLo = todayHighLow
        let isNight = ConditionSymbol.isNight(for: store.payload)
        let heroSymbol = ConditionSymbol.resolvedSystemName(for: store.payload, nowcast: resolvedNowcast)
        let heroTint = ConditionSymbol.tint(for: heroSymbol, isNight: isNight)
        let resolvedDesc = ConditionSymbol.resolvedDescription(for: store.payload)
        let locationLabel = (conditions?.location?.trimmingCharacters(in: .whitespacesAndNewlines)).flatMap { $0.isEmpty ? nil : $0 } ?? store.place

        return VStack(spacing: 8) {
            HStack(alignment: .center, spacing: 12) {
                Image(systemName: heroSymbol)
                    .font(.system(size: 40, weight: .medium))
                    .symbolRenderingMode(.hierarchical)
                    .foregroundStyle(heroTint)
                    .shadow(color: heroTint.opacity(0.45), radius: 10)

                Text(IOSFormat.observation(conditions, units: store.units))
                    .font(.system(size: min(heroTempSize, 76), weight: .bold, design: .rounded))
                    .foregroundStyle(WxGlassTheme.textPrimary)
                    .shadow(color: Color.black.opacity(0.28), radius: 8, y: 3)
                    .accessibilityIdentifier("surface.temp")
            }

            if let description = resolvedDesc, !description.isEmpty {
                Text(description)
                    .font(.system(size: 16, weight: .medium, design: .rounded))
                    .foregroundStyle(WxGlassTheme.textSecondary)
                    .accessibilityIdentifier("surface.condition")
            }

            // High / Low & Feels Like Sub-Line
            HStack(spacing: 10) {
                if let hi = hiLo.high, let lo = hiLo.low {
                    Text("H:\(hi)  L:\(lo)")
                        .font(.system(size: 13.5, weight: .semibold, design: .rounded))
                        .foregroundStyle(WxGlassTheme.textPrimary)
                } else if let lo = hiLo.low {
                    Text("Low \(lo)")
                        .font(.system(size: 13.5, weight: .semibold, design: .rounded))
                        .foregroundStyle(WxGlassTheme.textPrimary)
                } else if let hi = hiLo.high {
                    Text("High \(hi)")
                        .font(.system(size: 13.5, weight: .semibold, design: .rounded))
                        .foregroundStyle(WxGlassTheme.textPrimary)
                }
                if let feels = feelsLikeString(conditions) {
                    Text("Feels \(feels)")
                        .font(.system(size: 13.5, weight: .medium, design: .rounded))
                        .foregroundStyle(WxGlassTheme.textSecondary)
                }
            }

            // Quick Atmospheric Sensor Pills
            HStack(spacing: 8) {
                if let windText = quickWind(conditions) {
                    quickHeroPill(symbol: "wind", text: windText, tint: WxGlassTheme.cyan)
                }
                if let hum = conditions?.humidityPct {
                    quickHeroPill(symbol: "drop.fill", text: String(format: "%.0f%%", hum), tint: WxGlassTheme.cyanSoft)
                }
                if let aqi = air?.aqi {
                    quickHeroPill(symbol: "aqi.medium", text: "AQI \(aqi)", tint: WxTheme.aqiColor(aqi))
                }
            }
            .padding(.top, 2)
        }
        .frame(maxWidth: .infinity)
        .padding(.top, 4)
        .padding(.bottom, 8)
        .background(alignment: .top) {
            Text(locationLabel)
                .font(.system(size: 1))
                .foregroundStyle(.clear)
                .allowsHitTesting(false)
                .accessibilityIdentifier("surface.location")
                .accessibilityLabel(locationLabel)
        }
    }

    private func quickHeroPill(symbol: String, text: String, tint: Color) -> some View {
        HStack(spacing: 4) {
            Image(systemName: symbol)
                .font(.system(size: 10.5, weight: .semibold))
                .foregroundStyle(tint)
            Text(text)
                .font(.system(size: 11.5, weight: .semibold, design: .rounded))
                .foregroundStyle(WxGlassTheme.textPrimary)
                .lineLimit(1)
                .minimumScaleFactor(0.80)
        }
        .padding(.horizontal, 10)
        .padding(.vertical, 5)
        .background(Color.white.opacity(0.08), in: Capsule())
        .overlay(Capsule().strokeBorder(Color.white.opacity(0.14), lineWidth: 0.8))
    }

    // MARK: - Sculpted Alert Ribbon (Jumps to Severe Tab)

    @ViewBuilder
    private var alertRibbon: some View {
        let alerts = store.payload?.alerts ?? []
        if let lead = alerts.first {
            let isWarn = lead.isWarning
            let accent = isWarn ? WxGlassTheme.conditionRed : WxGlassTheme.gold
            Button {
                store.tab = .severe
            } label: {
                HStack(spacing: 10) {
                    Image(systemName: isWarn ? "exclamationmark.triangle.fill" : "bolt.shield.fill")
                        .font(.system(size: 15, weight: .bold))
                        .foregroundStyle(isWarn ? Color.white : accent)

                    VStack(alignment: .leading, spacing: 2) {
                        Text(lead.event)
                            .font(.system(size: 13.5, weight: .bold, design: .rounded))
                            .foregroundStyle(Color.white)
                            .lineLimit(1)
                        if let headline = lead.headline, !headline.isEmpty {
                            Text(headline)
                                .font(.system(size: 11.5, weight: .medium))
                                .foregroundStyle(Color.white.opacity(0.85))
                                .lineLimit(1)
                        }
                    }

                    Spacer(minLength: 4)

                    if alerts.count > 1 {
                        Text("+\(alerts.count - 1)")
                            .font(.system(size: 11, weight: .bold, design: .monospaced))
                            .foregroundStyle(Color.white)
                            .padding(.horizontal, 6)
                            .padding(.vertical, 2)
                            .background(Color.black.opacity(0.25), in: Capsule())
                    }

                    Image(systemName: "chevron.right")
                        .font(.system(size: 11, weight: .bold))
                        .foregroundStyle(Color.white.opacity(0.85))
                }
                .padding(.horizontal, 14)
                .padding(.vertical, 11)
                .background {
                    ZStack {
                        RoundedRectangle(cornerRadius: 18, style: .continuous).fill(.ultraThinMaterial)
                        RoundedRectangle(cornerRadius: 18, style: .continuous)
                            .fill(
                                LinearGradient(
                                    colors: isWarn
                                        ? [WxGlassTheme.conditionRed.opacity(0.86), WxGlassTheme.conditionRed.opacity(0.58)]
                                        : [WxGlassTheme.gold.opacity(0.32), WxGlassTheme.amber.opacity(0.18)],
                                    startPoint: .topLeading,
                                    endPoint: .bottomTrailing
                                )
                            )
                    }
                }
                .overlay(
                    RoundedRectangle(cornerRadius: 18, style: .continuous)
                        .strokeBorder(isWarn ? Color.white.opacity(0.35) : accent.opacity(0.65), lineWidth: 1)
                )
                .shadow(color: accent.opacity(0.32), radius: 12, y: 4)
            }
            .buttonStyle(.plain)
        }
    }

    // MARK: - Precipitation Nowcast Card (With 15-Minute Capsule Spark-Bars)

    /// Phase "none" uppercases to NONE, which reads as missing data next to a
    /// real dry headline such as "Clear". Hide that sentinel.
    private var nowcastPhaseTag: String? {
        let phase = store.nowcast?.nowcast?.primaryPhase?
            .trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if phase.isEmpty || phase.caseInsensitiveCompare("none") == .orderedSame {
            return nil
        }
        return phase
    }

    private var resolvedNowcast: NowcastDTO? {
        store.nowcast?.nowcast ?? store.payload?.conditions?.nowcast ?? store.payload?.nowcast
    }

    private var nowcastCard: some View {
        let nc = resolvedNowcast
        let intervals = nc?.intervals ?? []
        let isActive = nc?.isActivePrecip == true
        let hasPrecipSignal = isActive
            || nc?.nextPrecipTime != nil
            || (nc?.totalLiquidMm ?? 0) > 0.05
            || intervals.contains {
                ($0.rateInh ?? 0) > 0.001 || ($0.rateMmh ?? 0) > 0.02 || ($0.probability ?? 0) >= 15
            }
        let accent = isActive ? WxGlassTheme.cyan : WxGlassTheme.gold

        return WxGlassCard(title: "Nowcast", tag: nowcastPhaseTag, accent: accent) {
            VStack(alignment: .leading, spacing: 8) {
                Text(store.nowcastHeadline ?? "—")
                    .font(.system(size: 16, weight: .semibold, design: .rounded))
                    .foregroundStyle(WxGlassTheme.textPrimary)
                    .accessibilityIdentifier("surface.nowcast")

                if let summary = nc?.summary, !summary.isEmpty {
                    Text(summary)
                        .font(.system(size: 13))
                        .foregroundStyle(WxGlassTheme.textSecondary)
                        .fixedSize(horizontal: false, vertical: true)
                }

                if !intervals.isEmpty && hasPrecipSignal {
                    IOSNowcastBarsView(intervals: Array(intervals.prefix(24)))
                        .padding(.top, 2)
                }
            }
        }
    }

    // MARK: - Hourly Strip (Promoted Above Multi-Day Forecast)

    private var hourlyStrip: some View {
        let hours = store.payload?.forecast?.activeHourlyPeriods() ?? []
        return WxGlassCard(title: "Hourly Forecast", tag: hours.isEmpty ? nil : "\(min(hours.count, 24))H", accent: WxGlassTheme.cyan) {
            if hours.isEmpty {
                Text("No hourly periods")
                    .font(.system(size: 13))
                    .foregroundStyle(WxGlassTheme.textSecondary)
            } else {
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: 8) {
                        ForEach(hours) { period in
                            VStack(spacing: 6) {
                                Text(IOSFormat.hourLabel(period))
                                    .font(.system(size: 11.5, weight: .semibold, design: .rounded))
                                    .foregroundStyle(WxGlassTheme.textSecondary)

                                Image(systemName: ConditionSymbol.periodSymbol(for: period))
                                    .font(.system(size: 17, weight: .medium))
                                    .symbolRenderingMode(.hierarchical)
                                    .foregroundStyle(hourIconTint(period))
                                    .frame(height: 20)

                                if let pop = period.probabilityOfPrecipitation, pop >= 5 {
                                    Text(String(format: "%.0f%%", pop))
                                        .font(.system(size: 10, weight: .bold, design: .rounded))
                                        .foregroundStyle(WxGlassTheme.cyan)
                                } else {
                                    Color.clear.frame(height: 12)
                                }

                                Text(IOSFormat.period(period, units: store.units))
                                    .font(.system(size: 15, weight: .bold, design: .rounded))
                                    .foregroundStyle(WxGlassTheme.textPrimary)
                            }
                            .frame(minWidth: 54)
                            .padding(.vertical, 9)
                            .padding(.horizontal, 6)
                            .background(Color.white.opacity(0.05), in: WxGlassTheme.nestedShape())
                            .overlay(
                                WxGlassTheme.nestedShape()
                                    .strokeBorder(Color.white.opacity(0.10), lineWidth: 0.7)
                            )
                        }
                    }
                }
            }
        }
        .accessibilityIdentifier("surface.hourly")
    }

    private func hourIconTint(_ period: Period) -> Color {
        let sym = ConditionSymbol.periodSymbol(for: period)
        return ConditionSymbol.tint(for: sym, isNight: !period.resolvedIsDaytime)
    }

    // MARK: - 7-Day Synoptic Forecast With Thermal Range Bars

    private var dayList: some View {
        let rows = store.payload?.forecast?.synopticRows(units: store.units) ?? []
        let allTemps = rows.flatMap { [$0.lowValue, $0.highValue].compactMap { $0 } }
        let weekMin = allTemps.min() ?? 40
        let weekMax = max(weekMin + 1, allTemps.max() ?? 80)

        return WxGlassCard(title: "7-Day Forecast", tag: "NWS", accent: WxGlassTheme.cyan) {
            if rows.isEmpty {
                Text("No day periods")
                    .font(.system(size: 13))
                    .foregroundStyle(WxGlassTheme.textSecondary)
            } else {
                VStack(alignment: .leading, spacing: 0) {
                    ForEach(Array(rows.enumerated()), id: \.element.id) { idx, row in
                        VStack(alignment: .leading, spacing: 4) {
                            HStack(spacing: 6) {
                                Text(row.name)
                                    .font(.system(size: 14, weight: .semibold, design: .rounded))
                                    .foregroundStyle(WxGlassTheme.textPrimary)
                                    .lineLimit(1)
                                    .minimumScaleFactor(0.82)
                                    .layoutPriority(1)
                                    .frame(maxWidth: .infinity, alignment: .leading)

                                HStack(spacing: 3) {
                                    Image(systemName: ConditionSymbol.periodSymbol(for: row.primaryPeriod))
                                        .font(.system(size: 14, weight: .medium))
                                        .symbolRenderingMode(.hierarchical)
                                        .foregroundStyle(hourIconTint(row.primaryPeriod))
                                    if let pop = row.precipPct, pop >= 10 {
                                        Text(String(format: "%.0f%%", pop))
                                            .font(.system(size: 10, weight: .bold, design: .rounded))
                                            .foregroundStyle(WxGlassTheme.cyan)
                                    }
                                }
                                .frame(width: 44, alignment: .leading)

                                switch row.kind {
                                case .paired:
                                    Text(row.lowText ?? "—")
                                        .font(.system(size: 13.5, weight: .semibold, design: .rounded))
                                        .foregroundStyle(WxGlassTheme.textMuted)
                                        .frame(width: 32, alignment: .trailing)

                                    if let lo = row.lowValue, let hi = row.highValue {
                                        WxThermalRangeBar(
                                            dayLow: min(lo, hi),
                                            dayHigh: max(lo, hi),
                                            weekMin: weekMin,
                                            weekMax: weekMax
                                        )
                                        .frame(width: 56)
                                    } else {
                                        Color.clear.frame(width: 56, height: 1)
                                    }

                                    Text(row.highText ?? "—")
                                        .font(.system(size: 14, weight: .bold, design: .rounded))
                                        .foregroundStyle(WxGlassTheme.textPrimary)
                                        .frame(width: 34, alignment: .trailing)

                                case .nightOnly:
                                    HStack(spacing: 5) {
                                        Text("Low")
                                            .font(.system(size: 11.5, weight: .semibold, design: .rounded))
                                            .foregroundStyle(WxGlassTheme.periwinkle)
                                        Text(row.lowText ?? "—")
                                            .font(.system(size: 14, weight: .bold, design: .rounded))
                                            .foregroundStyle(WxGlassTheme.textPrimary)
                                    }
                                    .padding(.horizontal, 10)
                                    .padding(.vertical, 3)
                                    .background(WxGlassTheme.periwinkle.opacity(0.12), in: Capsule())
                                    .overlay(Capsule().strokeBorder(WxGlassTheme.periwinkle.opacity(0.32), lineWidth: 0.8))
                                    .frame(width: 134, alignment: .trailing)

                                case .dayOnly:
                                    HStack(spacing: 5) {
                                        Text("High")
                                            .font(.system(size: 11.5, weight: .semibold, design: .rounded))
                                            .foregroundStyle(WxGlassTheme.gold)
                                        Text(row.highText ?? "—")
                                            .font(.system(size: 14, weight: .bold, design: .rounded))
                                            .foregroundStyle(WxGlassTheme.textPrimary)
                                    }
                                    .padding(.horizontal, 10)
                                    .padding(.vertical, 3)
                                    .background(WxGlassTheme.gold.opacity(0.12), in: Capsule())
                                    .overlay(Capsule().strokeBorder(WxGlassTheme.gold.opacity(0.32), lineWidth: 0.8))
                                    .frame(width: 134, alignment: .trailing)
                                }
                            }

                            if let blurb = row.shortDescription, !blurb.isEmpty {
                                Text(blurb)
                                    .font(.system(size: 12))
                                    .foregroundStyle(WxGlassTheme.textMuted)
                                    .lineLimit(1)
                            }
                        }
                        .padding(.vertical, 8)

                        if idx < rows.count - 1 {
                            Divider()
                                .overlay(Color.white.opacity(0.09))
                        }
                    }
                }
            }
        }
        .accessibilityIdentifier("surface.days")
    }

    // MARK: - Formatting Helpers

    private var todayHighLow: (high: String?, low: String?) {
        let hiLo = store.payload?.forecast?.leadHighLow(units: store.units) ?? (highText: nil, lowText: nil)
        return (hiLo.highText, hiLo.lowText)
    }

    private func feelsLikeString(_ c: Conditions?) -> String? {
        guard let c else { return nil }
        let formatted = IOSFormat.temperature(f: c.feelsLikeF, c: c.feelsLikeC, units: store.units)
        return formatted == "—" ? nil : formatted
    }

    private func quickWind(_ c: Conditions?) -> String? {
        guard let c else { return nil }
        let dir = c.windDirection.map { " \($0)" } ?? ""
        if store.units == "metric", let kph = c.windKph {
            return String(format: "%.0f km/h%@", kph, dir)
        }
        if let mph = c.windMph {
            return String(format: "%.0f mph%@", mph, dir)
        }
        return nil
    }
}

// MARK: - 15-Minute Nowcast Capsule Spark-Bars (iOS)

struct IOSNowcastBarsView: View {
    let intervals: [PrecipIntervalDTO]

    private var maxRate: Double {
        max(0.20, intervals.map { $0.rateInh ?? 0 }.max() ?? 0)
    }

    var body: some View {
        VStack(spacing: 5) {
            GeometryReader { geo in
                let count = max(1, intervals.count)
                let spacing: CGFloat = geo.size.width < 220 ? 2.0 : 3.0
                let barWidth = max(2.0, (geo.size.width - CGFloat(count - 1) * spacing) / CGFloat(count))
                HStack(alignment: .bottom, spacing: spacing) {
                    ForEach(intervals) { iv in
                        let rate = iv.rateInh ?? 0
                        let ratio = min(1.0, rate / maxRate)
                        let height = max(4.0, CGFloat(ratio) * geo.size.height)
                        Capsule(style: .continuous)
                            .fill(barTint(rate: rate, phase: iv.phase))
                            .frame(width: barWidth, height: height)
                            .shadow(color: rate > 0.03 ? WxGlassTheme.cyan.opacity(0.45) : .clear, radius: 3)
                    }
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .bottomLeading)
            }
            .frame(height: 36)

            HStack {
                if let first = intervals.first {
                    Text(first.formattedTime)
                }
                Spacer()
                if intervals.count > 6 {
                    Text(intervals[intervals.count / 2].formattedTime)
                }
                Spacer()
                if let last = intervals.last {
                    Text(last.formattedTime)
                }
            }
            .font(.system(size: 10, weight: .medium, design: .monospaced))
            .foregroundStyle(WxGlassTheme.textMuted)
        }
        .padding(10)
        .background(Color.white.opacity(0.05), in: WxGlassTheme.nestedShape())
        .overlay(WxGlassTheme.nestedShape().strokeBorder(Color.white.opacity(0.10), lineWidth: 0.7))
    }

    private func barTint(rate: Double, phase: String?) -> Color {
        if (phase ?? "").lowercased() == "snow" {
            return Color.white
        }
        if rate >= 0.35 {
            return WxGlassTheme.conditionRed
        }
        if rate >= 0.15 {
            return WxGlassTheme.gold
        }
        if rate >= 0.02 {
            return WxGlassTheme.cyan
        }
        return Color.white.opacity(0.14)
    }
}
