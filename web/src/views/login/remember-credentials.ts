export interface RememberedCredentials {
  username: string
  password: string
}

const REMEMBERED_CREDENTIALS_KEY = 'login.remembered.credentials'

export function loadRememberedCredentials(storage: Storage = localStorage): RememberedCredentials | null {
  const raw = storage.getItem(REMEMBERED_CREDENTIALS_KEY)
  if (!raw) return null

  try {
    const parsed = JSON.parse(raw) as Partial<RememberedCredentials>
    if (typeof parsed.username !== 'string' || typeof parsed.password !== 'string') {
      return null
    }
    return {
      username: parsed.username,
      password: parsed.password
    }
  } catch {
    return null
  }
}

export function persistRememberedCredentials(
  storage: Storage = localStorage,
  remember: boolean,
  username: string,
  password: string
): void {
  if (!remember) {
    storage.removeItem(REMEMBERED_CREDENTIALS_KEY)
    return
  }

  storage.setItem(
    REMEMBERED_CREDENTIALS_KEY,
    JSON.stringify({
      username,
      password
    })
  )
}
