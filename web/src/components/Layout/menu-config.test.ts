import { describe, expect, it } from 'vitest'

import { getMenuItems } from './menu-config'

describe('layout menu config', () => {
  it('returns 12 base menu items for basic edition', () => {
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

    expect(menus).toHaveLength(12)
    expect(menus.some((m) => m.path === '/settings')).toBe(true)
  })
})
