import SwiftUI
import AppKit
import MapKit

/// Overlay representing a geographic radar frame image over an exact WGS84 bounding box.
final class RadarMapOverlay: NSObject, MKOverlay {
    let coordinate: CLLocationCoordinate2D
    let boundingMapRect: MKMapRect
    var image: NSImage
    let bbox: RadarBBox

    init(image: NSImage, bbox: RadarBBox, center: RadarCenter) {
        self.image = image
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

/// Custom MKMapView subclass that detects frame and live window resize events to trigger viewport updates.
final class WxMapView: MKMapView {
    var onViewportChanged: (() -> Void)?

    override func viewDidEndLiveResize() {
        super.viewDidEndLiveResize()
        onViewportChanged?()
    }

    override func setFrameSize(_ newSize: NSSize) {
        let oldSize = frame.size
        super.setFrameSize(newSize)
        if abs(newSize.width - oldSize.width) > 4 || abs(newSize.height - oldSize.height) > 4 {
            onViewportChanged?()
        }
    }
}

/// High-resolution vector map view displaying transparent radar overlays with Apple Maps dark basemap.
/// Automatically expands the radar composite overlay to cover 100% of the visible map viewport.
struct RadarMapView: NSViewRepresentable {
    let payload: RadarPayload
    let image: NSImage
    let recenterID: Int
    var onVisibleRadiusChanged: ((Double) -> Void)? = nil
    var onBBoxNeedsUpdate: ((RadarBBox) -> Void)? = nil

    func makeCoordinator() -> Coordinator {
        Coordinator(self)
    }

    func makeNSView(context: Context) -> WxMapView {
        let mapView = WxMapView()
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
        mapView.autoresizingMask = [.width, .height]

        let coordinator = context.coordinator
        mapView.onViewportChanged = { [weak coordinator, weak mapView] in
            guard let coordinator = coordinator, let mv = mapView else { return }
            coordinator.checkViewportCoverage(mv)
        }

        context.coordinator.update(mapView: mapView, payload: payload, image: image, forceCenter: true)

        // As soon as view is added and laid out, check viewport bounds
        DispatchQueue.main.async { [weak coordinator, weak mapView] in
            guard let mv = mapView, let coord = coordinator else { return }
            coord.checkViewportCoverage(mv)
        }

        return mapView
    }

    func updateNSView(_ mapView: WxMapView, context: Context) {
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
        var debounceWorkItem: DispatchWorkItem?
        var isSettingRegion: Bool = false

        init(_ parent: RadarMapView) {
            self.parent = parent
            self.lastRecenterID = parent.recenterID
        }

        func update(mapView: MKMapView, payload: RadarPayload, image: NSImage, forceCenter: Bool) {
            guard let bbox = payload.bbox, let center = payload.center else { return }

            let locationKey = "\(payload.location):\(payload.product)"
            let locationChanged = (lastLocationKey != locationKey)
            lastLocationKey = locationKey

            // If the overlay exists and spatial bounds match, update image in-place for flicker-free frame transitions
            if let existing = currentOverlay,
               !locationChanged,
               abs(existing.coordinate.latitude - center.lat) < 0.0001,
               abs(existing.coordinate.longitude - center.lon) < 0.0001,
               abs(existing.bbox.minLat - bbox.minLat) < 0.0001,
               abs(existing.bbox.maxLat - bbox.maxLat) < 0.0001 {
                existing.image = image
                if let renderer = mapView.renderer(for: existing) as? RadarMapOverlayRenderer {
                    renderer.setNeedsDisplay()
                }
            } else {
                if let existing = currentOverlay {
                    mapView.removeOverlay(existing)
                }
                let newOverlay = RadarMapOverlay(image: image, bbox: bbox, center: center)
                currentOverlay = newOverlay
                mapView.addOverlay(newOverlay, level: .aboveRoads)
            }

            // Center target pin only if location changed or annotation missing
            if locationChanged || currentLocationAnnotation == nil {
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
            }

            if forceCenter || locationChanged {
                isSettingRegion = true
                let centerCoord = CLLocationCoordinate2D(latitude: center.lat, longitude: center.lon)
                let targetRadiusKm = max(100.0, payload.radiusKm ?? 250.0)
                let latDelta = (targetRadiusKm * 2.0) / 111.0
                let span = MKCoordinateSpan(
                    latitudeDelta: latDelta,
                    longitudeDelta: latDelta
                )
                let region = MKCoordinateRegion(center: centerCoord, span: span)
                mapView.setRegion(region, animated: !forceCenter)
                DispatchQueue.main.asyncAfter(deadline: .now() + 0.35) { [weak self, weak mapView] in
                    guard let self = self, let mv = mapView else { return }
                    self.isSettingRegion = false
                    self.checkViewportCoverage(mv)
                }
            } else {
                checkViewportCoverage(mapView)
            }
        }

        func mapView(_ mapView: MKMapView, regionDidChangeAnimated animated: Bool) {
            guard !isSettingRegion else { return }
            checkViewportCoverage(mapView)
        }

        func checkViewportCoverage(_ mapView: MKMapView) {
            guard !isSettingRegion else { return }
            guard mapView.bounds.width > 30, mapView.bounds.height > 30 else { return }

            let visibleRegion = mapView.convert(mapView.bounds, toRegionFrom: mapView)
            guard visibleRegion.span.latitudeDelta > 0.001, visibleRegion.span.longitudeDelta > 0.001 else { return }

            let latDelta = visibleRegion.span.latitudeDelta
            let visibleRadiusKm = (latDelta * 111.0) / 2.0
            parent.onVisibleRadiusChanged?(visibleRadiusKm)

            let visMinLat = visibleRegion.center.latitude - visibleRegion.span.latitudeDelta / 2.0
            let visMaxLat = visibleRegion.center.latitude + visibleRegion.span.latitudeDelta / 2.0
            let visMinLon = visibleRegion.center.longitude - visibleRegion.span.longitudeDelta / 2.0
            let visMaxLon = visibleRegion.center.longitude + visibleRegion.span.longitudeDelta / 2.0

            if let overlay = currentOverlay {
                // Buffer margin check: if visible map is completely enclosed within overlay bounds with safety margin, no update needed
                let marginLat = visibleRegion.span.latitudeDelta * 0.02
                let marginLon = visibleRegion.span.longitudeDelta * 0.02
                let isUncovered = (visMinLat < overlay.bbox.minLat - marginLat) ||
                                  (visMaxLat > overlay.bbox.maxLat + marginLat) ||
                                  (visMinLon < overlay.bbox.minLon - marginLon) ||
                                  (visMaxLon > overlay.bbox.maxLon + marginLon)
                let isOverzoomed = (visibleRegion.span.latitudeDelta < (overlay.bbox.maxLat - overlay.bbox.minLat) * 0.35)

                if !isUncovered && !isOverzoomed {
                    return
                }
            }

            debounceWorkItem?.cancel()
            let item = DispatchWorkItem { [weak self, weak mapView] in
                guard let self = self, let mv = mapView else { return }
                guard mv.bounds.width > 30, mv.bounds.height > 30 else { return }
                let curRegion = mv.convert(mv.bounds, toRegionFrom: mv)
                guard curRegion.span.latitudeDelta > 0.001, curRegion.span.longitudeDelta > 0.001 else { return }

                // Padded bounding box: 20% margin around all 4 edges so panning doesn't immediately uncover edge
                let padLat = curRegion.span.latitudeDelta * 0.20
                let padLon = curRegion.span.longitudeDelta * 0.20

                let newBBox = RadarBBox(
                    minLat: max(-85.0, curRegion.center.latitude - curRegion.span.latitudeDelta / 2.0 - padLat),
                    minLon: max(-179.0, curRegion.center.longitude - curRegion.span.longitudeDelta / 2.0 - padLon),
                    maxLat: min(85.0, curRegion.center.latitude + curRegion.span.latitudeDelta / 2.0 + padLat),
                    maxLon: min(179.0, curRegion.center.longitude + curRegion.span.longitudeDelta / 2.0 + padLon)
                )
                self.parent.onBBoxNeedsUpdate?(newBBox)
            }
            debounceWorkItem = item
            DispatchQueue.main.asyncAfter(deadline: .now() + 0.35, execute: item)
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
