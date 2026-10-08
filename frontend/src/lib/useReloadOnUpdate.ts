import { useEffect } from 'react'

/**
 * Reloads the page when a new build is installed. Every minute it fetches
 * index.html and compares the script it points at with the one running.
 * This replaces closing the kiosk browser on update, which could leave
 * the screen blinking while Chromium restarted.
 */
export function useReloadOnUpdate(intervalMs = 60 * 1000) {
  useEffect(() => {
    const current = document.querySelector<HTMLScriptElement>('script[type="module"][src]')?.getAttribute('src')
    if (!current) return
    // Tell the Pi what's on screen, so `mirror-status` can report it
    // without anyone walking over to look.
    const beat = () => {
      const seen = {
        scene: document.querySelector('main')?.getAttribute('data-scene') ?? null,
        headline: Boolean(document.querySelector('.t-headline')?.textContent?.trim()),
        calendarConnected: !document.querySelector('.t-acts[hidden]'),
        transitRows: document.querySelectorAll('.t-rows li').length,
        width: window.innerWidth,
        height: window.innerHeight,
      }
      fetch('/api/heartbeat', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ build: current, seen }),
      }).catch(() => {})
    }
    const firstBeat = window.setTimeout(beat, 5000)
    const beatTimer = window.setInterval(beat, intervalMs)

    const timer = window.setInterval(async () => {
      try {
        const html = await (await fetch('/', { cache: 'no-store' })).text()
        const latest = html.match(/<script[^>]+type="module"[^>]+src="([^"]+)"/)?.[1]
        if (latest && latest !== current) window.location.reload()
      } catch {
        /* server restarting; try again next tick */
      }
    }, intervalMs)
    return () => {
      window.clearInterval(timer)
      window.clearInterval(beatTimer)
      window.clearTimeout(firstBeat)
    }
  }, [intervalMs])
}
