import SwiftUI

/// The app's one screen: the kitchen page, phone-sized, with a way to
/// put a message on the mirror at the bottom.
struct TodayScreen: View {
    @EnvironmentObject var model: MirrorModel

    var body: some View {
        TimelineView(.everyMinute) { tl in
            let now = tl.date
            let events = model.snapshot.calendar?.events ?? []
            let people = model.snapshot.calendar?.people ?? []
            let today = Day.timedEvents(events, on: now)
            let tomorrowDate = Calendar.current.date(byAdding: .day, value: 1, to: now)!
            let tomorrow = Day.timedEvents(events, on: tomorrowDate)

            ScrollView {
                VStack(alignment: .leading, spacing: 22) {
                    header(now)
                    Text(model.snapshot.calendar == nil ? "Not connected yet." : Day.headline(today))
                        .font(Theme.serif(30))
                        .foregroundStyle(Theme.ink)
                        .fixedSize(horizontal: false, vertical: true)

                    VStack(spacing: 6) {
                        DayLine(events: today, people: people, me: Store.name, now: now)
                            .frame(height: 70)
                        HStack {
                            ForEach(["6a", "noon", "6p", "12a"], id: \.self) { t in
                                Text(t)
                                if t != "12a" { Spacer() }
                            }
                        }
                        .font(Theme.sans(11))
                        .foregroundStyle(Theme.inkGrey)
                    }

                    EventList(heading: "Today",
                              allDay: Day.allDay(events, on: now),
                              events: today.filter { $0.end > now },
                              people: people,
                              empty: today.isEmpty ? "Nothing on the calendar." : "That’s everything for today.")
                    EventList(heading: "Tomorrow",
                              allDay: Day.allDay(events, on: tomorrowDate),
                              events: tomorrow,
                              people: people,
                              empty: "Nothing yet.")

                    if let e = model.error {
                        Text(staleLine(e))
                            .font(Theme.sans(13))
                            .foregroundStyle(Theme.inkGrey)
                    }
                }
                .padding(.horizontal, 20)
                .padding(.top, 8)
                .padding(.bottom, 24)
            }
            .refreshable { await model.refresh() }
        }
        .background(Theme.bg.ignoresSafeArea())
        .safeAreaInset(edge: .bottom) { ComposeBar() }
        .sheet(isPresented: $model.showSettings) {
            SettingsScreen().environmentObject(model)
        }
        .task { await model.refresh() }
    }

    private func header(_ now: Date) -> some View {
        HStack(alignment: .firstTextBaseline) {
            VStack(alignment: .leading, spacing: 2) {
                Text(now.formatted(.dateTime.weekday(.wide)))
                    .font(Theme.sans(15, .semibold))
                    .foregroundStyle(Theme.ink)
                Text(now.formatted(.dateTime.month(.wide).day()))
                    .font(Theme.sans(15))
                    .foregroundStyle(Theme.inkSoft)
            }
            Spacer()
            if let w = model.snapshot.weather {
                HStack(spacing: 6) {
                    Image(systemName: WeatherWords.symbol(w.code))
                    Text("\(Int(w.temp.rounded()))°")
                }
                .font(Theme.sans(17))
                .foregroundStyle(Theme.ink)
            }
            Button { model.showSettings = true } label: {
                Image(systemName: "gearshape")
                    .foregroundStyle(Theme.inkGrey)
                    .padding(.leading, 14)
            }
            .accessibilityLabel("Settings")
        }
    }

    private func staleLine(_ error: String) -> String {
        guard let at = model.snapshot.fetchedAt else { return error }
        return "\(error) Showing what it said at \(Day.clock(at))."
    }
}

struct EventList: View {
    var heading: String
    var allDay: [CalendarEvent]
    var events: [DayEvent]
    var people: [String]
    var empty: String

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text(heading.uppercased())
                .font(Theme.sans(11, .semibold))
                .tracking(1.2)
                .foregroundStyle(Theme.clay)
            Rectangle().fill(Theme.hairline).frame(height: 1)
            if allDay.isEmpty && events.isEmpty {
                Text(empty).font(Theme.sans(15)).foregroundStyle(Theme.inkGrey)
            }
            ForEach(allDay, id: \.self) { e in
                row(time: "All day", title: e.title, who: e.who ?? [])
            }
            ForEach(events, id: \.self) { e in
                row(time: Day.clock(e.start), title: e.title, who: e.who)
            }
        }
    }

    private func row(time: String, title: String, who: [String]) -> some View {
        HStack(alignment: .firstTextBaseline, spacing: 12) {
            Text(time)
                .font(Theme.sans(14).monospacedDigit())
                .foregroundStyle(Theme.inkSoft)
                .frame(width: 70, alignment: .leading)
            Text(title)
                .font(Theme.sans(16))
                .foregroundStyle(Theme.ink)
            Spacer(minLength: 8)
            if people.count >= 2, !who.isEmpty {
                Text(who.count >= 2 ? "Both" : who[0])
                    .font(Theme.sans(12))
                    .foregroundStyle(Theme.inkGrey)
            }
        }
    }
}
