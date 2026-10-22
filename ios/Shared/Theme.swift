import SwiftUI

/// The kitchen page's closed palette (top of kitchen.css): husk-brown
/// ground, coconut-cream ink, toasted coconut as the one accent.
enum Theme {
    static let bg = Color(hex: 0x2B221B)
    static let ink = Color(hex: 0xEBDFCB)
    static let inkSoft = Color(hex: 0xB6A693)
    static let inkGrey = Color(hex: 0x7D6E5F)
    static let hairline = Color(hex: 0x40352B)
    static let clay = Color(hex: 0xDC965C)

    /// Young Serif on the mirror; the phone's own serif (New York) here, so
    /// there's no font to bundle.
    static func serif(_ size: CGFloat) -> Font { .system(size: size, weight: .regular, design: .serif) }
    static func sans(_ size: CGFloat, _ weight: Font.Weight = .regular) -> Font { .system(size: size, weight: weight) }
}

extension Color {
    init(hex: UInt32) {
        self.init(red: Double((hex >> 16) & 0xFF) / 255,
                  green: Double((hex >> 8) & 0xFF) / 255,
                  blue: Double(hex & 0xFF) / 255)
    }
}

/// The day from 6 AM to midnight as a line, one per person: this phone's
/// owner solid, everyone else dashed, a dot at now. Same shape as the
/// strip along the bottom of the mirror.
struct DayLine: View {
    var events: [DayEvent]
    var people: [String]
    var me: String
    var now: Date

    var body: some View {
        Canvas { ctx, size in
            let pad: CGFloat = 4
            let w = size.width, h = size.height - pad
            func point(_ p: (x: Double, y: Double)) -> CGPoint {
                CGPoint(x: p.x * w, y: h - p.y * (h - pad) )
            }
            // Ground.
            var ground = Path()
            ground.move(to: CGPoint(x: 0, y: h))
            ground.addLine(to: CGPoint(x: w, y: h))
            ctx.stroke(ground, with: .color(Theme.hairline), lineWidth: 1)

            let names = people.count >= 2 ? people : [me.isEmpty ? "" : me]
            // Draw the others first so this phone's line sits on top.
            for name in names.filter({ $0 != me }) + names.filter({ $0 == me }) {
                let mine = names.count < 2 || name == me
                let theirs = names.count < 2 ? events : events.filter { Day.belongs($0, to: name) }
                let pts = Day.line(theirs).map(point)
                var path = Path()
                path.move(to: pts[0])
                for p in pts.dropFirst() { path.addLine(to: p) }
                ctx.stroke(path,
                           with: .color(mine ? Theme.ink : Theme.inkSoft),
                           style: StrokeStyle(lineWidth: mine ? 1.6 : 1.2, lineCap: .round, lineJoin: .round,
                                              dash: mine ? [] : [4, 4]))
            }
            // Now.
            let hNow = Day.hours(now)
            if hNow >= Day.start {
                let mineEvents = names.count < 2 || me.isEmpty ? events : events.filter { Day.belongs($0, to: me) }
                let p = point((Day.x(hNow), Day.rise(hNow, mineEvents)))
                ctx.fill(Path(ellipseIn: CGRect(x: p.x - 3.5, y: p.y - 3.5, width: 7, height: 7)), with: .color(Theme.clay))
            }
        }
    }
}
