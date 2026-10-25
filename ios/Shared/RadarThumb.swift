import CoreGraphics
import Foundation
import ImageIO
import UniformTypeIdentifiers

/// One downsampled radar frame. The widget never sees the loop or a full-resolution scan.
enum RadarThumb {
    /// A PNG that is already small enough to decode in the widget.
    static func accepted(_ data: Data) -> Data? {
        guard !data.isEmpty, data.count <= WxWidgetLimits.radarMaxBytes else { return nil }
        guard let edge = pixelEdge(data), edge <= WxWidgetLimits.radarMaxPixel else { return nil }
        return data
    }

    static func png(base64: String) -> Data? {
        let trimmed = base64.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty,
              let data = Data(base64Encoded: trimmed, options: .ignoreUnknownCharacters),
              !data.isEmpty else {
            return nil
        }
        return png(data: data)
    }

    /// ImageIO thumbnail of a single frame. Returns nil when the result is still too big.
    static func png(data: Data) -> Data? {
        let sourceOptions: [CFString: Any] = [kCGImageSourceShouldCache: false]
        guard let source = CGImageSourceCreateWithData(data as CFData, sourceOptions as CFDictionary) else {
            return nil
        }
        let thumbOptions: [CFString: Any] = [
            kCGImageSourceThumbnailMaxPixelSize: WxWidgetLimits.radarMaxPixel,
            kCGImageSourceCreateThumbnailFromImageAlways: true,
            kCGImageSourceCreateThumbnailWithTransform: true,
            kCGImageSourceShouldCache: false,
        ]
        guard let image = CGImageSourceCreateThumbnailAtIndex(source, 0, thumbOptions as CFDictionary) else {
            return nil
        }
        guard max(image.width, image.height) <= WxWidgetLimits.radarMaxPixel else { return nil }
        let out = NSMutableData()
        guard let dest = CGImageDestinationCreateWithData(out, UTType.png.identifier as CFString, 1, nil) else {
            return nil
        }
        CGImageDestinationAddImage(dest, image, nil)
        guard CGImageDestinationFinalize(dest) else { return nil }
        return accepted(out as Data)
    }

    static func pixelEdge(_ data: Data) -> Int? {
        let sourceOptions: [CFString: Any] = [kCGImageSourceShouldCache: false]
        guard let source = CGImageSourceCreateWithData(data as CFData, sourceOptions as CFDictionary),
              let raw = CGImageSourceCopyPropertiesAtIndex(source, 0, nil) else {
            return nil
        }
        let props = raw as NSDictionary
        guard let width = intValue(props[kCGImagePropertyPixelWidth]),
              let height = intValue(props[kCGImagePropertyPixelHeight]),
              width > 0, height > 0 else {
            return nil
        }
        return max(width, height)
    }

    private static func intValue(_ raw: Any?) -> Int? {
        switch raw {
        case let value as Int:
            return value
        case let value as NSNumber:
            return value.intValue
        default:
            return nil
        }
    }
}
