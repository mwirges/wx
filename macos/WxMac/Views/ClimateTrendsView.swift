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
        .background(WxTheme.bg)
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

            // Range Segmented Buttons
            HStack(spacing: 2) {
                rangeButton(days: 7, label: "7 DAYS")
                rangeButton(days: 14, label: "14 DAYS")
                rangeButton(days: 30, label: "30 DAYS")
            }
            .padding(3)
            .background(WxTheme.snwChassis, in: RoundedRectangle(cornerRadius: 6))
            .overlay(RoundedRectangle(cornerRadius: 6).strokeBorder(WxTheme.border.opacity(0.35), lineWidth: 0.8))

            Button {
                Task {
                    await store.refreshHistory()
                    await store.refreshClimate()
                }
            } label: {
                HStack(spacing: 4) {
                    if store.isHistoryLoading || store.isClimateLoading {
                        ProgressView().controlSize(.small).tint(WxTheme.snwCyan)
                    } else {
                        Image(systemName: "arrow.triangle.2.circlepath")
                    }
                    Text("SYNC")
                }
                .font(.system(size: 9, weight: .bold, design: .monospaced))
                .foregroundStyle(WxTheme.snwCyan)
                .padding(.horizontal, 9)
                .padding(.vertical, 5)
                .background(WxTheme.snwCyan.opacity(0.12), in: RoundedRectangle(cornerRadius: 5))
                .overlay(RoundedRectangle(cornerRadius: 5).strokeBorder(WxTheme.snwCyan.opacity(0.35), lineWidth: 0.8))
            }
            .buttonStyle(.plain)
            .disabled(store.isHistoryLoading || store.isClimateLoading)
        }
        .padding(10)
        .background(WxTheme.snwPanel, in: RoundedRectangle(cornerRadius: 8))
        .overlay(RoundedRectangle(cornerRadius: 8).strokeBorder(WxTheme.border.opacity(0.4), lineWidth: 0.8))
    }

    private func rangeButton(days: Int, label: String) -> some View {
        Button {
            selectedRange = days
            store.historyDaysCount = days
            Task { await store.refreshHistory() }
        } label: {
            Text(label)
                .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                .padding(.horizontal, 8)
                .padding(.vertical, 4)
                .background(
                    selectedRange == days ? WxTheme.snwCyan.opacity(0.25) : Color.clear,
                    in: RoundedRectangle(cornerRadius: 4)
                )
                .overlay(
                    RoundedRectangle(cornerRadius: 4)
                        .strokeBorder(selectedRange == days ? WxTheme.snwCyan : Color.clear, lineWidth: 0.8)
                )
                .foregroundStyle(selectedRange == days ? WxTheme.snwCyan : WxTheme.textSecondary)
        }
        .buttonStyle(.plain)
    }

    // ── NOAA Climate Normals & Daily Historical Records ───────────────────────────

    @ViewBuilder
    private var climateLoadingCard: some View {
        HStack(spacing: 8) {
            ProgressView().controlSize(.small).tint(WxTheme.snwAmber)
            Text("ACQUIRING NOAA ACIS CLIMATE NORMALS & HISTORICAL EXTREMES...")
                .font(.system(size: 9.5, weight: .semibold, design: .monospaced))
                .foregroundStyle(WxTheme.snwSilver)
            Spacer()
        }
        .padding(10)
        .background(WxTheme.snwPanel, in: RoundedRectangle(cornerRadius: 6))
        .overlay(RoundedRectangle(cornerRadius: 6).strokeBorder(WxTheme.border.opacity(0.3), lineWidth: 0.8))
    }

    @ViewBuilder
    private func climateNormalsCard(_ c: ClimateReportDTO) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            // Header Bar
            HStack(spacing: 8) {
                Image(systemName: "gauge.with.needle")
                    .font(.system(size: 11, weight: .bold))
                    .foregroundStyle(WxTheme.snwAmber)
                Text("NOAA 30-YEAR CLIMATE NORMALS & ALL-TIME RECORDS")
                    .font(.system(size: 10, weight: .bold, design: .monospaced))
                    .tracking(0.5)
                    .foregroundStyle(WxTheme.text)

                Spacer()

                if let station = c.stationName {
                    Text(station)
                        .font(.system(size: 8.5, weight: .medium, design: .monospaced))
                        .foregroundStyle(WxTheme.snwSilver)
                }

                Text(c.normalsPeriod ?? "1991–2020")
                    .font(.system(size: 8, weight: .bold, design: .monospaced))
                    .foregroundStyle(WxTheme.snwAmber)
                    .padding(.horizontal, 6)
                    .padding(.vertical, 2)
                    .background(WxTheme.snwAmber.opacity(0.12), in: RoundedRectangle(cornerRadius: 4))
                    .overlay(RoundedRectangle(cornerRadius: 4).strokeBorder(WxTheme.snwAmber.opacity(0.35), lineWidth: 0.8))
            }

            // Departure Anomaly Banner
            if let dep = c.departure {
                departureBanner(dep)
            }

            // Normals and Records Grid
            HStack(alignment: .top, spacing: 10) {
                // Today's 30-Year Normals
                if let norm = c.todayNormals {
                    todayNormalsPanel(norm)
                }

                // All-Time Records
                if let rec = c.records {
                    recordsPanel(rec)
                }

                // Monthly Normals
                if let monthly = c.monthlyNormals {
                    monthlyNormalsPanel(monthly)
                }
            }
        }
        .padding(12)
        .background(WxTheme.snwPanel, in: RoundedRectangle(cornerRadius: 8))
        .overlay(RoundedRectangle(cornerRadius: 8).strokeBorder(WxTheme.snwAmber.opacity(0.35), lineWidth: 0.8))
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
        .padding(.horizontal, 10)
        .padding(.vertical, 6)
        .background(badgeColor.opacity(0.12), in: RoundedRectangle(cornerRadius: 5))
        .overlay(RoundedRectangle(cornerRadius: 5).strokeBorder(badgeColor.opacity(0.35), lineWidth: 0.8))
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
        .padding(9)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(WxTheme.snwChassis, in: RoundedRectangle(cornerRadius: 6))
        .overlay(RoundedRectangle(cornerRadius: 6).strokeBorder(WxTheme.border.opacity(0.3), lineWidth: 0.8))
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
        .padding(9)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(WxTheme.snwChassis, in: RoundedRectangle(cornerRadius: 6))
        .overlay(RoundedRectangle(cornerRadius: 6).strokeBorder(WxTheme.border.opacity(0.3), lineWidth: 0.8))
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
        .padding(9)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(WxTheme.snwChassis, in: RoundedRectangle(cornerRadius: 6))
        .overlay(RoundedRectangle(cornerRadius: 6).strokeBorder(WxTheme.border.opacity(0.3), lineWidth: 0.8))
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
        VStack(alignment: .leading, spacing: 5) {
            HStack {
                Image(systemName: icon)
                    .font(.system(size: 10))
                    .foregroundStyle(accent)
                Text(title)
                    .font(.system(size: 8, weight: .bold, design: .monospaced))
                    .foregroundStyle(WxTheme.snwSilver)
                Spacer()
            }
            Text(value)
                .font(.system(size: 16, weight: .heavy, design: .monospaced))
                .foregroundStyle(WxTheme.text)
        }
        .padding(10)
        .background(WxTheme.snwPanel, in: RoundedRectangle(cornerRadius: 6))
        .overlay(RoundedRectangle(cornerRadius: 6).strokeBorder(WxTheme.border.opacity(0.35), lineWidth: 0.8))
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
                            .foregroundStyle(
                                LinearGradient(
                                    colors: [WxTheme.snwRed.opacity(0.25), WxTheme.snwCyan.opacity(0.15)],
                                    startPoint: .top,
                                    endPoint: .bottom
                                )
                            )

                            LineMark(
                                x: .value("Date", dateLabel),
                                y: .value("High", h)
                            )
                            .foregroundStyle(WxTheme.snwRed)
                            .lineStyle(StrokeStyle(lineWidth: 2))

                            PointMark(
                                x: .value("Date", dateLabel),
                                y: .value("High", h)
                            )
                            .foregroundStyle(WxTheme.snwRed)
                            .symbolSize(18)

                            LineMark(
                                x: .value("Date", dateLabel),
                                y: .value("Low", l)
                            )
                            .foregroundStyle(WxTheme.snwCyan)
                            .lineStyle(StrokeStyle(lineWidth: 2))

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
                                colors: [WxTheme.snwGreen, WxTheme.snwCyan.opacity(0.8)],
                                startPoint: .top,
                                endPoint: .bottom
                            )
                        )
                        .cornerRadius(3)
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
        SNWConsoleCard(title: "Surface Observation Telemetry Log", tag: "CLIM.LOG") {
            VStack(spacing: 0) {
                // Table Header
                HStack(spacing: 6) {
                    Text("DATE")
                        .frame(width: 75, alignment: .leading)
                    Text("CONDITIONS")
                        .frame(maxWidth: .infinity, alignment: .leading)
                    Text("HIGH / LOW")
                        .frame(width: 95, alignment: .trailing)
                    Text("PRECIP")
                        .frame(width: 65, alignment: .trailing)
                    Text("WIND")
                        .frame(width: 65, alignment: .trailing)
                }
                .font(.system(size: 8, weight: .bold, design: .monospaced))
                .foregroundStyle(WxTheme.snwSilver)
                .padding(.horizontal, 10)
                .padding(.vertical, 6)
                .background(WxTheme.snwChassis.opacity(0.8))

                Divider().background(WxTheme.border.opacity(0.4))

                // Table Rows
                ForEach(Array(days.reversed())) { d in
                    HStack(spacing: 6) {
                        Text(formatRowDate(d.date))
                            .font(.system(size: 9, weight: .bold, design: .monospaced))
                            .foregroundStyle(WxTheme.text)
                            .frame(width: 75, alignment: .leading)

                        HStack(spacing: 5) {
                            Image(systemName: ConditionSymbol.systemName(for: d.conditionCode))
                                .font(.system(size: 9.5))
                                .foregroundStyle(WxTheme.snwCyan)
                            Text(d.description?.uppercased() ?? "OBSERVED")
                                .font(.system(size: 8.5, weight: .medium, design: .monospaced))
                                .foregroundStyle(WxTheme.snwSilver)
                                .lineLimit(1)
                        }
                        .frame(maxWidth: .infinity, alignment: .leading)

                        let h = isMetric ? d.tempMaxC : d.tempMaxF
                        let l = isMetric ? d.tempMinC : d.tempMinF
                        HStack(spacing: 3) {
                            Text(h.map { String(format: "%.0f°", $0) } ?? "--")
                                .font(.system(size: 9.5, weight: .bold, design: .monospaced))
                                .foregroundStyle(WxTheme.snwRed)
                            Text("/")
                                .font(.system(size: 8, design: .monospaced))
                                .foregroundStyle(WxTheme.border)
                            Text(l.map { String(format: "%.0f°", $0) } ?? "--")
                                .font(.system(size: 9.5, weight: .bold, design: .monospaced))
                                .foregroundStyle(WxTheme.snwCyan)
                        }
                        .frame(width: 95, alignment: .trailing)

                        let p = (isMetric ? d.precipMm : d.precipIn) ?? 0
                        Text(p > 0 ? String(format: isMetric ? "%.1fmm" : "%.2f\"", p) : "0.00")
                            .font(.system(size: 9, weight: p > 0 ? .bold : .regular, design: .monospaced))
                            .foregroundStyle(p > 0 ? WxTheme.snwGreen : WxTheme.snwSilver.opacity(0.5))
                            .frame(width: 65, alignment: .trailing)

                        let w = (isMetric ? d.windMaxKph : d.windMaxMph) ?? 0
                        Text(String(format: "%.0f %@", w, windUnit))
                            .font(.system(size: 9, design: .monospaced))
                            .foregroundStyle(WxTheme.textSecondary)
                            .frame(width: 65, alignment: .trailing)
                    }
                    .padding(.horizontal, 10)
                    .padding(.vertical, 6.5)

                    Divider().background(WxTheme.border.opacity(0.18))
                }
            }
        }
    }

    private func legendItem(color: Color, label: String) -> some View {
        HStack(spacing: 4) {
            RoundedRectangle(cornerRadius: 2)
                .fill(color)
                .frame(width: 8, height: 8)
            Text(label)
                .font(.system(size: 8, weight: .bold, design: .monospaced))
                .foregroundStyle(WxTheme.snwSilver)
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
