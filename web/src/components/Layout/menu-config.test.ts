import { describe, expect, it } from 'vitest'

import { getMenuItems } from './menu-config'

describe('layout menu config', () => {
  it('basic edition should hide ollama menu', () => {
    const menus = getMenuItems({
      type: 'basic',
      features: {
        vector_cache: false,
        vector_db_management: false,
        knowledge_base: false,
        cold_hot_tiering: false
      },
      display_name: '基础版',
      description: '',
      dependencies: ['redis']
    })

    expect(menus.some((m) => m.path === '/ollama')).toBe(false)
    expect(menus.some((m) => m.path === '/settings')).toBe(true)

    const providersAccountsMenu = menus.find((m) => m.path === '/providers-accounts')
    expect(providersAccountsMenu?.title).toBe('AI服务商')
  })

  it('standard edition should show ollama menu', () => {
    const menus = getMenuItems({
      type: 'standard',
      features: {
        vector_cache: true,
        vector_db_management: false,
        knowledge_base: false,
        cold_hot_tiering: false
      },
      display_name: '标准版',
      description: '',
      dependencies: ['redis', 'ollama']
    })

    expect(menus.some((m) => m.path === '/ollama')).toBe(true)
  })

  it('should keep providers accounts path stable', () => {
    const menus = getMenuItems(null)
    expect(menus.some((m) => m.path === '/providers-accounts' && m.title === 'AI服务商')).toBe(true)
  })
})
