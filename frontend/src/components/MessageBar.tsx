import type { MirrorMessage } from '../lib/useMessages'

export function MessageBar({ message }: { message: MirrorMessage | null }) {
  return (
    <div className={message ? 'message is-visible' : 'message'} role="status" aria-live="polite">
      {message && (
        <>
          {message.source && <p className="message-source">{message.source}</p>}
          <p className="message-text">{message.text}</p>
        </>
      )}
    </div>
  )
}
