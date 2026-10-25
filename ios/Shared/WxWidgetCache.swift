import Foundation

enum WxCacheDirectories {
    /// App Group container when the simulator (or a signed build) provides one.
    /// Otherwise the app caches directory, which the widget cannot see.
    static func resolve(groupURL: URL?, cachesURL: URL) -> URL {
        if let groupURL {
            return groupURL
        }
        return cachesURL.appendingPathComponent(WxAppGroup.fallbackFolder, isDirectory: true)
    }
}

enum WxWidgetCache {
    static func liveDirectory(fileManager: FileManager = .default) -> URL {
        let caches = fileManager.urls(for: .cachesDirectory, in: .userDomainMask).first
            ?? URL(fileURLWithPath: NSTemporaryDirectory(), isDirectory: true)
        return WxCacheDirectories.resolve(
            groupURL: fileManager.containerURL(forSecurityApplicationGroupIdentifier: WxAppGroup.id),
            cachesURL: caches
        )
    }

    static func read(from directory: URL) -> (WxWidgetSnapshot, Data?) {
        let snapshotURL = directory.appendingPathComponent(WxAppGroup.snapshotFile)
        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .iso8601
        let snapshot: WxWidgetSnapshot
        if let data = try? Data(contentsOf: snapshotURL),
           let decoded = try? decoder.decode(WxWidgetSnapshot.self, from: data) {
            snapshot = decoded
        } else {
            snapshot = .placeholder
        }
        let radarURL = directory.appendingPathComponent(WxAppGroup.radarFile)
        let radar = (try? Data(contentsOf: radarURL)).flatMap { RadarThumb.accepted($0) }
        return (snapshot, radar)
    }

    static func write(
        _ snapshot: WxWidgetSnapshot,
        radarPNG: Data?,
        to directory: URL,
        fileManager: FileManager = .default
    ) throws {
        try fileManager.createDirectory(at: directory, withIntermediateDirectories: true)
        let encoder = JSONEncoder()
        encoder.outputFormatting = [.sortedKeys]
        encoder.dateEncodingStrategy = .iso8601
        let data = try encoder.encode(snapshot)
        let snapshotURL = directory.appendingPathComponent(WxAppGroup.snapshotFile)
        try data.write(to: snapshotURL, options: .atomic)
        let radarURL = directory.appendingPathComponent(WxAppGroup.radarFile)
        let stored = radarPNG.flatMap { RadarThumb.accepted($0) ?? RadarThumb.png(data: $0) }
        if let stored {
            try stored.write(to: radarURL, options: .atomic)
        } else if fileManager.fileExists(atPath: radarURL.path) {
            try fileManager.removeItem(at: radarURL)
        }
    }
}
