import Foundation
import CoreLocation
import Combine

enum LocationManagerError: LocalizedError {
    case denied
    case restricted
    case notAvailable
    case locationUnknown
    case network
    case headingFailure
    case unavailable
    case custom(String)

    /// Shown for denied, restricted, and `CLError.denied`. Exact UI copy.
    static let accessOffMessage = "Location access is off for wx"

    static let settingsURL = URL(string: "x-apple.systempreferences:com.apple.preference.security?Privacy_LocationServices")!

    var errorDescription: String? {
        switch self {
        case .denied, .restricted:
            return Self.accessOffMessage
        case .notAvailable:
            return "Unable to determine current coordinates."
        case .locationUnknown:
            return "Location is temporarily unavailable. Try again."
        case .network:
            return "Location lookup needs a network connection."
        case .headingFailure:
            return "Compass heading is unavailable."
        case .unavailable:
            return "Could not read your location. Try again."
        case .custom(let msg):
            return msg
        }
    }

    /// Friendly text for the location UI. CoreLocation failures never pass through
    /// `localizedDescription` (`kCLErrorDomain`, "error 1").
    static func message(for error: Error) -> String {
        let text = mapped(error).errorDescription ?? "Could not read your location. Try again."
        if text.contains("kCLErrorDomain") || text.contains("error 1") {
            return "Could not read your location. Try again."
        }
        return text
    }

    static func mapped(_ error: Error) -> LocationManagerError {
        if let existing = error as? LocationManagerError {
            return existing
        }
        let ns = error as NSError
        if ns.domain == kCLErrorDomain {
            guard let code = CLError.Code(rawValue: ns.code) else {
                return .unavailable
            }
            return mapped(code)
        }
        let raw = ns.localizedDescription
        if raw.contains("kCLErrorDomain") || raw.contains("error 1") {
            return ns.code == CLError.denied.rawValue ? .denied : .unavailable
        }
        return .custom(raw)
    }

    static func mapped(_ code: CLError.Code) -> LocationManagerError {
        switch code {
        case .denied, .promptDeclined:
            return .denied
        case .locationUnknown:
            return .locationUnknown
        case .network:
            return .network
        case .headingFailure:
            return .headingFailure
        case .geocodeFoundNoResult, .geocodeFoundPartialResult, .geocodeCanceled:
            return .notAvailable
        default:
            return .unavailable
        }
    }
}

/// Manages system CoreLocation access, authorization, and reverse-geocoding for wx.
@MainActor
final class LocationManager: NSObject, ObservableObject, CLLocationManagerDelegate {
    @Published var authorizationStatus: CLAuthorizationStatus
    @Published var isLocating: Bool = false
    @Published var errorMessage: String?

    /// WeatherStore clears a stale "location access is off" banner when this flips to authorized.
    var onAuthorizationStatusChange: ((CLAuthorizationStatus) -> Void)?

    private let manager = CLLocationManager()
    private var locationCompletion: ((Result<String, Error>) -> Void)?
    private var awaitingAuthorization = false
    private let uiTestAuthInjected: Bool

    override init() {
        let injected = Self.injectedStatus()
        self.uiTestAuthInjected = injected != nil
        self.authorizationStatus = injected ?? manager.authorizationStatus
        super.init()
        manager.delegate = self
        manager.desiredAccuracy = kCLLocationAccuracyHundredMeters
        if uiTestAuthInjected {
            installAuthInjectionObserver()
        }
    }

    deinit {
        let raw = Unmanaged.passUnretained(self).toOpaque()
        CFNotificationCenterRemoveEveryObserver(CFNotificationCenterGetDarwinNotifyCenter(), raw)
    }

    func requestLocation(completion: @escaping (Result<String, Error>) -> Void) {
        self.locationCompletion = completion
        self.errorMessage = nil

        if !uiTestAuthInjected {
            setAuthorization(manager.authorizationStatus)
        }
        let status = authorizationStatus

        if status == .notDetermined {
            // Do not call requestLocation until the user answers the prompt.
            awaitingAuthorization = true
            isLocating = true
            requestUsageAuthorization()
            return
        }
        if status == .denied {
            fail(.denied)
            return
        }
        if status == .restricted {
            fail(.restricted)
            return
        }

        awaitingAuthorization = false
        beginHardwareRequest()
    }

    // MARK: - CLLocationManagerDelegate

    nonisolated func locationManagerDidChangeAuthorization(_ manager: CLLocationManager) {
        let status = manager.authorizationStatus
        Task { @MainActor in
            // Injected UI tests own the status. The real TCC state must not overwrite it.
            guard !self.uiTestAuthInjected else { return }
            self.handleAuthorization(status)
        }
    }

    nonisolated func locationManager(_ manager: CLLocationManager, didUpdateLocations locations: [CLLocation]) {
        guard let location = locations.last else { return }
        Task { @MainActor in
            guard !self.uiTestAuthInjected else { return }
            self.reverseGeocode(location)
        }
    }

    nonisolated func locationManager(_ manager: CLLocationManager, didFailWithError error: Error) {
        Task { @MainActor in
            guard !self.uiTestAuthInjected else { return }
            self.handleFailure(error)
        }
    }

    // MARK: - Authorization

    /// Prefer When-In-Use. This SDK marks `CLAuthorizationStatus.authorizedWhenInUse`
    /// unavailable on macOS, but `requestWhenInUseAuthorization` is available and
    /// the grant arrives as `.authorizedAlways`. Fall back to Always only if the
    /// manager does not respond to the When-In-Use selector.
    private func requestUsageAuthorization() {
        guard !uiTestAuthInjected else { return }
        let whenInUse = #selector(CLLocationManager.requestWhenInUseAuthorization)
        if manager.responds(to: whenInUse) {
            manager.requestWhenInUseAuthorization()
        } else {
            manager.requestAlwaysAuthorization()
        }
    }

    /// `.authorizedAlways` is the granted state on macOS. Raw value 4 is
    /// `authorizedWhenInUse`, which this SDK will not let us name.
    private func isUsageAuthorized(_ status: CLAuthorizationStatus) -> Bool {
        status == .authorizedAlways || status.rawValue == 4
    }

    private func handleAuthorization(_ status: CLAuthorizationStatus) {
        setAuthorization(status)
        let allowed = isUsageAuthorized(status)
        if allowed {
            if errorMessage == LocationManagerError.accessOffMessage {
                errorMessage = nil
            }
            if awaitingAuthorization {
                awaitingAuthorization = false
                beginHardwareRequest()
            }
            return
        }
        if (status == .denied || status == .restricted) && (awaitingAuthorization || locationCompletion != nil) {
            awaitingAuthorization = false
            fail(status == .denied ? .denied : .restricted)
        }
    }

    private func setAuthorization(_ status: CLAuthorizationStatus) {
        authorizationStatus = status
        onAuthorizationStatusChange?(status)
    }

    private func beginHardwareRequest() {
        isLocating = true
        if uiTestAuthInjected {
            if let code = Self.injectedFailCode() {
                // Same shape CoreLocation puts in localizedDescription. The UI must not show it.
                let raw = "The operation couldn’t be completed. (kCLErrorDomain error \(code.rawValue).)"
                let err = NSError(
                    domain: kCLErrorDomain,
                    code: code.rawValue,
                    userInfo: [NSLocalizedDescriptionKey: raw]
                )
                handleFailure(err)
                return
            }
            // Fixed coordinate so a granted-auth tap resolves without the host's TCC decision.
            // The place name still comes from the production reverse-geocode path.
            reverseGeocode(CLLocation(latitude: 41.0706, longitude: -84.9739))
            return
        }
        manager.requestLocation()
    }

    private func fail(_ error: LocationManagerError) {
        isLocating = false
        awaitingAuthorization = false
        let message = LocationManagerError.message(for: error)
        errorMessage = message
        let completion = locationCompletion
        locationCompletion = nil
        completion?(.failure(error))
    }

    private func handleFailure(_ error: Error) {
        isLocating = false
        awaitingAuthorization = false
        let mapped = LocationManagerError.mapped(error)
        errorMessage = LocationManagerError.message(for: mapped)
        let completion = locationCompletion
        locationCompletion = nil
        completion?(.failure(mapped))
    }

    private func reverseGeocode(_ location: CLLocation) {
        let fallback = String(format: "%.4f,%.4f", location.coordinate.latitude, location.coordinate.longitude)
        let geocoder = CLGeocoder()
        if uiTestAuthInjected {
            // Don't hang the suite if the geocoder never calls back. Production has no timeout.
            Task { @MainActor in
                try? await Task.sleep(nanoseconds: 8_000_000_000)
                guard self.locationCompletion != nil else { return }
                geocoder.cancelGeocode()
                self.finishResolved(fallback)
            }
        }
        geocoder.reverseGeocodeLocation(location) { [weak self] placemarks, _ in
            Task { @MainActor in
                guard let self else { return }
                var resolved = ""
                if let p = placemarks?.first {
                    if let locality = p.locality, let admin = p.administrativeArea {
                        resolved = "\(locality), \(admin)"
                    } else if let name = p.name {
                        resolved = name
                    }
                }
                if resolved.isEmpty {
                    resolved = fallback
                }
                self.finishResolved(resolved)
            }
        }
    }

    private func finishResolved(_ resolved: String) {
        guard locationCompletion != nil else { return }
        isLocating = false
        let completion = locationCompletion
        locationCompletion = nil
        completion?(.success(resolved))
    }

    // MARK: - UI-test authorization injection
    //
    // Honored only when WX_UI_TEST=1 and WX_LOCATION_AUTH is set. This injects
    // CLAuthorizationStatus (and, optionally, the CLError a failed request would
    // deliver). It does not draw UI. The host TCC database is not writable from
    // XCUITest without changing this Mac's privacy settings, so the real prompt
    // cannot be driven. Darwin notes named
    // com.mwirges.wx.uitest.location-auth.<status> flip the injected status.

    private static func injectedStatus() -> CLAuthorizationStatus? {
        guard ProcessInfo.processInfo.environment["WX_UI_TEST"] == "1" else { return nil }
        guard let raw = ProcessInfo.processInfo.environment["WX_LOCATION_AUTH"] else { return nil }
        return status(from: raw)
    }

    private static func injectedFailCode() -> CLError.Code? {
        guard ProcessInfo.processInfo.environment["WX_UI_TEST"] == "1" else { return nil }
        switch ProcessInfo.processInfo.environment["WX_LOCATION_FAIL"]?
            .trimmingCharacters(in: .whitespacesAndNewlines) {
        case "denied": return .denied
        case "locationUnknown": return .locationUnknown
        case "network": return .network
        case "headingFailure": return .headingFailure
        case "promptDeclined": return .promptDeclined
        default: return nil
        }
    }

    private static func status(from raw: String) -> CLAuthorizationStatus? {
        switch raw.trimmingCharacters(in: .whitespacesAndNewlines) {
        case "denied": return .denied
        case "restricted": return .restricted
        case "notDetermined": return .notDetermined
        // macOS reports a When-In-Use grant as .authorizedAlways.
        case "authorized", "authorizedWhenInUse", "authorizedAlways": return .authorizedAlways
        default: return nil
        }
    }

    fileprivate func applyInjectedAuthorization(_ token: String) {
        guard uiTestAuthInjected, let status = Self.status(from: token) else { return }
        handleAuthorization(status)
    }

    private func installAuthInjectionObserver() {
        let center = CFNotificationCenterGetDarwinNotifyCenter()
        let observer = Unmanaged.passUnretained(self).toOpaque()
        let tokens = [
            "denied", "restricted", "notDetermined",
            "authorized", "authorizedWhenInUse", "authorizedAlways",
        ]
        for token in tokens {
            let name = "com.mwirges.wx.uitest.location-auth.\(token)" as CFString
            CFNotificationCenterAddObserver(
                center,
                observer,
                uiTestLocationAuthNote,
                name,
                nil,
                .deliverImmediately
            )
        }
    }
}

private func uiTestLocationAuthNote(
    _ center: CFNotificationCenter?,
    _ observer: UnsafeMutableRawPointer?,
    _ name: CFNotificationName?,
    _ object: UnsafeRawPointer?,
    _ userInfo: CFDictionary?
) {
    guard let observer else { return }
    _ = (center, object, userInfo)
    let token = ((name?.rawValue as String?) ?? "").split(separator: ".").last.map(String.init) ?? ""
    let manager = Unmanaged<LocationManager>.fromOpaque(observer).takeUnretainedValue()
    Task { @MainActor in
        manager.applyInjectedAuthorization(token)
    }
}
