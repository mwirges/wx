import SwiftUI
import AppKit
import MapKit

/// Overlay representing a geographic radar frame image over an exact WGS84 bounding box.
final class RadarMapOverlay: NSObject, MKOverlay {
    let coordinate: CLLocationCoordinate2D
    let boundingMapRect: MKMapRect
    let image: NSImage

    init(image: NSImage, bbox: RadarBBox, center: RadarCenter) {
        self.image = image
        self.coordinate = CLLocationCoordinate2D(latitude: center.lat, longitude: center.lon)

        let topLeft = MKMapPoint(CLLocationCoordinate2D(latitude: bbox.maxLat, longitude: bbox.minLon))
        let bottomRight = MKMapPoint(CLLocationCoordinate2D(latitude: bbox.minLat, longitude: bbox.maxLon))

        self.boundingMapRect = MKMapRect(
            x: min(topLeft.x, bottomRight.x),
            y: min(topLeft.y, bottomRight.y),
            width: abs(bottomRight.x - topLeft.x),
            height: abs(bottomRight.y - topLeft.y)
        )
        super.init()
    }
}

/// Renderer that draws the transparent radar image onto the map rect.
final class RadarMapOverlayRenderer: MKOverlayRenderer {
    override func draw(_ mapRect: MKMapRect, zoomScale: MKZoomScale, in context: CGContext) {
        guard let radarOverlay = self.overlay as? RadarMapOverlay else { return }
        var imageRect = CGRect(x: 0, y: 0, width: radarOverlay.image.size.width, height: radarOverlay.image.size.height)
        guard let cgImage = radarOverlay.image.cgImage(forProposedRect: &imageRect, context: nil, hints: nil) else { return }

        let rect = self.rect(for: radarOverlay.boundingMapRect)

        context.saveGState()
        // Quartz 2D drawing coordinates have origin at bottom-left; flip vertically to align
        // top-down raster coordinates with MapKit's Mercator projection.
        context.translateBy(x: rect.origin.x, y: rect.origin.y + rect.size.height)
        context.scaleBy(x: 1.0, y: -1.0)
        context.draw(cgImage, in: CGRect(x: 0, y: 0, width: rect.size.width, height: rect.size.height))
        context.restoreGState()
    }
}

/// High-resolution vector map view displaying transparent radar overlays with Apple Maps dark basemap.
struct RadarMapView: NSViewRepresentable {
    let payload: RadarPayload
    let image: NSImage
    let recenterID: Int

    func makeCoordinator() -> Coordinator {
        Coordinator(self)
    }

    func makeNSView(context: Context) -> MKMapView {
        let mapView = MKMapView()
        mapView.delegate = context.coordinator
        mapView.appearance = NSAppearance(named: .darkAqua)

        let config = MKStandardMapConfiguration(elevationStyle: .flat, emphasisStyle: .muted)
        config.pointOfInterestFilter = .excludingAll
        mapView.preferredConfiguration = config

        mapView.isPitchEnabled = false
        mapView.isRotateEnabled = false
        mapView.showsZoomControls = true
        mapView.showsCompass = false
        mapView.showsScale = false

        context.coordinator.update(mapView: mapView, payload: payload, image: image, forceCenter: true)
        return mapView
    }

    func updateNSView(_ mapView: MKMapView, context: Context) {
        context.coordinator.parent = self
        let shouldForceCenter = context.coordinator.lastRecenterID != recenterID
        context.coordinator.lastRecenterID = recenterID
        context.coordinator.update(mapView: mapView, payload: payload, image: image, forceCenter: shouldForceCenter)
    }

    final class Coordinator: NSObject, MKMapViewDelegate {
        var parent: RadarMapView
        var currentOverlay: RadarMapOverlay?
        var currentLocationAnnotation: MKPointAnnotation?
        var lastLocationKey: String = ""
        var lastRecenterID: Int = 0

        init(_ parent: RadarMapView) {
            self.parent = parent
            self.lastRecenterID = parent.recenterID
        }

        func update(mapView: MKMapView, payload: RadarPayload, image: NSImage, forceCenter: Bool) {
            guard let bbox = payload.bbox, let center = payload.center else { return }

            let locationKey = "\(payload.location):\(payload.product)"
            let locationChanged = (lastLocationKey != locationKey)
            lastLocationKey = locationKey

            // Swap radar overlay
            if let existing = currentOverlay {
                mapView.removeOverlay(existing)
            }
            let newOverlay = RadarMapOverlay(image: image, bbox: bbox, center: center)
            currentOverlay = newOverlay
            mapView.addOverlay(newOverlay, level: .aboveRoads)

            // Center target pin
            if let existingAnnotation = currentLocationAnnotation {
                mapView.removeAnnotation(existingAnnotation)
            }
            let pin = MKPointAnnotation()
            pin.coordinate = CLLocationCoordinate2D(latitude: center.lat, longitude: center.lon)
            pin.title = payload.location
            if let station = payload.station, !station.isEmpty {
                pin.subtitle = "Radar Station: \(station)"
            }
            currentLocationAnnotation = pin
            mapView.addAnnotation(pin)

            if forceCenter || locationChanged {
                let centerCoord = CLLocationCoordinate2D(latitude: center.lat, longitude: center.lon)
                let span = MKCoordinateSpan(
                    latitudeDelta: (bbox.maxLat - bbox.minLat) * 1.05,
                    longitudeDelta: (bbox.maxLon - bbox.minLon) * 1.05
                )
                let region = MKCoordinateRegion(center: centerCoord, span: span)
                mapView.setRegion(region, animated: !forceCenter)
            }
        }

        func mapView(_ mapView: MKMapView, rendererFor overlay: MKOverlay) -> MKOverlayRenderer {
            if let radarOverlay = overlay as? RadarMapOverlay {
                return RadarMapOverlayRenderer(overlay: radarOverlay)
            }
            return MKOverlayRenderer(overlay: overlay)
        }

        func mapView(_ mapView: MKMapView, viewFor annotation: MKAnnotation) -> MKAnnotationView? {
            guard !(annotation is MKUserLocation) else { return nil }
            let identifier = "RadarTargetLocation"
            var view = mapView.dequeueReusableAnnotationView(withIdentifier: identifier) as? MKMarkerAnnotationView
            if view == nil {
                view = MKMarkerAnnotationView(annotation: annotation, reuseIdentifier: identifier)
                view?.canShowCallout = true
                view?.markerTintColor = NSColor(calibratedRed: 0.12, green: 0.60, blue: 1.0, alpha: 1.0)
                view?.glyphImage = NSImage(systemSymbolName: "location.fill", accessibilityDescription: "Target Location")
                view?.displayPriority = .required
            } else {
                view?.annotation = annotation
            }
            return view
        }
    }
}
