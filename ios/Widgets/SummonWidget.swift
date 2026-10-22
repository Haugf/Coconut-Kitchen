import WidgetKit
import SwiftUI
import AppIntents

struct SummonEntry: TimelineEntry {
    let date: Date
    let last: LastSent?
}

struct SummonProvider: TimelineProvider {
    func placeholder(in context: Context) -> SummonEntry { SummonEntry(date: Date(), last: nil) }

    func getSnapshot(in context: Context, completion: @escaping (SummonEntry) -> Void) {
        completion(SummonEntry(date: Date(), last: context.isPreview ? nil : Store.lastSummon))
    }

    func getTimeline(in context: Context, completion: @escaping (Timeline<SummonEntry>) -> Void) {
        let now = Date()
        let last = Store.lastSummon
        var entries = [SummonEntry(date: now, last: last)]
        // The "on their way" line clears after a few seconds.
        if let last, now.timeIntervalSince(last.at) < 8 {
            entries.append(SummonEntry(date: last.at.addingTimeInterval(8), last: nil))
        }
        completion(Timeline(entries: entries, policy: .never))
    }
}

struct SummonWidget: Widget {
    var body: some WidgetConfiguration {
        StaticConfiguration(kind: SummonWidgetKind, provider: SummonProvider()) { entry in
            SummonWidgetView(entry: entry)
                .containerBackground(for: .widget) {
                    LinearGradient(colors: [Haunted.sky, Haunted.bg], startPoint: .top, endPoint: .bottom)
                }
        }
        .configurationDisplayName("Summon")
        .description("Send bats, a zombie or a witch across the haunted mirror.")
        .supportedFamilies([.systemSmall, .systemMedium])
    }
}

struct SummonWidgetView: View {
    @Environment(\.widgetFamily) private var family
    let entry: SummonEntry

    private var small: Bool { family == .systemSmall }
    private var visitors: [MirrorVisitor] {
        small ? [.bats, .zombie, .witch, .hand] : [.bats, .zombie, .witch, .hand, .cat, .crow, .balloon]
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack {
                if let last = entry.last {
                    Text(last.ok ? "\(last.text), on their way" : last.text)
                        .font(.system(size: 12, weight: .semibold, design: .serif).italic())
                        .foregroundStyle(last.ok ? Haunted.ink : Haunted.glow)
                        .lineLimit(1)
                } else {
                    Text("Summon")
                        .font(.system(size: 13, design: .serif).italic())
                        .foregroundStyle(Haunted.inkSoft)
                }
                Spacer(minLength: 0)
            }
            Spacer(minLength: 0)
            LazyVGrid(columns: Array(repeating: GridItem(.flexible(), spacing: 4), count: small ? 2 : 4), spacing: small ? 6 : 4) {
                ForEach(visitors, id: \.self) { v in
                    Button(intent: SummonVisitorIntent(v)) {
                        VStack(spacing: 2) {
                            v.image
                                .resizable()
                                .scaledToFit()
                                .foregroundStyle(v == .balloon ? Haunted.balloon : Haunted.ink)
                                .frame(height: small ? 30 : 26)
                            if !small {
                                Text(v.label)
                                    .font(.system(size: 10))
                                    .foregroundStyle(Haunted.inkSoft)
                            }
                        }
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, small ? 4 : 2)
                        .contentShape(Rectangle())
                    }
                    .buttonStyle(.plain)
                    .accessibilityLabel("Summon \(v.label)")
                }
            }
        }
    }
}
