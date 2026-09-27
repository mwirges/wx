import SwiftUI

/// Location text field with GPS button, Favorite star toggle, and Favorites/Recents dropdown menu.
struct LocationBarView: View {
    @EnvironmentObject var store: WeatherStore
    var isHUD: Bool = false
    var onSubmit: (() -> Void)? = nil

    private var isCurrentFavorite: Bool {
        store.isFavorite(store.locationInput)
    }

    var body: some View {
        HStack(spacing: 5) {
            Image(systemName: "scope")
                .font(.system(size: isHUD ? 10.5 : 11))
                .foregroundStyle(WxTheme.snwCyan.opacity(0.85))

            TextField(isHUD ? "Zip / City, ST" : "Zip or City, ST", text: $store.locationInput)
                .textFieldStyle(.plain)
                .font(.system(size: isHUD ? 11.5 : 12, weight: .medium, design: .monospaced))
                .foregroundStyle(WxTheme.text)
                .onSubmit {
                    if let onSubmit {
                        onSubmit()
                    } else {
                        Task { await store.applyLocationAndUnits() }
                    }
                }

            // Clear text button if user is typing
            if !store.locationInput.isEmpty {
                Button {
                    store.locationInput = ""
                } label: {
                    Image(systemName: "xmark.circle.fill")
                        .font(.system(size: 10))
                        .foregroundStyle(WxTheme.snwSilver.opacity(0.5))
                }
                .buttonStyle(.plain)
                .help("Clear search")
            }

            // Favorite toggle button
            Button {
                store.toggleFavorite(store.locationInput)
            } label: {
                Image(systemName: isCurrentFavorite ? "star.fill" : "star")
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundStyle(isCurrentFavorite ? WxTheme.snwGold : WxTheme.snwSilver.opacity(0.6))
            }
            .buttonStyle(.plain)
            .disabled(store.locationInput.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            .help(isCurrentFavorite ? "Remove from Favorites" : "Add current location to Favorites")

            // CoreLocation GPS button
            Button {
                store.useCurrentLocation()
            } label: {
                if store.isLocating {
                    ProgressView()
                        .controlSize(.mini)
                } else {
                    Image(systemName: "location.fill")
                        .font(.system(size: 10.5, weight: .medium))
                        .foregroundStyle(WxTheme.snwCyan)
                }
            }
            .buttonStyle(.plain)
            .disabled(store.isLocating)
            .help("Acquire current location via CoreLocation")

            // Saved Locations & Recents Menu
            LocationsDropdownMenu()
        }
        .padding(.horizontal, 8)
        .padding(.vertical, isHUD ? 5 : 6)
        .background(
            RoundedRectangle(cornerRadius: 6, style: .continuous)
                .fill(isHUD ? WxTheme.snwChassis.opacity(0.92) : WxTheme.snwChassis.opacity(0.85))
                .overlay(
                    RoundedRectangle(cornerRadius: 6, style: .continuous)
                        .strokeBorder(WxTheme.border.opacity(0.4), lineWidth: 0.8)
                )
        )
    }
}

/// Dropdown menu showing Favorite Locations, Recent History, and GPS trigger.
struct LocationsDropdownMenu: View {
    @EnvironmentObject var store: WeatherStore

    var body: some View {
        Menu {
            Button {
                store.useCurrentLocation()
            } label: {
                Label("Use Current Location", systemImage: "location.fill")
            }

            if !store.favorites.isEmpty {
                Section("FAVORITES") {
                    ForEach(store.favorites) { fav in
                        Button {
                            store.selectLocation(fav.value)
                        } label: {
                            if store.locationInput.caseInsensitiveCompare(fav.value) == .orderedSame ||
                               store.locationInput.caseInsensitiveCompare(fav.name) == .orderedSame {
                                Text("✓ ★ \(fav.name) (\(fav.value))")
                            } else {
                                Text("★ \(fav.name) (\(fav.value))")
                            }
                        }
                    }
                }
            }

            if !store.recentLocations.isEmpty {
                Section("RECENT LOCATIONS") {
                    ForEach(store.recentLocations, id: \.self) { recent in
                        Button {
                            store.selectLocation(recent)
                        } label: {
                            if store.locationInput.caseInsensitiveCompare(recent) == .orderedSame {
                                Text("✓ \(recent)")
                            } else {
                                Text(recent)
                            }
                        }
                    }
                    Divider()
                    Button(role: .destructive) {
                        store.clearRecents()
                    } label: {
                        Label("Clear Recent Locations", systemImage: "trash")
                    }
                }
            }
        } label: {
            Image(systemName: "chevron.down.circle.fill")
                .font(.system(size: 11))
                .foregroundStyle(WxTheme.snwSilver.opacity(0.75))
        }
        .menuStyle(.borderlessButton)
        .fixedSize()
        .help("Favorite and Recent Locations")
    }
}

/// Horizontal quick-switch chips for favorite locations.
struct FavoritesQuickBarView: View {
    @EnvironmentObject var store: WeatherStore

    var body: some View {
        if !store.favorites.isEmpty {
            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 5) {
                    Text("FAVORITES //")
                        .font(.system(size: 8, weight: .bold, design: .monospaced))
                        .foregroundStyle(WxTheme.snwSilver.opacity(0.65))
                        .padding(.trailing, 2)

                    ForEach(store.favorites) { fav in
                        let isSelected = store.locationInput.caseInsensitiveCompare(fav.value) == .orderedSame ||
                                         store.locationInput.caseInsensitiveCompare(fav.name) == .orderedSame
                        Button {
                            store.selectLocation(fav.value)
                        } label: {
                            HStack(spacing: 3.5) {
                                Image(systemName: "star.fill")
                                    .font(.system(size: 7.5))
                                    .foregroundStyle(isSelected ? WxTheme.snwGold : WxTheme.snwGold.opacity(0.8))
                                Text(fav.name.uppercased())
                                    .font(.system(size: 8.5, weight: isSelected ? .bold : .medium, design: .monospaced))
                            }
                            .padding(.horizontal, 7)
                            .padding(.vertical, 3.5)
                            .background(
                                isSelected ? WxTheme.snwGold.opacity(0.2) : WxTheme.snwChassis.opacity(0.7),
                                in: RoundedRectangle(cornerRadius: 4)
                            )
                            .overlay(
                                RoundedRectangle(cornerRadius: 4)
                                    .strokeBorder(
                                        isSelected ? WxTheme.snwGold.opacity(0.8) : WxTheme.border.opacity(0.4),
                                        lineWidth: isSelected ? 1.0 : 0.8
                                    )
                            )
                            .foregroundStyle(isSelected ? WxTheme.text : WxTheme.textSecondary)
                        }
                        .buttonStyle(.plain)
                        .contextMenu {
                            Button("Switch to \(fav.name)") {
                                store.selectLocation(fav.value)
                            }
                            Button("Set as Default Location") {
                                Task {
                                    store.locationInput = fav.value
                                    await store.applyLocationAndUnits()
                                }
                            }
                            Divider()
                            Button("Remove Favorite", role: .destructive) {
                                store.removeFavorite(fav.name)
                            }
                        }
                    }
                }
                .padding(.vertical, 2)
            }
        } else {
            HStack(spacing: 4) {
                Text("FAVORITES //")
                    .font(.system(size: 8, weight: .bold, design: .monospaced))
                    .foregroundStyle(WxTheme.snwSilver.opacity(0.45))
                Button {
                    store.toggleFavorite(store.locationInput)
                } label: {
                    HStack(spacing: 3) {
                        Image(systemName: "star")
                            .font(.system(size: 8))
                        Text("PIN CURRENT AS FAVORITE")
                            .font(.system(size: 8, weight: .medium, design: .monospaced))
                    }
                    .padding(.horizontal, 6)
                    .padding(.vertical, 2.5)
                    .background(WxTheme.snwChassis.opacity(0.5), in: RoundedRectangle(cornerRadius: 3))
                    .overlay(RoundedRectangle(cornerRadius: 3).strokeBorder(WxTheme.border.opacity(0.3), lineWidth: 0.6))
                    .foregroundStyle(WxTheme.snwSilver.opacity(0.7))
                }
                .buttonStyle(.plain)
                .disabled(store.locationInput.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                Spacer()
            }
            .padding(.vertical, 2)
        }
    }
}
