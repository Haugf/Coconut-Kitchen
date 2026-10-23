import Foundation

// A Swift copy of the mirror's day logic (frontend/src/widgets/terrain/
// day.ts, shape.ts, voice.ts), so the phone says the same thing the
// kitchen does. If those change, change these.

struct DayEvent: Hashable {
    var title: String
    var start: Date
    var end: Date
    var startH: Double
    var endH: Double
    var who: [String]
}

enum Day {
    static let start = 6.0   // the drawing spans 6 AM
    static let end = 24.0    // to midnight

    static var cal: Calendar { Calendar.current }

    static func hours(_ d: Date) -> Double {
        let c = cal.dateComponents([.hour, .minute], from: d)
        return Double(c.hour ?? 0) + Double(c.minute ?? 0) / 60
    }

    /// Timed events that start on `day`, sorted. All-day events are left out.
    static func timedEvents(_ events: [CalendarEvent], on day: Date) -> [DayEvent] {
        events
            .filter { !$0.allDay && cal.isDate($0.start, inSameDayAs: day) }
            .map { e in
                let end = e.end ?? e.start.addingTimeInterval(30 * 60)
                let endH = cal.isDate(end, inSameDayAs: day) ? hours(end) : 24
                let sH = hours(e.start)
                return DayEvent(title: e.title.trimmingCharacters(in: .whitespaces), start: e.start, end: end,
                                startH: sH, endH: max(endH, sH + 0.25), who: e.who ?? [])
            }
            .sorted { $0.startH < $1.startH }
    }

    static func allDay(_ events: [CalendarEvent], on day: Date) -> [CalendarEvent] {
        events.filter { $0.allDay && cal.isDate($0.start, inSameDayAs: day) }
    }

    /// Events on one person's calendar. Unattributed events count for everyone.
    static func belongs(_ e: DayEvent, to name: String) -> Bool {
        e.who.isEmpty || e.who.contains(name)
    }

    /// "2 PM", "2:30 PM"
    static func clock(_ d: Date) -> String {
        let c = cal.dateComponents([.hour, .minute], from: d)
        let h24 = c.hour ?? 0, m = c.minute ?? 0
        let h = h24 % 12 == 0 ? 12 : h24 % 12
        let ap = h24 < 12 ? "AM" : "PM"
        return m == 0 ? "\(h) \(ap)" : "\(h):\(String(format: "%02d", m)) \(ap)"
    }

    /// "2", "2:30" inside a sentence where AM/PM is obvious.
    static func shortClock(_ d: Date) -> String {
        let c = cal.dateComponents([.hour, .minute], from: d)
        let h24 = c.hour ?? 0, m = c.minute ?? 0
        let h = h24 % 12 == 0 ? 12 : h24 % 12
        return m == 0 ? "\(h)" : "\(h):\(String(format: "%02d", m))"
    }

    // MARK: voice.ts

    private static func at(_ e: DayEvent) -> String { "\(e.title) at \(shortClock(e.start))" }

    /// One line. Name the one distinct thing if there is one, otherwise the shape.
    static func headline(_ events: [DayEvent]) -> String {
        if events.isEmpty { return "Open all day." }
        if events.count == 1 { return "One thing today: \(at(events[0]))." }

        let longest = events.max { ($0.endH - $0.startH) < ($1.endH - $1.startH) }!
        let longestDur = longest.endH - longest.startH
        let others = events.filter { $0 != longest }
        if others.allSatisfy({ $0.endH - $0.startH <= longestDur * 0.6 }) {
            let after = events.filter { $0.startH >= longest.endH }
            return after.isEmpty
                ? "A climb to \(at(longest)), then the day opens up."
                : "The day peaks with \(at(longest))."
        }
        if events[0].startH >= 16 { return "Open until \(shortClock(events[0].start)), then \(events[0].title)." }
        let last = events[events.count - 1]
        if last.endH <= 13 { return "A busy morning, then open from \(shortClock(last.end))." }
        return events.count >= 5 ? "A full day, steady from start to finish." : "A steady day with room between things."
    }

    // MARK: shape.ts — the day as a line, in unit coordinates (0…1 across, 0 = ground, 1 = highest)

    /// Elevation is load: each event raises the land around it, longer events higher and wider.
    static func rise(_ hour: Double, _ events: [DayEvent]) -> Double {
        let maxRise = 104.0
        var total = 0.0
        for e in events {
            let dur = min(max(e.endH - e.startH, 0.25), 4)
            let mid = (e.startH + e.endH) / 2
            let spread = max(dur / 2, 0.5) + 0.6
            let weight = 40 + dur * 40
            total += weight * exp(-pow((hour - mid) / spread, 2))
        }
        return (1 - exp(-total / maxRise)) // 0…1
    }

    static func x(_ hour: Double) -> Double {
        (min(max(hour, start), end) - start) / (end - start)
    }

    /// Points along the line, 0…1 in both axes.
    static func line(_ events: [DayEvent], samples: Int = 72) -> [(x: Double, y: Double)] {
        (0...samples).map { i in
            let h = start + (end - start) * Double(i) / Double(samples)
            return (x(h), rise(h, events))
        }
    }
}

enum WeatherWords {
    static func describe(_ code: Int) -> String {
        switch code {
        case 0: return "Clear"
        case 1: return "Mostly clear"
        case 2: return "Partly cloudy"
        case 3: return "Overcast"
        case 45, 48: return "Fog"
        case 51, 53, 55, 56, 57: return "Drizzle"
        case 61: return "Light rain"
        case 63: return "Rain"
        case 65: return "Heavy rain"
        case 66, 67: return "Freezing rain"
        case 71, 73, 75, 77: return "Snow"
        case 80, 81, 82: return "Showers"
        case 85, 86: return "Snow showers"
        case 95, 96, 99: return "Thunderstorms"
        default: return ""
        }
    }

    static func symbol(_ code: Int) -> String {
        switch code {
        case 0, 1: return "sun.max"
        case 2: return "cloud.sun"
        case 3: return "cloud"
        case 45, 48: return "cloud.fog"
        case 51...57: return "cloud.drizzle"
        case 61...67, 80...82: return "cloud.rain"
        case 71...77, 85, 86: return "cloud.snow"
        case 95...99: return "cloud.bolt.rain"
        default: return "cloud"
        }
    }
}
