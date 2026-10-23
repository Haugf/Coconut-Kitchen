import SwiftUI

/// One-tap presets, then a line to write anything. Messages stay on the
/// mirror for 20 seconds, signed with this phone's name.
struct ComposeBar: View {
    @EnvironmentObject var model: MirrorModel
    @State private var text = ""
    @State private var speak = false
    @State private var sending = false
    @State private var status: String?
    @FocusState private var focused: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 8) {
                    ForEach(Store.presets, id: \.self) { p in
                        Button(p) { Task { await send(p) } }
                            .font(Theme.sans(14))
                            .foregroundStyle(Theme.ink)
                            .padding(.horizontal, 12)
                            .padding(.vertical, 7)
                            .overlay(Rectangle().stroke(Theme.hairline, lineWidth: 1))
                    }
                }
            }
            HStack(spacing: 10) {
                TextField("Write on the mirror", text: $text)
                    .focused($focused)
                    .font(Theme.sans(16))
                    .foregroundStyle(Theme.ink)
                    .submitLabel(.send)
                    .onSubmit { Task { await send(text) } }
                Button { speak.toggle() } label: {
                    Image(systemName: speak ? "speaker.wave.2.fill" : "speaker.slash")
                        .foregroundStyle(speak ? Theme.clay : Theme.inkGrey)
                }
                .accessibilityLabel(speak ? "Also say it out loud: on" : "Also say it out loud: off")
                Button { Task { await send(text) } } label: {
                    Image(systemName: "arrow.up.circle.fill")
                        .font(.system(size: 28))
                        .foregroundStyle(text.trimmingCharacters(in: .whitespaces).isEmpty ? Theme.inkGrey : Theme.clay)
                }
                .disabled(sending || text.trimmingCharacters(in: .whitespaces).isEmpty)
                .accessibilityLabel("Send")
            }
            if let s = status {
                Text(s).font(Theme.sans(12)).foregroundStyle(Theme.inkSoft)
            }
        }
        .padding(.horizontal, 20)
        .padding(.vertical, 12)
        .background(Theme.bg.overlay(alignment: .top) { Rectangle().fill(Theme.hairline).frame(height: 1) })
        .onChange(of: model.composing) { _, on in
            if on { focused = true; model.composing = false }
        }
    }

    private func send(_ raw: String) async {
        let t = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !t.isEmpty, !sending else { return }
        sending = true
        defer { sending = false }
        if let err = await model.send(t, speak: speak) {
            status = err
        } else {
            status = "On the mirror: “\(t)”"
            if t == text.trimmingCharacters(in: .whitespacesAndNewlines) { text = "" }
            focused = false
        }
    }
}
