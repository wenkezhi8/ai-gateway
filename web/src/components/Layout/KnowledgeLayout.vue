<template>
  <el-container class="knowledge-layout">
    <el-header class="knowledge-header">
      <div class="brand">
        <el-icon :size="20"><Document /></el-icon>
        <span class="brand-text">知识库管理</span>
      </div>

      <nav class="nav-links">
        <router-link
          v-for="item in navItems"
          :key="item.path"
          :to="item.path"
          class="nav-link"
          :class="{ active: isActive(item.path) }"
        >
          {{ item.title }}
        </router-link>
      </nav>

      <div class="actions">
        <el-button type="primary" plain @click="goGateway">返回 AI Gateway</el-button>
      </div>
    </el-header>

    <el-main class="knowledge-main">
      <router-view v-slot="{ Component }">
        <transition name="page-fade" mode="out-in">
          <component :is="Component" />
        </transition>
      </router-view>
    </el-main>
  </el-container>
</template>

<script setup lang="ts">
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()

const navItems = [
  { path: '/knowledge/documents', title: '文档' },
  { path: '/knowledge/chat', title: '问答' },
  { path: '/knowledge/config', title: '配置' }
]

const isActive = (path: string) => route.path === path || route.path.startsWith(`${path}/`)

const goGateway = () => {
  router.push('/console')
}
</script>

<style scoped lang="scss">
.knowledge-layout {
  min-height: 100vh;
  background: linear-gradient(135deg, #fff7f0 0%, #fff3e6 45%, #fefbea 100%);
}

.knowledge-header {
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  border-bottom: 1px solid var(--border-primary);
  background: rgba(255, 255, 255, 0.88);
  backdrop-filter: blur(8px);
  padding: 0 20px;
}

.brand {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--text-primary);
}

.brand-text {
  font-weight: 700;
}

.nav-links {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.nav-link {
  text-decoration: none;
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1;
  padding: 8px 10px;
  border-radius: 10px;
  border: 1px solid transparent;
  transition: all 0.2s ease;

  &:hover {
    color: var(--text-primary);
    background: rgba(245, 108, 45, 0.08);
  }

  &.active {
    color: #a63f0a;
    background: rgba(245, 108, 45, 0.14);
    border-color: rgba(245, 108, 45, 0.28);
  }
}

.actions {
  flex-shrink: 0;
}

.knowledge-main {
  padding: 20px;
}

@media (max-width: 900px) {
  .knowledge-header {
    height: auto;
    padding: 10px 12px;
    align-items: flex-start;
    flex-direction: column;
  }

  .knowledge-main {
    padding: 12px;
  }
}
</style>
