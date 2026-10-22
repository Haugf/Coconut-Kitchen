import AppIntents
import WidgetKit

/// Puts a message on the mirror. The Send widget's buttons run it in the
/// background, and it shows up in Shortcuts and Siri ("Send to the
/// mirror in Coconut Kitchen").
struct SendToMirrorIntent: AppIntent {
    static var title: LocalizedStringResource = "Send to the Mirror"
    static var description = IntentDescription("Shows a message on the Coconut Kitchen mirror.")

    @Parameter(title: "Message")
    var text: String

    @Parameter(title: "Say it out loud", default: false)
    var speak: Bool

    init() {}

    init(text: String, speak: Bool = false) {
        self.text = text
        self.speak = speak
    }

    static var parameterSummary: some ParameterSummary {
        Summary("Send \(\.$text) to the mirror") {
            \.$speak
        }
    }

    func perform() async throws -> some IntentResult & ProvidesDialog {
        let message = text.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !message.isEmpty else { return .result(dialog: "Nothing to send.") }
        do {
            try await MirrorAPI().send(message, speak: speak)
            Store.lastSent = LastSent(text: message, at: Date(), ok: true)
            WidgetCenter.shared.reloadTimelines(ofKind: SendWidgetKind)
            return .result(dialog: "On the mirror.")
        } catch {
            Store.lastSent = LastSent(text: message, at: Date(), ok: false)
            WidgetCenter.shared.reloadTimelines(ofKind: SendWidgetKind)
            return .result(dialog: "Couldn’t reach the mirror. Are you on the home Wi-Fi?")
        }
    }
}

let SendWidgetKind = "SendWidget"
let TodayWidgetKind = "TodayWidget"

/// What the Send widget shows under its buttons for a little while.
struct LastSent: Codable {
    var text: String
    var at: Date
    var ok: Bool
}

extension Store {
    static var lastSent: LastSent? {
        get {
            guard let d = defaults.data(forKey: "lastSent") else { return nil }
            return try? JSONDecoder.mirror.decode(LastSent.self, from: d)
        }
        set {
            let e = JSONEncoder()
            e.dateEncodingStrategy = .iso8601
            defaults.set(newValue.flatMap { try? e.encode($0) }, forKey: "lastSent")
        }
    }
}
