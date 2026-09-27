import Foundation
import CoreLocation
import Combine

enum LocationManagerError: LocalizedError {
    case denied
    case restricted
    case notAvailable
    case custom(String)

    var errorDescription: String? {
        switch self {
        case .denied:
            return "Location access denied. Enable in System Settings → Privacy & Security → Location Services."
        case .restricted:
            return "Location access restricted by system policy."
        case .notAvailable:
            return "Unable to determine current coordinates."
        case .custom(let msg):
            return msg
        }
    }
}

/// Manages system CoreLocation access, authorization, and reverse-geocoding for wx.
@MainActor
final class LocationManager: NSObject, ObservableObject, CLLocationManagerDelegate {
    @Published var authorizationStatus: CLAuthorizationStatus
    @Published var isLocating: Bool = false
    @Published var errorMessage: String?

    private let manager = CLLocationManager()
    private var locationCompletion: ((Result<String, Error>) -> Void)?

    override init() {
        self.authorizationStatus = manager.authorizationStatus
        super.init()
        manager.delegate = self
        manager.desiredAccuracy = kCLLocationAccuracyHundredMeters
    }

    func requestLocation(completion: @escaping (Result<String, Error>) -> Void) {
        self.locationCompletion = completion
        self.errorMessage = nil

        let status = manager.authorizationStatus
        if status == .notDetermined {
            isLocating = true
            manager.requestAlwaysAuthorization()
            manager.requestLocation()
            return
        } else if status == .denied {
            let err = LocationManagerError.denied
            self.errorMessage = err.localizedDescription
            completion(.failure(err))
            self.locationCompletion = nil
            return
        } else if status == .restricted {
            let err = LocationManagerError.restricted
            self.errorMessage = err.localizedDescription
            completion(.failure(err))
            self.locationCompletion = nil
            return
        }

        isLocating = true
        manager.requestLocation()
    }

    // MARK: - CLLocationManagerDelegate

    nonisolated func locationManagerDidChangeAuthorization(_ manager: CLLocationManager) {
        let status = manager.authorizationStatus
        Task { @MainActor in
            self.authorizationStatus = status
            if status == .authorizedAlways && self.isLocating {
                manager.requestLocation()
            } else if status == .denied || status == .restricted {
                self.isLocating = false
                let err: LocationManagerError = (status == .denied) ? .denied : .restricted
                self.errorMessage = err.localizedDescription
                self.locationCompletion?(.failure(err))
                self.locationCompletion = nil
            }
        }
    }

    nonisolated func locationManager(_ manager: CLLocationManager, didUpdateLocations locations: [CLLocation]) {
        guard let location = locations.last else { return }
        Task { @MainActor in
            self.reverseGeocode(location)
        }
    }

    nonisolated func locationManager(_ manager: CLLocationManager, didFailWithError error: Error) {
        Task { @MainActor in
            self.isLocating = false
            self.errorMessage = error.localizedDescription
            self.locationCompletion?(.failure(error))
            self.locationCompletion = nil
        }
    }

    private func reverseGeocode(_ location: CLLocation) {
        let fallback = String(format: "%.4f,%.4f", location.coordinate.latitude, location.coordinate.longitude)
        let geocoder = CLGeocoder()
        geocoder.reverseGeocodeLocation(location) { [weak self] placemarks, _ in
            Task { @MainActor in
                guard let self else { return }
                self.isLocating = false
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
                self.locationCompletion?(.success(resolved))
                self.locationCompletion = nil
            }
        }
    }
}
