# wx Mac shell — architecture

**Core:** Go `wx` CLI JSON (`wx --json --forecast --alerts`). Shared with a future GTK Linux port.

**Shell:** SwiftUI only. Decode DTOs for display. No NWS, cache, or location-resolution logic in Swift.

**Boundary:** `WeatherBackend` protocol. Default implementation `WxCLIBackend` (Process → PATH `wx`). Swap later for c-shared without changing views.

**Surfaces:** NSStatusItem + popover, Desk window — both bind `WeatherStore`.
