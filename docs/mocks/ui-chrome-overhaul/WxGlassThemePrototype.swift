import SwiftUI

// MARK: - WxGlass v2 Multiplatform Design System Prototype
// Compiles cleanly across both macOS (AppKit/SwiftUI) and iOS/iPadOS (UIKit/SwiftUI).
// Verify with:
//   xcrun --sdk macosx swiftc -typecheck docs/mocks/ui-chrome-overhaul/WxGlassThemePrototype.swift
//   xcrun --sdk iphonesimulator swiftc -target arm64-apple-ios17.0-simulator -typecheck docs/mocks/ui-chrome-overhaul/WxGlassThemePrototype.swift

public enum WxGlassDirection: String, CaseIterable, Identifiable, Sendable {
    case enterpriseGlass = "snw-glass"      // Direction A (Recommended): Strange New Worlds × Apple Liquid Glass
    case atmosphericHorizon = "atmos-glass" // Direction B: Weather-forward ultra-translucent glass
    case curvedLCARS = "lcars-curved"       // Direction C: Swept LCARS L-elbows + concave fillets + smoked quartz

    public var id: String { rawValue }
}

public enum WxAtmosphereRegime: String, CaseIterable, Sendable {
    case clearDay
    case rain
    case severe
    case goldenHour
    case orbitalNight
}

public enum WxGlassTheme {
    // Core Luminous Telemetry Accents (Preserved Starfleet DNA + 25th-Century LCARS Periwinkle)
    public static let cyan = Color(red: 0x38 / 255, green: 0xE1 / 255, blue: 0xFF / 255)
    public static let cyanSoft = Color(red: 0x7C / 255, green: 0xEC / 255, blue: 0xFF / 255)
    public static let periwinkle = Color(red: 0x9A / 255, green: 0xAE / 255, blue: 0xFF / 255)
    public static let gold = Color(red: 0xF5 / 255, green: 0xA6 / 255, blue: 0x23 / 255)
    public static let amber = Color(red: 0xFF / 255, green: 0x9F / 255, blue: 0x0A / 255)
    public static let conditionRed = Color(red: 0xFF / 255, green: 0x3B / 255, blue: 0x56 / 255)
    public static let nominalGreen = Color(red: 0x30 / 255, green: 0xD1 / 255, blue: 0x58 / 255)

    // WCAG AA Compliant Text Hierarchy over Translucent Dark Glass
    public static let textPrimary = Color(red: 0xF3 / 255, green: 0xF8 / 255, blue: 0xFF / 255)
    public static let textSecondary = Color(red: 0xB4 / 255, green: 0xC5 / 255, blue: 0xDF / 255)
    public static let textMuted = Color(red: 0x8C / 255, green: 0xA1 / 255, blue: 0xC1 / 255)

    // Concentric Continuous Radii: R_inner = max(10, R_outer - cardPadding)
    public static let cardPadding: CGFloat = 12

    public static func cardRadius(for direction: WxGlassDirection = .enterpriseGlass) -> CGFloat {
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

    public static func nestedRadius(for direction: WxGlassDirection = .enterpriseGlass) -> CGFloat {
        max(10, cardRadius(for: direction) - cardPadding)
    }

    public static func cardShape(for direction: WxGlassDirection) -> AnyShape {
        let r = cardRadius(for: direction)
        if direction == .curvedLCARS {
            return AnyShape(
                UnevenRoundedRectangle(
                    topLeadingRadius: r + 6,
                    bottomLeadingRadius: r + 2,
                    bottomTrailingRadius: max(16, r - 4),
                    topTrailingRadius: max(14, r - 8),
                    style: .continuous
                )
            )
        }
        return AnyShape(RoundedRectangle(cornerRadius: r, style: .continuous))
    }

    public static func nestedShape(for direction: WxGlassDirection) -> AnyShape {
        let r = nestedRadius(for: direction)
        if direction == .curvedLCARS {
            return AnyShape(
                UnevenRoundedRectangle(
                    topLeadingRadius: r + 4,
                    bottomLeadingRadius: r + 4,
                    bottomTrailingRadius: r + 1,
                    topTrailingRadius: max(10, r - 3),
                    style: .continuous
                )
            )
        }
        return AnyShape(RoundedRectangle(cornerRadius: r, style: .continuous))
    }

    public static func cardTintOverlay(for direction: WxGlassDirection) -> Color {
        switch direction {
        case .enterpriseGlass:
            return Color(red: 0x0C / 255, green: 0x18 / 255, blue: 0x2E / 255).opacity(0.42)
        case .atmosphericHorizon:
            return Color(red: 0x10 / 255, green: 0x20 / 255, blue: 0x3A / 255).opacity(0.24)
        case .curvedLCARS:
            return Color(red: 0x0A / 255, green: 0x12 / 255, blue: 0x26 / 255).opacity(0.54)
        }
    }

    public static func specularRim(accent: Color = cyan, direction: WxGlassDirection = .enterpriseGlass) -> LinearGradient {
        if direction == .curvedLCARS {
            return LinearGradient(
                colors: [
                    gold.opacity(0.52),
                    periwinkle.opacity(0.38),
                    cyan.opacity(0.14)
                ],
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )
        }
        return LinearGradient(
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

// MARK: - Layer 0: Dynamic Weather-Reactive Atmospheric Canvas
// Uses analytic RadialGradient stops rather than offscreen Circle().blur(radius:) passes,
// avoiding multi-megabyte CAGaussianBlur render targets so WidgetKit stays well below its 30 MB Jetsam ceiling.

public struct WxAtmosphereCanvas: View {
    public var regime: WxAtmosphereRegime = .rain

    public init(regime: WxAtmosphereRegime = .rain) {
        self.regime = regime
    }

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

    public var body: some View {
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

public struct WxHorizonRail: View {
    public var accent: Color = WxGlassTheme.cyan

    public init(accent: Color = WxGlassTheme.cyan) {
        self.accent = accent
    }

    public var body: some View {
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

// MARK: - Direction C Swept Glass LCARS Elbow Frame (Zero 90° Corners, C1-Continuous Concave Fillets)

public struct WxLCARSElbowOverlay: View {
    public var primary: Color = WxGlassTheme.gold
    public var secondary: Color = WxGlassTheme.periwinkle
    public var tertiary: Color = WxGlassTheme.cyan

    public init(
        primary: Color = WxGlassTheme.gold,
        secondary: Color = WxGlassTheme.periwinkle,
        tertiary: Color = WxGlassTheme.cyan
    ) {
        self.primary = primary
        self.secondary = secondary
        self.tertiary = tertiary
    }

    public var body: some View {
        Canvas { ctx, size in
            let spineW: CGFloat = 11
            let topH: CGFloat = 7
            let botH: CGFloat = 5.5
            let topFillet: CGFloat = 16
            let botFillet: CGFloat = 12
            let splitY = max(topH + topFillet + 10, size.height * 0.58)
            let gapH: CGFloat = 5
            let capR = spineW * 0.5
            let shoulderEnd = max(spineW + topFillet + 10, size.width * 0.42)

            // 1. Upper Swept L-Elbow (Spine + C1-Continuous Concave Fillet + Rounded Top Shoulder & Rounded Gap Pill Cap — Zero 90° Corners!)
            var upper = Path()
            upper.move(to: CGPoint(x: 0, y: 0))
            upper.addLine(to: CGPoint(x: shoulderEnd, y: 0))
            upper.addArc(
                center: CGPoint(x: shoulderEnd, y: topH * 0.5),
                radius: topH * 0.5,
                startAngle: .degrees(-90),
                endAngle: .degrees(90),
                clockwise: false
            )
            upper.addLine(to: CGPoint(x: spineW + topFillet, y: topH))
            upper.addArc(
                center: CGPoint(x: spineW + topFillet, y: topH + topFillet),
                radius: topFillet,
                startAngle: .degrees(-90),
                endAngle: .degrees(180),
                clockwise: true
            )
            let upperCapCenterY = max(topH + topFillet + 1, splitY - capR)
            upper.addLine(to: CGPoint(x: spineW, y: upperCapCenterY))
            upper.addArc(
                center: CGPoint(x: capR, y: upperCapCenterY),
                radius: capR,
                startAngle: .degrees(0),
                endAngle: .degrees(180),
                clockwise: false
            )
            upper.closeSubpath()
            ctx.fill(upper, with: .color(primary.opacity(0.86)))

            // 2. Segmented Secondary & Tertiary Top Pill Bars
            let seg1Start = shoulderEnd + 7
            let seg1Width = max(0, size.width * 0.21)
            if seg1Start + seg1Width < size.width - 12, seg1Width > 10 {
                let r1 = CGRect(x: seg1Start, y: 0, width: seg1Width, height: 6)
                ctx.fill(Path(roundedRect: r1, cornerRadius: 3), with: .color(secondary.opacity(0.84)))
            }

            let seg2Start = seg1Start + seg1Width + 6
            let seg2Width = max(0, size.width * 0.14)
            if seg2Start + seg2Width < size.width - 12, seg2Width > 10 {
                let r2 = CGRect(x: seg2Start, y: 0, width: seg2Width, height: 6)
                ctx.fill(Path(roundedRect: r2, cornerRadius: 3), with: .color(tertiary.opacity(0.84)))
            }

            // 3. Lower Swept Heel (Rounded Top Pill Cap Below Gap + C1-Continuous Concave Bottom Fillet — Zero 90° Corners!)
            let lowerTop = min(size.height - botH - botFillet - 2, splitY + gapH)
            let heelEnd = min(size.width * 0.55, 68)
            if lowerTop + capR < size.height - botH {
                let lowerCapCenterY = lowerTop + capR
                let filletStartY = max(lowerCapCenterY, size.height - botH - botFillet)
                var lower = Path()
                lower.move(to: CGPoint(x: 0, y: lowerCapCenterY))
                lower.addArc(
                    center: CGPoint(x: capR, y: lowerCapCenterY),
                    radius: capR,
                    startAngle: .degrees(180),
                    endAngle: .degrees(0),
                    clockwise: false
                )
                lower.addLine(to: CGPoint(x: spineW, y: filletStartY))
                lower.addArc(
                    center: CGPoint(x: spineW + botFillet, y: filletStartY),
                    radius: botFillet,
                    startAngle: .degrees(180),
                    endAngle: .degrees(90),
                    clockwise: true
                )
                lower.addLine(to: CGPoint(x: heelEnd, y: size.height - botH))
                lower.addArc(
                    center: CGPoint(x: heelEnd, y: size.height - botH * 0.5),
                    radius: botH * 0.5,
                    startAngle: .degrees(-90),
                    endAngle: .degrees(90),
                    clockwise: false
                )
                lower.addLine(to: CGPoint(x: 0, y: size.height))
                lower.closeSubpath()
                ctx.fill(lower, with: .color(secondary.opacity(0.84)))
            }
        }
        .allowsHitTesting(false)
    }
}

// MARK: - Primary Sculpted Liquid Glass Card

public struct WxGlassCard<Content: View>: View {
    public var title: String?
    public var tag: String?
    public var accent: Color
    public var direction: WxGlassDirection
    @ViewBuilder public var content: Content

    #if os(iOS)
    @ScaledMetric(relativeTo: .caption) private var kickerSize: CGFloat = 11.5
    @ScaledMetric(relativeTo: .caption2) private var tagSize: CGFloat = 10.5
    #else
    @ScaledMetric(relativeTo: .caption) private var kickerSize: CGFloat = 10.5
    @ScaledMetric(relativeTo: .caption2) private var tagSize: CGFloat = 10.0
    #endif

    public init(
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

    public var body: some View {
        let shape = WxGlassTheme.cardShape(for: direction)
        let pad = WxGlassTheme.cardPadding
        let isLCARS = (direction == .curvedLCARS)

        VStack(alignment: .leading, spacing: 10) {
            if title != nil || tag != nil {
                HStack(spacing: 7) {
                    Circle()
                        .fill(isLCARS ? WxGlassTheme.gold : accent)
                        .frame(width: 6, height: 6)
                        .shadow(color: (isLCARS ? WxGlassTheme.gold : accent).opacity(0.8), radius: 3)

                    if let title {
                        Text(title.uppercased())
                            .font(.system(size: min(kickerSize, 16), weight: .bold, design: .rounded))
                            .tracking(isLCARS ? 0.8 : 0.6)
                            .foregroundStyle(isLCARS ? WxGlassTheme.gold.opacity(0.92) : WxGlassTheme.textSecondary)
                            .lineLimit(1)
                    }

                    Spacer(minLength: 4)

                    if let tag {
                        Text(tag.uppercased())
                            .font(.system(size: min(tagSize, 14), weight: isLCARS ? .heavy : .semibold, design: .monospaced))
                            .foregroundStyle(isLCARS ? Color(red: 0.03, green: 0.07, blue: 0.14) : WxGlassTheme.cyanSoft)
                            .lineLimit(1)
                            .padding(.horizontal, isLCARS ? 10 : 8)
                            .padding(.vertical, 2.5)
                            .background {
                                if isLCARS {
                                    Capsule().fill(
                                        LinearGradient(
                                            colors: [WxGlassTheme.gold, WxGlassTheme.periwinkle],
                                            startPoint: .leading,
                                            endPoint: .trailing
                                        )
                                    )
                                } else {
                                    Capsule().fill(accent.opacity(0.12))
                                }
                            }
                            .overlay(Capsule().strokeBorder(accent.opacity(0.28), lineWidth: 0.8))
                    }
                }
            }

            content
        }
        .padding(.top, isLCARS ? pad + 5 : pad + 2)
        .padding(.bottom, pad + 2)
        .padding(.leading, isLCARS ? pad + 11 : pad)
        .padding(.trailing, pad)
        .background {
            ZStack {
                shape.fill(.ultraThinMaterial)
                shape.fill(WxGlassTheme.cardTintOverlay(for: direction))
            }
        }
        .overlay(alignment: .top) {
            if direction == .enterpriseGlass {
                WxHorizonRail(accent: accent)
            }
        }
        .overlay {
            if isLCARS {
                WxLCARSElbowOverlay(primary: WxGlassTheme.gold, secondary: WxGlassTheme.periwinkle, tertiary: accent)
            }
        }
        .overlay {
            shape.stroke(WxGlassTheme.specularRim(accent: accent, direction: direction), lineWidth: 1)
        }
        .clipShape(shape)
        .shadow(color: Color.black.opacity(0.38), radius: 18, x: 0, y: 10)
    }
}

// MARK: - Concentric Nested Sensor Capsule (Replaces SNWMetricTile + 4 Right-Angle Brackets)

public struct WxSensorCapsule: View {
    public let label: String
    public let value: String
    public var subtitle: String?
    public var symbol: String?
    public var accent: Color = WxGlassTheme.cyan
    public var direction: WxGlassDirection = .enterpriseGlass

    @ScaledMetric(relativeTo: .caption2) private var labelSize: CGFloat = 10.5
    @ScaledMetric(relativeTo: .title3) private var valueSize: CGFloat = 20.0
    @ScaledMetric(relativeTo: .caption) private var subtitleSize: CGFloat = 11.5

    public init(
        label: String,
        value: String,
        subtitle: String? = nil,
        symbol: String? = nil,
        accent: Color = WxGlassTheme.cyan,
        direction: WxGlassDirection = .enterpriseGlass
    ) {
        self.label = label
        self.value = value
        self.subtitle = subtitle
        self.symbol = symbol
        self.accent = accent
        self.direction = direction
    }

    public var body: some View {
        let shape = WxGlassTheme.nestedShape(for: direction)
        let isLCARS = (direction == .curvedLCARS)

        VStack(alignment: .leading, spacing: 4) {
            HStack {
                Text(label.uppercased())
                    .font(.system(size: min(labelSize, 15), weight: .bold, design: .rounded))
                    .tracking(0.4)
                    .foregroundStyle(WxGlassTheme.textSecondary)
                    .lineLimit(1)
                Spacer(minLength: 2)
                if let symbol {
                    Image(systemName: symbol)
                        .font(.system(size: min(labelSize + 1, 16), weight: .semibold))
                        .foregroundStyle(accent)
                }
            }

            Text(value)
                .font(.system(size: min(valueSize, 28), weight: .bold, design: .rounded))
                .foregroundStyle(WxGlassTheme.textPrimary)
                .lineLimit(1)
                .minimumScaleFactor(0.72)

            if let subtitle {
                Text(subtitle)
                    .font(.system(size: min(subtitleSize, 16), weight: .medium))
                    .foregroundStyle(WxGlassTheme.textSecondary)
                    .lineLimit(1)
                    .minimumScaleFactor(0.8)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.leading, isLCARS ? 16 : 12)
        .padding(.trailing, 12)
        .padding(.vertical, 10)
        .background(Color.white.opacity(0.06), in: shape)
        .overlay(alignment: .leading) {
            if isLCARS {
                let barColor = (accent == WxGlassTheme.cyan) ? WxGlassTheme.gold : accent
                Capsule(style: .continuous)
                    .fill(barColor.opacity(0.88))
                    .frame(width: 4)
                    .padding(.vertical, 7)
                    .padding(.leading, 5)
            }
        }
        .overlay(shape.stroke(isLCARS ? WxGlassTheme.periwinkle.opacity(0.24) : Color.white.opacity(0.12), lineWidth: 0.8))
        .clipShape(shape)
    }
}

// MARK: - 7-Day Thermal Range Bar

public struct WxThermalRangeBar: View {
    public let dayLow: Double
    public let dayHigh: Double
    public let weekMin: Double
    public let weekMax: Double

    public init(dayLow: Double, dayHigh: Double, weekMin: Double, weekMax: Double) {
        self.dayLow = dayLow
        self.dayHigh = dayHigh
        self.weekMin = weekMin
        self.weekMax = weekMax
    }

    public var body: some View {
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

// MARK: - ScrollView Bottom Dock Fade Clearance (Zero Text Bleed Behind Floating Dock)

public struct WxDockScrollClearanceModifier: ViewModifier {
    public var dockTopFromBottom: CGFloat = 76
    public var fadeBandHeight: CGFloat = 40

    public init(dockTopFromBottom: CGFloat = 76, fadeBandHeight: CGFloat = 40) {
        self.dockTopFromBottom = dockTopFromBottom
        self.fadeBandHeight = fadeBandHeight
    }

    public func body(content: Content) -> some View {
        content
            // 6pt above-dock clearance + fadeBandHeight + 6pt extra bottom breathing room so final card sits 100% above opaqueStop at max scroll.
            .safeAreaPadding(.bottom, dockTopFromBottom + fadeBandHeight + 12)
            .mask {
                GeometryReader { geo in
                    let h = max(1, geo.size.height)
                    // Reaches 0% alpha 6pt ABOVE dockTopFromBottom so scrolling rows never bleed into dock glass.
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

public extension View {
    func wxDockScrollClearance(dockTopFromBottom: CGFloat = 76, fadeBandHeight: CGFloat = 40) -> some View {
        modifier(WxDockScrollClearanceModifier(dockTopFromBottom: dockTopFromBottom, fadeBandHeight: fadeBandHeight))
    }
}

// MARK: - iOS 26 Floating Capsule + Circular Config Orb Dock (100% XCUITest & 320pt–430pt Compatible)

public struct WxFloatingTabItem: Identifiable, Equatable, Sendable {
    public let id: String // Matches IOSTab.rawValue: "Surface", "Tactical", "Radar", "Severe", "Tropics", "Config"
    public let symbol: String

    public init(id: String, symbol: String) {
        self.id = id
        self.symbol = symbol
    }
}

public struct WxFloatingTabDock: View {
    public let primaryTabs: [WxFloatingTabItem]
    public let orbTab: WxFloatingTabItem
    public var direction: WxGlassDirection
    @Binding public var selection: String

    @ScaledMetric(relativeTo: .caption2) private var iconSize: CGFloat = 15.5
    @ScaledMetric(relativeTo: .caption2) private var labelSize: CGFloat = 10.0

    public init(
        primaryTabs: [WxFloatingTabItem],
        orbTab: WxFloatingTabItem = WxFloatingTabItem(id: "Config", symbol: "slider.horizontal.3"),
        direction: WxGlassDirection = .enterpriseGlass,
        selection: Binding<String>
    ) {
        self.primaryTabs = primaryTabs
        self.orbTab = orbTab
        self.direction = direction
        self._selection = selection
    }

    public var body: some View {
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
        let isLCARS = (direction == .curvedLCARS)
        return HStack(spacing: compact ? 6 : 8) {
            // Primary 5-Tab Frosted Capsule
            HStack(spacing: compact ? 1 : 2) {
                ForEach(primaryTabs) { tab in
                    dockButton(tab, isOrb: false, compact: compact)
                }
            }
            .padding(.vertical, compact ? 4 : 5)
            .padding(.horizontal, isLCARS ? (compact ? 6 : 8) : (compact ? 4 : 5))
            .background {
                ZStack {
                    Capsule().fill(.ultraThinMaterial)
                    Capsule().fill(Color(red: 0.06, green: 0.11, blue: 0.22).opacity(isLCARS ? 0.78 : 0.66))
                    if isLCARS {
                        HStack(spacing: 0) {
                            Rectangle().fill(WxGlassTheme.gold.opacity(0.82)).frame(width: 7)
                            Spacer(minLength: 0)
                            Rectangle().fill(WxGlassTheme.periwinkle.opacity(0.74)).frame(width: 7)
                        }
                        .clipShape(Capsule())
                    }
                }
            }
            .overlay(
                Capsule().strokeBorder(
                    isLCARS ? WxGlassTheme.gold.opacity(0.46) : Color.white.opacity(0.22),
                    lineWidth: 1
                )
            )
            .shadow(color: Color.black.opacity(0.45), radius: 18, y: 8)

            // Detached Circular Liquid Glass Config/Stations Orb
            dockButton(orbTab, isOrb: true, compact: compact)
                .frame(width: orbSide, height: orbSide)
                .background {
                    ZStack {
                        Circle().fill(.ultraThinMaterial)
                        Circle().fill(Color(red: 0.06, green: 0.11, blue: 0.22).opacity(isLCARS ? 0.76 : 0.66))
                    }
                }
                .overlay(
                    Circle().strokeBorder(
                        selection == orbTab.id
                            ? WxGlassTheme.cyan
                            : (isLCARS ? WxGlassTheme.gold.opacity(0.50) : Color.white.opacity(0.22)),
                        lineWidth: 1
                    )
                )
                .shadow(color: Color.black.opacity(0.45), radius: 18, y: 8)
        }
        .padding(.horizontal, compact ? 8 : 12)
        .padding(.bottom, 6)
    }

    @ViewBuilder
    private func dockButton(_ tab: WxFloatingTabItem, isOrb: Bool, compact: Bool) -> some View {
        let isSelected = (selection == tab.id)
        let isLCARS = (direction == .curvedLCARS)
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
                        .fill(
                            isLCARS
                                ? AnyShapeStyle(LinearGradient(colors: [WxGlassTheme.gold.opacity(0.34), WxGlassTheme.periwinkle.opacity(0.22)], startPoint: .topLeading, endPoint: .bottomTrailing))
                                : AnyShapeStyle(WxGlassTheme.cyan.opacity(0.22))
                        )
                        .overlay(Capsule().strokeBorder(Color.white.opacity(0.28), lineWidth: 0.8))
                }
            }
            .overlay(alignment: .bottom) {
                if isSelected {
                    Capsule()
                        .fill(isLCARS ? WxGlassTheme.gold : WxGlassTheme.cyan)
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
