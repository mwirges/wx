import MapKit
import SwiftUI
import UIKit

final class IOSRadarMapOverlay: NSObject, MKOverlay {
    let coordinate: CLLocationCoordinate2D
    let boundingMapRect: MKMapRect
    var cgImage: CGImage
    let bbox: RadarBBox

    init(cgImage: CGImage, bbox: RadarBBox, center: RadarCenter) {
        self.cgImage = cgImage
        self.bbox = bbox
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

final class IOSRadarMapOverlayRenderer: MKOverlayRenderer {
    override func draw(_ mapRect: MKMapRect, zoomScale: MKZoomScale, in context: CGContext) {
        guard let radarOverlay = self.overlay as? IOSRadarMapOverlay else { return }
        let rect = self.rect(for: radarOverlay.boundingMapRect)

        context.saveGState()
        context.translateBy(x: rect.origin.x, y: rect.origin.y + rect.size.height)
        context.scaleBy(x: 1.0, y: -1.0)
        context.draw(radarOverlay.cgImage, in: CGRect(x: 0, y: 0, width: rect.size.width, height: rect.size.height))
        context.restoreGState()
    }
}

/// Custom MKMapView that records genuine user touch interactions and triggers initial region fit once laid out.
final class IOSWxMapView: MKMapView {
    var userInteracted = false
    var onDidLayoutForInitialRegion: (() -> Void)?
    private var didApplyInitialLayout = false

    override func touchesBegan(_ touches: Set<UITouch>, with event: UIEvent?) {
        userInteracted = true
        super.touchesBegan(touches, with: event)
    }

    override func layoutSubviews() {
        super.layoutSubviews()
        if !didApplyInitialLayout, bounds.width > 30, bounds.height > 30 {
            didApplyInitialLayout = true
            onDidLayoutForInitialRegion?()
        }
    }
}

struct IOSRadarMapView: UIViewRepresentable {
    let payload: RadarPayload
    let cgImage: CGImage
    let recenterID: Int
    var viewportRevision: Int = 0
    var targetRadiusKm: Double = 200
    var onBBoxNeedsUpdate: ((RadarBBox) -> Void)? = nil

    func makeCoordinator() -> Coordinator {
        Coordinator(self)
    }

    func makeUIView(context: Context) -> IOSWxMapView {
        let mapView = IOSWxMapView()
        mapView.delegate = context.coordinator
        mapView.overrideUserInterfaceStyle = .dark

        let config = MKStandardMapConfiguration(elevationStyle: .flat, emphasisStyle: .muted)
        config.pointOfInterestFilter = .excludingAll
        mapView.preferredConfiguration = config

        mapView.isPitchEnabled = false
        mapView.isRotateEnabled = false
        mapView.showsCompass = false
        mapView.showsScale = false

        let coord = context.coordinator
        mapView.onDidLayoutForInitialRegion = { [weak coord, weak mapView] in
            guard let coord, let mv = mapView else { return }
            coord.applyRegionIfPossible(mapView: mv, animated: false)
        }

        context.coordinator.update(
            mapView: mapView,
            payload: payload,
            cgImage: cgImage,
            forceCenter: true,
            animatedRegion: false
        )
        return mapView
    }

    func updateUIView(_ mapView: IOSWxMapView, context: Context) {
        context.coordinator.parent = self
        let recentered = context.coordinator.lastRecenterID != recenterID
        let revChanged = context.coordinator.lastViewportRevision != viewportRevision
        context.coordinator.lastRecenterID = recenterID
        context.coordinator.lastViewportRevision = viewportRevision
        context.coordinator.update(
            mapView: mapView,
            payload: payload,
            cgImage: cgImage,
            forceCenter: recentered || revChanged,
            animatedRegion: !recentered && revChanged
        )
    }

    final class Coordinator: NSObject, MKMapViewDelegate {
        var parent: IOSRadarMapView
        var currentOverlay: IOSRadarMapOverlay?
        var currentPin: MKPointAnnotation?
        var lastLocationKey: String = ""
        var lastRecenterID: Int
        var lastViewportRevision: Int
        var isSettingRegion = false
        var debounceWorkItem: DispatchWorkItem?
        var regionToken = 0

        init(_ parent: IOSRadarMapView) {
            self.parent = parent
            self.lastRecenterID = parent.recenterID
            self.lastViewportRevision = parent.viewportRevision
        }

        func update(
            mapView: IOSWxMapView,
            payload: RadarPayload,
            cgImage: CGImage,
            forceCenter: Bool,
            animatedRegion: Bool
        ) {
            guard let bbox = payload.bbox, let center = payload.center else { return }

            let locationKey = "\(payload.location):\(payload.product)"
            let locationChanged = (lastLocationKey != locationKey)
            lastLocationKey = locationKey

            if let existing = currentOverlay,
               !locationChanged,
               abs(existing.coordinate.latitude - center.lat) < 0.0001,
               abs(existing.coordinate.longitude - center.lon) < 0.0001,
               abs(existing.bbox.minLat - bbox.minLat) < 0.0001,
               abs(existing.bbox.maxLat - bbox.maxLat) < 0.0001,
               abs(existing.bbox.minLon - bbox.minLon) < 0.0001,
               abs(existing.bbox.maxLon - bbox.maxLon) < 0.0001 {
                existing.cgImage = cgImage
                if let renderer = mapView.renderer(for: existing) as? IOSRadarMapOverlayRenderer {
                    renderer.setNeedsDisplay()
                }
            } else {
                if let existing = currentOverlay {
                    mapView.removeOverlay(existing)
                }
                let overlay = IOSRadarMapOverlay(cgImage: cgImage, bbox: bbox, center: center)
                currentOverlay = overlay
                mapView.addOverlay(overlay, level: .aboveRoads)
            }

            if locationChanged || currentPin == nil {
                if let existingPin = currentPin {
                    mapView.removeAnnotation(existingPin)
                }
                let pin = MKPointAnnotation()
                pin.coordinate = CLLocationCoordinate2D(latitude: center.lat, longitude: center.lon)
                pin.title = payload.location
                currentPin = pin
                mapView.addAnnotation(pin)
            }

            if forceCenter || locationChanged {
                mapView.userInteracted = false
                debounceWorkItem?.cancel()
                applyRegionIfPossible(mapView: mapView, animated: animatedRegion)
            }
        }

        func applyRegionIfPossible(mapView: IOSWxMapView, animated: Bool) {
            guard mapView.bounds.width > 30, mapView.bounds.height > 30 else { return }
            guard let center = parent.payload.center else { return }

            isSettingRegion = true
            regionToken += 1
            let token = regionToken

            let centerCoord = CLLocationCoordinate2D(latitude: center.lat, longitude: center.lon)
            let radiusKm = max(80.0, parent.payload.radiusKm ?? parent.targetRadiusKm)
            let latDelta = (radiusKm * 2.0) / 111.0
            let region = MKCoordinateRegion(
                center: centerCoord,
                span: MKCoordinateSpan(latitudeDelta: latDelta, longitudeDelta: latDelta)
            )
            mapView.setRegion(region, animated: animated)
            let settleDelay: TimeInterval = animated ? 0.75 : 0.25
            DispatchQueue.main.asyncAfter(deadline: .now() + settleDelay) { [weak self] in
                guard let self, self.regionToken == token else { return }
                self.isSettingRegion = false
            }
        }

        func mapView(_ mapView: MKMapView, regionDidChangeAnimated animated: Bool) {
            guard let wxMap = mapView as? IOSWxMapView, wxMap.userInteracted else { return }
            guard !isSettingRegion, parent.onBBoxNeedsUpdate != nil else { return }
            guard mapView.bounds.width > 30, mapView.bounds.height > 30 else { return }

            let region = mapView.region
            guard region.span.latitudeDelta > 0.001, region.span.longitudeDelta > 0.001 else { return }

            let visMinLat = region.center.latitude - region.span.latitudeDelta / 2.0
            let visMaxLat = region.center.latitude + region.span.latitudeDelta / 2.0
            let visMinLon = region.center.longitude - region.span.longitudeDelta / 2.0
            let visMaxLon = region.center.longitude + region.span.longitudeDelta / 2.0

            if let overlay = currentOverlay {
                let marginLat = region.span.latitudeDelta * 0.02
                let marginLon = region.span.longitudeDelta * 0.02
                let uncovered = (visMinLat < overlay.bbox.minLat - marginLat) ||
                    (visMaxLat > overlay.bbox.maxLat + marginLat) ||
                    (visMinLon < overlay.bbox.minLon - marginLon) ||
                    (visMaxLon > overlay.bbox.maxLon + marginLon)
                let overzoomed = region.span.latitudeDelta < (overlay.bbox.maxLat - overlay.bbox.minLat) * 0.35
                if !uncovered && !overzoomed { return }
            }

            debounceWorkItem?.cancel()
            let item = DispatchWorkItem { [weak self, weak mapView] in
                guard let self, let mv = mapView else { return }
                let cur = mv.region
                let padLat = cur.span.latitudeDelta * 0.20
                let padLon = cur.span.longitudeDelta * 0.20
                let newBBox = RadarBBox(
                    minLat: max(-85.0, cur.center.latitude - cur.span.latitudeDelta / 2.0 - padLat),
                    minLon: max(-179.0, cur.center.longitude - cur.span.longitudeDelta / 2.0 - padLon),
                    maxLat: min(85.0, cur.center.latitude + cur.span.latitudeDelta / 2.0 + padLat),
                    maxLon: min(179.0, cur.center.longitude + cur.span.longitudeDelta / 2.0 + padLon)
                )
                self.parent.onBBoxNeedsUpdate?(newBBox)
            }
            debounceWorkItem = item
            DispatchQueue.main.asyncAfter(deadline: .now() + 0.35, execute: item)
        }

        func mapView(_ mapView: MKMapView, rendererFor overlay: MKOverlay) -> MKOverlayRenderer {
            if let radarOverlay = overlay as? IOSRadarMapOverlay {
                return IOSRadarMapOverlayRenderer(overlay: radarOverlay)
            }
            return MKOverlayRenderer(overlay: overlay)
        }

        func mapView(_ mapView: MKMapView, viewFor annotation: MKAnnotation) -> MKAnnotationView? {
            guard !(annotation is MKUserLocation) else { return nil }
            let id = "IOSRadarTargetPin"
            let view = (mapView.dequeueReusableAnnotationView(withIdentifier: id) as? MKMarkerAnnotationView)
                ?? MKMarkerAnnotationView(annotation: annotation, reuseIdentifier: id)
            view.annotation = annotation
            view.canShowCallout = true
            view.markerTintColor = UIColor(red: 0.22, green: 0.88, blue: 1.0, alpha: 1.0)
            view.glyphImage = UIImage(systemName: "location.fill")
            return view
        }
    }
}
