import AppKit
import Foundation
import SwiftUI
import WebKit

@MainActor
final class MockVerifier: NSObject, WKNavigationDelegate {
    private let window: NSWindow
    private let webView: WKWebView
    private var navContinuation: CheckedContinuation<Void, Error>?

    override init() {
        let config = WKWebViewConfiguration()
        let rect = NSRect(x: -4000, y: -4000, width: 1380, height: 1020)
        self.webView = WKWebView(frame: NSRect(origin: .zero, size: rect.size), configuration: config)
        self.window = NSWindow(contentRect: rect, styleMask: [.borderless], backing: .buffered, defer: false)
        self.window.contentView = self.webView
        self.window.orderBack(nil)
        super.init()
        self.webView.navigationDelegate = self
    }

    func webView(_ webView: WKWebView, didFinish navigation: WKNavigation!) {
        navContinuation?.resume()
        navContinuation = nil
    }

    func webView(_ webView: WKWebView, didFail navigation: WKNavigation!, withError error: Error) {
        navContinuation?.resume(throwing: error)
        navContinuation = nil
    }

    func resize(width: CGFloat, height: CGFloat) {
        window.setContentSize(NSSize(width: width, height: height))
        webView.setFrameSize(NSSize(width: width, height: height))
        webView.displayIfNeeded()
    }

    func load(fileURL: URL) async throws {
        try await withCheckedThrowingContinuation { (cont: CheckedContinuation<Void, Error>) in
            self.navContinuation = cont
            self.webView.loadFileURL(fileURL, allowingReadAccessTo: fileURL.deletingLastPathComponent())
        }
        _ = try await eval("""
            window.__wxErrors = [];
            window.addEventListener('error', e => window.__wxErrors.push(String(e.message)));
            true;
        """)
    }

    func eval(_ script: String) async throws -> Any? {
        try await withCheckedThrowingContinuation { (cont: CheckedContinuation<Any?, Error>) in
            webView.evaluateJavaScript(script) { result, error in
                if let error {
                    cont.resume(throwing: error)
                } else {
                    cont.resume(returning: result)
                }
            }
        }
    }

    func runSwiftUIPrototypeAudit() throws {
        let tabs = [
            WxFloatingTabItem(id: "Surface", symbol: "sun.max.fill"),
            WxFloatingTabItem(id: "Tactical", symbol: "safari.fill"),
            WxFloatingTabItem(id: "Radar", symbol: "dot.radiowaves.left.and.right"),
            WxFloatingTabItem(id: "Severe", symbol: "bolt.trianglebadge.exclamationmark.fill"),
            WxFloatingTabItem(id: "Tropics", symbol: "hurricane")
        ]

        for width in [CGFloat(320), CGFloat(375), CGFloat(430)] {
            for dt in [DynamicTypeSize.large, DynamicTypeSize.xxxLarge, DynamicTypeSize.accessibility3] {
                for dir in WxGlassDirection.allCases {
                    let view = ZStack(alignment: .bottom) {
                        WxAtmosphereCanvas(regime: .severe)
                        ScrollView {
                            VStack(spacing: 12) {
                                WxGlassCard(title: "Atmospheric Telemetry", tag: "KFWA", direction: dir) {
                                    WxSensorCapsule(label: "Humidity", value: "78%", subtitle: "Dew point 64°", symbol: "drop.fill", direction: dir)
                                    WxThermalRangeBar(dayLow: 54, dayHigh: 76, weekMin: 48, weekMax: 84)
                                }
                            }
                            .padding(.horizontal, 12)
                        }
                        .wxDockScrollClearance()
                        WxFloatingTabDock(primaryTabs: tabs, direction: dir, selection: .constant("Tactical"))
                    }
                    .environment(\.dynamicTypeSize, dt)
                    .frame(width: width, height: 420)

                    let host = NSHostingView(rootView: view)
                    host.frame = NSRect(x: 0, y: 0, width: width, height: 420)
                    host.layoutSubtreeIfNeeded()
                    let fitting = host.fittingSize
                    if fitting.width > width + 0.5 {
                        throw NSError(
                            domain: "MockVerifier",
                            code: 10,
                            userInfo: [NSLocalizedDescriptionKey: "SwiftUI overflow at width=\(width) dt=\(dt) dir=\(dir.rawValue): fitting=\(fitting)"]
                        )
                    }
                }
            }
        }
        print("SWIFTUI_PROTOTYPE_AUDIT: OK (320pt/375pt/430pt x large/xxxLarge/accessibility3 x 3 directions incl. WxLCARSElbowOverlay & wxDockScrollClearance)")
    }

    func runResponsiveAndAccessibilityAudit(baseFileURL: URL) async throws {
        try await load(fileURL: baseFileURL)

        // 1. Multi-viewport horizontal reflow audit across ALL directions (incl. lcars-curved!), all 5 stages & all 8 Mac modes (WCAG 1.4.10 down to 320px)
        for vpWidth in [CGFloat(320), CGFloat(375), CGFloat(768), CGFloat(1360)] {
            resize(width: vpWidth, height: 900)
            try await Task.sleep(nanoseconds: 60_000_000)
            let overflowJSON = try await eval("""
            (() => {
              const bad = [];
              for (const dir of ['snw-glass', 'lcars-curved']) {
                window.wxMocks.setDirection(dir);
                for (const id of ['iphone-lineup', 'iphone-interactive', 'widgets-island', 'mac-desk', 'popover-ipad']) {
                  window.wxMocks.selectStage(id);
                  if (id === 'mac-desk') {
                    for (const m of ['Tactical', 'Surface', 'Radar', 'Outlooks', 'Chase', 'Climate', 'Tropics', 'Grid']) {
                      window.wxMocks.setMacMode(m);
                      const sw = document.documentElement.scrollWidth;
                      const cw = document.documentElement.clientWidth;
                      if (sw > cw + 1) bad.push(`${dir}/mac-desk/${m}@${cw}px -> scrollWidth=${sw}`);
                    }
                    window.wxMocks.setMacMode('Tactical');
                  } else {
                    const sw = document.documentElement.scrollWidth;
                    const cw = document.documentElement.clientWidth;
                    if (sw > cw + 1) bad.push(`${dir}/${id}@${cw}px -> scrollWidth=${sw}`);
                  }
                }
              }
              window.wxMocks.setDirection('snw-glass');
              window.wxMocks.selectStage('iphone-lineup');
              return JSON.stringify(bad);
            })()
            """) as? String ?? "[]"
            if overflowJSON != "[]" {
                throw NSError(domain: "MockVerifier", code: 11, userInfo: [NSLocalizedDescriptionKey: "Responsive overflow: \(overflowJSON)"])
            }
        }

        // 2. Full DOM, dock mask clearance, emoji ban, leaked-ID ban, weather regime coherence, iPad split, 8 Mac modes & target-size audit
        resize(width: 1360, height: 1020)
        try await Task.sleep(nanoseconds: 60_000_000)

        let reportJSON = try await eval("""
        (() => {
          const issues = [];

          // A. Studio ARIA tablist + Interactive iPhone roving tabindex
          const tabs = Array.from(document.querySelectorAll('[data-studio-tablist] [role="tab"]'));
          if (tabs.length !== 5) issues.push(`Expected 5 studio tabs, found ${tabs.length}`);
          const protoTabs = Array.from(document.querySelectorAll('#proto-tablist [role="tab"]'));
          if (protoTabs.length !== 6) issues.push(`Expected 6 proto tabs, found ${protoTabs.length}`);
          const protoTabIndexZero = protoTabs.filter(t => t.tabIndex === 0);
          if (protoTabIndexZero.length !== 1) issues.push(`Expected 1 proto tab with tabindex=0, found ${protoTabIndexZero.length}`);

          // B. Zero OS Color Emojis (ban U+FE0F variation selector & U+1F300..U+1FAFF pictographs & 🌅/🌫️)
          const bodyText = document.body.textContent || '';
          if (/[\\uFE0F\\u{1F300}-\\u{1FAFF}\\u26C5\\u26C8\\u2699\\u23F8]/u.test(bodyText)) {
            const matches = Array.from(bodyText).filter(ch => /[\\uFE0F\\u{1F300}-\\u{1FAFF}\\u26C5\\u26C8\\u2699\\u23F8]/u.test(ch));
            issues.push(`Raw OS emoji found in DOM text: ${Array.from(new Set(matches)).join(' ')}`);
          }

          // C. Zero Leaked XCUITest Identifiers inside Simulated Device Screens
          const deviceSelector = '.iphone-screen, .mac-window, .mac-popover-frame, .ipad-screen, .widget-stage-card';
          for (const dev of document.querySelectorAll(deviceSelector)) {
            const txt = dev.textContent || '';
            if (txt.includes('alert-banner')) issues.push('Leaked alert-banner test ID in visible device text');
            if (txt.includes('desk.split')) issues.push('Leaked desk.split test ID in visible device text');
          }

          // D. Dock Scroll Mask Clearance + 7-Day Completeness + Scroll-to-Bottom Unfaded Check across Directions A/B/C
          window.wxMocks.selectStage('iphone-lineup');
          const lineupDays = document.querySelectorAll('#panel-iphone-lineup [data-day-rows] .day-row').length;
          if (lineupDays !== 7) issues.push(`Expected 7 lineup day rows, found ${lineupDays}`);
          const bodyEl = document.querySelector('#panel-iphone-lineup .ios-body');
          const dockEl = document.querySelector('#panel-iphone-lineup .ios-dock-wrap');
          const screenEl = document.querySelector('#panel-iphone-lineup .iphone-screen');
          if (bodyEl && dockEl && screenEl) {
            const screenRect = screenEl.getBoundingClientRect();
            const dockRect = dockEl.getBoundingClientRect();
            const dockTopFromBottom = screenRect.bottom - dockRect.top;
            const maskStr = getComputedStyle(bodyEl).webkitMaskImage || getComputedStyle(bodyEl).maskImage || '';
            if (!maskStr.includes('82px') || dockTopFromBottom > 80) {
              issues.push(`Dock mask clearance insufficient: dockTopFromBottom=${dockTopFromBottom.toFixed(1)}px, mask=${maskStr}`);
            }
            for (const dir of ['snw-glass', 'atmos-glass', 'lcars-curved']) {
              window.wxMocks.setDirection(dir);
              bodyEl.scrollTop = bodyEl.scrollHeight;
              const cards = Array.from(bodyEl.querySelectorAll('.wx-glass-card.modern-only')).filter(c => c.offsetParent !== null);
              const lastCard = cards[cards.length - 1];
              if (lastCard) {
                const clearanceFromBottom = screenEl.getBoundingClientRect().bottom - lastCard.getBoundingClientRect().bottom;
                if (clearanceFromBottom < 122) {
                  issues.push(`Max-scroll bottom card enters fade zone in ${dir}: clearance=${clearanceFromBottom.toFixed(1)}px (< 122px)`);
                }
              }
              bodyEl.scrollTop = 0;
            }
            window.wxMocks.setDirection('snw-glass');
          } else {
            issues.push('Missing .ios-body or .ios-dock-wrap for dock clearance check');
          }

          window.wxMocks.selectStage('iphone-interactive');
          window.wxMocks.setProtoTab('Surface');
          const protoSurfaceBody = document.getElementById('proto-panel-Surface');
          const protoScreen = document.getElementById('interactive-phone-screen');
          const protoDays = protoSurfaceBody.querySelectorAll('[data-day-rows] .day-row').length;
          if (protoDays !== 7) issues.push(`Expected 7 interactive Surface day rows, found ${protoDays}`);
          protoSurfaceBody.scrollTop = protoSurfaceBody.scrollHeight;
          const protoCards = protoSurfaceBody.querySelectorAll('.wx-glass-card');
          const lastProtoCard = protoCards[protoCards.length - 1];
          if (lastProtoCard && (protoScreen.getBoundingClientRect().bottom - lastProtoCard.getBoundingClientRect().bottom) < 122) {
            issues.push('Interactive iPhone Surface max-scroll card enters fade zone (< 122px)');
          }
          protoSurfaceBody.scrollTop = 0;

          // E. Direction C Curved LCARS Full-Surface Spine & Capsule Bar Clearance across ALL 5 Stages + 8 Mac Modes + Popover Tactical
          window.wxMocks.setDirection('lcars-curved');
          window.wxMocks.setPopoverMode('tactical');
          const allStages = ['iphone-lineup', 'iphone-interactive', 'widgets-island', 'mac-desk', 'popover-ipad'];
          for (const st of allStages) {
            window.wxMocks.selectStage(st);
            const subs = st === 'iphone-interactive'
              ? ['Surface', 'Tactical', 'Radar', 'Severe', 'Tropics', 'Config']
              : st === 'mac-desk'
                ? ['Tactical', 'Surface', 'Radar', 'Outlooks', 'Chase', 'Climate', 'Tropics', 'Grid']
                : [null];
            for (const sub of subs) {
              if (st === 'iphone-interactive' && sub) window.wxMocks.setProtoTab(sub);
              if (st === 'mac-desk' && sub) window.wxMocks.setMacMode(sub);
              const visCards = Array.from(document.querySelectorAll('.wx-glass-card')).filter(el => el.offsetParent !== null);
              for (const card of visCards) {
                const cs = getComputedStyle(card);
                const afterBg = getComputedStyle(card, '::after').backgroundImage || '';
                if (parseFloat(cs.borderTopLeftRadius) < 32 || parseFloat(cs.paddingLeft) < 25 || !afterBg.includes('radial-gradient')) {
                  issues.push(`[lcars ${st}/${sub || '-'}] card spine collision: tl=${cs.borderTopLeftRadius}, pl=${cs.paddingLeft}`);
                }
                if (card.scrollHeight > card.clientHeight + 1) {
                  issues.push(`[lcars ${st}/${sub || '-'}] card clipped: scrollH=${card.scrollHeight} > clientH=${card.clientHeight}`);
                }
              }
              const visCapsules = Array.from(document.querySelectorAll('.wx-sensor-capsule')).filter(el => el.offsetParent !== null);
              for (const cap of visCapsules) {
                const pl = parseFloat(getComputedStyle(cap).paddingLeft);
                if (pl < 14.5) {
                  issues.push(`[lcars ${st}/${sub || '-'}] capsule bar collision: pl=${pl}px`);
                }
              }
            }
          }
          window.wxMocks.setPopoverMode('standard');
          window.wxMocks.setDirection('snw-glass');

          // F. Weather Regime Consistency across Radar, Solar Orbit, UV, Severe Tab, Chase, & Pinned New Haven Rows
          // 1) Rain: stratiform echo visible, severe supercell hidden, Severe tab shows Condition Yellow Areal Flood Watch
          window.wxMocks.setWeather('rain');
          if (getComputedStyle(document.querySelector('.echo-rain')).display === 'none') issues.push('Rain echo hidden in rain preset');
          if (getComputedStyle(document.querySelector('.echo-severe')).display !== 'none') issues.push('Severe echo leaked into rain preset');
          if (document.querySelector('[data-bind="severeTag"]').textContent !== 'CONDITION · YELLOW') {
            issues.push(`Expected Severe tab CONDITION · YELLOW in rain preset, got ${document.querySelector('[data-bind="severeTag"]').textContent}`);
          }

          // 2) Severe: supercell echo visible, rain echo hidden, Condition Red alert active, Chase shows 4 Active Severe Cells
          window.wxMocks.setWeather('severe');
          if (getComputedStyle(document.querySelector('.echo-severe')).display === 'none') issues.push('Severe echo hidden in severe preset');
          if (getComputedStyle(document.querySelector('.echo-rain')).display !== 'none') issues.push('Rain echo leaked into severe preset');
          if (document.querySelector('[data-bind="capeVal"]').textContent !== '3,250') issues.push('Severe CAPE not updated');
          if (document.querySelector('[data-bind="severeTag"]').textContent !== 'CONDITION · RED') issues.push('Severe tab not CONDITION · RED in severe preset');
          if (!document.querySelector('[data-bind="chaseClusterHeadline"]').textContent.includes('4 Active Severe Cells')) issues.push('Chase cluster headline not updated in severe preset');

          // 3) Golden Hour: clear-air echo visible, zero rain/severe echoes, low golden sun +4.6° WNW, Nominal Severe & Chase
          window.wxMocks.setWeather('golden');
          if (getComputedStyle(document.querySelector('.echo-golden')).display === 'none') issues.push('Golden clear-air echo hidden in golden preset');
          if (getComputedStyle(document.querySelector('.echo-severe')).display !== 'none') issues.push('Severe supercell leaked into golden preset');
          if (getComputedStyle(document.querySelector('.echo-rain')).display !== 'none') issues.push('Rain echo leaked into golden preset');
          if (document.querySelector('[data-bind="solarTag"]').textContent !== '+4.6° WNW') issues.push('Golden solarTag mismatch');
          const nonDryInGolden = document.querySelectorAll('[data-nowcast-bars] .nowcast-bar:not(.dry)').length;
          if (nonDryInGolden !== 0) issues.push(`Expected 0 rain bars in golden preset, found ${nonDryInGolden}`);
          if (!document.querySelector('[data-bind="severeTag"]').textContent.includes('ALL CLEAR')) issues.push('Severe tab leaked warning in golden preset');
          if (!document.querySelector('[data-bind="chaseClusterHeadline"]').textContent.includes('0 Active Severe Cells')) issues.push('Chase leaked active cells in golden preset');

          // 4) Night: nocturnal clear-air echo visible, UV === 0, solarTag negative, sun orb below horizon, pinned New Haven === 44°
          window.wxMocks.setWeather('night');
          if (getComputedStyle(document.querySelector('.echo-night')).display === 'none') issues.push('Night clear-air echo hidden in night preset');
          if (getComputedStyle(document.querySelector('.echo-severe')).display !== 'none') issues.push('Severe supercell leaked into night preset');
          const nightUV = document.querySelector('[data-bind="uvVal"]').textContent;
          if (nightUV !== '0') issues.push(`Expected UV 0 at night, got ${nightUV}`);
          const nightSolar = document.querySelector('[data-bind="solarTag"]').textContent;
          if (!nightSolar.startsWith('-')) issues.push(`Expected negative solar altitude at night, got ${nightSolar}`);
          const nightOrbY = Number(document.querySelector('[data-solar-orb]').getAttribute('cy'));
          if (!(nightOrbY > 52)) issues.push(`Expected solar orb below horizon (cy > 52) at night, got ${nightOrbY}`);
          for (const nh of document.querySelectorAll('[data-newhaven-temp]')) {
            if (nh.textContent !== '44°') issues.push(`Pinned New Haven contradicted night temp: got ${nh.textContent}`);
          }

          // G. Complete iPad Split Desk Check (5 hourly slots + 5 forecast days + interactive radar transport)
          window.wxMocks.selectStage('popover-ipad');
          const ipadHours = document.querySelectorAll('.ipad-screen [data-hourly-strip] .hour-pill').length;
          const ipadDays = document.querySelectorAll('.ipad-screen [data-ipad-days] .day-row').length;
          const ipadPlay = document.querySelectorAll('.ipad-screen [data-radar-play]').length;
          const ipadScrub = document.querySelectorAll('.ipad-screen [data-radar-slider]').length;
          if (ipadHours !== 5) issues.push(`Expected 5 iPad hourly pills, found ${ipadHours}`);
          if (ipadDays !== 5) issues.push(`Expected 5 iPad day rows, found ${ipadDays}`);
          if (ipadPlay !== 1 || ipadScrub !== 1) issues.push(`Expected iPad radar play+scrubber controls, found play=${ipadPlay} scrub=${ipadScrub}`);

          // H. Interactive 8-Mode macOS Desk Console + Target Size >= 24x24 CSS px across ALL visible controls
          const stages = ['iphone-lineup', 'iphone-interactive', 'widgets-island', 'mac-desk', 'popover-ipad'];
          for (const st of stages) {
            window.wxMocks.selectStage(st);
            const subStates = st === 'iphone-interactive'
              ? ['Surface', 'Tactical', 'Radar', 'Severe', 'Tropics', 'Config']
              : st === 'mac-desk'
                ? ['Tactical', 'Surface', 'Radar', 'Outlooks', 'Chase', 'Climate', 'Tropics', 'Grid']
                : [null];
            for (const sub of subStates) {
              if (st === 'iphone-interactive' && sub) window.wxMocks.setProtoTab(sub);
              if (st === 'mac-desk' && sub) {
                window.wxMocks.setMacMode(sub);
                const visiblePanes = Array.from(document.querySelectorAll('[data-mac-pane]')).filter(p => !p.hidden);
                if (visiblePanes.length !== 1 || visiblePanes[0].dataset.macPane !== sub) {
                  issues.push(`Mac mode ${sub} failed to activate exclusively`);
                }
              }
              const controls = Array.from(document.querySelectorAll('button, input, select')).filter(el => el.offsetParent !== null);
              for (const c of controls) {
                const r = c.getBoundingClientRect();
                if (r.width < 24 || r.height < 24) {
                  issues.push(`[${st}/${sub || '-'}] ${c.tagName}#${c.id || c.textContent.trim().slice(0,18)} below 24x24px: ${r.width.toFixed(1)}x${r.height.toFixed(1)}`);
                }
              }
            }
          }
          window.wxMocks.setMacMode('Tactical');

          // I. Full Live Metric Unit Conversion (Climate, Tropics, Grid, Severe) + Station Radar Label Sync + Legacy Mode Buttons
          window.wxMocks.selectStage('iphone-lineup');
          window.wxMocks.setWeather('golden');
          window.wxMocks.setUnits('metric');
          const metricTemp = document.querySelector('[data-bind="temp"]').textContent;
          if (metricTemp !== '20°') issues.push(`Expected 68F -> 20° in metric, got ${metricTemp}`);
          if (document.querySelector('[data-bind="climNormal"]').textContent !== '23° / 12°') {
            issues.push(`Expected Climate 30-yr normal 23° / 12° in metric, got ${document.querySelector('[data-bind="climNormal"]').textContent}`);
          }
          if (document.querySelector('[data-bind="climPrecip"]').textContent !== '46.7 mm') {
            issues.push(`Expected Climate precip 46.7 mm in metric, got ${document.querySelector('[data-bind="climPrecip"]').textContent}`);
          }
          for (const sel of ['[data-bind="tropicsMeta"]', '[data-bind="macTropicsSub"]', '[data-bind="macTropicsMax"]', '[data-bind="macInvestSub"]', '[data-bind="gridMetaNH"]', '[data-bind="gridMetaMKE"]', '[data-bind="gridMetaAMA"]']) {
            const t = document.querySelector(sel)?.textContent || '';
            if (t.includes('mph') || !t.includes('km/h')) issues.push(`Leaked mph in metric mode on ${sel}: ${t}`);
          }

          window.wxMocks.setStation('milwaukee');
          if (document.querySelector('[data-bind="icao"]').textContent !== 'KMKE') {
            issues.push('Station switch to Milwaukee failed');
          }
          if (document.querySelector('[data-radar-station]').textContent !== 'MILWAUKEE ★') {
            issues.push(`Radar map station label failed to update: ${document.querySelector('[data-radar-station]').textContent}`);
          }

          // Verify Legacy v1 mode buttons are real interactive controls
          window.wxMocks.selectStage('mac-desk');
          window.wxMocks.setDirection('legacy-v1');
          const legacyClimateBtn = document.querySelector('.legacy-mode-btn[data-mac-mode="Climate"]');
          if (!legacyClimateBtn) {
            issues.push('Missing interactive .legacy-mode-btn in Before (v1 Audit) mode');
          } else {
            legacyClimateBtn.click();
            if (document.querySelector('[data-mac-pane="Climate"]').hidden) {
              issues.push('Clicking Legacy v1 Climate button failed to switch Mac pane');
            }
          }
          window.wxMocks.setMacMode('Tactical');
          window.wxMocks.setDirection('snw-glass');

          // Restore defaults
          window.wxMocks.setStation('newhaven');
          window.wxMocks.setUnits('imperial');
          window.wxMocks.setWeather('rain');
          window.wxMocks.setProtoTab('Severe');

          // Contrast check
          function lum(hex) {
            const n = parseInt(hex.slice(1), 16);
            const rgb = [(n >> 16) & 255, (n >> 8) & 255, n & 255].map(v => {
              const s = v / 255;
              return s <= 0.03928 ? s / 12.92 : Math.pow((s + 0.055) / 1.055, 2.4);
            });
            return 0.2126 * rgb[0] + 0.7152 * rgb[1] + 0.0722 * rgb[2];
          }
          function ratio(a, b) {
            const l1 = lum(a), l2 = lum(b);
            return (Math.max(l1, l2) + 0.05) / (Math.min(l1, l2) + 0.05);
          }
          const rPrimary = ratio('#f3f8ff', '#0e1a2d');
          const rSecondary = ratio('#b4c5df', '#0e1a2d');
          const rSliderTrack = ratio('#4c668a', '#0c172b');
          if (rPrimary < 4.5) issues.push(`Primary contrast too low: ${rPrimary.toFixed(2)}`);
          if (rSecondary < 4.5) issues.push(`Secondary contrast too low: ${rSecondary.toFixed(2)}`);
          if (rSliderTrack < 3.0) issues.push(`Slider track contrast below 3:1: ${rSliderTrack.toFixed(2)}`);

          return JSON.stringify({
            issues,
            primaryContrast: Number(rPrimary.toFixed(2)),
            secondaryContrast: Number(rSecondary.toFixed(2)),
            sliderTrackContrast: Number(rSliderTrack.toFixed(2))
          });
        })()
        """) as? String ?? "{}"

        print("WEB_AUDIT_SUITE:", reportJSON)
        guard let data = reportJSON.data(using: .utf8),
              let obj = try JSONSerialization.jsonObject(with: data) as? [String: Any],
              let issues = obj["issues"] as? [String] else {
            throw NSError(domain: "MockVerifier", code: 1, userInfo: [NSLocalizedDescriptionKey: "Invalid audit JSON"])
        }
        if !issues.isEmpty {
            throw NSError(domain: "MockVerifier", code: 2, userInfo: [NSLocalizedDescriptionKey: "Audit issues: \(issues.joined(separator: "; "))"])
        }
    }

    func captureQuery(
        baseFileURL: URL,
        query: String,
        width: CGFloat,
        height: CGFloat,
        outputURL: URL
    ) async throws {
        resize(width: width, height: height)
        var comps = URLComponents(url: baseFileURL.standardizedFileURL, resolvingAgainstBaseURL: true)!
        comps.query = query
        try await withCheckedThrowingContinuation { (cont: CheckedContinuation<Void, Error>) in
            self.navContinuation = cont
            self.webView.loadFileURL(comps.url!, allowingReadAccessTo: baseFileURL.deletingLastPathComponent())
        }
        try await Task.sleep(nanoseconds: 120_000_000)

        // Verify BOTH DOM ARIA exclusivity AND computed paint background exclusivity!
        let paintCheck = try await eval("""
        (() => {
          const isPainted = el => {
            const bg = getComputedStyle(el).backgroundColor;
            return bg && bg !== 'rgba(0, 0, 0, 0)' && bg !== 'transparent';
          };
          const domDirs = document.querySelectorAll('[data-set-direction][aria-pressed="true"]').length;
          const domSkies = document.querySelectorAll('[data-set-weather][aria-pressed="true"]').length;
          const domTabs = document.querySelectorAll('[data-studio-tablist] [role="tab"][aria-selected="true"]').length;
          const paintDirs = Array.from(document.querySelectorAll('[data-set-direction]')).filter(isPainted).length;
          const paintSkies = Array.from(document.querySelectorAll('[data-set-weather]')).filter(isPainted).length;
          const paintTabs = Array.from(document.querySelectorAll('[data-studio-tablist] [role="tab"]')).filter(isPainted).length;
          return `${domDirs}:${domSkies}:${domTabs}|${paintDirs}:${paintSkies}:${paintTabs}`;
        })()
        """) as? String ?? ""
        if paintCheck != "1:1:1|1:1:1" {
            throw NSError(
                domain: "MockVerifier",
                code: 6,
                userInfo: [NSLocalizedDescriptionKey: "Active pill DOM/paint mismatch on \(query): \(paintCheck)"]
            )
        }

        let docHeight = (try await eval("Math.ceil(document.documentElement.scrollHeight)") as? Double) ?? Double(height)
        let finalHeight = max(height, min(1320, CGFloat(docHeight)))
        resize(width: width, height: finalHeight)
        try await Task.sleep(nanoseconds: 100_000_000)

        let snapConfig = WKSnapshotConfiguration()
        snapConfig.rect = CGRect(x: 0, y: 0, width: width, height: finalHeight)
        snapConfig.snapshotWidth = NSNumber(value: Int(width * 2)) // 2x Retina

        let image: NSImage = try await withCheckedThrowingContinuation { cont in
            webView.takeSnapshot(with: snapConfig) { img, err in
                if let err {
                    cont.resume(throwing: err)
                } else if let img {
                    cont.resume(returning: img)
                } else {
                    cont.resume(throwing: NSError(domain: "MockVerifier", code: 3))
                }
            }
        }

        guard let tiff = image.tiffRepresentation,
              let rep = NSBitmapImageRep(data: tiff),
              let png = rep.representation(using: .png, properties: [:]) else {
            throw NSError(domain: "MockVerifier", code: 4, userInfo: [NSLocalizedDescriptionKey: "PNG encode failed"])
        }
        try png.write(to: outputURL)
        print("CAPTURED:", outputURL.lastPathComponent, "(\(rep.pixelsWide)x\(rep.pixelsHigh) px)")
    }

    func checkZeroJSErrors() async throws {
        let errs = (try await eval("JSON.stringify(window.__wxErrors || [])") as? String) ?? "[]"
        if errs != "[]" {
            throw NSError(domain: "MockVerifier", code: 5, userInfo: [NSLocalizedDescriptionKey: "JS runtime errors: \(errs)"])
        }
    }
}

@main
struct Main {
    static func main() async throws {
        let app = NSApplication.shared
        app.setActivationPolicy(.prohibited)

        let rawBase = CommandLine.arguments.count > 1 ? CommandLine.arguments[1] : FileManager.default.currentDirectoryPath
        let baseDir = URL(fileURLWithPath: rawBase).standardizedFileURL
        let indexURL = baseDir.appendingPathComponent("index.html")
        let rendersDir = baseDir.appendingPathComponent("renders")
        try FileManager.default.createDirectory(at: rendersDir, withIntermediateDirectories: true)

        let verifier = MockVerifier()
        try verifier.runSwiftUIPrototypeAudit()
        try await verifier.runResponsiveAndAccessibilityAudit(baseFileURL: indexURL)

        let shots: [(String, CGFloat, CGFloat, String)] = [
            ("01-iphone-lineup-enterprise-glass.png", 1360, 1020, "tab=iphone-lineup&dir=snw-glass&wx=rain"),
            ("02-iphone-lineup-horizon-glass.png", 1360, 1020, "tab=iphone-lineup&dir=atmos-glass&wx=golden"),
            ("03-iphone-lineup-curved-lcars.png", 1360, 1020, "tab=iphone-lineup&dir=lcars-curved&wx=night"),
            ("04-iphone-lineup-legacy-v1-before.png", 1360, 1020, "tab=iphone-lineup&dir=legacy-v1&wx=rain"),
            ("05-iphone-severe-condition-red.png", 1180, 980, "tab=iphone-interactive&dir=snw-glass&wx=severe&proto=Severe"),
            ("06-iphone-config-dark-glass.png", 1180, 980, "tab=iphone-interactive&dir=snw-glass&wx=rain&proto=Config"),
            ("07-widgets-and-dynamic-island.png", 1220, 760, "tab=widgets-island&dir=snw-glass&wx=rain"),
            ("08-macos-desk-enterprise-glass.png", 1320, 920, "tab=mac-desk&dir=snw-glass&wx=rain"),
            ("09-macos-popover-and-ipad-split.png", 1340, 860, "tab=popover-ipad&dir=snw-glass&wx=rain&hud=tactical")
        ]

        for (name, w, h, query) in shots {
            try await verifier.captureQuery(baseFileURL: indexURL, query: query, width: w, height: h, outputURL: rendersDir.appendingPathComponent(name))
        }

        try await verifier.checkZeroJSErrors()
        print("ALL_CHECKS_AND_RENDERS_OK")
    }
}
