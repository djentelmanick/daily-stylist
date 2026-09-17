export function remember(key: string, value: unknown): void {
  try {
    sessionStorage.setItem(key, JSON.stringify(value))
  } catch {
    // Память кончилась или хранилище запрещено настройками - тогда просто не помним.
  }
}

export function recall<T>(key: string): T | null {
  try {
    const stored = sessionStorage.getItem(key)
    return stored === null ? null : (JSON.parse(stored) as T)
  } catch {
    return null
  }
}

export function forget(key: string): void {
  try {
    sessionStorage.removeItem(key)
  } catch {
    // Забыть не вышло - неважно: перезаписывать всё равно будет чем.
  }
}
