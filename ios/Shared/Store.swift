import Foundation

/// Settings and the last good snapshot, shared between the app and the
/// widgets through the App Group. Nothing here leaves the phone.
enum Store {
    static let appGroup = "group.com.haugf.coconutkitchen"
    static var defaults: UserDefaults { UserDefaults(suiteName: appGroup) ?? .standard }

    private enum Key {
        static let host = "host"
        static let name = "name"
        static let token = "eventsToken"
        static let presets = "presets"
        static let snapshot = "snapshot"
    }

    /// The Pi on the home Wi-Fi. The README's address by default.
    static var host: String {
        get { defaults.string(forKey: Key.host) ?? "mirror.local:8080" }
        set { defaults.set(newValue.trimmingCharacters(in: .whitespaces), forKey: Key.host) }
    }

    /// Who this phone belongs to ("Fred" or "Ally"). Signs messages and
    /// picks which line is solid in the day drawing.
    static var name: String {
        get { defaults.string(forKey: Key.name) ?? "" }
        set { defaults.set(newValue.trimmingCharacters(in: .whitespaces), forKey: Key.name) }
    }

    /// Only needed if `eventsToken` is set in the Pi's config.json.
    static var token: String {
        get { defaults.string(forKey: Key.token) ?? "" }
        set { defaults.set(newValue.trimmingCharacters(in: .whitespaces), forKey: Key.token) }
    }

    /// The one-tap messages on the Send widget, in order.
    static var presets: [String] {
        get { defaults.stringArray(forKey: Key.presets) ?? ["Dinner’s ready", "Home soon", "Love you"] }
        set { defaults.set(newValue.map { $0.trimmingCharacters(in: .whitespaces) }.filter { !$0.isEmpty }, forKey: Key.presets) }
    }

    static var snapshot: Snapshot {
        get {
            guard let data = defaults.data(forKey: Key.snapshot),
                  let s = try? JSONDecoder.mirror.decode(Snapshot.self, from: data) else { return .empty }
            return s
        }
        set {
            let e = JSONEncoder()
            e.dateEncodingStrategy = .iso8601
            if let data = try? e.encode(newValue) { defaults.set(data, forKey: Key.snapshot) }
        }
    }
}
