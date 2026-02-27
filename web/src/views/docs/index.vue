<template>
  <div class="docs-layout-page">
    <div class="ambient-grid"></div>
    <div class="ambient-orb orb-left"></div>
    <div class="ambient-orb orb-right"></div>

    <header class="docs-topbar">
      <router-link to="/" class="brand-link">
        <span class="brand-mark">AG</span>
        <span class="brand-text">AI Gateway Docs</span>
      </router-link>

      <nav class="top-nav" aria-label="文档导航">
        <router-link to="/docs/getting-started">入门指南</router-link>
        <router-link to="/docs/api">API 参考</router-link>
        <router-link to="/docs/sdk">SDK 示例</router-link>
      </nav>

      <div class="top-actions">
        <router-link :to="loginRoute" class="btn btn-outline">控制台登录</router-link>
      </div>
    </header>

    <main class="docs-layout-shell">
      <aside class="docs-sidebar" aria-label="文档侧边栏">
        <section v-for="group in navGroups" :key="group.title" class="sidebar-group">
          <h2>{{ group.title }}</h2>
          <router-link
            v-for="item in group.items"
            :key="item.to"
            :to="item.to"
            class="sidebar-link"
            :class="{ active: isActive(item.to) }"
          >
            {{ item.label }}
          </router-link>
        </section>
      </aside>

      <section class="docs-content">
        <div class="mobile-nav" aria-label="移动端文档导航">
          <router-link
            v-for="item in flatNavItems"
            :key="item.to"
            :to="item.to"
            class="mobile-nav-item"
            :class="{ active: isActive(item.to) }"
          >
            {{ item.label }}
          </router-link>
        </div>

        <router-view />
      </section>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { LOGIN_ROUTE } from '@/constants/navigation'

interface NavItem {
  label: string
  to: string
}

interface NavGroup {
  title: string
  items: NavItem[]
}

const route = useRoute()
const loginRoute = LOGIN_ROUTE

const navGroups: NavGroup[] = [
  {
    title: '快速开始',
    items: [
      { label: '入门指南', to: '/docs/getting-started' },
      { label: '安装向导', to: '/docs/wizard' }
    ]
  },
  {
    title: '开发参考',
    items: [
      { label: 'API 参考', to: '/docs/api' },
      { label: 'SDK 示例', to: '/docs/sdk' },
      { label: '服务商', to: '/docs/providers' }
    ]
  },
  {
    title: '运维管理',
    items: [
      { label: '管理 API', to: '/docs/admin' },
      { label: '错误码', to: '/docs/errors' }
    ]
  }
]

const flatNavItems = computed(() => navGroups.flatMap((group) => group.items))

const isActive = (targetPath: string) => route.path === targetPath
</script>

<style scoped>
.docs-layout-page {
  --bg-0: #f7fbff;
  --bg-1: #eef6ff;
  --bg-2: #e4f0ff;
  --text-primary: #0f2438;
  --text-secondary: #4f667b;
  --accent: #0ea5a0;
  --accent-strong: #1388c9;
  --line: rgba(98, 136, 164, 0.24);
  position: relative;
  min-height: 100vh;
  overflow: hidden;
  padding: 1rem 1.1rem 2rem;
  background:
    radial-gradient(circle at 10% 12%, rgba(14, 165, 160, 0.2), transparent 38%),
    radial-gradient(circle at 88% 10%, rgba(244, 183, 64, 0.14), transparent 34%),
    linear-gradient(145deg, var(--bg-0) 0%, var(--bg-1) 46%, var(--bg-2) 100%);
  color: var(--text-primary);
  font-family: 'Space Grotesk', 'PingFang SC', 'Microsoft YaHei', sans-serif;
}

.ambient-grid {
  position: absolute;
  inset: 0;
  pointer-events: none;
  background-image:
    linear-gradient(rgba(114, 152, 181, 0.1) 1px, transparent 1px),
    linear-gradient(90deg, rgba(114, 152, 181, 0.1) 1px, transparent 1px);
  background-size: 32px 32px;
  mask-image: radial-gradient(circle at center, black 36%, transparent 84%);
}

.ambient-orb {
  position: absolute;
  width: 22rem;
  height: 22rem;
  border-radius: 999px;
  filter: blur(64px);
  opacity: 0.5;
  pointer-events: none;
}

.orb-left {
  left: -8rem;
  top: 22rem;
  background: rgba(14, 165, 160, 0.24);
}

.orb-right {
  right: -7rem;
  top: 12rem;
  background: rgba(244, 183, 64, 0.2);
}

.docs-topbar,
.docs-layout-shell {
  position: relative;
  z-index: 2;
  max-width: 1240px;
  margin: 0 auto;
}

.docs-topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border: 1px solid var(--line);
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.76);
  backdrop-filter: blur(10px);
  padding: 0.7rem 0.9rem;
}

.brand-link {
  display: inline-flex;
  align-items: center;
  gap: 0.65rem;
  text-decoration: none;
  color: inherit;
}

.brand-mark {
  display: grid;
  place-items: center;
  width: 2rem;
  height: 2rem;
  border-radius: 10px;
  color: #f4feff;
  font-size: 0.78rem;
  font-weight: 700;
  background: linear-gradient(135deg, var(--accent), #46b7e5);
}

.brand-text {
  font-weight: 700;
}

.top-nav {
  display: flex;
  gap: 1rem;
}

.top-nav a {
  color: var(--text-secondary);
  text-decoration: none;
  transition: color 0.2s ease;
}

.top-nav a:hover,
.top-nav a.router-link-active {
  color: var(--text-primary);
}

.docs-layout-shell {
  margin-top: 1rem;
  border: 1px solid var(--line);
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.78);
  backdrop-filter: blur(8px);
  padding: 1rem;
  display: grid;
  grid-template-columns: 280px 1fr;
  gap: 1rem;
}

.docs-sidebar {
  position: sticky;
  top: 1rem;
  align-self: start;
  border: 1px solid var(--line);
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.84);
  padding: 0.85rem;
}

.sidebar-group + .sidebar-group {
  margin-top: 0.85rem;
  padding-top: 0.75rem;
  border-top: 1px solid rgba(98, 136, 164, 0.2);
}

.sidebar-group h2 {
  margin: 0;
  font-size: 0.84rem;
  color: var(--text-secondary);
}

.sidebar-link {
  margin-top: 0.45rem;
  display: block;
  border: 1px solid rgba(98, 136, 164, 0.24);
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.82);
  padding: 0.5rem 0.58rem;
  color: var(--text-secondary);
  text-decoration: none;
  font-size: 0.9rem;
  transition: border-color 0.2s ease, color 0.2s ease, box-shadow 0.2s ease;
}

.sidebar-link.active {
  border-color: rgba(14, 165, 160, 0.62);
  color: var(--text-primary);
  box-shadow: 0 8px 18px rgba(19, 136, 201, 0.14);
}

.docs-content {
  min-width: 0;
}

.mobile-nav {
  display: none;
}

.mobile-nav-item {
  display: inline-flex;
  align-items: center;
  border: 1px solid rgba(98, 136, 164, 0.24);
  border-radius: 999px;
  padding: 0.34rem 0.66rem;
  background: rgba(255, 255, 255, 0.86);
  text-decoration: none;
  color: var(--text-secondary);
  font-size: 0.82rem;
}

.mobile-nav-item.active {
  border-color: rgba(14, 165, 160, 0.62);
  color: var(--text-primary);
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 2.45rem;
  padding: 0 1rem;
  border-radius: 12px;
  text-decoration: none;
  font-weight: 600;
}

.btn-outline {
  border: 1px solid rgba(14, 165, 160, 0.34);
  background: rgba(255, 255, 255, 0.86);
  color: #0f5f66;
}

@media (max-width: 1080px) {
  .docs-layout-shell {
    grid-template-columns: 1fr;
  }

  .docs-sidebar {
    display: none;
  }

  .mobile-nav {
    display: flex;
    flex-wrap: wrap;
    gap: 0.45rem;
    margin-bottom: 0.85rem;
  }
}

@media (max-width: 820px) {
  .docs-layout-page {
    padding: 0.8rem 0.8rem 1.5rem;
  }

  .top-nav {
    display: none;
  }

  .brand-text {
    font-size: 0.95rem;
  }
}
</style>
