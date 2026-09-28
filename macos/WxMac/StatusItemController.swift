import AppKit
import SwiftUI

@MainActor
final class StatusItemController: NSObject {
    private var statusItem: NSStatusItem?
    private var popover: NSPopover?
    private var statusMenu: NSMenu?
    private let store: WeatherStore
    private var updateObserver: NSObjectProtocol?

    // Ambient hazard alert pip pulse
    private var pipTimer: Timer?
    private var pipPulseOn: Bool = true

    init(store: WeatherStore) {
        self.store = store
        super.init()
    }

    func install() {
        let item = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
        statusItem = item
        if let button = item.button {
            button.target = self
            button.action = #selector(statusItemClick(_:))
            button.sendAction(on: [.leftMouseUp, .rightMouseUp])
        }

        statusMenu = buildMenu()

        let pop = NSPopover()
        pop.behavior = .transient
        pop.contentSize = WxTheme.popoverSize
        pop.contentViewController = NSHostingController(rootView: PopoverView().environmentObject(store))
        popover = pop

        updateObserver = NotificationCenter.default.addObserver(
            forName: .wxWeatherDidUpdate,
            object: nil,
            queue: .main
        ) { [weak self] _ in
            Task { @MainActor in
                self?.refreshPipState()
                self?.refreshButton()
            }
        }
        refreshPipState()
        refreshButton()
    }

    func tearDown() {
        pipTimer?.invalidate()
        pipTimer = nil
        if let updateObserver {
            NotificationCenter.default.removeObserver(updateObserver)
        }
        popover?.performClose(nil)
        if let statusItem {
            NSStatusBar.system.removeStatusItem(statusItem)
        }
        statusItem = nil
        popover = nil
        statusMenu = nil
    }

    // ── Hazard Alert Pip ─────────────────────────────────────────────────────────

    private func refreshPipState() {
        let hasHazard = store.hasActiveWarning || store.hasActiveWatchOrAdvisory
        if hasHazard {
            if pipTimer == nil {
                pipPulseOn = true
                pipTimer = Timer.scheduledTimer(withTimeInterval: 0.8, repeats: true) { [weak self] _ in
                    Task { @MainActor in
                        self?.pipPulseOn.toggle()
                        self?.refreshButton()
                    }
                }
            }
        } else {
            pipTimer?.invalidate()
            pipTimer = nil
            pipPulseOn = true
        }
    }

    private var pipGlyph: String {
        if store.hasActiveWarning {
            return pipPulseOn ? "🔴 " : "⭕ "
        } else if store.hasActiveWatchOrAdvisory {
            return pipPulseOn ? "🟠 " : "○ "
        }
        return ""
    }

    // ── Status Item Rendering ───────────────────────────────────────────────────

    private func refreshButton() {
        guard let button = statusItem?.button else { return }

        if store.errorMessage != nil && store.payload?.conditions == nil {
            button.image = nil
            button.title = " wx?"
            return
        }

        let pip = pipGlyph

        switch store.menuBarFormat {
        case .compact:
            button.image = nil
            button.title = "\(pip)\(store.displayTemp)"

        case .standard:
            button.image = NSImage(systemSymbolName: store.statusSymbol, accessibilityDescription: "wx")
            button.imagePosition = .imageLeading
            if pip.isEmpty {
                button.title = " \(store.displayTemp)"
            } else {
                button.title = " \(pip)\(store.displayTemp)"
            }

        case .tactical:
            button.image = nil
            button.title = "\(pip)\(store.tacticalStatusText)"
        }
    }

    // ── Dropdown Menu ───────────────────────────────────────────────────────────

    private func buildMenu() -> NSMenu {
        let menu = NSMenu()

        // 1. Active Location Header
        if let current = store.payload?.conditions?.location {
            let activeItem = NSMenuItem(
                title: "ACTIVE // \(current.uppercased()) (\(store.displayTemp))",
                action: nil,
                keyEquivalent: ""
            )
            activeItem.isEnabled = false
            menu.addItem(activeItem)
            menu.addItem(.separator())
        }

        // 2. Command Grid / Pinned Locations Header
        let gridHeader = NSMenuItem(title: "PINNED LOCATIONS // COMMAND GRID", action: nil, keyEquivalent: "")
        gridHeader.isEnabled = false
        menu.addItem(gridHeader)

        let cards = store.gridCards
        if cards.isEmpty {
            let emptyItem = NSMenuItem(title: "  No pinned locations (add in Command Grid)", action: nil, keyEquivalent: "")
            emptyItem.isEnabled = false
            menu.addItem(emptyItem)
        } else {
            for card in cards {
                let isCurrent = (store.locationInput.caseInsensitiveCompare(card.locationKey) == .orderedSame) ||
                    (store.payload?.conditions?.location?.caseInsensitiveCompare(card.displayName) == .orderedSame)

                let cardItem = NSMenuItem()
                let hostingView = NSHostingView(
                    rootView: MenuBarGridCardView(
                        card: card,
                        units: store.units,
                        isCurrent: isCurrent,
                        onSelect: { [weak self] in
                            self?.selectLocation(card.locationKey)
                        }
                    )
                )
                hostingView.frame = NSRect(x: 0, y: 0, width: 300, height: 44)
                cardItem.view = hostingView
                cardItem.representedObject = card.locationKey
                cardItem.target = self
                cardItem.action = #selector(didSelectPinnedLocationItem(_:))
                menu.addItem(cardItem)
            }
        }

        let manageGrid = NSMenuItem(title: "Manage Command Grid...", action: #selector(openCommandGrid(_:)), keyEquivalent: "g")
        manageGrid.keyEquivalentModifierMask = [.command]
        manageGrid.target = self
        menu.addItem(manageGrid)

        menu.addItem(.separator())

        // 3. Menu Bar Format Submenu
        let formatMenu = NSMenu(title: "Menu Bar Format")
        for fmt in MenuBarFormat.allCases {
            let item = NSMenuItem(title: fmt.displayName, action: #selector(selectMenuBarFormatItem(_:)), keyEquivalent: "")
            item.target = self
            item.representedObject = fmt
            item.state = (store.menuBarFormat == fmt) ? .on : .off
            formatMenu.addItem(item)
        }
        let formatParent = NSMenuItem(title: "Menu Bar Format", action: nil, keyEquivalent: "")
        formatParent.submenu = formatMenu
        menu.addItem(formatParent)

        menu.addItem(.separator())

        // 4. Open Desk
        let openDesk = NSMenuItem(
            title: "Open Desk Window",
            action: #selector(openDesk(_:)),
            keyEquivalent: "d"
        )
        openDesk.keyEquivalentModifierMask = [.command]
        openDesk.target = self
        menu.addItem(openDesk)

        // 5. Refresh Telemetry
        let refreshItem = NSMenuItem(
            title: "Refresh Telemetry",
            action: #selector(refreshAllTelemetry(_:)),
            keyEquivalent: "r"
        )
        refreshItem.keyEquivalentModifierMask = [.command]
        refreshItem.target = self
        menu.addItem(refreshItem)

        menu.addItem(.separator())

        // 6. Quit
        let quit = NSMenuItem(
            title: "Quit wx",
            action: #selector(quitWx(_:)),
            keyEquivalent: "q"
        )
        quit.keyEquivalentModifierMask = [.command]
        quit.target = self
        menu.addItem(quit)

        return menu
    }

    // ── Interaction Actions ─────────────────────────────────────────────────────

    @objc private func statusItemClick(_ sender: Any?) {
        guard let event = NSApp.currentEvent else {
            togglePopover(sender)
            return
        }
        switch event.type {
        case .rightMouseUp:
            showStatusMenu()
        default:
            togglePopover(sender)
        }
    }

    private func showStatusMenu() {
        guard let button = statusItem?.button else { return }
        popover?.performClose(nil)
        statusMenu = buildMenu()
        guard let menu = statusMenu else { return }
        let point = NSPoint(x: button.bounds.midX, y: button.bounds.maxY + 2)
        menu.popUp(positioning: nil, at: point, in: button)
    }

    @objc private func togglePopover(_ sender: Any?) {
        guard let button = statusItem?.button, let popover else { return }
        if popover.isShown {
            popover.performClose(sender)
        } else {
            popover.show(relativeTo: button.bounds, of: button, preferredEdge: .minY)
            NSApp.activate()
        }
    }

    private func selectLocation(_ locationKey: String) {
        statusMenu?.cancelTracking()
        popover?.performClose(nil)
        store.selectLocation(locationKey)
    }

    @objc private func didSelectPinnedLocationItem(_ sender: NSMenuItem) {
        guard let key = sender.representedObject as? String else { return }
        selectLocation(key)
    }

    @objc private func openCommandGrid(_ sender: Any?) {
        statusMenu?.cancelTracking()
        popover?.performClose(nil)
        store.selectedDeskTab = .grid
        NotificationCenter.default.post(name: .wxOpenDeskWindow, object: nil)
    }

    @objc private func selectMenuBarFormatItem(_ sender: NSMenuItem) {
        guard let format = sender.representedObject as? MenuBarFormat else { return }
        store.setMenuBarFormat(format)
        refreshButton()
    }

    @objc private func refreshAllTelemetry(_ sender: Any?) {
        Task {
            await store.refresh()
            await store.refreshCPC()
            await store.refreshChase()
            if !store.favorites.isEmpty {
                await store.refreshGrid()
            }
        }
    }

    @objc private func openDesk(_ sender: Any?) {
        popover?.performClose(nil)
        NotificationCenter.default.post(name: .wxOpenDeskWindow, object: nil)
    }

    @objc private func quitWx(_ sender: Any?) {
        NSApp.terminate(nil)
    }
}

// ── Dropdown Menu Quick Preview Card ──────────────────────────────────────────

struct MenuBarGridCardView: View {
    let card: LocationGridCardData
    let units: String
    let isCurrent: Bool
    let onSelect: () -> Void

    @State private var isHovered = false

    private var tempString: String {
        guard let c = card.payload?.conditions else { return "--" }
        if units == "metric" {
            if let t = c.temperatureC { return String(format: "%.0f°", t) }
        } else {
            if let t = c.temperatureF { return String(format: "%.0f°", t) }
        }
        return "--"
    }

    private var symbol: String {
        ConditionSymbol.systemName(for: card.payload?.conditions?.conditionCode)
    }

    private var windString: String {
        guard let c = card.payload?.conditions else { return "" }
        let speed: String
        if units == "metric" {
            guard let w = c.windKph else { return "" }
            speed = String(format: "%.0fkm/h", w)
        } else {
            guard let w = c.windMph else { return "" }
            speed = String(format: "%.0fmph", w)
        }
        let arrow = WeatherStore.windArrow(for: c.windDirection)
        return "\(arrow)\(speed)"
    }

    private var alertBadge: (text: String, color: Color)? {
        guard let alerts = card.payload?.alerts, !alerts.isEmpty else { return nil }
        if alerts.contains(where: { $0.isWarning }) {
            return ("WARN", WxTheme.snwRed)
        } else if alerts.contains(where: { $0.isWatch || $0.isAdvisory }) {
            return ("WATCH", WxTheme.snwAmber)
        }
        return nil
    }

    var body: some View {
        Button(action: onSelect) {
            HStack(spacing: 8) {
                // Current station active dot
                if isCurrent {
                    Circle()
                        .fill(WxTheme.snwCyan)
                        .frame(width: 5, height: 5)
                        .shadow(color: WxTheme.snwCyan.opacity(0.8), radius: 2)
                } else {
                    Circle()
                        .fill(WxTheme.snwSilver.opacity(0.3))
                        .frame(width: 5, height: 5)
                }

                // Station / Location Name
                VStack(alignment: .leading, spacing: 1) {
                    Text(card.displayName)
                        .font(.system(size: 11, weight: isCurrent ? .bold : .medium))
                        .foregroundStyle(isCurrent ? WxTheme.snwCyan : WxTheme.text)
                        .lineLimit(1)

                    if let desc = card.payload?.conditions?.description, !desc.isEmpty {
                        Text(desc)
                            .font(.system(size: 8.5))
                            .foregroundStyle(WxTheme.textSecondary)
                            .lineLimit(1)
                    }
                }

                Spacer(minLength: 4)

                // Warning / Watch Badge
                if let badge = alertBadge {
                    HStack(spacing: 2) {
                        Image(systemName: "exclamationmark.triangle.fill")
                            .font(.system(size: 8))
                        Text(badge.text)
                            .font(.system(size: 7.5, weight: .heavy, design: .monospaced))
                    }
                    .foregroundStyle(badge.color)
                    .padding(.horizontal, 4)
                    .padding(.vertical, 2)
                    .background(badge.color.opacity(0.18), in: RoundedRectangle(cornerRadius: 3))
                    .overlay(RoundedRectangle(cornerRadius: 3).strokeBorder(badge.color.opacity(0.5), lineWidth: 0.5))
                }

                // Wind Telemetry
                if !windString.isEmpty {
                    Text(windString)
                        .font(.system(size: 9.5, weight: .medium, design: .monospaced))
                        .foregroundStyle(WxTheme.snwSilver)
                }

                // Weather Symbol + Temp
                HStack(spacing: 4) {
                    if card.isLoading && card.payload == nil {
                        ProgressView().controlSize(.mini).tint(WxTheme.snwCyan)
                    } else {
                        Image(systemName: symbol)
                            .font(.system(size: 12))
                            .foregroundStyle(WxTheme.snwCyan)
                        Text(tempString)
                            .font(.system(size: 12, weight: .bold, design: .monospaced))
                            .foregroundStyle(WxTheme.text)
                    }
                }
            }
            .padding(.horizontal, 10)
            .padding(.vertical, 6)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(
                isHovered
                    ? WxTheme.snwCyan.opacity(0.12)
                    : (isCurrent ? WxTheme.snwChassis.opacity(0.6) : Color.clear),
                in: RoundedRectangle(cornerRadius: 4)
            )
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .onHover { hovering in
            isHovered = hovering
        }
    }
}
