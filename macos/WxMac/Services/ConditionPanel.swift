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

    var conditionLabel: String {
        switch self {
        case .red: return "CONDITION: RED"
        case .yellow: return "CONDITION: YELLOW"
        }
    }

    var pulseDuration: Double {
        switch self {
        case .red: return 0.6
        case .yellow: return 1.1
        }
    }
}

/// Original tactical ConditionPanel — brackets + end frames + bar modules + ALERT stack.
/// Compact badge style for Option A. Whole-panel brightness pulse; Reduce Motion → static.
struct ConditionPanel: View {
    let band: ConditionBand
    var size: CGFloat = 88

    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @State private var lit = false

    var body: some View {
        let fill = band.fill
        ZStack {
            Color.black
            VStack(spacing: size * 0.04) {
                ConditionEndFrame(fill: fill, barCount: 4)
                    .frame(height: size * 0.18)
                HStack(spacing: size * 0.06) {
                    ConditionSideBracket(fill: fill, facing: .leading)
                        .frame(width: size * 0.12)
                    VStack(spacing: size * 0.015) {
                        Text("ALERT")
                            .font(.system(size: size * 0.20, weight: .heavy, design: .rounded))
                            .tracking(-1)
                            .foregroundStyle(fill)
                            .minimumScaleFactor(0.8)
                            .lineLimit(1)
                        Text(band.conditionLabel)
                            .font(.system(size: size * 0.078, weight: .bold, design: .rounded))
                            .tracking(-0.2)
                            .foregroundStyle(fill)
                            .minimumScaleFactor(0.55)
                            .lineLimit(1)
                            .fixedSize(horizontal: false, vertical: true)
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.horizontal, 1)
                    ConditionSideBracket(fill: fill, facing: .trailing)
                        .frame(width: size * 0.12)
                }
                .frame(maxHeight: .infinity)
                ConditionEndFrame(fill: fill, barCount: 4)
                    .frame(height: size * 0.18)
            }
            .padding(size * 0.04)
        }
        .frame(width: size, height: size)
        .clipShape(RoundedRectangle(cornerRadius: size * 0.06, style: .continuous))
        .opacity(reduceMotion ? 1.0 : (lit ? 1.0 : 0.55))
        .brightness(reduceMotion ? 0 : (lit ? 0.08 : -0.12))
        .accessibilityElement(children: .ignore)
        .accessibilityLabel("Alert, \(band.conditionLabel)")
        .onAppear {
            guard !reduceMotion else { return }
            withAnimation(.easeInOut(duration: band.pulseDuration).repeatForever(autoreverses: true)) {
                lit = true
            }
        }
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
                let r = w * 0.45
                if facing == .leading {
                    p.move(to: CGPoint(x: w, y: 0))
                    p.addLine(to: CGPoint(x: r, y: 0))
                    p.addQuadCurve(to: CGPoint(x: 0, y: r), control: CGPoint(x: 0, y: 0))
                    p.addLine(to: CGPoint(x: 0, y: h - r))
                    p.addQuadCurve(to: CGPoint(x: r, y: h), control: CGPoint(x: 0, y: h))
                    p.addLine(to: CGPoint(x: w, y: h))
                    p.addLine(to: CGPoint(x: w, y: h * 0.78))
                    p.addLine(to: CGPoint(x: w * 0.45, y: h * 0.78))
                    p.addLine(to: CGPoint(x: w * 0.45, y: h * 0.22))
                    p.addLine(to: CGPoint(x: w, y: h * 0.22))
                    p.closeSubpath()
                } else {
                    p.move(to: CGPoint(x: 0, y: 0))
                    p.addLine(to: CGPoint(x: w - r, y: 0))
                    p.addQuadCurve(to: CGPoint(x: w, y: r), control: CGPoint(x: w, y: 0))
                    p.addLine(to: CGPoint(x: w, y: h - r))
                    p.addQuadCurve(to: CGPoint(x: w - r, y: h), control: CGPoint(x: w, y: h))
                    p.addLine(to: CGPoint(x: 0, y: h))
                    p.addLine(to: CGPoint(x: 0, y: h * 0.78))
                    p.addLine(to: CGPoint(x: w * 0.55, y: h * 0.78))
                    p.addLine(to: CGPoint(x: w * 0.55, y: h * 0.22))
                    p.addLine(to: CGPoint(x: 0, y: h * 0.22))
                    p.closeSubpath()
                }
            }
            .fill(fill)
        }
    }
}

struct ConditionEndFrame: View {
    let fill: Color
    var barCount: Int = 4

    var body: some View {
        GeometryReader { geo in
            let w = geo.size.width
            let h = geo.size.height
            let wing = w * 0.18
            let cutW = w * 0.42
            let cutH = h * 0.72
            ZStack {
                Path { p in
                    // Soft hexagon / trapezoid bar
                    p.move(to: CGPoint(x: wing * 0.3, y: h * 0.5))
                    p.addLine(to: CGPoint(x: wing, y: 0))
                    p.addLine(to: CGPoint(x: w - wing, y: 0))
                    p.addLine(to: CGPoint(x: w - wing * 0.3, y: h * 0.5))
                    p.addLine(to: CGPoint(x: w - wing, y: h))
                    p.addLine(to: CGPoint(x: wing, y: h))
                    p.closeSubpath()
                }
                .fill(fill)

                // Center cutout
                Rectangle()
                    .fill(Color.black)
                    .frame(width: cutW, height: cutH)

                // Horizontal bar modules
                VStack(spacing: max(1, cutH * 0.08)) {
                    ForEach(0..<barCount, id: \.self) { i in
                        let t = CGFloat(i) / CGFloat(max(barCount - 1, 1))
                        Rectangle()
                            .fill(fill.opacity(1.0 - t * 0.45))
                            .frame(width: cutW * 0.88, height: max(1.5, cutH * 0.12))
                    }
                }
            }
            .frame(width: w, height: h)
        }
    }
}
