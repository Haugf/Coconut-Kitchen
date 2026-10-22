import Foundation

// The shapes the Pi already serves. See the API table in AGENTS.md.

struct CalendarEvent: Codable, Hashable {
    var title: String
    var start: Date
    var end: Date?
    var allDay: Bool
    var location: String?
    var who: [String]?
}

struct CalendarResponse: Codable {
    var people: [String]?
    var events: [CalendarEvent]
    var failed: [String]?
}

struct WeatherDay: Codable, Hashable {
    var date: String
    var high: Double
    var low: Double
    var code: Int
    var rainChance: Int
}

struct Weather: Codable {
    var temp: Double
    var feelsLike: Double
    var code: Int
    var units: String
    var days: [WeatherDay]?
}

/// What the widgets and the app draw: the last good copy of the Pi's data.
struct Snapshot: Codable {
    var calendar: CalendarResponse?
    var weather: Weather?
    /// When the Pi last answered. Shown as "from 9:40" once it's stale.
    var fetchedAt: Date?

    static let empty = Snapshot()
}

/// Body of POST /api/events.
struct OutgoingMessage: Encodable {
    var text: String
    var source: String?
    var ttl: Int?
    var speak: Bool?
}

extension JSONDecoder {
    /// Go writes times as RFC 3339, sometimes with fractional seconds.
    static let mirror: JSONDecoder = {
        let d = JSONDecoder()
        let plain = ISO8601DateFormatter()
        plain.formatOptions = [.withInternetDateTime]
        let fractional = ISO8601DateFormatter()
        fractional.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        d.dateDecodingStrategy = .custom { decoder in
            let s = try decoder.singleValueContainer().decode(String.self)
            if let date = fractional.date(from: s) ?? plain.date(from: s) { return date }
            throw DecodingError.dataCorrupted(.init(codingPath: decoder.codingPath, debugDescription: "Not an RFC 3339 time: \(s)"))
        }
        return d
    }()
}
