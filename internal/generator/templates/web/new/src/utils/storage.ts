/**
 * 本地存储工具
 */
export const storage = {
  get(key: string): string | null {
    try {
      return localStorage.getItem(key)
    } catch {
      return null
    }
  },

  set(key: string, value: string): void {
    try {
      localStorage.setItem(key, value)
    } catch {
      // storage unavailable
    }
  },

  remove(key: string): void {
    try {
      localStorage.removeItem(key)
    } catch {
      // storage unavailable
    }
  },

  clear(): void {
    try {
      localStorage.clear()
    } catch {
      // storage unavailable
    }
  },
}

/**
 * JSON 存储工具
 */
export const jsonStorage = {
  get<T>(key: string): T | null {
    const value = storage.get(key)
    if (!value) return null
    try {
      return JSON.parse(value) as T
    } catch {
      return null
    }
  },

  set<T>(key: string, value: T): void {
    try {
      storage.set(key, JSON.stringify(value))
    } catch {
      // serialization failed
    }
  },

  remove(key: string): void {
    storage.remove(key)
  },
}
