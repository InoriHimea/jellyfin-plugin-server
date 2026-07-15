import { useEffect, useState } from 'react'

// Debounce a fast-changing value (e.g. search input) so consumers only see
// it after the user pauses typing — keeps every keystroke from firing a
// server-side query.
export function useDebounce<T>(value: T, delayMs = 300): T {
  const [debounced, setDebounced] = useState(value)
  useEffect(() => {
    const t = setTimeout(() => setDebounced(value), delayMs)
    return () => clearTimeout(t)
  }, [value, delayMs])
  return debounced
}
