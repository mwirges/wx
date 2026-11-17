import SwiftUI
import Charts

struct ClimateTrendsView: View {
    @EnvironmentObject var store: WeatherStore
    @State private var selectedRange: Int = 14

    private var isMetric: Bool {
        store.units == "metric"
    }

    private var tempUnit: String {
        isMetric ? "°C" : "°F"
    }

    private var precipUnit: String {
        isMetric ? "mm" : "in"
    }

    private var windUnit: String {
        isMetric ? "km/h" : "mph"
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 14) {
                // Header & Range Control
                headerBar

                // NOAA 30-Year Normals & Daily Historical Records
                if let climate = store.climatePayload?.climate {
                    climateNormalsCard(climate)
                } else if store.isClimateLoading {
                    climateLoadingCard
                }

                if store.isHistoryLoading && store.historyPayload == nil {
                    loadingView
                } else if let err = store.historyErrorMessage {
                    errorView(err)
                } else if let payload = store.historyPayload {
                    // 1. Executive Telemetry Summary Cards
                    summaryCards(payload.summary)

                    // 2. High/Low Temperature Envelope Chart
                    temperatureEnvelopeCard(payload.days, summary: payload.summary)

                    // 3. Precipitation Totals Bar Chart
                    precipitationCard(payload.days, summary: payload.summary)

                    // 4. Detailed Daily Observation Table
                    observationsTable(payload.days)
                } else {
                    emptyStateView
                }
            }
            .padding(14)
        }
        .onAppear {
            if store.historyPayload == nil && !store.isHistoryLoading {
                Task { await store.refreshHistory() }
            }
            if store.climatePayload == nil && !store.isClimateLoading {
                Task { await store.refreshClimate() }
            }
        }
    }

    // ── Header & Range Bar ──────────────────────────────────────────────────────────

    @ViewBuilder
    private var headerBar: some View {
        HStack(spacing: 8) {
            VStack(alignment: .leading, spacing: 3) {
                HStack(spacing: 6) {
                    Image(systemName: "calendar.day.timeline.left")
                        .font(.system(size: 11, weight: .bold))
                        .foregroundStyle(WxTheme.snwCyan)
                    Text("CLIMATOLOGICAL OBSERVATION ARCHIVE")
                        .font(.system(size: 10.5, weight: .bold, design: .monospaced))
                        .tracking(0.5)
                        .foregroundStyle(WxTheme.text)
                }
                if let loc = store.historyPayload?.location ?? store.locationInput.nilIfEmpty {
                    Text(loc.uppercased())
                        .font(.system(size: 9, design: .monospaced))
                        .foregroundStyle(WxTheme.snwSilver)
                }
            }

            Spacer()

            // Range Segmented Capsule
            HStack(spacing: 2) {
                rangeButton(days: 7, label: "7 Days")
                rangeButton(days: 14, label: "14 Days")
                rangeButton(days: 30, label: "30 Days")
            }
            .padding(3)
            .background(Color.black.opacity(0.28), in: Capsule())
            .overlay(Capsule().strokeBorder(Color.white.opacity(0.16), lineWidth: 0.8))

            Button {
                Task {
                    await store.refreshHistory()
                    await store.refreshClimate()
                }
            } label: {
                HStack(spacing: 4) {
                    if store.isHistoryLoading || store.isClimateLoading {
                        ProgressView().controlSize(.small).tint(WxGlassTheme.cyan)
                    } else {
                        Image(systemName: "arrow.triangle.2.circlepath")
                    }
                    Text("Sync")
                }
                .font(.system(size: 10.5, weight: .semibold, design: .rounded))
                .foregroundStyle(WxGlassTheme.cyan)
                .padding(.horizontal, 12)
                .padding(.vertical, 5.5)
                .wxFrostedCapsule(accent: WxGlassTheme.cyan, active: true)
            }
            .buttonStyle(.plain)
            .disabled(store.isHistoryLoading || store.isClimateLoading)
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

    private func rangeButton(days: Int, label: String) -> some View {
        Button {
            selectedRange = days
            store.historyDaysCount = days
            Task { await store.refreshHistory() }
        } label: {
            Text(label)
                .font(.system(size: 10, weight: .semibold, design: .rounded))
                .padding(.horizontal, 10)
                .padding(.vertical, 4.5)
                .background(
                    selectedRange == days ? WxGlassTheme.cyan.opacity(0.24) : Color.clear,
                    in: Capsule()
                )
                .overlay(
                    Capsule()
                        .strokeBorder(selectedRange == days ? WxGlassTheme.cyan.opacity(0.65) : Color.clear, lineWidth: 0.8)
                )
                .foregroundStyle(selectedRange == days ? WxGlassTheme.textPrimary : WxGlassTheme.textSecondary)
        }
        .buttonStyle(.plain)
    }

    // ── NOAA Climate Normals & Daily Historical Records ───────────────────────────

    @ViewBuilder
    private var climateLoadingCard: some View {
        HStack(spacing: 8) {
            ProgressView().controlSize(.small).tint(WxGlassTheme.amber)
            Text("Acquiring NOAA ACIS climate normals & historical extremes…")
                .font(.system(size: 11, weight: .semibold, design: .rounded))
                .foregroundStyle(WxGlassTheme.textSecondary)
            Spacer()
        }
        .padding(10)
        .background(Color.white.opacity(0.05), in: WxGlassTheme.nestedShape())
        .overlay(WxGlassTheme.nestedShape().strokeBorder(Color.white.opacity(0.12), lineWidth: 0.8))
    }

    @ViewBuilder
    private func climateNormalsCard(_ c: ClimateReportDTO) -> some View {
        WxGlassCard(
            title: "NOAA 30-Year Climate Normals & Records",
            tag: c.normalsPeriod ?? "1991–2020",
            accent: WxGlassTheme.amber
        ) {
            VStack(alignment: .leading, spacing: 10) {
                if let station = c.stationName {
                    Text(station)
                        .font(.system(size: 11, weight: .semibold, design: .rounded))
                        .foregroundStyle(WxGlassTheme.textSecondary)
                }

                if let dep = c.departure {
                    departureBanner(dep)
                }

                HStack(alignment: .top, spacing: 10) {
                    if let norm = c.todayNormals {
                        todayNormalsPanel(norm)
                    }
                    if let rec = c.records {
                        recordsPanel(rec)
                    }
                    if let monthly = c.monthlyNormals {
                        monthlyNormalsPanel(monthly)
                    }
                }
            }
        }
    }

    @ViewBuilder
    private func departureBanner(_ dep: ClimateDepartureDTO) -> some View {
        let val = isMetric ? dep.departureCurrentC : dep.departureCurrentF
        let badgeColor: Color = {
            guard let v = val else { return WxTheme.snwGreen }
            let thresh = isMetric ? 1.7 : 3.0
            if v >= thresh { return WxTheme.snwRed }
            if v <= -thresh { return WxTheme.snwCyan }
            return WxTheme.snwGreen
        }()

        HStack(spacing: 8) {
            Image(systemName: "thermometer.transmission")
                .font(.system(size: 10, weight: .bold))
                .foregroundStyle(badgeColor)

            Text(dep.summary ?? "Climatological Departure Nominal")
                .font(.system(size: 9.5, weight: .bold, design: .monospaced))
                .foregroundStyle(badgeColor)

            Spacer()

            if let obs = isMetric ? dep.observedCurrentC : dep.observedCurrentF {
                Text(String(format: "OBSERVED: %.1f%@", obs, tempUnit))
                    .font(.system(size: 8.5, weight: .semibold, design: .monospaced))
                    .foregroundStyle(WxTheme.textSecondary)
            }
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 7)
        .background(badgeColor.opacity(0.12), in: Capsule())
        .overlay(Capsule().strokeBorder(badgeColor.opacity(0.38), lineWidth: 0.8))
    }

    @ViewBuilder
    private func todayNormalsPanel(_ norm: DailyNormalsDTO) -> some View {
        let hi = isMetric ? norm.normalHighC : norm.normalHighF
        let lo = isMetric ? norm.normalLowC : norm.normalLowF
        let mean = isMetric ? norm.normalMeanC : norm.normalMeanF
        let pcpn = isMetric ? norm.normalPrecipMm : norm.normalPrecipIn

        VStack(alignment: .leading, spacing: 6) {
            Text("TODAY'S 30-YR NORMALS")
                .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                .foregroundStyle(WxTheme.snwAmber)

            Divider().overlay(WxTheme.border.opacity(0.3))

            climateStatRow(label: "NORMAL HIGH", value: hi.map { String(format: "%.0f%@", $0, tempUnit) } ?? "--", color: WxTheme.snwRed)
            climateStatRow(label: "NORMAL LOW", value: lo.map { String(format: "%.0f%@", $0, tempUnit) } ?? "--", color: WxTheme.snwCyan)
            climateStatRow(label: "NORMAL MEAN", value: mean.map { String(format: "%.1f%@", $0, tempUnit) } ?? "--", color: WxTheme.text)
            climateStatRow(label: "NORMAL PRECIP", value: pcpn.map { String(format: "%.2f %@", $0, precipUnit) } ?? "--", color: WxTheme.snwGreen)
        }
        .padding(10)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color.white.opacity(0.05), in: WxGlassTheme.nestedShape())
        .overlay(WxGlassTheme.nestedShape().strokeBorder(Color.white.opacity(0.10), lineWidth: 0.7))
    }

    @ViewBuilder
    private func recordsPanel(_ rec: DailyRecordsDTO) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack {
                Text("ALL-TIME DAILY RECORDS")
                    .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                    .foregroundStyle(WxTheme.snwGold)
                Spacer()
                if let sampled = rec.totalYearsSampled {
                    Text("\(sampled) YRS")
                        .font(.system(size: 7.5, weight: .bold, design: .monospaced))
                        .foregroundStyle(WxTheme.snwSilver)
                }
            }

            Divider().overlay(WxTheme.border.opacity(0.3))

            climateRecordRow(label: "RECORD HIGH", record: rec.recordHigh, isTemp: true, color: WxTheme.snwRed)
            climateRecordRow(label: "RECORD LOW", record: rec.recordLow, isTemp: true, color: WxTheme.snwCyan)
            climateRecordRow(label: "MAX PRECIP", record: rec.recordPrecip, isTemp: false, color: WxTheme.snwGreen)
            climateRecordRow(label: "COLDEST HIGH", record: rec.coldestHigh, isTemp: true, color: WxTheme.snwSilver)
            climateRecordRow(label: "WARMEST LOW", record: rec.warmestLow, isTemp: true, color: WxTheme.snwSilver)
        }
        .padding(10)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color.white.opacity(0.05), in: WxGlassTheme.nestedShape())
        .overlay(WxGlassTheme.nestedShape().strokeBorder(Color.white.opacity(0.10), lineWidth: 0.7))
    }

    @ViewBuilder
    private func monthlyNormalsPanel(_ monthly: MonthlyNormalsDTO) -> some View {
        let hi = isMetric ? monthly.normalAvgHighC : monthly.normalAvgHighF
        let lo = isMetric ? monthly.normalAvgLowC : monthly.normalAvgLowF
        let pcpn = isMetric ? monthly.normalTotalPrecipMm : monthly.normalTotalPrecipIn
        let month = (monthly.monthName ?? "MONTH").uppercased()

        VStack(alignment: .leading, spacing: 6) {
            Text("\(month) NORMALS")
                .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                .foregroundStyle(WxTheme.snwCyan)

            Divider().overlay(WxTheme.border.opacity(0.3))

            climateStatRow(label: "AVG HIGH", value: hi.map { String(format: "%.1f%@", $0, tempUnit) } ?? "--", color: WxTheme.snwRed)
            climateStatRow(label: "AVG LOW", value: lo.map { String(format: "%.1f%@", $0, tempUnit) } ?? "--", color: WxTheme.snwCyan)
            climateStatRow(label: "TOTAL PRECIP", value: pcpn.map { String(format: "%.2f %@", $0, precipUnit) } ?? "--", color: WxTheme.snwGreen)
            climateStatRow(label: "STATUS", value: "30-YR BASELINE", color: WxTheme.snwSilver)
        }
        .padding(10)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color.white.opacity(0.05), in: WxGlassTheme.nestedShape())
        .overlay(WxGlassTheme.nestedShape().strokeBorder(Color.white.opacity(0.10), lineWidth: 0.7))
    }

    private func climateStatRow(label: String, value: String, color: Color) -> some View {
        HStack {
            Text(label)
                .font(.system(size: 8, weight: .medium, design: .monospaced))
                .foregroundStyle(WxTheme.snwSilver)
            Spacer()
            Text(value)
                .font(.system(size: 9, weight: .bold, design: .monospaced))
                .foregroundStyle(color)
        }
    }

    private func climateRecordRow(label: String, record: DailyRecordDTO?, isTemp: Bool, color: Color) -> some View {
        let valStr: String = {
            guard let r = record else { return "--" }
            if isTemp {
                let v = isMetric ? r.valueC : r.valueF
                return v.map { String(format: "%.0f%@", $0, tempUnit) } ?? "--"
            } else {
                let v = isMetric ? r.valueMm : r.valueIn
                return v.map { String(format: "%.2f %@", $0, precipUnit) } ?? "--"
            }
        }()
        let yrsStr: String = {
            guard let yrs = record?.years, !yrs.isEmpty else { return "" }
            return "(" + yrs.map { String($0) }.joined(separator: ", ") + ")"
        }()

        return HStack(spacing: 4) {
            Text(label)
                .font(.system(size: 8, weight: .medium, design: .monospaced))
                .foregroundStyle(WxTheme.snwSilver)
            Spacer()
            Text(valStr)
                .font(.system(size: 9, weight: .bold, design: .monospaced))
                .foregroundStyle(color)
            if !yrsStr.isEmpty {
                Text(yrsStr)
                    .font(.system(size: 7.5, design: .monospaced))
                    .foregroundStyle(WxTheme.textSecondary)
            }
        }
    }

    // ── Executive Summary KPI Cards ───────────────────────────────────────────────

    @ViewBuilder
    private func summaryCards(_ s: HistorySummaryDTO) -> some View {
        let avgHigh = isMetric ? s.avgTempMaxC : s.avgTempMaxF
        let avgLow = isMetric ? s.avgTempMinC : s.avgTempMinF
        let totalP = isMetric ? s.totalPrecipMm : s.totalPrecipIn
        let maxW = isMetric ? s.maxWindKph : s.maxWindMph

        LazyVGrid(columns: [GridItem(.adaptive(minimum: 150), spacing: 10)], spacing: 10) {
            kpiCard(
                title: "MEAN HIGH TEMP",
                value: avgHigh.map { String(format: "%.1f%@", $0, tempUnit) } ?? "--",
                icon: "thermometer.sun.fill",
                accent: WxTheme.snwRed
            )
            kpiCard(
                title: "MEAN LOW TEMP",
                value: avgLow.map { String(format: "%.1f%@", $0, tempUnit) } ?? "--",
                icon: "thermometer.snowflake",
                accent: WxTheme.snwCyan
            )
            kpiCard(
                title: "TOTAL PRECIP",
                value: totalP.map { String(format: "%.2f %@", $0, precipUnit) } ?? "--",
                icon: "drop.fill",
                accent: WxTheme.snwGreen
            )
            kpiCard(
                title: "PEAK GUST",
                value: maxW.map { String(format: "%.0f %@", $0, windUnit) } ?? "--",
                icon: "wind",
                accent: WxTheme.snwAmber
            )
        }
    }

    private func kpiCard(title: String, value: String, icon: String, accent: Color) -> some View {
        WxSensorCapsule(label: title, value: value, symbol: icon, accent: accent)
    }

    // ── Chart 1: Temperature High/Low Range Envelope ──────────────────────────────

    @ViewBuilder
    private func temperatureEnvelopeCard(_ days: [HistoryDayDTO], summary: HistorySummaryDTO) -> some View {
        SNWConsoleCard(title: "Thermal Range Envelope (Daily High / Low)", tag: "CLIM.TEMP") {
            VStack(alignment: .leading, spacing: 8) {
                HStack(spacing: 12) {
                    legendItem(color: WxTheme.snwRed, label: "HIGH")
                    legendItem(color: WxTheme.snwCyan, label: "LOW")
                    legendItem(color: WxTheme.snwCyan.opacity(0.2), label: "ENVELOPE")
                    Spacer()
                    if let avgH = (isMetric ? summary.avgTempMaxC : summary.avgTempMaxF) {
                        Text(String(format: "AVG MAX: %.1f%@", avgH, tempUnit))
                            .font(.system(size: 8.5, weight: .medium, design: .monospaced))
                            .foregroundStyle(WxTheme.snwRed.opacity(0.85))
                    }
                }
                .padding(.bottom, 4)

                Chart {
                    ForEach(days) { d in
                        let dateLabel = formatChartDate(d.date)
                        let high = isMetric ? d.tempMaxC : d.tempMaxF
                        let low = isMetric ? d.tempMinC : d.tempMinF

                        if let h = high, let l = low {
                            AreaMark(
                                x: .value("Date", dateLabel),
                                yStart: .value("Low", l),
                                yEnd: .value("High", h)
                            )
                            .interpolationMethod(.catmullRom)
                            .foregroundStyle(
                                LinearGradient(
                                    colors: [WxGlassTheme.conditionRed.opacity(0.26), WxGlassTheme.cyan.opacity(0.16)],
                                    startPoint: .top,
                                    endPoint: .bottom
                                )
                            )

                            LineMark(
                                x: .value("Date", dateLabel),
                                y: .value("High", h)
                            )
                            .interpolationMethod(.catmullRom)
                            .foregroundStyle(WxGlassTheme.conditionRed)
                            .lineStyle(StrokeStyle(lineWidth: 2.2))

                            PointMark(
                                x: .value("Date", dateLabel),
                                y: .value("High", h)
                            )
                            .foregroundStyle(WxGlassTheme.conditionRed)
                            .symbolSize(18)

                            LineMark(
                                x: .value("Date", dateLabel),
                                y: .value("Low", l)
                            )
                            .interpolationMethod(.catmullRom)
                            .foregroundStyle(WxGlassTheme.cyan)
                            .lineStyle(StrokeStyle(lineWidth: 2.2))

                            PointMark(
                                x: .value("Date", dateLabel),
                                y: .value("Low", l)
                            )
                            .foregroundStyle(WxTheme.snwCyan)
                            .symbolSize(18)
                        }
                    }
                }
                .chartYAxis {
                    AxisMarks(position: .leading) { value in
                        AxisGridLine(stroke: StrokeStyle(lineWidth: 0.5, dash: [2, 3]))
                            .foregroundStyle(WxTheme.border.opacity(0.4))
                        AxisTick(stroke: StrokeStyle(lineWidth: 0.5))
                            .foregroundStyle(WxTheme.border)
                        AxisValueLabel {
                            if let doubleValue = value.as(Double.self) {
                                Text("\(Int(round(doubleValue)))°")
                                    .font(.system(size: 8, design: .monospaced))
                                    .foregroundStyle(WxTheme.snwSilver)
                            }
                        }
                    }
                }
                .chartXAxis {
                    AxisMarks(values: .automatic(desiredCount: min(days.count, 8))) { _ in
                        AxisGridLine(stroke: StrokeStyle(lineWidth: 0.5, dash: [2, 3]))
                            .foregroundStyle(WxTheme.border.opacity(0.3))
                        AxisTick(stroke: StrokeStyle(lineWidth: 0.5))
                            .foregroundStyle(WxTheme.border)
                        AxisValueLabel()
                            .font(.system(size: 8, design: .monospaced))
                            .foregroundStyle(WxTheme.snwSilver)
                    }
                }
                .frame(height: 180)
            }
        }
    }

    // ── Chart 2: Daily Precipitation Totals ───────────────────────────────────────

    @ViewBuilder
    private func precipitationCard(_ days: [HistoryDayDTO], summary: HistorySummaryDTO) -> some View {
        SNWConsoleCard(title: "Hydrologic Accumulation (Precipitation Totals)", tag: "CLIM.PRECIP") {
            VStack(alignment: .leading, spacing: 8) {
                HStack {
                    legendItem(color: WxTheme.snwGreen, label: "PRECIPITATION (\(precipUnit.uppercased()))")
                    Spacer()
                    if let tot = (isMetric ? summary.totalPrecipMm : summary.totalPrecipIn) {
                        Text(String(format: "TOTAL: %.2f %@", tot, precipUnit))
                            .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                            .foregroundStyle(WxTheme.snwGreen)
                    }
                }
                .padding(.bottom, 4)

                Chart {
                    ForEach(days) { d in
                        let dateLabel = formatChartDate(d.date)
                        let precip = (isMetric ? d.precipMm : d.precipIn) ?? 0

                        BarMark(
                            x: .value("Date", dateLabel),
                            y: .value("Precipitation", precip)
                        )
                        .foregroundStyle(
                            LinearGradient(
                                colors: [WxGlassTheme.nominalGreen, WxGlassTheme.cyan.opacity(0.85)],
                                startPoint: .top,
                                endPoint: .bottom
                            )
                        )
                        .cornerRadius(6)
                    }
                }
                .chartYAxis {
                    AxisMarks(position: .leading) { value in
                        AxisGridLine(stroke: StrokeStyle(lineWidth: 0.5, dash: [2, 3]))
                            .foregroundStyle(WxTheme.border.opacity(0.4))
                        AxisTick(stroke: StrokeStyle(lineWidth: 0.5))
                            .foregroundStyle(WxTheme.border)
                        AxisValueLabel {
                            if let doubleValue = value.as(Double.self) {
                                Text(String(format: isMetric ? "%.0f" : "%.2f", doubleValue))
                                    .font(.system(size: 8, design: .monospaced))
                                    .foregroundStyle(WxTheme.snwSilver)
                            }
                        }
                    }
                }
                .chartXAxis {
                    AxisMarks(values: .automatic(desiredCount: min(days.count, 8))) { _ in
                        AxisGridLine(stroke: StrokeStyle(lineWidth: 0.5, dash: [2, 3]))
                            .foregroundStyle(WxTheme.border.opacity(0.3))
                        AxisTick(stroke: StrokeStyle(lineWidth: 0.5))
                            .foregroundStyle(WxTheme.border)
                        AxisValueLabel()
                            .font(.system(size: 8, design: .monospaced))
                            .foregroundStyle(WxTheme.snwSilver)
                    }
                }
                .frame(height: 120)
            }
        }
    }

    // ── Table: Detailed Observations Data Log ─────────────────────────────────────

    @ViewBuilder
    private func observationsTable(_ days: [HistoryDayDTO]) -> some View {
        WxGlassCard(title: "Surface Observation Telemetry Log", tag: "CLIM.LOG") {
            VStack(spacing: 2) {
                // Table Header
                HStack(spacing: 6) {
                    Text("DATE")
                        .frame(width: 82, alignment: .leading)
                    Text("CONDITIONS")
                        .frame(maxWidth: .infinity, alignment: .leading)
                    Text("HIGH / LOW")
                        .frame(width: 95, alignment: .trailing)
                    Text("PRECIP")
                        .frame(width: 65, alignment: .trailing)
                    Text("WIND")
                        .frame(width: 68, alignment: .trailing)
                }
                .font(.system(size: 9, weight: .bold, design: .rounded))
                .foregroundStyle(WxGlassTheme.textMuted)
                .padding(.horizontal, 10)
                .padding(.vertical, 6)
                .background(Color.white.opacity(0.06), in: WxGlassTheme.nestedShape())

                // Table Rows (Subtle Translucent Row Striping)
                let reversedDays = Array(days.reversed())
                ForEach(Array(reversedDays.enumerated()), id: \.element.id) { idx, d in
                    HStack(spacing: 6) {
                        Text(formatRowDate(d.date))
                            .font(.system(size: 10, weight: .semibold, design: .monospaced))
                            .foregroundStyle(WxGlassTheme.textPrimary)
                            .frame(width: 82, alignment: .leading)

                        HStack(spacing: 6) {
                            Image(systemName: ConditionSymbol.systemName(for: d.conditionCode))
                                .font(.system(size: 11))
                                .symbolRenderingMode(.hierarchical)
                                .foregroundStyle(WxGlassTheme.cyan)
                            Text(d.description ?? "Observed")
                                .font(.system(size: 11, weight: .medium, design: .rounded))
                                .foregroundStyle(WxGlassTheme.textSecondary)
                                .lineLimit(1)
                        }
                        .frame(maxWidth: .infinity, alignment: .leading)

                        let h = isMetric ? d.tempMaxC : d.tempMaxF
                        let l = isMetric ? d.tempMinC : d.tempMinF
                        HStack(spacing: 3) {
                            Text(h.map { String(format: "%.0f°", $0) } ?? "--")
                                .font(.system(size: 11.5, weight: .bold, design: .rounded))
                                .foregroundStyle(WxGlassTheme.conditionRed)
                            Text("/")
                                .font(.system(size: 9.5, design: .rounded))
                                .foregroundStyle(WxGlassTheme.textMuted)
                            Text(l.map { String(format: "%.0f°", $0) } ?? "--")
                                .font(.system(size: 11.5, weight: .bold, design: .rounded))
                                .foregroundStyle(WxGlassTheme.cyan)
                        }
                        .frame(width: 95, alignment: .trailing)

                        let p = (isMetric ? d.precipMm : d.precipIn) ?? 0
                        Text(p > 0 ? String(format: isMetric ? "%.1f mm" : "%.2f\"", p) : "0.00")
                            .font(.system(size: 10, weight: p > 0 ? .bold : .regular, design: .monospaced))
                            .foregroundStyle(p > 0 ? WxGlassTheme.nominalGreen : WxGlassTheme.textMuted.opacity(0.6))
                            .frame(width: 65, alignment: .trailing)

                        let w = (isMetric ? d.windMaxKph : d.windMaxMph) ?? 0
                        Text(String(format: "%.0f %@", w, windUnit))
                            .font(.system(size: 10, design: .monospaced))
                            .foregroundStyle(WxGlassTheme.textSecondary)
                            .frame(width: 68, alignment: .trailing)
                    }
                    .padding(.horizontal, 10)
                    .padding(.vertical, 6.5)
                    .background(
                        idx.isMultiple(of: 2) ? Color.clear : Color.white.opacity(0.035),
                        in: RoundedRectangle(cornerRadius: 8, style: .continuous)
                    )
                }
            }
        }
    }

    private func legendItem(color: Color, label: String) -> some View {
        HStack(spacing: 4) {
            Circle()
                .fill(color)
                .frame(width: 7, height: 7)
            Text(label)
                .font(.system(size: 8.5, weight: .semibold, design: .rounded))
                .foregroundStyle(WxGlassTheme.textSecondary)
        }
    }

    private func formatChartDate(_ isoDate: String) -> String {
        // e.g. "2026-09-21" -> "9/21"
        let parts = isoDate.split(separator: "-")
        if parts.count == 3, let m = Int(parts[1]), let d = Int(parts[2]) {
            return "\(m)/\(d)"
        }
        return isoDate
    }

    private func formatRowDate(_ isoDate: String) -> String {
        let f = DateFormatter()
        f.dateFormat = "yyyy-MM-dd"
        if let d = f.date(from: isoDate) {
            let out = DateFormatter()
            out.dateFormat = "EEE MM/dd"
            return out.string(from: d).uppercased()
        }
        return isoDate
    }

    // ── Placeholder States ────────────────────────────────────────────────────────

    @ViewBuilder
    private var loadingView: some View {
        VStack(spacing: 8) {
            ProgressView().controlSize(.regular).tint(WxTheme.snwCyan)
            Text("DOWNLINKING HISTORICAL REANALYSIS TELEMETRY…")
                .font(.system(size: 10, weight: .bold, design: .monospaced))
                .foregroundStyle(WxTheme.snwCyan)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 60)
    }

    @ViewBuilder
    private func errorView(_ err: String) -> some View {
        VStack(spacing: 8) {
            Image(systemName: "exclamationmark.triangle")
                .font(.title2)
                .foregroundStyle(WxTheme.snwRed)
            Text("// TELEMETRY FAULT: \(err)")
                .font(.system(size: 10.5, design: .monospaced))
                .foregroundStyle(WxTheme.snwRed)
            Button("RETRY ARCHIVE FETCH") {
                Task { await store.refreshHistory() }
            }
            .font(.system(size: 9.5, weight: .bold, design: .monospaced))
            .foregroundStyle(WxTheme.snwCyan)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 40)
    }

    @ViewBuilder
    private var emptyStateView: some View {
        VStack(spacing: 8) {
            Image(systemName: "calendar.badge.clock")
                .font(.title)
                .foregroundStyle(WxTheme.snwCyan.opacity(0.6))
            Text("CLIMATOLOGICAL ARCHIVE READY")
                .font(.system(size: 11, weight: .bold, design: .monospaced))
                .foregroundStyle(WxTheme.textSecondary)
            Button("ACQUIRE OBSERVATIONS") {
                Task { await store.refreshHistory() }
            }
            .font(.system(size: 9.5, weight: .bold, design: .monospaced))
            .foregroundStyle(WxTheme.snwCyan)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 60)
    }
}

private extension String {
    var nilIfEmpty: String? {
        let trimmed = trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? nil : trimmed
    }
}
