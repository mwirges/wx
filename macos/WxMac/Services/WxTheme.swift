import SwiftUI

// MARK: - WxGlass v2 Design System (Direction A: Enterprise Liquid Glass)

enum WxGlassDirection: String, CaseIterable, Identifiable, Sendable {
    case enterpriseGlass = "snw-glass"      // Direction A (Locked): Strange New Worlds × Apple Liquid Glass
    case atmosphericHorizon = "atmos-glass"
    case curvedLCARS = "lcars-curved"

    var id: String { rawValue }
}

enum WxAtmosphereRegime: String, CaseIterable, Sendable {
    case clearDay
    case rain
    case severe
    case goldenHour
    case orbitalNight

    static func resolve(
        conditions: Conditions?,
        alerts: [Alert] = [],
        astronomy: AstronomyDTO? = nil,
        nowcast: NowcastDTO? = nil,
        forecast: Forecast? = nil,
        now: Date = Date()
    ) -> WxAtmosphereRegime {
        if alerts.contains(where: { $0.isWarning }) {
            return .severe
        }
        let code = (conditions?.conditionCode ?? "").lowercased()
        if code == "thunder" {
            return .severe
        }
        let activePrecip = (nowcast ?? conditions?.nowcast)?.isActivePrecip == true
        if ["rain", "heavy-rain", "snow", "sleet"].contains(code) || activePrecip {
            return .rain
        }
        let astro = astronomy ?? conditions?.astronomy
        if astro?.isPolarNight == true {
            return .orbitalNight
        }
        if let elev = astro?.solarElevationDeg {
            if elev < -2.0 {
                return .orbitalNight
            }
            if elev <= 8.0 {
                return .goldenHour
            }
            return .clearDay
        }
        if let period = astro?.currentPeriod?.lowercased() {
            if period.contains("golden") {
                return .goldenHour
            }
            if period.contains("night") || period.contains("twilight") || period.contains("blue") {
                return .orbitalNight
            }
        }
        if ConditionSymbol.isNight(conditions: conditions, astronomy: astro, forecast: forecast, now: now) {
            return .orbitalNight
        }
        return .clearDay
    }
}

enum WxGlassTheme {
    // Core Luminous Telemetry Accents (Starfleet Bridge DNA)
    static let cyan = Color(red: 0x38 / 255, green: 0xE1 / 255, blue: 0xFF / 255)
    static let cyanSoft = Color(red: 0x7C / 255, green: 0xEC / 255, blue: 0xFF / 255)
    static let periwinkle = Color(red: 0x9A / 255, green: 0xAE / 255, blue: 0xFF / 255)
    static let gold = Color(red: 0xF5 / 255, green: 0xA6 / 255, blue: 0x23 / 255)
    static let amber = Color(red: 0xFF / 255, green: 0x9F / 255, blue: 0x0A / 255)
    static let conditionRed = Color(red: 0xFF / 255, green: 0x3B / 255, blue: 0x56 / 255)
    static let nominalGreen = Color(red: 0x30 / 255, green: 0xD1 / 255, blue: 0x58 / 255)

    // WCAG AA Compliant Text Hierarchy over Translucent Dark Glass
    static let textPrimary = Color(red: 0xF3 / 255, green: 0xF8 / 255, blue: 0xFF / 255)
    static let textSecondary = Color(red: 0xB4 / 255, green: 0xC5 / 255, blue: 0xDF / 255)
    static let textMuted = Color(red: 0x8C / 255, green: 0xA1 / 255, blue: 0xC1 / 255)

    // Concentric Continuous Radii: R_inner = max(10, R_outer - cardPadding)
    static let cardPadding: CGFloat = 12

    static func cardRadius(for direction: WxGlassDirection = .enterpriseGlass) -> CGFloat {
        #if os(iOS)
        switch direction {
        case .enterpriseGlass: return 24
        case .atmosphericHorizon: return 28
        case .curvedLCARS: return 28
        }
        #else
        switch direction {
        case .enterpriseGlass: return 20
        case .atmosphericHorizon: return 24
        case .curvedLCARS: return 24
        }
        #endif
    }

    static func nestedRadius(for direction: WxGlassDirection = .enterpriseGlass) -> CGFloat {
        max(10, cardRadius(for: direction) - cardPadding)
    }

    static func cardShape(for direction: WxGlassDirection = .enterpriseGlass) -> RoundedRectangle {
        RoundedRectangle(cornerRadius: cardRadius(for: direction), style: .continuous)
    }

    static func nestedShape(for direction: WxGlassDirection = .enterpriseGlass) -> RoundedRectangle {
        RoundedRectangle(cornerRadius: nestedRadius(for: direction), style: .continuous)
    }

    static func cardTintOverlay(for direction: WxGlassDirection = .enterpriseGlass) -> Color {
        switch direction {
        case .enterpriseGlass:
            return Color(red: 0x0C / 255, green: 0x18 / 255, blue: 0x2E / 255).opacity(0.42)
        case .atmosphericHorizon:
            return Color(red: 0x10 / 255, green: 0x20 / 255, blue: 0x3A / 255).opacity(0.24)
        case .curvedLCARS:
            return Color(red: 0x0A / 255, green: 0x12 / 255, blue: 0x26 / 255).opacity(0.54)
        }
    }

    static func pillTintOverlay() -> Color {
        Color(red: 0x08 / 255, green: 0x12 / 255, blue: 0x24 / 255).opacity(0.58)
    }

    static func specularRim(accent: Color = cyan, direction: WxGlassDirection = .enterpriseGlass) -> LinearGradient {
        LinearGradient(
            colors: [
                Color.white.opacity(0.34),
                accent.opacity(0.28),
                Color.white.opacity(0.06)
            ],
            startPoint: .topLeading,
            endPoint: .bottomTrailing
        )
    }
}

/// Visual design tokens shared across macOS and iOS shells (v2 Enterprise Liquid Glass).
enum WxTheme {
    // Core Backgrounds (Translucent Liquid Glass)
    static let bg = Color(red: 0x06 / 255, green: 0x0C / 255, blue: 0x18 / 255).opacity(0.72)
    static let snwChassis = Color(red: 0x08 / 255, green: 0x12 / 255, blue: 0x24 / 255)
    static let snwPanel = WxGlassTheme.cardTintOverlay(for: .enterpriseGlass)

    // Console Signature Palette
    static let accent = WxGlassTheme.cyan
    static let snwCyan = WxGlassTheme.cyan
    static let snwGold = WxGlassTheme.gold
    static let snwAmber = WxGlassTheme.amber
    static let snwRed = WxGlassTheme.conditionRed
    static let snwGreen = WxGlassTheme.nominalGreen
    static let snwSilver = WxGlassTheme.textSecondary

    static let accentSecondary = WxGlassTheme.periwinkle
    static let text = WxGlassTheme.textPrimary
    static let textSecondary = WxGlassTheme.textSecondary
    static let textTertiary = WxGlassTheme.textMuted
    static let warn = snwGold
    static let alert = snwRed
    static let conditionRed = snwRed
    static let conditionYellow = snwGold
    static let border = snwCyan.opacity(0.32)
    static let corner: CGFloat = WxGlassTheme.cardRadius(for: .enterpriseGlass)
    static let nestedCorner: CGFloat = WxGlassTheme.nestedRadius(for: .enterpriseGlass)
    static let popoverSize = CGSize(width: 380, height: 460)

    static func popoverSize(for format: MenuBarFormat) -> CGSize {
        switch format {
        case .compact:
            return CGSize(width: 360, height: 380)
        case .standard:
            return CGSize(width: 380, height: 460)
        case .tactical:
            return CGSize(width: 410, height: 540)
        }
    }

    static func severityColor(_ s: String?) -> Color {
        switch (s ?? "").lowercased() {
        case "extreme": return snwRed
        case "severe": return snwRed.opacity(0.90)
        case "moderate": return snwGold
        case "minor": return snwCyan
        default: return textSecondary
        }
    }

    static func aqiColor(_ aqi: Int?) -> Color {
        guard let aqi = aqi else { return snwSilver }
        switch aqi {
        case ..<51:
            return snwGreen
        case 51...100:
            return snwAmber
        case 101...150:
            return Color.orange
        case 151...200:
            return snwRed
        case 201...300:
            return Color.purple
        default:
            return Color(red: 0.5, green: 0.0, blue: 0.15)
        }
    }
}

// MARK: - Layer 0: Dynamic Weather-Reactive Atmospheric Canvas
// Uses analytic RadialGradient stops rather than offscreen Circle().blur(radius:) passes,
// keeping GPU fill rate minimal and WidgetKit well below its 30 MB Jetsam ceiling.

struct WxAtmosphereCanvas: View {
    var regime: WxAtmosphereRegime = .clearDay

    private var palette: (top: Color, mid: Color, bottom: Color, bloomA: Color, bloomB: Color) {
        switch regime {
        case .rain:
            return (
                Color(red: 0.04, green: 0.11, blue: 0.23),
                Color(red: 0.05, green: 0.16, blue: 0.31),
                Color(red: 0.02, green: 0.05, blue: 0.10),
                WxGlassTheme.cyan.opacity(0.32),
                Color(red: 0.31, green: 0.45, blue: 0.98).opacity(0.26)
            )
        case .severe:
            return (
                Color(red: 0.12, green: 0.05, blue: 0.15),
                Color(red: 0.16, green: 0.07, blue: 0.19),
                Color(red: 0.04, green: 0.03, blue: 0.08),
                WxGlassTheme.conditionRed.opacity(0.36),
                WxGlassTheme.gold.opacity(0.24)
            )
        case .goldenHour:
            return (
                Color(red: 0.10, green: 0.15, blue: 0.27),
                Color(red: 0.22, green: 0.16, blue: 0.27),
                Color(red: 0.04, green: 0.06, blue: 0.12),
                WxGlassTheme.gold.opacity(0.38),
                Color.orange.opacity(0.24)
            )
        case .orbitalNight:
            return (
                Color(red: 0.02, green: 0.04, blue: 0.09),
                Color(red: 0.04, green: 0.08, blue: 0.18),
                Color(red: 0.01, green: 0.02, blue: 0.05),
                Color.indigo.opacity(0.30),
                WxGlassTheme.cyan.opacity(0.18)
            )
        case .clearDay:
            return (
                Color(red: 0.05, green: 0.16, blue: 0.32),
                Color(red: 0.07, green: 0.22, blue: 0.40),
                Color(red: 0.03, green: 0.07, blue: 0.15),
                WxGlassTheme.cyan.opacity(0.36),
                WxGlassTheme.gold.opacity(0.16)
            )
        }
    }

    var body: some View {
        let p = palette
        GeometryReader { geo in
            let maxDim = max(geo.size.width, geo.size.height)
            ZStack {
                LinearGradient(colors: [p.top, p.mid, p.bottom], startPoint: .top, endPoint: .bottom)

                RadialGradient(
                    colors: [p.bloomA, p.bloomA.opacity(0.35), .clear],
                    center: UnitPoint(x: 0.20, y: 0.10),
                    startRadius: 0,
                    endRadius: maxDim * 0.62
                )

                RadialGradient(
                    colors: [p.bloomB, p.bloomB.opacity(0.30), .clear],
                    center: UnitPoint(x: 0.86, y: 0.28),
                    startRadius: 0,
                    endRadius: maxDim * 0.56
                )
            }
            .allowsHitTesting(false)
        }
        .ignoresSafeArea()
    }
}

// MARK: - Swept Starfleet Horizon Light-Rail (Direction A — Replaces 90° SNWCornerBrackets)

struct WxHorizonRail: View {
    var accent: Color = WxGlassTheme.cyan

    var body: some View {
        Capsule(style: .continuous)
            .fill(
                LinearGradient(
                    colors: [
                        .clear,
                        accent.opacity(0.85),
                        Color.white.opacity(0.75),
                        accent.opacity(0.85),
                        .clear
                    ],
                    startPoint: .leading,
                    endPoint: .trailing
                )
            )
            .frame(height: 2)
            .padding(.horizontal, 22)
            .shadow(color: accent.opacity(0.55), radius: 5, y: 1)
            .allowsHitTesting(false)
    }
}

// MARK: - Primary Sculpted Liquid Glass Card

struct WxGlassCard<Content: View>: View {
    var title: String?
    var tag: String?
    var accent: Color
    var direction: WxGlassDirection
    @ViewBuilder var content: Content

    #if os(iOS)
    @ScaledMetric(relativeTo: .caption) private var kickerSize: CGFloat = 11.5
    @ScaledMetric(relativeTo: .caption2) private var tagSize: CGFloat = 10.5
    #else
    @ScaledMetric(relativeTo: .caption) private var kickerSize: CGFloat = 10.5
    @ScaledMetric(relativeTo: .caption2) private var tagSize: CGFloat = 10.0
    #endif

    init(
        title: String? = nil,
        tag: String? = nil,
        accent: Color = WxGlassTheme.cyan,
        direction: WxGlassDirection = .enterpriseGlass,
        @ViewBuilder content: () -> Content
    ) {
        self.title = title
        self.tag = tag
        self.accent = accent
        self.direction = direction
        self.content = content()
    }

    var body: some View {
        let shape = WxGlassTheme.cardShape(for: direction)
        let pad = WxGlassTheme.cardPadding

        VStack(alignment: .leading, spacing: 10) {
            if title != nil || tag != nil {
                HStack(spacing: 7) {
                    Circle()
                        .fill(accent)
                        .frame(width: 6, height: 6)
                        .shadow(color: accent.opacity(0.8), radius: 3)

                    if let title {
                        Text(title.uppercased())
                            .font(.system(size: min(kickerSize, 16), weight: .bold, design: .rounded))
                            .tracking(0.6)
                            .foregroundStyle(WxGlassTheme.textSecondary)
                            .lineLimit(1)
                    }

                    Spacer(minLength: 4)

                    if let tag, !tag.isEmpty {
                        Text(tag.uppercased())
                            .font(.system(size: min(tagSize, 14), weight: .semibold, design: .monospaced))
                            .foregroundStyle(WxGlassTheme.cyanSoft)
                            .lineLimit(1)
                            .padding(.horizontal, 8)
                            .padding(.vertical, 2.5)
                            .background(Capsule().fill(accent.opacity(0.12)))
                            .overlay(Capsule().strokeBorder(accent.opacity(0.28), lineWidth: 0.8))
                    }
                }
            }

            content
        }
        .padding(.top, pad + 2)
        .padding(.bottom, pad + 2)
        .padding(.horizontal, pad)
        .background {
            ZStack {
                shape.fill(.ultraThinMaterial)
                shape.fill(WxGlassTheme.cardTintOverlay(for: direction))
            }
        }
        .overlay(alignment: .top) {
            WxHorizonRail(accent: accent)
        }
        .overlay {
            shape.strokeBorder(WxGlassTheme.specularRim(accent: accent, direction: direction), lineWidth: 1)
        }
        .clipShape(shape)
        .shadow(color: Color.black.opacity(0.32), radius: 16, x: 0, y: 8)
    }
}

// MARK: - Concentric Nested Sensor Capsule (Replaces SNWMetricTile + 4 Right-Angle Brackets)

struct WxSensorCapsule: View {
    let label: String
    let value: String
    var subtitle: String? = nil
    var symbol: String? = nil
    var accent: Color = WxGlassTheme.cyan
    var compact: Bool = false

    #if os(iOS)
    @ScaledMetric(relativeTo: .caption2) private var labelSize: CGFloat = 10.5
    @ScaledMetric(relativeTo: .title3) private var valueSize: CGFloat = 19.0
    @ScaledMetric(relativeTo: .caption) private var subtitleSize: CGFloat = 11.5
    #else
    @ScaledMetric(relativeTo: .caption2) private var labelSize: CGFloat = 9.5
    @ScaledMetric(relativeTo: .body) private var valueSize: CGFloat = 15.0
    @ScaledMetric(relativeTo: .caption) private var subtitleSize: CGFloat = 10.5
    #endif

    var body: some View {
        let shape = WxGlassTheme.nestedShape(for: .enterpriseGlass)

        VStack(alignment: .leading, spacing: compact ? 2 : 4) {
            HStack(spacing: 4) {
                Text(label.uppercased())
                    .font(.system(size: min(compact ? labelSize - 1 : labelSize, 14), weight: .bold, design: .rounded))
                    .tracking(0.35)
                    .foregroundStyle(WxGlassTheme.textSecondary)
                    .lineLimit(1)
                Spacer(minLength: 2)
                if let symbol {
                    Image(systemName: symbol)
                        .font(.system(size: min(compact ? labelSize : labelSize + 1, 15), weight: .semibold))
                        .symbolRenderingMode(.hierarchical)
                        .foregroundStyle(accent)
                }
            }

            Text(value)
                .font(.system(size: min(compact ? valueSize - 2.5 : valueSize, 26), weight: .bold, design: .rounded))
                .foregroundStyle(WxGlassTheme.textPrimary)
                .lineLimit(1)
                .minimumScaleFactor(0.70)

            if let subtitle, !subtitle.isEmpty {
                Text(subtitle)
                    .font(.system(size: min(subtitleSize, 15), weight: .medium))
                    .foregroundStyle(WxGlassTheme.textMuted)
                    .lineLimit(1)
                    .minimumScaleFactor(0.78)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.horizontal, compact ? 9 : 11)
        .padding(.vertical, compact ? 7 : 9)
        .background(Color.white.opacity(0.06), in: shape)
        .overlay(shape.strokeBorder(Color.white.opacity(0.12), lineWidth: 0.8))
        .clipShape(shape)
    }
}

// MARK: - 7-Day Thermal Range Bar

struct WxThermalRangeBar: View {
    let dayLow: Double
    let dayHigh: Double
    let weekMin: Double
    let weekMax: Double

    var body: some View {
        GeometryReader { geo in
            let span = max(1.0, weekMax - weekMin)
            let startFrac = max(0, min(0.92, (dayLow - weekMin) / span))
            let endFrac = max(startFrac + 0.08, min(1, (dayHigh - weekMin) / span))
            let width = geo.size.width * (endFrac - startFrac)
            let offset = geo.size.width * startFrac

            ZStack(alignment: .leading) {
                Capsule()
                    .fill(Color.white.opacity(0.12))
                Capsule()
                    .fill(
                        LinearGradient(
                            colors: [WxGlassTheme.cyan, Color.mint, WxGlassTheme.gold],
                            startPoint: .leading,
                            endPoint: .trailing
                        )
                    )
                    .frame(width: width)
                    .offset(x: offset)
            }
        }
        .frame(height: 6)
    }
}

// MARK: - Smooth Bézier Solar Horizon Curve (Shared macOS + iOS)

struct WxSolarOrbitGraph: View {
    let solarElevationDeg: Double?
    var solarAzimuthDeg: Double? = nil

    /// Maps solar elevation (-90°..+90°) and azimuth (<180° morning vs >=180° afternoon)
    /// continuously onto 0..1 progress along the diurnal sine curve.
    private var progress: CGFloat {
        guard let elev = solarElevationDeg else { return 0.50 }
        let clamped = max(-60.0, min(75.0, elev))
        let isAfternoon = (solarAzimuthDeg ?? 135.0) >= 180.0
        let riseT = 0.2227
        let setT = 0.7773

        if clamped >= 0 {
            let norm = min(1.0, clamped / 65.0)
            if isAfternoon {
                return CGFloat(setT - (setT - 0.50) * norm)
            } else {
                return CGFloat(riseT + (0.50 - riseT) * norm)
            }
        } else {
            let norm = min(1.0, abs(clamped) / 60.0)
            if isAfternoon {
                return CGFloat(setT + (0.98 - setT) * norm)
            } else {
                return CGFloat(riseT - (riseT - 0.02) * norm)
            }
        }
    }

    var body: some View {
        Canvas { ctx, size in
            let w = size.width
            let h = size.height
            let horizonY = h * 0.62
            let amplitude = h * 0.42

            // 1. Subtle horizon reference line
            var horizon = Path()
            horizon.move(to: CGPoint(x: 4, y: horizonY))
            horizon.addLine(to: CGPoint(x: w - 4, y: horizonY))
            ctx.stroke(
                horizon,
                with: .color(Color.white.opacity(0.18)),
                style: StrokeStyle(lineWidth: 1, dash: [4, 4])
            )

            // 2. Smooth sine trajectory (-cos so trough at edges, peak at center)
            func yAt(_ t: CGFloat) -> CGFloat {
                let angle = Double(t) * 2.0 * .pi
                let wave = -cos(angle) // -1 at 0, +1 at 0.5, -1 at 1.0
                return horizonY - CGFloat(wave) * amplitude * 0.82 - amplitude * 0.14
            }

            var curve = Path()
            let steps = 64
            for i in 0...steps {
                let t = CGFloat(i) / CGFloat(steps)
                let pt = CGPoint(x: 6 + t * (w - 12), y: yAt(t))
                if i == 0 {
                    curve.move(to: pt)
                } else {
                    curve.addLine(to: pt)
                }
            }

            ctx.stroke(
                curve,
                with: .linearGradient(
                    Gradient(colors: [
                        WxGlassTheme.periwinkle.opacity(0.45),
                        WxGlassTheme.cyan.opacity(0.75),
                        WxGlassTheme.gold,
                        WxGlassTheme.amber,
                        WxGlassTheme.periwinkle.opacity(0.45)
                    ]),
                    startPoint: CGPoint(x: 0, y: horizonY),
                    endPoint: CGPoint(x: w, y: horizonY)
                ),
                style: StrokeStyle(lineWidth: 2.2, lineCap: .round, lineJoin: .round)
            )

            // 3. Glowing Sun / Orbital Position Orb
            let sunX = 6 + progress * (w - 12)
            let sunY = yAt(progress)
            let isDay = (solarElevationDeg ?? 15) >= 0
            let orbColor = isDay ? WxGlassTheme.gold : WxGlassTheme.cyan

            let haloRect = CGRect(x: sunX - 9, y: sunY - 9, width: 18, height: 18)
            ctx.fill(Path(ellipseIn: haloRect), with: .color(orbColor.opacity(0.28)))

            let dotRect = CGRect(x: sunX - 4.5, y: sunY - 4.5, width: 9, height: 9)
            ctx.fill(Path(ellipseIn: dotRect), with: .color(orbColor))
            ctx.stroke(Path(ellipseIn: dotRect), with: .color(Color.white.opacity(0.85)), lineWidth: 1.2)
        }
        .frame(height: 54)
        .allowsHitTesting(false)
    }
}

// MARK: - ScrollView Bottom Dock Fade Clearance (Zero Text Bleed Behind Floating Dock)

struct WxDockScrollClearanceModifier: ViewModifier {
    var dockTopFromBottom: CGFloat = 74
    var fadeBandHeight: CGFloat = 34

    func body(content: Content) -> some View {
        content
            .safeAreaPadding(.bottom, dockTopFromBottom + fadeBandHeight + 12)
            .mask {
                GeometryReader { geo in
                    let h = max(1, geo.size.height)
                    let clearStop = max(0, min(1, (h - (dockTopFromBottom + 6)) / h))
                    let opaqueStop = max(0, min(clearStop, (h - (dockTopFromBottom + 6 + fadeBandHeight)) / h))
                    LinearGradient(
                        stops: [
                            .init(color: .black, location: 0),
                            .init(color: .black, location: opaqueStop),
                            .init(color: .clear, location: clearStop),
                            .init(color: .clear, location: 1)
                        ],
                        startPoint: .top,
                        endPoint: .bottom
                    )
                }
            }
    }
}

extension View {
    func wxDockScrollClearance(dockTopFromBottom: CGFloat = 74, fadeBandHeight: CGFloat = 34) -> some View {
        modifier(WxDockScrollClearanceModifier(dockTopFromBottom: dockTopFromBottom, fadeBandHeight: fadeBandHeight))
    }

    func wxFrostedCapsule(accent: Color = WxGlassTheme.cyan, active: Bool = false) -> some View {
        self
            .background {
                ZStack {
                    Capsule().fill(.ultraThinMaterial)
                    Capsule().fill(active ? accent.opacity(0.22) : WxGlassTheme.pillTintOverlay())
                }
            }
            .overlay(
                Capsule().strokeBorder(
                    active ? accent.opacity(0.75) : Color.white.opacity(0.18),
                    lineWidth: active ? 1.1 : 0.8
                )
            )
    }
}

// MARK: - Floating Capsule + Circular Config Orb Dock (iOS & iPadOS)

struct WxFloatingTabItem: Identifiable, Equatable, Sendable {
    let id: String
    let symbol: String
}

struct WxFloatingTabDock: View {
    let primaryTabs: [WxFloatingTabItem]
    var orbTab: WxFloatingTabItem = WxFloatingTabItem(id: "Config", symbol: "slider.horizontal.3")
    @Binding var selection: String

    @ScaledMetric(relativeTo: .caption2) private var iconSize: CGFloat = 15.5
    @ScaledMetric(relativeTo: .caption2) private var labelSize: CGFloat = 10.0

    var body: some View {
        ViewThatFits(in: .horizontal) {
            dockContent(compact: false)
            dockContent(compact: true)
        }
        .dynamicTypeSize(...DynamicTypeSize.xxxLarge)
        .accessibilityElement(children: .contain)
        .accessibilityIdentifier("tabbar.custom")
    }

    private func dockContent(compact: Bool) -> some View {
        let orbSide: CGFloat = compact ? 46 : 52
        return HStack(spacing: compact ? 6 : 8) {
            HStack(spacing: compact ? 1 : 2) {
                ForEach(primaryTabs) { tab in
                    dockButton(tab, isOrb: false, compact: compact)
                }
            }
            .padding(.vertical, compact ? 4 : 5)
            .padding(.horizontal, compact ? 4 : 5)
            .background {
                ZStack {
                    Capsule().fill(.ultraThinMaterial)
                    Capsule().fill(Color(red: 0.05, green: 0.10, blue: 0.20).opacity(0.68))
                }
            }
            .overlay(
                Capsule().strokeBorder(Color.white.opacity(0.22), lineWidth: 1)
            )
            .shadow(color: Color.black.opacity(0.42), radius: 18, y: 8)

            dockButton(orbTab, isOrb: true, compact: compact)
                .frame(width: orbSide, height: orbSide)
                .background {
                    ZStack {
                        Circle().fill(.ultraThinMaterial)
                        Circle().fill(Color(red: 0.05, green: 0.10, blue: 0.20).opacity(0.68))
                    }
                }
                .overlay(
                    Circle().strokeBorder(
                        selection == orbTab.id ? WxGlassTheme.cyan : Color.white.opacity(0.22),
                        lineWidth: 1
                    )
                )
                .shadow(color: Color.black.opacity(0.42), radius: 18, y: 8)
        }
        .padding(.horizontal, compact ? 8 : 12)
        .padding(.bottom, 6)
    }

    @ViewBuilder
    private func dockButton(_ tab: WxFloatingTabItem, isOrb: Bool, compact: Bool) -> some View {
        let isSelected = (selection == tab.id)
        Button {
            selection = tab.id
        } label: {
            VStack(spacing: 2) {
                Image(systemName: tab.symbol)
                    .font(.system(size: min(iconSize, compact ? 15 : 17), weight: .semibold))
                Text(tab.id)
                    .font(.system(size: min(labelSize, compact ? 9.5 : 11.5), weight: .semibold, design: .rounded))
                    .lineLimit(1)
                    .minimumScaleFactor(0.72)
            }
            .padding(.horizontal, isOrb ? 2 : (compact ? 1 : 4))
            .frame(minWidth: isOrb ? nil : (compact ? 42 : 50), maxWidth: isOrb ? nil : .infinity, minHeight: 44)
            .foregroundStyle(isSelected ? WxGlassTheme.textPrimary : WxGlassTheme.textSecondary)
            .background {
                if isSelected && !isOrb {
                    Capsule()
                        .fill(WxGlassTheme.cyan.opacity(0.22))
                        .overlay(Capsule().strokeBorder(Color.white.opacity(0.28), lineWidth: 0.8))
                }
            }
            .overlay(alignment: .bottom) {
                if isSelected {
                    Capsule()
                        .fill(WxGlassTheme.cyan)
                        .frame(height: 2)
                        .padding(.horizontal, isOrb ? 12 : 10)
                        .padding(.bottom, 3)
                }
            }
        }
        .buttonStyle(.plain)
        .accessibilityIdentifier("tab.\(tab.id)")
        .accessibilityLabel(tab.id)
        .accessibilityAddTraits(isSelected ? .isSelected : [])
    }
}

// MARK: - Upgraded Legacy Adapters (Zero Right-Angle Brackets Anywhere)

struct SNWConsoleCard<Content: View>: View {
    var title: String?
    var tag: String?
    var statusColor: Color = WxGlassTheme.cyan
    @ViewBuilder var content: Content

    var body: some View {
        WxGlassCard(title: title, tag: tag, accent: statusColor, direction: .enterpriseGlass) {
            content
        }
    }
}

struct SNWHeaderPill: View {
    let title: String
    let subtitle: String?
    var badge: String = "WX.DESK"

    var body: some View {
        HStack(spacing: 8) {
            HStack(spacing: 5) {
                Circle()
                    .fill(WxGlassTheme.nominalGreen)
                    .frame(width: 6, height: 6)
                    .shadow(color: WxGlassTheme.nominalGreen.opacity(0.8), radius: 3)
                Text(badge)
                    .font(.system(size: 9.5, weight: .bold, design: .monospaced))
                    .foregroundStyle(WxGlassTheme.gold)
            }
            .padding(.horizontal, 8)
            .padding(.vertical, 3.5)
            .background(WxGlassTheme.gold.opacity(0.14), in: Capsule())
            .overlay(Capsule().strokeBorder(WxGlassTheme.gold.opacity(0.35), lineWidth: 0.8))

            Text(title)
                .font(.system(size: 12, weight: .semibold, design: .rounded))
                .foregroundStyle(WxGlassTheme.textPrimary)
                .lineLimit(1)

            Spacer(minLength: 4)

            if let subtitle, !subtitle.isEmpty {
                Text(subtitle.uppercased())
                    .font(.system(size: 9.5, weight: .semibold, design: .monospaced))
                    .foregroundStyle(WxGlassTheme.cyanSoft)
                    .lineLimit(1)
            }
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 7)
        .wxFrostedCapsule(accent: WxGlassTheme.cyan)
    }
}

struct SNWMetricTile: View {
    let label: String
    let value: String
    var accent: Color = WxGlassTheme.cyan

    var body: some View {
        WxSensorCapsule(label: label, value: value, accent: accent, compact: true)
    }
}

struct WarpGlassBackground: View {
    var regime: WxAtmosphereRegime = .clearDay

    var body: some View {
        ZStack {
            WxAtmosphereCanvas(regime: regime)
            Rectangle()
                .fill(.ultraThinMaterial)
                .opacity(0.22)
                .ignoresSafeArea()
        }
    }
}

struct PillHeader: View {
    let title: String
    let subtitle: String?

    var body: some View {
        SNWHeaderPill(title: title, subtitle: subtitle, badge: "WX")
    }
}
