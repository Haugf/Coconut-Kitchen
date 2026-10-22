import Foundation

enum MirrorError: LocalizedError {
    case badHost
    case status(Int, String)

    var errorDescription: String? {
        switch self {
        case .badHost: return "The mirror’s address doesn’t look right. Check it in Settings."
        case .status(401, _): return "The mirror wants a token. Add it in Settings."
        case .status(403, let body) where body.contains("party"): return "The calendar is private during the party."
        case .status(let code, let body): return body.isEmpty ? "The mirror answered \(code)." : body
        }
    }
}

/// Talks to the Pi over the home Wi-Fi. Away from home every call fails
/// fast and the widgets keep showing the last good snapshot.
struct MirrorAPI {
    var host: String = Store.host
    var token: String = Store.token

    private var session: URLSession {
        let c = URLSessionConfiguration.ephemeral
        // Short: a widget has seconds to refresh, and off the home Wi-Fi
        // mirror.local never answers.
        c.timeoutIntervalForRequest = 6
        c.timeoutIntervalForResource = 10
        c.waitsForConnectivity = false
        return URLSession(configuration: c)
    }

    func url(_ path: String) throws -> URL {
        var h = host
        if !h.hasPrefix("http://") && !h.hasPrefix("https://") { h = "http://" + h }
        while h.hasSuffix("/") { h.removeLast() }
        guard let u = URL(string: h + path), u.host != nil else { throw MirrorError.badHost }
        return u
    }

    private func get<T: Decodable>(_ path: String, as: T.Type) async throws -> T {
        let (data, resp) = try await session.data(from: url(path))
        try check(resp, data)
        return try JSONDecoder.mirror.decode(T.self, from: data)
    }

    private func check(_ resp: URLResponse, _ data: Data) throws {
        guard let http = resp as? HTTPURLResponse else { return }
        guard (200..<300).contains(http.statusCode) else {
            let body = String(data: data, encoding: .utf8)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            throw MirrorError.status(http.statusCode, body)
        }
    }

    func calendar() async throws -> CalendarResponse { try await get("/api/calendar", as: CalendarResponse.self) }
    func weather() async throws -> Weather { try await get("/api/weather", as: Weather.self) }

    /// Shows a message on the mirror (and every open mirror page).
    func send(_ text: String, from name: String? = nil, speak: Bool = false) async throws {
        var req = URLRequest(url: try url("/api/events"))
        req.httpMethod = "POST"
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        if !token.isEmpty { req.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization") }
        let who = (name ?? Store.name)
        req.httpBody = try JSONEncoder().encode(OutgoingMessage(
            text: text,
            source: who.isEmpty ? nil : who,
            ttl: nil,
            speak: speak ? true : nil
        ))
        let (data, resp) = try await session.data(for: req)
        try check(resp, data)
    }

    /// Fetches what it can and keeps the last good copy of anything that
    /// fails, the same way the Pi's cached[T] does. Returns the merged
    /// snapshot and the first error, if any.
    @discardableResult
    func refresh() async -> (Snapshot, Error?) {
        var snap = Store.snapshot
        async let cal = attempt { try await calendar() }
        async let wx = attempt { try await weather() }
        let (c, w) = await (cal, wx)
        var firstError: Error?
        switch c {
        case .success(let v): snap.calendar = v; snap.fetchedAt = Date()
        case .failure(let e): firstError = e
        }
        switch w {
        case .success(let v): snap.weather = v; snap.fetchedAt = Date()
        case .failure(let e): firstError = firstError ?? e
        }
        Store.snapshot = snap
        return (snap, firstError)
    }
}

private func attempt<T>(_ body: () async throws -> T) async -> Result<T, Error> {
    do { return .success(try await body()) } catch { return .failure(error) }
}
