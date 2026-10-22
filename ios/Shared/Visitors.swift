import AppIntents
import CryptoKit
import SwiftUI
import WidgetKit

/// The haunted page's visitors (frontend/src/widgets/haunted/Visitors.tsx).
/// Their silhouettes are in Visitors.xcassets, drawn from the mirror's own
/// paths until Ally redraws them (docs/drawing-for-the-mirror.md).
enum MirrorVisitor: String, AppEnum, CaseIterable {
    case bats, zombie, witch, hand, cat, crow, balloon

    static var typeDisplayRepresentation: TypeDisplayRepresentation = "Visitor"
    static var caseDisplayRepresentations: [MirrorVisitor: DisplayRepresentation] = [
        .bats: "Bats", .zombie: "Zombie", .witch: "Witch", .hand: "Hand",
        .cat: "Black cat", .crow: "Crow", .balloon: "Red balloon",
    ]

    var label: String {
        switch self {
        case .bats: return "Bats"
        case .zombie: return "Zombie"
        case .witch: return "Witch"
        case .hand: return "Hand"
        case .cat: return "Cat"
        case .crow: return "Crow"
        case .balloon: return "Balloon"
        }
    }

    var image: Image { Image("visitor-\(rawValue)") }
}

/// The haunted page's palette (top of haunted.css).
enum Haunted {
    static let bg = Color(hex: 0x0E0A10)
    static let sky = Color(hex: 0x05040B)
    static let ink = Color(hex: 0xEADFC8)
    static let inkSoft = Color(hex: 0xAE9F88)
    static let hairline = Color(hex: 0x2B2129)
    static let glow = Color(hex: 0xE8812F)
    static let balloon = Color(hex: 0xB81D25)
}

extension Store {
    /// What the settings page's cookie holds: a hash of the PIN, never the
    /// PIN itself (pinToken in backend/settings.go). Empty until set.
    static var settingsToken: String {
        get { defaults.string(forKey: "settingsToken") ?? "" }
        set { defaults.set(newValue, forKey: "settingsToken") }
    }

    static func setPin(_ pin: String) {
        let p = pin.trimmingCharacters(in: .whitespaces).lowercased()
        guard !p.isEmpty else { settingsToken = ""; return }
        let digest = SHA256.hash(data: Data("coconut-kitchen-settings:\(p)".utf8))
        settingsToken = digest.map { String(format: "%02x", $0) }.joined()
    }

    static var lastSummon: LastSent? {
        get { defaults.data(forKey: "lastSummon").flatMap { try? JSONDecoder.mirror.decode(LastSent.self, from: $0) } }
        set {
            let e = JSONEncoder()
            e.dateEncodingStrategy = .iso8601
            defaults.set(newValue.flatMap { try? e.encode($0) }, forKey: "lastSummon")
        }
    }
}

enum SummonError: LocalizedError {
    case noPin, wrongPin, notHaunted, unreachable

    var errorDescription: String? {
        switch self {
        case .noPin: return "Add the settings PIN in the app to summon visitors."
        case .wrongPin: return "The settings PIN didn’t work. Check it in the app."
        case .notHaunted: return "Visitors only come when the mirror is haunted."
        case .unreachable: return "Couldn’t reach the mirror. Are you on the home Wi-Fi?"
        }
    }

    /// Short, for the widget.
    var short: String {
        switch self {
        case .noPin, .wrongPin: return "Needs the PIN"
        case .notHaunted: return "Not haunted"
        case .unreachable: return "Not home?"
        }
    }
}

extension MirrorAPI {
    /// Sends a visitor across the haunted page: POST /api/settings/summon,
    /// with the settings page's PIN cookie.
    func summon(_ v: MirrorVisitor) async throws {
        let token = Store.settingsToken
        guard !token.isEmpty else { throw SummonError.noPin }
        var req = URLRequest(url: try url("/api/settings/summon"))
        req.httpMethod = "POST"
        req.timeoutInterval = 6
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        req.setValue("mirror_settings=\(token)", forHTTPHeaderField: "Cookie")
        req.httpBody = try JSONEncoder().encode(["visitor": v.rawValue])
        let c = URLSessionConfiguration.ephemeral
        c.httpShouldSetCookies = false
        c.timeoutIntervalForResource = 10
        let resp: URLResponse
        do {
            (_, resp) = try await URLSession(configuration: c).data(for: req)
        } catch {
            throw SummonError.unreachable
        }
        switch (resp as? HTTPURLResponse)?.statusCode ?? 0 {
        case 200..<300: return
        case 401: throw SummonError.wrongPin
        case 409: throw SummonError.notHaunted
        case 403: throw SummonError.unreachable // not from the home network
        default: throw SummonError.unreachable
        }
    }
}

/// Calls a visitor onto the haunted page. The Summon widget's buttons run
/// it; it's in Shortcuts and Siri too ("Summon bats in Coconut Kitchen").
struct SummonVisitorIntent: AppIntent {
    static var title: LocalizedStringResource = "Summon a Visitor"
    static var description = IntentDescription("Sends a visitor across the haunted mirror.")

    @Parameter(title: "Visitor", default: .bats)
    var visitor: MirrorVisitor

    init() {}
    init(_ visitor: MirrorVisitor) { self.visitor = visitor }

    static var parameterSummary: some ParameterSummary { Summary("Summon \(\.$visitor)") }

    func perform() async throws -> some IntentResult & ProvidesDialog {
        do {
            try await MirrorAPI().summon(visitor)
            Store.lastSummon = LastSent(text: visitor.label, at: Date(), ok: true)
            WidgetCenter.shared.reloadTimelines(ofKind: SummonWidgetKind)
            return .result(dialog: "\(visitor.label), on their way.")
        } catch {
            let e = error as? SummonError ?? .unreachable
            Store.lastSummon = LastSent(text: e.short, at: Date(), ok: false)
            WidgetCenter.shared.reloadTimelines(ofKind: SummonWidgetKind)
            return .result(dialog: IntentDialog(stringLiteral: e.errorDescription ?? ""))
        }
    }
}

let SummonWidgetKind = "SummonWidget"
