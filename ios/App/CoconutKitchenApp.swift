import SwiftUI
import AppIntents
import WidgetKit

@main
struct CoconutKitchenApp: App {
    @Environment(\.scenePhase) private var phase
    @StateObject private var model = MirrorModel()

    var body: some Scene {
        WindowGroup {
            TodayScreen()
                .environmentObject(model)
                .preferredColorScheme(.dark)
                .onOpenURL { url in
                    // coconutkitchen://send — the Send widget's "Write…" button.
                    if url.host == "send" { model.composing = true }
                }
        }
        .onChange(of: phase) { _, p in
            if p == .active { Task { await model.refresh() } }
        }
    }
}

struct MirrorShortcuts: AppShortcutsProvider {
    static var appShortcuts: [AppShortcut] {
        AppShortcut(intent: SendToMirrorIntent(),
                    phrases: ["Send to the mirror in \(.applicationName)",
                              "Message the mirror in \(.applicationName)"],
                    shortTitle: "Send to Mirror",
                    systemImageName: "text.bubble")
    }
}

@MainActor
final class MirrorModel: ObservableObject {
    @Published var snapshot = Store.snapshot
    @Published var error: String?
    @Published var loading = false
    @Published var composing = false
    @Published var showSettings = Store.name.isEmpty

    func refresh() async {
        loading = true
        let (snap, err) = await MirrorAPI().refresh()
        snapshot = snap
        error = err.map(Self.explain)
        loading = false
        WidgetCenter.shared.reloadTimelines(ofKind: TodayWidgetKind)
    }

    /// Returns nil on success, or why it failed.
    func send(_ text: String, speak: Bool) async -> String? {
        do {
            try await MirrorAPI().send(text, speak: speak)
            Store.lastSent = LastSent(text: text, at: Date(), ok: true)
            WidgetCenter.shared.reloadTimelines(ofKind: SendWidgetKind)
            return nil
        } catch {
            return Self.explain(error)
        }
    }

    static func explain(_ e: Error) -> String {
        if let m = e as? MirrorError { return m.errorDescription ?? "Something went wrong." }
        if (e as? URLError) != nil { return "Can’t reach the mirror. It only answers on the home Wi-Fi." }
        return e.localizedDescription
    }
}
