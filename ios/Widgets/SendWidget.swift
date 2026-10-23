import WidgetKit
import SwiftUI
import AppIntents

struct SendEntry: TimelineEntry {
    let date: Date
    let presets: [String]
    let lastSent: LastSent?
}

struct SendProvider: TimelineProvider {
    func placeholder(in context: Context) -> SendEntry {
        SendEntry(date: Date(), presets: ["Dinner’s ready", "Home soon", "Love you"], lastSent: nil)
    }

    func getSnapshot(in context: Context, completion: @escaping (SendEntry) -> Void) {
        completion(SendEntry(date: Date(), presets: Store.presets, lastSent: Store.lastSent))
    }

    func getTimeline(in context: Context, completion: @escaping (Timeline<SendEntry>) -> Void) {
        let now = Date()
        let last = Store.lastSent
        var entries = [SendEntry(date: now, presets: Store.presets, lastSent: last)]
        // "On the mirror" shows for the 20 seconds the message is up, then clears.
        if let last, now.timeIntervalSince(last.at) < 20 {
            entries.append(SendEntry(date: last.at.addingTimeInterval(20), presets: Store.presets, lastSent: nil))
        }
        completion(Timeline(entries: entries, policy: .never))
    }
}

struct SendWidget: Widget {
    var body: some WidgetConfiguration {
        StaticConfiguration(kind: SendWidgetKind, provider: SendProvider()) { entry in
            SendWidgetView(entry: entry)
                .containerBackground(Theme.bg, for: .widget)
        }
        .configurationDisplayName("Send to Mirror")
        .description("One tap puts a message on the kitchen mirror. Works on the home Wi-Fi.")
        .supportedFamilies([.systemSmall, .systemMedium])
    }
}

struct SendWidgetView: View {
    @Environment(\.widgetFamily) private var family
    let entry: SendEntry

    private var small: Bool { family == .systemSmall }
    private var buttons: [String] { Array(entry.presets.prefix(small ? 2 : 4)) }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                status
                Spacer()
                Link(destination: URL(string: "coconutkitchen://send")!) {
                    Image(systemName: "square.and.pencil")
                        .font(Theme.sans(14))
                        .foregroundStyle(Theme.inkSoft)
                }
                .accessibilityLabel("Write a message")
            }
            Spacer(minLength: 0)
            LazyVGrid(columns: Array(repeating: GridItem(.flexible(), spacing: 8), count: small ? 1 : 2), spacing: 8) {
                ForEach(buttons, id: \.self) { p in
                    Button(intent: SendToMirrorIntent(text: p)) {
                        Text(p)
                            .font(Theme.serif(15))
                            .foregroundStyle(Theme.ink)
                            .lineLimit(1)
                            .minimumScaleFactor(0.75)
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .padding(.vertical, 9)
                            .padding(.horizontal, 8)
                            .overlay(Rectangle().stroke(Theme.hairline, lineWidth: 1))
                    }
                    .buttonStyle(.plain)
                }
            }
        }
    }

    /// "TO THE MIRROR", or for 20 seconds after a tap, whether it got there.
    @ViewBuilder private var status: some View {
        if let last = entry.lastSent {
            Text(last.ok ? "On the mirror" : "Couldn’t reach it")
                .font(Theme.sans(12, .semibold))
                .foregroundStyle(last.ok ? Theme.ink : Theme.clay)
        } else {
            Text("TO THE MIRROR")
                .font(Theme.sans(10, .semibold)).tracking(1)
                .foregroundStyle(Theme.clay)
        }
    }
}

@main
struct MirrorWidgets: WidgetBundle {
    var body: some Widget {
        TodayWidget()
        SendWidget()
        SummonWidget()
    }
}
