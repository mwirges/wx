import AppKit
import ImageIO
import UniformTypeIdentifiers

@MainActor
enum RadarExportService {
    /// Copies the given NSImage to the system general pasteboard.
    @discardableResult
    static func copyImageToPasteboard(_ image: NSImage) -> Bool {
        let pb = NSPasteboard.general
        pb.clearContents()
        return pb.writeObjects([image])
    }

    /// Prompts the user with an NSSavePanel to save the image as a PNG file.
    static func saveImageAsPNG(_ image: NSImage, suggestedFilename: String) {
        guard let tiff = image.tiffRepresentation,
              let rep = NSBitmapImageRep(data: tiff),
              let pngData = rep.representation(using: .png, properties: [:]) else {
            return
        }

        let panel = NSSavePanel()
        panel.title = "Save Radar Frame"
        panel.nameFieldStringValue = sanitizeFilename(suggestedFilename, ext: "png")
        panel.allowedContentTypes = [.png]
        panel.canCreateDirectories = true

        if panel.runModal() == .OK, let url = panel.url {
            try? pngData.write(to: url)
        }
    }

    /// Compiles an array of NSImages into an animated GIF.
    static func createAnimatedGIF(images: [NSImage], frameDelay: Double = 0.38) -> Data? {
        guard !images.isEmpty else { return nil }

        let data = NSMutableData()
        guard let destination = CGImageDestinationCreateWithData(
            data as CFMutableData,
            UTType.gif.identifier as CFString,
            images.count,
            nil
        ) else {
            return nil
        }

        let gifFileProperties: [CFString: Any] = [
            kCGImagePropertyGIFDictionary: [
                kCGImagePropertyGIFLoopCount: 0 // infinite loop
            ]
        ]
        CGImageDestinationSetProperties(destination, gifFileProperties as CFDictionary)

        for (idx, img) in images.enumerated() {
            var rect = NSRect(origin: .zero, size: img.size)
            guard let cgImage = img.cgImage(forProposedRect: &rect, context: nil, hints: nil) else {
                continue
            }

            // Dwell on the final frame (live radar scan)
            let isLast = (idx == images.count - 1)
            let delay = isLast ? (frameDelay * 2.2) : frameDelay

            let frameProperties: [CFString: Any] = [
                kCGImagePropertyGIFDictionary: [
                    kCGImagePropertyGIFDelayTime: delay
                ]
            ]
            CGImageDestinationAddImage(destination, cgImage, frameProperties as CFDictionary)
        }

        guard CGImageDestinationFinalize(destination) else { return nil }
        return data as Data
    }

    /// Prompts the user with an NSSavePanel to export an animated GIF loop.
    static func exportLoopAsGIF(images: [NSImage], frameDelay: Double = 0.38, suggestedFilename: String) {
        guard let gifData = createAnimatedGIF(images: images, frameDelay: frameDelay) else {
            return
        }

        let panel = NSSavePanel()
        panel.title = "Export Animated Radar Loop"
        panel.nameFieldStringValue = sanitizeFilename(suggestedFilename, ext: "gif")
        panel.allowedContentTypes = [.gif]
        panel.canCreateDirectories = true

        if panel.runModal() == .OK, let url = panel.url {
            try? gifData.write(to: url)
        }
    }

    private static func sanitizeFilename(_ name: String, ext: String) -> String {
        let base = name
            .replacingOccurrences(of: "/", with: "-")
            .replacingOccurrences(of: ":", with: "-")
            .replacingOccurrences(of: " ", with: "-")
            .trimmingCharacters(in: .whitespacesAndNewlines)
        if base.lowercased().hasSuffix(".\(ext)") {
            return base
        }
        return "\(base).\(ext)"
    }
}
