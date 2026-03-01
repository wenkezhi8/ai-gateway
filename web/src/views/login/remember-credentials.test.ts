import { describe, expect, it } from 'vitest'
import { loadRememberedCredentials, persistRememberedCredentials } from './remember-credentials'

class MemoryStorage implements Storage {
  private data = new Map<string, string>()

  get length() {
    return this.data.size
  }

  clear(): void {
    this.data.clear()
  }

  getItem(key: string): string | null {
    return this.data.has(key) ? this.data.get(key)! : null
  }

  key(index: number): string | null {
    return Array.from(this.data.keys())[index] ?? null
  }

  removeItem(key: string): void {
    this.data.delete(key)
  }

  setItem(key: string, value: string): void {
    this.data.set(key, value)
  }
}

describe('remember-credentials', () => {
  it('returns null when no remembered credentials', () => {
    const storage = new MemoryStorage()
    expect(loadRememberedCredentials(storage)).toBeNull()
  })

  it('returns null when saved value is invalid JSON', () => {
    const storage = new MemoryStorage()
    storage.setItem('login.remembered.credentials', '{bad json')
    expect(loadRememberedCredentials(storage)).toBeNull()
  })

  it('saves and loads remembered credentials when remember is true', () => {
    const storage = new MemoryStorage()
    persistRememberedCredentials(storage, true, 'admin', 'admin123')
    expect(loadRememberedCredentials(storage)).toEqual({
      username: 'admin',
      password: 'admin123'
    })
  })

  it('clears remembered credentials when remember is false', () => {
    const storage = new MemoryStorage()
    persistRememberedCredentials(storage, true, 'admin', 'admin123')
    persistRememberedCredentials(storage, false, 'admin', 'admin123')
    expect(loadRememberedCredentials(storage)).toBeNull()
  })
})
