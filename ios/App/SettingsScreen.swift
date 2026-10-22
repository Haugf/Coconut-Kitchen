import SwiftUI
import WidgetKit

struct SettingsScreen: View {
    @EnvironmentObject var model: MirrorModel
    @Environment(\.dismiss) private var dismiss

    @State private var name = Store.name
    @State private var host = Store.host
    @State private var token = Store.token
    @State private var presets = Store.presets.joined(separator: "\n")
    @State private var test: String?

    private var people: [String] { model.snapshot.calendar?.people ?? ["Fred", "Ally"] }

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    Picker("This phone is", selection: $name) {
                        Text("Choose").tag("")
                        ForEach(people, id: \.self) { Text($0).tag($0) }
                    }
                } footer: {
                    Text("Signs your messages and makes your line solid in the day drawing.")
                }

                Section {
                    TextField("One per line", text: $presets, axis: .vertical)
                        .lineLimit(3...6)
                } header: {
                    Text("One-tap messages")
                } footer: {
                    Text("The first three are the buttons on the Send widget.")
                }

                Section {
                    TextField("mirror.local:8080", text: $host)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                        .keyboardType(.URL)
                    SecureField("Token (only if set on the Pi)", text: $token)
                    Button("Test the connection") { Task { await runTest() } }
                    if let t = test { Text(t).font(.footnote).foregroundStyle(.secondary) }
                } header: {
                    Text("Mirror")
                } footer: {
                    Text("The mirror only answers on the home Wi-Fi. Away from home, the app and widgets show the last thing it said.")
                }
            }
            .navigationTitle("Settings")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("Done") { save(); dismiss() }
                }
            }
        }
        .preferredColorScheme(.dark)
    }

    private func save() {
        Store.name = name
        Store.host = host
        Store.token = token
        Store.presets = presets.components(separatedBy: .newlines)
        WidgetCenter.shared.reloadAllTimelines()
        Task { await model.refresh() }
    }

    private func runTest() async {
        test = "Checking…"
        do {
            let cal = try await MirrorAPI(host: host, token: token).calendar()
            test = "Connected. \(cal.events.count) events this week."
        } catch {
            test = MirrorModel.explain(error)
        }
    }
}
