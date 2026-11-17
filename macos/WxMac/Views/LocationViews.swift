import SwiftUI

/// Sculpted frosted capsule location input with GPS button, Favorite star toggle, and Favorites/Recents menu.
struct LocationBarView: View {
    @EnvironmentObject var store: WeatherStore
    var isHUD: Bool = false
    var onSubmit: (() -> Void)? = nil

    private var isCurrentFavorite: Bool {
        store.isFavorite(store.locationInput)
    }

    var body: some View {
        HStack(spacing: 6) {
            Image(systemName: "magnifyingglass")
                .font(.system(size: isHUD ? 10.5 : 11, weight: .semibold))
                .foregroundStyle(WxGlassTheme.cyan)

            TextField(isHUD ? "Zip / City, ST" : "Zip or City, ST", text: $store.locationInput)
                .textFieldStyle(.plain)
                .accessibilityIdentifier("wx.location")
                .font(.system(size: isHUD ? 11.5 : 12, weight: .medium, design: .rounded))
                .foregroundStyle(WxGlassTheme.textPrimary)
                .onSubmit {
                    if let onSubmit {
                        onSubmit()
                    } else {
                        Task { await store.applyLocationAndUnits() }
                    }
                }

            if !store.locationInput.isEmpty {
                Button {
                    store.locationInput = ""
                } label: {
                    Image(systemName: "xmark.circle.fill")
                        .font(.system(size: 10.5))
                        .foregroundStyle(WxGlassTheme.textMuted)
                }
                .buttonStyle(.plain)
                .help("Clear search")
            }

            Button {
                store.toggleFavorite(store.locationInput)
            } label: {
                Image(systemName: isCurrentFavorite ? "star.fill" : "star")
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundStyle(isCurrentFavorite ? WxGlassTheme.gold : WxGlassTheme.textMuted)
            }
            .buttonStyle(.plain)
            .disabled(store.locationInput.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            .help(isCurrentFavorite ? "Remove from Favorites" : "Add current location to Favorites")

            Button {
                store.useCurrentLocation()
            } label: {
                if store.isLocating {
                    ProgressView()
                        .controlSize(.mini)
                } else {
                    Image(systemName: "location.fill")
                        .font(.system(size: 11, weight: .semibold))
                        .foregroundStyle(WxGlassTheme.cyan)
                }
            }
            .buttonStyle(.plain)
            .disabled(store.isLocating)
            .accessibilityIdentifier("wx.use-current-location")
            .accessibilityLabel("Use Current Location")
            .help("Acquire current location via CoreLocation")

            LocationsDropdownMenu()
        }
        .padding(.horizontal, 10)
        .padding(.vertical, isHUD ? 5.5 : 6.5)
        .wxFrostedCapsule(accent: WxGlassTheme.cyan)
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
                .font(.system(size: 11.5))
                .foregroundStyle(WxGlassTheme.textSecondary)
        }
        .menuStyle(.borderlessButton)
        .fixedSize()
        .accessibilityIdentifier("wx.locations-menu")
        .help("Favorite and Recent Locations")
    }
}

/// Horizontal quick-switch capsule chips for favorite locations.
struct FavoritesQuickBarView: View {
    @EnvironmentObject var store: WeatherStore

    var body: some View {
        if !store.favorites.isEmpty {
            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 6) {
                    ForEach(store.favorites) { fav in
                        let isSelected = store.locationInput.caseInsensitiveCompare(fav.value) == .orderedSame ||
                                         store.locationInput.caseInsensitiveCompare(fav.name) == .orderedSame
                        Button {
                            store.selectLocation(fav.value)
                        } label: {
                            HStack(spacing: 4) {
                                Image(systemName: "star.fill")
                                    .font(.system(size: 8.5))
                                    .foregroundStyle(WxGlassTheme.gold)
                                Text(fav.name)
                                    .font(.system(size: 10, weight: isSelected ? .bold : .semibold, design: .rounded))
                            }
                            .padding(.horizontal, 9)
                            .padding(.vertical, 4)
                            .background(
                                isSelected ? WxGlassTheme.gold.opacity(0.22) : Color.white.opacity(0.06),
                                in: Capsule()
                            )
                            .overlay(
                                Capsule()
                                    .strokeBorder(
                                        isSelected ? WxGlassTheme.gold.opacity(0.75) : Color.white.opacity(0.14),
                                        lineWidth: isSelected ? 1.0 : 0.7
                                    )
                            )
                            .foregroundStyle(isSelected ? WxGlassTheme.textPrimary : WxGlassTheme.textSecondary)
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
        }
    }
}
