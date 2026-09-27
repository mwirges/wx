import SwiftUI

enum ConditionBand {
    case red
    case yellow

    static func from(severity: String?) -> ConditionBand? {
        switch (severity ?? "").lowercased() {
        case "extreme", "severe": return .red
        case "moderate", "minor": return .yellow
        default: return nil
        }
    }

    static func highest(in severities: [String?]) -> ConditionBand? {
        var best: ConditionBand?
        for s in severities {
            guard let b = from(severity: s) else { continue }
            switch (best, b) {
            case (nil, _): best = b
            case (.yellow, .red): best = .red
            default: break
            }
        }
        return best
    }

    var fill: Color {
        switch self {
        case .red: return WxTheme.conditionRed
        case .yellow: return WxTheme.conditionYellow
        }
    }

    var flashFill: Color {
        switch self {
        case .red: return Color(red: 255 / 255.0, green: 34 / 255.0, blue: 34 / 255.0)
        case .yellow: return Color(red: 255 / 255.0, green: 220 / 255.0, blue: 50 / 255.0)
        }
    }

    var conditionLabel: String {
        switch self {
        case .red: return "CONDITION: RED"
        case .yellow: return "CONDITION: YELLOW"
        }
    }

    var stepDuration: Double {
        switch self {
        case .red: return 0.05
        case .yellow: return 0.09
        }
    }

    var pulseDuration: Double {
        switch self {
        case .red: return 0.6
        case .yellow: return 1.1
        }
    }
}

/// Tactical ConditionPanel matching authentic Star Trek / Strange New Worlds console graphic.
/// Features swept winged end frames with 5 horizontal level bars in center notch,
/// convex side brackets, bold ALERT + CONDITION: RED/YELLOW typography, and
/// dynamic energy level cascade + peak strobe flash animation on solid black background.
/// Respects accessibilityReduceMotion.
struct ConditionPanel: View {
    let band: ConditionBand
    var size: CGFloat = 88

    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    // 9 phases matching reference animation:
    // 0: all 5 bars illuminated with cascading level gradient
    // 1..4: outer bars collapse inward toward center
    // 5: deep beat pause (all bars extinguished)
    // 6..7: pre-charge / spark
    // 8: peak strobe flash (wings, brackets, text & inner bars surge to 100% luminous intensity)
    private func barOpacity(index: Int, phase: Int) -> Double {
        if reduceMotion {
            return [0.50, 0.70, 0.84, 0.94, 1.0][index]
        }
        let base = [0.50, 0.70, 0.84, 0.94, 1.0][index]
        switch phase {
        case 0: return base
        case 1: return index == 0 ? 0.0 : base
        case 2: return index <= 1 ? 0.0 : base
        case 3: return index <= 2 ? 0.0 : (index == 3 ? 0.65 : 0.90)
        case 4: return index <= 3 ? 0.0 : 0.48
        case 5: return 0.0
        case 6: return (index == 0 || index == 3) ? 0.40 : 0.0
        case 7: return (index == 0 || index == 3) ? 0.60 : 0.0
        case 8: return index == 4 ? 1.0 : (index >= 2 ? 0.75 : 0.0)
        default: return base
        }
    }

    var body: some View {
        let w = size
        let h = size * 0.75
        let baseFill = band.fill
        let flashFill = band.flashFill

        PhaseAnimator(Array(0..<9)) { phase in
            let isFlash = !reduceMotion && (phase == 8)
            let currentFill = isFlash ? flashFill : baseFill
            let elementOpacity = isFlash ? 1.0 : (reduceMotion ? 0.95 : 0.88)
            let glowRadius: CGFloat = isFlash ? max(3, w * 0.04) : (reduceMotion ? 1 : 0)

            ZStack {
                Color.black

                // Top Wing & Level Bars
                ConditionTopWing(
                    fill: currentFill.opacity(elementOpacity),
                    barOpacities: (0..<5).map { barOpacity(index: $0, phase: phase) }
                )
                .frame(width: w, height: h * 0.32)
                .position(x: w * 0.5, y: h * 0.16)

                // Center Brackets & Text
                HStack(spacing: 0) {
                    ConditionSideBracket(fill: currentFill.opacity(elementOpacity), facing: .leading)
                        .frame(width: w * 0.0775, height: h * 0.253)

                    Spacer(minLength: 0)

                    VStack(spacing: h * 0.02) {
                        Text("ALERT")
                            .font(.system(size: h * 0.185, weight: .heavy, design: .rounded))
                            .tracking(w * 0.02)
                            .foregroundStyle(currentFill.opacity(elementOpacity))
                            .lineLimit(1)
                            .minimumScaleFactor(0.7)

                        Text(band.conditionLabel)
                            .font(.system(size: h * 0.076, weight: .bold, design: .monospaced))
                            .tracking(w * 0.015)
                            .foregroundStyle(currentFill.opacity(elementOpacity))
                            .lineLimit(1)
                            .minimumScaleFactor(0.7)
                    }
                    .frame(maxWidth: .infinity)

                    Spacer(minLength: 0)

                    ConditionSideBracket(fill: currentFill.opacity(elementOpacity), facing: .trailing)
                        .frame(width: w * 0.0775, height: h * 0.253)
                }
                .padding(.horizontal, w * 0.025)
                .frame(width: w, height: h * 0.30)
                .position(x: w * 0.5, y: h * 0.50)

                // Bottom Wing & Level Bars (inverted bar order: 0 is innermost/top)
                ConditionBottomWing(
                    fill: currentFill.opacity(elementOpacity),
                    barOpacities: (0..<5).map { barOpacity(index: 4 - $0, phase: phase) }
                )
                .frame(width: w, height: h * 0.32)
                .position(x: w * 0.5, y: h * 0.84)
            }
            .frame(width: w, height: h)
            .shadow(color: currentFill.opacity(isFlash ? 0.9 : 0.0), radius: glowRadius)
            .clipShape(RoundedRectangle(cornerRadius: max(3, w * 0.03), style: .continuous))
            .overlay(
                RoundedRectangle(cornerRadius: max(3, w * 0.03), style: .continuous)
                    .strokeBorder(currentFill.opacity(isFlash ? 0.8 : 0.2), lineWidth: 1.0)
            )
        } animation: { _ in
            reduceMotion ? nil : .linear(duration: band.stepDuration)
        }
        .accessibilityElement(children: .ignore)
        .accessibilityLabel("Alert, \(band.conditionLabel)")
    }
}

struct ConditionSideBracket: View {
    let fill: Color
    enum Facing { case leading, trailing }
    var facing: Facing

    var body: some View {
        GeometryReader { geo in
            let w = geo.size.width
            let h = geo.size.height
            Path { p in
                if facing == .leading {
                    p.move(to: CGPoint(x: w, y: 0))
                    p.addLine(to: CGPoint(x: w * 0.22, y: 0))
                    p.addQuadCurve(to: CGPoint(x: w * 0.22, y: h), control: CGPoint(x: -w * 0.40, y: h * 0.5))
                    p.addLine(to: CGPoint(x: w, y: h))
                    p.closeSubpath()
                } else {
                    p.move(to: CGPoint(x: 0, y: 0))
                    p.addLine(to: CGPoint(x: w * 0.78, y: 0))
                    p.addQuadCurve(to: CGPoint(x: w * 0.78, y: h), control: CGPoint(x: w * 1.40, y: h * 0.5))
                    p.addLine(to: CGPoint(x: 0, y: h))
                    p.closeSubpath()
                }
            }
            .fill(fill)
        }
    }
}

struct ConditionTopWing: View {
    let fill: Color
    let barOpacities: [Double]

    var body: some View {
        GeometryReader { geo in
            let w = geo.size.width
            let h = geo.size.height
            let cutoutW = w * 0.4375
            let cutoutH = h * 0.833
            let cutoutLeft = (w - cutoutW) / 2
            let cutoutRight = cutoutLeft + cutoutW

            ZStack {
                // Left Wing
                Path { p in
                    p.move(to: CGPoint(x: cutoutLeft, y: 0))
                    p.addQuadCurve(
                        to: CGPoint(x: w * 0.1025, y: h * 0.50),
                        control: CGPoint(x: w * 0.16, y: 0)
                    )
                    p.addLine(to: CGPoint(x: w * 0.20, y: cutoutH))
                    p.addLine(to: CGPoint(x: cutoutLeft, y: cutoutH))
                    p.closeSubpath()
                }
                .fill(fill)

                // Right Wing
                Path { p in
                    p.move(to: CGPoint(x: cutoutRight, y: 0))
                    p.addQuadCurve(
                        to: CGPoint(x: w * 0.8975, y: h * 0.50),
                        control: CGPoint(x: w * 0.84, y: 0)
                    )
                    p.addLine(to: CGPoint(x: w * 0.80, y: cutoutH))
                    p.addLine(to: CGPoint(x: cutoutRight, y: cutoutH))
                    p.closeSubpath()
                }
                .fill(fill)

                // 5 horizontal bars inside notch
                VStack(spacing: max(1, h * 0.022)) {
                    ForEach(0..<min(5, barOpacities.count), id: \.self) { i in
                        Rectangle()
                            .fill(fill.opacity(barOpacities[i]))
                            .frame(width: cutoutW, height: max(1.5, h * 0.078))
                    }
                    Spacer(minLength: 0)
                }
                .frame(width: cutoutW, height: cutoutH)
                .position(x: w * 0.5, y: cutoutH * 0.5)
            }
        }
    }
}

struct ConditionBottomWing: View {
    let fill: Color
    let barOpacities: [Double]

    var body: some View {
        GeometryReader { geo in
            let w = geo.size.width
            let h = geo.size.height
            let cutoutW = w * 0.4375
            let cutoutH = h * 0.833
            let cutoutLeft = (w - cutoutW) / 2
            let cutoutRight = cutoutLeft + cutoutW
            let cutoutTop = h - cutoutH

            ZStack {
                // Left Wing
                Path { p in
                    p.move(to: CGPoint(x: cutoutLeft, y: h))
                    p.addQuadCurve(
                        to: CGPoint(x: w * 0.1025, y: h * 0.50),
                        control: CGPoint(x: w * 0.16, y: h)
                    )
                    p.addLine(to: CGPoint(x: w * 0.20, y: cutoutTop))
                    p.addLine(to: CGPoint(x: cutoutLeft, y: cutoutTop))
                    p.closeSubpath()
                }
                .fill(fill)

                // Right Wing
                Path { p in
                    p.move(to: CGPoint(x: cutoutRight, y: h))
                    p.addQuadCurve(
                        to: CGPoint(x: w * 0.8975, y: h * 0.50),
                        control: CGPoint(x: w * 0.84, y: h)
                    )
                    p.addLine(to: CGPoint(x: w * 0.80, y: cutoutTop))
                    p.addLine(to: CGPoint(x: cutoutRight, y: cutoutTop))
                    p.closeSubpath()
                }
                .fill(fill)

                // 5 horizontal bars inside notch
                VStack(spacing: max(1, h * 0.022)) {
                    Spacer(minLength: 0)
                    ForEach(0..<min(5, barOpacities.count), id: \.self) { i in
                        Rectangle()
                            .fill(fill.opacity(barOpacities[i]))
                            .frame(width: cutoutW, height: max(1.5, h * 0.078))
                    }
                }
                .frame(width: cutoutW, height: cutoutH)
                .position(x: w * 0.5, y: cutoutTop + cutoutH * 0.5)
            }
        }
    }
}
