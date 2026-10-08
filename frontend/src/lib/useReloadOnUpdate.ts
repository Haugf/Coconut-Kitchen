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
    const timer = window.setInterval(async () => {
      try {
        const html = await (await fetch('/', { cache: 'no-store' })).text()
        const latest = html.match(/<script[^>]+type="module"[^>]+src="([^"]+)"/)?.[1]
        if (latest && latest !== current) window.location.reload()
      } catch {
        /* server restarting; try again next tick */
      }
    }, intervalMs)
    return () => window.clearInterval(timer)
  }, [intervalMs])
}
