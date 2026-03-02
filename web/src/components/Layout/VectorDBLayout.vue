<template>
  <el-container class="vector-layout">
    <el-header class="vector-header">
      <div class="brand">
        <el-icon :size="20"><DataAnalysis /></el-icon>
        <span class="brand-text">向量数据管理</span>
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

    <el-main class="vector-main">
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
  { path: '/vector-db/collections', title: '集合' },
  { path: '/vector-db/search', title: '检索' },
  { path: '/vector-db/import', title: '导入' },
  { path: '/vector-db/monitoring', title: '监控' },
  { path: '/vector-db/permissions', title: '权限' },
  { path: '/vector-db/backup', title: '备份' },
  { path: '/vector-db/audit', title: '审计' },
  { path: '/vector-db/visualization', title: '可视化' }
]

const isActive = (path: string) => route.path === path || route.path.startsWith(`${path}/`)

const goGateway = () => {
  router.push('/console')
}
</script>

<style scoped lang="scss">
.vector-layout {
  min-height: 100vh;
  background: linear-gradient(135deg, #f4f7fb 0%, #eef3ff 45%, #eaf8f4 100%);
}

.vector-header {
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  border-bottom: 1px solid var(--border-primary);
  background: rgba(255, 255, 255, 0.85);
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
    background: rgba(64, 158, 255, 0.08);
  }

  &.active {
    color: #1155cc;
    background: rgba(64, 158, 255, 0.12);
    border-color: rgba(64, 158, 255, 0.24);
  }
}

.actions {
  flex-shrink: 0;
}

.vector-main {
  padding: 20px;
}

@media (max-width: 900px) {
  .vector-header {
    height: auto;
    padding: 10px 12px;
    align-items: flex-start;
    flex-direction: column;
  }

  .vector-main {
    padding: 12px;
  }
}
</style>
