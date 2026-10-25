import CoreLocation
import Foundation

enum IOSLocationError: LocalizedError {
    case denied
    case restricted
    case notAvailable

    var errorDescription: String? {
        switch self {
        case .denied:
            return "Location access denied."
        case .restricted:
            return "Location access restricted."
        case .notAvailable:
            return "Unable to determine the current place."
        }
    }
}

/// When-In-Use, one shot. Never requests Always and never starts significant-change updates.
@MainActor
final class IOSLocation: NSObject, CLLocationManagerDelegate {
    private let manager = CLLocationManager()
    private var pending: CheckedContinuation<String, Error>?
    private var timeout: Task<Void, Never>?

    override init() {
        super.init()
        manager.delegate = self
        manager.desiredAccuracy = kCLLocationAccuracyHundredMeters
    }

    func oneShot() async throws -> String {
        if ProcessInfo.processInfo.environment["WX_LOCATION_MODE"] == "denied" {
            throw IOSLocationError.denied
        }
        return try await withCheckedThrowingContinuation { continuation in
            if pending != nil {
                continuation.resume(throwing: IOSLocationError.notAvailable)
                return
            }
            pending = continuation
            timeout = Task { [weak self] in
                try? await Task.sleep(nanoseconds: 20_000_000_000)
                self?.finish(.failure(IOSLocationError.notAvailable))
            }
            start()
        }
    }

    private func start() {
        switch manager.authorizationStatus {
        case .notDetermined:
            manager.requestWhenInUseAuthorization()
        case .authorizedWhenInUse, .authorizedAlways:
            manager.requestLocation()
        case .denied:
            finish(.failure(IOSLocationError.denied))
        case .restricted:
            finish(.failure(IOSLocationError.restricted))
        @unknown default:
            finish(.failure(IOSLocationError.notAvailable))
        }
    }

    private func finish(_ result: Result<String, Error>) {
        guard let continuation = pending else { return }
        pending = nil
        timeout?.cancel()
        timeout = nil
        continuation.resume(with: result)
    }

    nonisolated func locationManagerDidChangeAuthorization(_ manager: CLLocationManager) {
        let status = manager.authorizationStatus
        Task { @MainActor in
            guard self.pending != nil else { return }
            switch status {
            case .authorizedWhenInUse, .authorizedAlways:
                manager.requestLocation()
            case .denied:
                self.finish(.failure(IOSLocationError.denied))
            case .restricted:
                self.finish(.failure(IOSLocationError.restricted))
            case .notDetermined:
                break
            @unknown default:
                self.finish(.failure(IOSLocationError.notAvailable))
            }
        }
    }

    nonisolated func locationManager(_ manager: CLLocationManager, didUpdateLocations locations: [CLLocation]) {
        guard let location = locations.last else { return }
        let fallback = String(format: "%.4f,%.4f", location.coordinate.latitude, location.coordinate.longitude)
        CLGeocoder().reverseGeocodeLocation(location) { marks, _ in
            let resolved: String
            if let place = marks?.first {
                if let locality = place.locality, let admin = place.administrativeArea {
                    resolved = "\(locality), \(admin)"
                } else if let name = place.name, !name.isEmpty {
                    resolved = name
                } else {
                    resolved = fallback
                }
            } else {
                resolved = fallback
            }
            Task { @MainActor in
                self.finish(.success(resolved))
            }
        }
    }

    nonisolated func locationManager(_ manager: CLLocationManager, didFailWithError error: Error) {
        Task { @MainActor in
            self.finish(.failure(error))
        }
    }
}
