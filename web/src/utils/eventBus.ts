import { onUnmounted } from 'vue'

type EventCallback = (payload?: any) => void

class EventBus {
  private events: Map<string, Set<EventCallback>> = new Map()

  on(event: string, callback: EventCallback) {
    if (!this.events.has(event)) {
      this.events.set(event, new Set())
    }
    this.events.get(event)!.add(callback)
    
    return () => this.off(event, callback)
  }

  off(event: string, callback: EventCallback) {
    const callbacks = this.events.get(event)
    if (callbacks) {
      callbacks.delete(callback)
    }
  }

  emit(event: string, payload?: any) {
    const callbacks = this.events.get(event)
    if (callbacks) {
      callbacks.forEach(cb => {
        try {
          cb(payload)
        } catch (e) {
          console.error(`EventBus callback error for ${event}:`, e)
        }
      })
    }
  }

  once(event: string, callback: EventCallback) {
    const wrapper = (payload?: any) => {
      this.off(event, wrapper)
      callback(payload)
    }
    this.on(event, wrapper)
  }

  clear() {
    this.events.clear()
  }
}

export const eventBus = new EventBus()

export const DATA_EVENTS = {
  PROVIDERS_CHANGED: 'data:providers:changed',
  ACCOUNTS_CHANGED: 'data:accounts:changed',
  MODELS_CHANGED: 'data:models:changed',
  ALERTS_CHANGED: 'data:alerts:changed',
  CACHE_CHANGED: 'data:cache:changed',
  ROUTING_CHANGED: 'data:routing:changed',
  STATS_CHANGED: 'data:stats:changed',
  ALL_DATA_REFRESH: 'data:all:refresh'
} as const

export function useEventBus(event: string, callback: EventCallback) {
  const unsubscribe = eventBus.on(event, callback)
  
  onUnmounted(() => {
    unsubscribe()
  })
  
  return { unsubscribe }
}

export function useDataRefresh(events: string[], callback: () => void) {
  const unsubscribers: (() => void)[] = []
  
  events.forEach(event => {
    unsubscribers.push(eventBus.on(event, callback))
  })
  
  onUnmounted(() => {
    unsubscribers.forEach(unsub => unsub())
  })
}
