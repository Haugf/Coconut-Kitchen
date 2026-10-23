import WidgetKit
import SwiftUI

struct TodayEntry: TimelineEntry {
    let date: Date
    let snapshot: Snapshot
    let me: String
}

struct TodayProvider: TimelineProvider {
    func placeholder(in context: Context) -> TodayEntry {
        TodayEntry(date: Date(), snapshot: .sample, me: "Fred")
    }

    func getSnapshot(in context: Context, completion: @escaping (TodayEntry) -> Void) {
        let s = Store.snapshot
        completion(TodayEntry(date: Date(), snapshot: context.isPreview && s.calendar == nil ? .sample : s, me: Store.name))
    }

    func getTimeline(in context: Context, completion: @escaping (Timeline<TodayEntry>) -> Void) {
        Task {
            // Off the home Wi-Fi this fails within a few seconds and the
            // last good snapshot is used.
            let (snap, _) = await MirrorAPI().refresh()
            let now = Date()
            // An entry every 15 minutes so the "now" dot moves and past
            // events drop off, even when the Pi can't be reached.
            let entries = (0..<12).map { i in
                TodayEntry(date: now.addingTimeInterval(Double(i) * 15 * 60), snapshot: snap, me: Store.name)
            }
            completion(Timeline(entries: entries, policy: .after(now.addingTimeInterval(30 * 60))))
        }
    }
}

struct TodayWidget: Widget {
    var body: some WidgetConfiguration {
        StaticConfiguration(kind: TodayWidgetKind, provider: TodayProvider()) { entry in
            TodayWidgetView(entry: entry)
                .containerBackground(Theme.bg, for: .widget)
                .widgetURL(URL(string: "coconutkitchen://today"))
        }
        .configurationDisplayName("Today")
        .description("Today on the kitchen mirror: the headline, the day as a line, and what’s next.")
        .supportedFamilies([.systemSmall, .systemMedium, .systemLarge, .accessoryRectangular, .accessoryInline])
    }
}

struct TodayWidgetView: View {
    @Environment(\.widgetFamily) private var family
    let entry: TodayEntry

    private var events: [CalendarEvent] { entry.snapshot.calendar?.events ?? [] }
    private var people: [String] { entry.snapshot.calendar?.people ?? [] }
    private var today: [DayEvent] { Day.timedEvents(events, on: entry.date) }
    private var upcoming: [DayEvent] { today.filter { $0.end > entry.date } }
    private var headline: String {
        entry.snapshot.calendar == nil ? "Open the app on the home Wi-Fi." : Day.headline(today)
    }

    var body: some View {
        switch family {
        case .accessoryInline:
            Text(upcoming.first.map { "\(Day.shortClock($0.start)) \($0.title)" } ?? headline)
        case .accessoryRectangular:
            VStack(alignment: .leading, spacing: 1) {
                Text(entry.date.formatted(.dateTime.weekday(.wide))).font(.caption2.weight(.semibold))
                Text(headline).font(.system(size: 13, design: .serif)).lineLimit(2)
                if let w = entry.snapshot.weather {
                    Text("\(Int(w.temp.rounded()))° \(WeatherWords.describe(w.code))").font(.caption2)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        case .systemSmall:
            small
        case .systemLarge:
            large
        default:
            medium
        }
    }

    private var top: some View {
        HStack(alignment: .firstTextBaseline) {
            Text(entry.date.formatted(.dateTime.weekday(.abbreviated).day()))
                .font(Theme.sans(12, .semibold))
                .foregroundStyle(Theme.clay)
                .textCase(.uppercase)
            Spacer()
            if let w = entry.snapshot.weather {
                Label("\(Int(w.temp.rounded()))°", systemImage: WeatherWords.symbol(w.code))
                    .labelStyle(.titleAndIcon)
                    .font(Theme.sans(12))
                    .foregroundStyle(Theme.inkSoft)
            }
        }
    }

    private var line: some View {
        DayLine(events: today, people: people, me: entry.me, now: entry.date)
    }

    private var small: some View {
        VStack(alignment: .leading, spacing: 6) {
            top
            Text(headline)
                .font(Theme.serif(15))
                .foregroundStyle(Theme.ink)
                .lineLimit(4)
                .minimumScaleFactor(0.85)
            Spacer(minLength: 0)
            line.frame(height: 26)
        }
    }

    private var medium: some View {
        HStack(alignment: .top, spacing: 14) {
            VStack(alignment: .leading, spacing: 6) {
                top
                Text(headline)
                    .font(Theme.serif(16))
                    .foregroundStyle(Theme.ink)
                    .lineLimit(3)
                    .minimumScaleFactor(0.85)
                Spacer(minLength: 0)
                line.frame(height: 30)
            }
            Rectangle().fill(Theme.hairline).frame(width: 1)
            VStack(alignment: .leading, spacing: 7) {
                Text("NEXT").font(Theme.sans(10, .semibold)).tracking(1).foregroundStyle(Theme.inkGrey)
                if upcoming.isEmpty {
                    Text(today.isEmpty ? "Nothing on the calendar." : "That’s everything.")
                        .font(Theme.sans(13)).foregroundStyle(Theme.inkGrey)
                }
                ForEach(upcoming.prefix(3), id: \.self) { e in row(e) }
                Spacer(minLength: 0)
            }
            .frame(width: 128, alignment: .leading)
        }
    }

    private var large: some View {
        VStack(alignment: .leading, spacing: 10) {
            top
            Text(headline)
                .font(Theme.serif(22))
                .foregroundStyle(Theme.ink)
                .lineLimit(3)
            line.frame(height: 56)
            Rectangle().fill(Theme.hairline).frame(height: 1)
            if upcoming.isEmpty {
                Text(today.isEmpty ? "Nothing on the calendar." : "That’s everything for today.")
                    .font(Theme.sans(14)).foregroundStyle(Theme.inkGrey)
            }
            ForEach(upcoming.prefix(6), id: \.self) { e in row(e, wide: true) }
            Spacer(minLength: 0)
            if let at = entry.snapshot.fetchedAt, entry.date.timeIntervalSince(at) > 3 * 3600 {
                Text("As of \(Day.clock(at))").font(Theme.sans(11)).foregroundStyle(Theme.inkGrey)
            }
        }
    }

    private func row(_ e: DayEvent, wide: Bool = false) -> some View {
        HStack(alignment: .firstTextBaseline, spacing: 8) {
            Text(Day.shortClock(e.start))
                .font(Theme.sans(wide ? 14 : 12).monospacedDigit())
                .foregroundStyle(e.start <= entry.date ? Theme.clay : Theme.inkSoft)
                .frame(width: wide ? 44 : 32, alignment: .leading)
            Text(e.title)
                .font(Theme.sans(wide ? 15 : 13))
                .foregroundStyle(Theme.ink)
                .lineLimit(1)
            if wide, people.count >= 2, !e.who.isEmpty {
                Spacer(minLength: 4)
                Text(e.who.count >= 2 ? "Both" : e.who[0])
                    .font(Theme.sans(12)).foregroundStyle(Theme.inkGrey)
            }
        }
    }
}

extension Snapshot {
    /// What the widget gallery shows before the Pi has ever answered.
    static var sample: Snapshot {
        let cal = Calendar.current
        func at(_ h: Int, _ m: Int = 0) -> Date { cal.date(bySettingHour: h, minute: m, second: 0, of: Date())! }
        return Snapshot(
            calendar: CalendarResponse(people: ["Fred", "Ally"], events: [
                CalendarEvent(title: "Standup", start: at(10), end: at(10, 30), allDay: false, who: ["Fred"]),
                CalendarEvent(title: "Studio", start: at(13), end: at(16), allDay: false, who: ["Ally"]),
                CalendarEvent(title: "Dinner", start: at(19), end: at(21), allDay: false, who: ["Fred", "Ally"]),
            ], failed: nil),
            weather: Weather(temp: 58, feelsLike: 56, code: 2, units: "fahrenheit", days: nil),
            fetchedAt: Date()
        )
    }
}
