import { useEffect, useState } from 'react'

/** Current time, re-rendering every `intervalMs`, aligned to the interval. */
export function useNow(intervalMs = 1000): Date {
  const [now, setNow] = useState(() => new Date())
  useEffect(() => {
    let timer: number
    const tick = () => {
      setNow(new Date())
      timer = window.setTimeout(tick, intervalMs - (Date.now() % intervalMs))
    }
    timer = window.setTimeout(tick, intervalMs - (Date.now() % intervalMs))
    return () => window.clearTimeout(timer)
  }, [intervalMs])
  return now
}
