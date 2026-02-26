<template>
  <el-container class="layout-container">
    <!-- 侧边栏 -->
    <el-aside class="sidebar" :class="{ 'is-collapsed': isCollapse }">
      <div class="logo">
        <div class="logo-icon">
          <el-icon :size="24"><Platform /></el-icon>
        </div>
        <transition name="fade">
          <span v-show="!isCollapse" class="logo-text">AI Gateway</span>
        </transition>
      </div>

      <nav class="sidebar-nav">
        <el-tooltip
          v-for="item in menuItems"
          :key="item.path"
          :content="item.title"
          placement="right"
          :disabled="!isCollapse"
          :show-after="300"
        >
          <router-link
            :to="item.path"
            class="nav-item"
            :class="{ active: isActive(item.path) }"
          >
            <el-icon :size="20"><component :is="item.icon" /></el-icon>
            <transition name="fade">
              <span v-show="!isCollapse" class="nav-text">{{ item.title }}</span>
            </transition>
          </router-link>
        </el-tooltip>
      </nav>

      <!-- 侧边栏底部 -->
      <div class="sidebar-footer">
        <button class="collapse-btn" @click="toggleCollapse">
          <el-icon :size="18">
            <Fold v-if="!isCollapse" />
            <Expand v-else />
          </el-icon>
        </button>
      </div>
    </el-aside>

    <!-- 主内容区 -->
    <el-container class="main-container">
      <!-- 顶部导航栏 -->
      <el-header class="header glass-header">
        <div class="header-left">
          <el-breadcrumb separator="/">
            <el-breadcrumb-item :to="{ path: '/dashboard' }">首页</el-breadcrumb-item>
            <el-breadcrumb-item v-if="currentTitle">{{ currentTitle }}</el-breadcrumb-item>
          </el-breadcrumb>
        </div>

        <div class="header-right">
          <!-- AI对话快捷入口 -->
          <el-tooltip content="独立对话页面 (新窗口)" placement="bottom">
            <a href="/p/chat" target="_blank" class="chat-btn">
              <el-icon :size="18"><ChatDotRound /></el-icon>
              <span class="chat-text">AI 对话</span>
              <span class="external-badge">↗</span>
            </a>
          </el-tooltip>

          <!-- 文档中心 -->
          <el-tooltip content="文档中心" placement="bottom">
            <router-link to="/docs" class="docs-btn" :class="{ active: isActive('/docs') }">
              <el-icon :size="18"><Document /></el-icon>
              <span class="docs-text">文档</span>
            </router-link>
          </el-tooltip>

          <!-- GitHub 仓库 -->
          <el-tooltip content="GitHub 仓库" placement="bottom">
            <a href="https://github.com/wenkezhi8/ai-gateway" target="_blank" class="github-btn">
              <svg height="18" width="18" viewBox="0 0 16 16" fill="currentColor">
                <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z"></path>
              </svg>
              <span class="github-text">GitHub</span>
            </a>
          </el-tooltip>

          <div class="header-divider"></div>

          <!-- 主题切换 -->
          <el-tooltip :content="themeTooltip" placement="bottom">
            <button class="theme-btn" @click="toggleTheme">
              <el-icon :size="18">
                <Sunny v-if="isDarkMode" />
                <Moon v-else />
              </el-icon>
            </button>
          </el-tooltip>

          <!-- 通知 -->
          <el-badge :value="notificationCount" :hidden="notificationCount === 0" class="notification-badge">
            <button class="icon-btn">
              <el-icon :size="18"><Bell /></el-icon>
            </button>
          </el-badge>

          <!-- 用户菜单 -->
          <el-dropdown trigger="click" @command="handleUserCommand">
            <div class="user-dropdown">
              <el-avatar :size="32" class="user-avatar">
                <el-icon><UserFilled /></el-icon>
              </el-avatar>
              <span class="username">Admin</span>
              <el-icon :size="14"><ArrowDown /></el-icon>
            </div>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="profile">
                  <el-icon><User /></el-icon>
                  个人设置
                </el-dropdown-item>
                <el-dropdown-item command="settings">
                  <el-icon><Setting /></el-icon>
                  系统设置
                </el-dropdown-item>
                <el-dropdown-item divided command="logout">
                  <el-icon><SwitchButton /></el-icon>
                  退出登录
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>

      <!-- 内容区域 -->
      <el-main class="main-content">
        <router-view v-slot="{ Component }">
          <transition name="page-fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useTheme } from '@/composables/useTheme'
import { useUserStore } from '@/store/user'

const route = useRoute()
const router = useRouter()
const { currentTheme, toggleTheme, isDark } = useTheme()
const userStore = useUserStore()

const isCollapse = ref(false)
const notificationCount = ref(3)

const menuItems = [
  { path: '/dashboard', title: '监控仪表盘', icon: 'Monitor' },
  { path: '/ops', title: '运维监控', icon: 'Operation' },
  { path: '/chat', title: 'AI 对话', icon: 'ChatDotRound' },
  { path: '/api-management', title: 'API 管理', icon: 'Connection' },
  { path: '/model-management', title: '模型管理', icon: 'Collection' },
  { path: '/providers-accounts', title: '账号与限额', icon: 'Key' },
  { path: '/usage', title: 'API 使用统计', icon: 'DataLine' },
  { path: '/routing', title: '路由策略', icon: 'Guide' },
  { path: '/cache', title: '缓存管理', icon: 'Box' },
  { path: '/alerts', title: '告警管理', icon: 'Bell' },
  { path: '/settings', title: '系统设置', icon: 'Setting' }
]

const resolvePath = (path: string) => {
  return path.startsWith('/') ? path : `/${path}`
}

const currentTitle = computed(() => route.meta.title as string || '')

const isDarkMode = computed(() => isDark())

const themeTooltip = computed(() => {
  const themeMap: Record<string, string> = {
    light: '当前：亮色模式',
    dark: '当前：暗色模式',
    auto: '当前：跟随系统'
  }
  return themeMap[currentTheme.value] + ' (点击切换)'
})

const isActive = (path: string) => {
  const resolvedPath = resolvePath(path)
  // 精确匹配
  if (route.path === resolvedPath) {
    return true
  }
  // 前缀匹配：确保是子路径（避免 /test 匹配 /test-center）
  if (resolvedPath !== '/' && route.path.startsWith(resolvedPath + '/')) {
    return true
  }
  return false
}

const toggleCollapse = () => {
  isCollapse.value = !isCollapse.value
}

const handleUserCommand = (command: string) => {
  switch (command) {
    case 'profile':
      router.push('/settings')
      break
    case 'settings':
      router.push('/settings')
      break
    case 'logout':
      userStore.logout()
      router.push('/login')
      break
  }
}
</script>

<style scoped lang="scss">
.layout-container {
  height: 100vh;
  overflow: hidden;
}

// ===== 侧边栏 =====
.sidebar {
  width: 240px !important;
  min-width: 240px;
  max-width: 240px;
  background: var(--sidebar-bg);
  backdrop-filter: blur(20px) saturate(180%);
  -webkit-backdrop-filter: blur(20px) saturate(180%);
  display: flex;
  flex-direction: column;
  transition: all 0.35s cubic-bezier(0.4, 0, 0.2, 1);
  overflow: hidden;
  border-right: 1px solid rgba(255, 255, 255, 0.1);
  flex-shrink: 0;
  flex-basis: 240px;

  &.is-collapsed {
    width: 64px !important;
    min-width: 64px !important;
    max-width: 64px !important;
    flex-basis: 64px;

    .logo-text,
    .nav-text {
      display: none !important;
    }

    .nav-item {
      justify-content: center;
      padding: var(--spacing-md);
      gap: 0;

      &:hover {
        transform: scale(1.08);
      }
    }

    .logo {
      justify-content: center;
      padding: 0 var(--spacing-sm);
      gap: 0;
    }

    .sidebar-nav {
      padding: var(--spacing-sm);
    }

    .sidebar-footer {
      padding: var(--spacing-sm);
    }
  }
}

.logo {
  height: 64px;
  display: flex;
  align-items: center;
  padding: 0 var(--spacing-xl);
  gap: var(--spacing-lg);
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  background: linear-gradient(135deg, rgba(0, 122, 255, 0.08), transparent);

  .logo-icon {
    width: 36px;
    height: 36px;
    background: linear-gradient(135deg, var(--color-primary), var(--color-primary-light));
    border-radius: var(--border-radius-lg);
    display: flex;
    align-items: center;
    justify-content: center;
    color: white;
    flex-shrink: 0;
    box-shadow: 0 0 20px rgba(0, 122, 255, 0.3);
    transition: box-shadow 0.3s ease;

    &:hover {
      box-shadow: 0 0 30px rgba(0, 122, 255, 0.5);
    }
  }

  .logo-text {
    font-size: var(--font-size-lg);
    font-weight: var(--font-weight-semibold);
    color: white;
    white-space: nowrap;
    letter-spacing: 0.5px;
  }
}

.sidebar-nav {
  flex: 1;
  padding: var(--spacing-lg);
  overflow-y: auto;

  &::-webkit-scrollbar {
    width: 6px;
  }

  &::-webkit-scrollbar-track {
    background: transparent;
  }

  &::-webkit-scrollbar-thumb {
    background: rgba(255, 255, 255, 0.15);
    border-radius: 3px;

    &:hover {
      background: rgba(255, 255, 255, 0.25);
    }
  }
}

.nav-divider {
  height: 1px;
  background: rgba(255, 255, 255, 0.1);
  margin: var(--spacing-md) 0;
}

.docs-entry {
  background: linear-gradient(135deg, rgba(103, 194, 58, 0.15), rgba(64, 158, 255, 0.15));
  border: 1px solid rgba(255, 255, 255, 0.1);
  
  &:hover {
    background: linear-gradient(135deg, rgba(103, 194, 58, 0.25), rgba(64, 158, 255, 0.25));
    transform: translateX(4px);
  }
  
  &.active {
    background: linear-gradient(135deg, rgba(103, 194, 58, 0.3), rgba(64, 158, 255, 0.3));
    box-shadow: 0 4px 16px rgba(103, 194, 58, 0.2);
  }
}

.nav-item {
  display: flex;
  align-items: center;
  gap: var(--spacing-lg);
  padding: var(--spacing-lg) var(--spacing-xl);
  margin-bottom: var(--spacing-xs);
  color: var(--sidebar-text);
  text-decoration: none;
  border-radius: var(--border-radius-lg);
  transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
  position: relative;

  &:hover {
    background: rgba(255, 255, 255, 0.1);
    color: var(--sidebar-text-hover);
    transform: translateX(4px);

    .el-icon {
      transform: scale(1.1);
    }
  }

  &.active {
    background: linear-gradient(135deg, var(--color-primary), var(--color-primary-light));
    color: white;
    font-weight: var(--font-weight-semibold);
    box-shadow: 0 4px 16px rgba(0, 122, 255, 0.35), inset 0 1px 0 rgba(255, 255, 255, 0.2);
    transform: translateX(0);

    // 左侧指示条
    &::before {
      content: '';
      position: absolute;
      left: 0;
      top: 50%;
      transform: translateY(-50%);
      width: 3px;
      height: 60%;
      background: white;
      border-radius: 0 2px 2px 0;
    }

    .el-icon {
      color: white;
    }
  }

  .el-icon {
    transition: transform 0.2s ease;
  }

  .nav-text {
    white-space: nowrap;
    font-size: var(--font-size-md);
  }
}

.sidebar-footer {
  padding: var(--spacing-lg);
  border-top: 1px solid rgba(255, 255, 255, 0.1);
  position: relative;
  z-index: 10;

  .collapse-btn {
    width: 100%;
    height: 44px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--sidebar-active-bg);
    border: none;
    border-radius: var(--border-radius-lg);
    color: var(--sidebar-text);
    cursor: pointer;
    transition: all var(--transition-spring);
    pointer-events: auto;

    &:hover {
      background: rgba(255, 255, 255, 0.2);
      color: white;
      transform: scale(1.02);
    }

    &:active {
      transform: scale(0.98);
    }
  }
}

// ===== 顶部导航栏 =====
.glass-header {
  background: var(--bg-glass);
  backdrop-filter: var(--blur-lg);
  -webkit-backdrop-filter: var(--blur-lg);
  border-bottom: 1px solid var(--border-primary);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 var(--spacing-xl);
  height: 64px !important;
}

.header-left {
  display: flex;
  align-items: center;
}

.header-right {
  display: flex;
  align-items: center;
  gap: var(--spacing-md);
}

.chat-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 14px;
  background: linear-gradient(135deg, var(--color-primary), var(--color-primary-light));
  border: none;
  border-radius: var(--border-radius-lg);
  color: white;
  cursor: pointer;
  transition: all var(--transition-fast);
  text-decoration: none;
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);

  &:hover {
    transform: translateY(-1px);
    box-shadow: 0 4px 12px rgba(0, 122, 255, 0.4);
  }

  .external-badge {
    font-size: 10px;
    opacity: 0.8;
    margin-left: 2px;
  }

  .chat-text {
    @media (max-width: 768px) {
      display: none;
    }
  }
}

.docs-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 14px;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-primary);
  border-radius: var(--border-radius-lg);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
  text-decoration: none;
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);

  &:hover {
    background: var(--color-gray-300);
    color: var(--text-primary);
    transform: translateY(-1px);
  }

  &.active {
    background: linear-gradient(135deg, rgba(103, 194, 58, 0.15), rgba(64, 158, 255, 0.15));
    color: var(--color-primary);
    border-color: var(--color-primary);
  }

  .docs-text {
    @media (max-width: 768px) {
      display: none;
    }
  }
}

.github-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 14px;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-primary);
  border-radius: var(--border-radius-lg);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
  text-decoration: none;
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);

  &:hover {
    background: #24292e;
    color: #fff;
    border-color: #24292e;
    transform: translateY(-1px);
  }

  svg {
    flex-shrink: 0;
  }

  .github-text {
    @media (max-width: 768px) {
      display: none;
    }
  }
}

.header-divider {
  width: 1px;
  height: 24px;
  background: var(--border-primary);
  margin: 0 var(--spacing-xs);
}

.theme-btn,
.icon-btn {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-tertiary);
  border: none;
  border-radius: var(--border-radius-md);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);

  &:hover {
    background: var(--color-gray-300);
    color: var(--text-primary);
  }
}

.notification-badge {
  :deep(.el-badge__content) {
    background-color: var(--color-danger);
  }
}

.user-dropdown {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  padding: 6px 12px;
  background: var(--bg-tertiary);
  border-radius: var(--border-radius-lg);
  cursor: pointer;
  transition: all var(--transition-fast);

  &:hover {
    background: var(--color-gray-300);
  }

  .user-avatar {
    background: var(--color-primary);
    color: white;
  }

  .username {
    font-weight: var(--font-weight-medium);
    color: var(--text-primary);
  }
}

// ===== 主内容区 =====
.main-container {
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.main-content {
  flex: 1;
  background-color: var(--bg-secondary);
  padding: var(--spacing-xl);
  overflow-y: auto;

  &::-webkit-scrollbar {
    width: 6px;
  }

  &::-webkit-scrollbar-track {
    background: transparent;
  }

  &::-webkit-scrollbar-thumb {
    background: var(--color-gray-400);
    border-radius: 3px;
  }
}

// ===== 动画 =====
.fade-enter-active,
.fade-leave-active {
  transition: opacity var(--transition-fast);
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

.page-fade-enter-active,
.page-fade-leave-active {
  transition: all var(--transition-normal);
}

.page-fade-enter-from {
  opacity: 0;
  transform: translateY(10px);
}

.page-fade-leave-to {
  opacity: 0;
  transform: translateY(-10px);
}

// ===== 响应式设计 =====
@media (max-width: 1440px) {
  // 在大屏笔记本上稍微缩小侧边栏
  .sidebar:not(.el-aside--collapsed) {
    width: 260px !important;
  }
}

@media (max-width: 1280px) {
  // 在标准笔记本上使用适中宽度
  .sidebar:not(.el-aside--collapsed) {
    width: 240px !important;
  }
  
  .logo {
    padding: 0 var(--spacing-lg);
    
    .logo-text {
      font-size: var(--font-size-md);
    }
  }
  
  .nav-item {
    padding: var(--spacing-md) var(--spacing-lg);
    gap: var(--spacing-md);
  }
}

@media (max-width: 1024px) {
  // 在平板设备上自动折叠侧边栏
  .sidebar {
    width: 72px !important;
    
    .logo-text, .nav-text {
      display: none !important;
    }
    
    .logo {
      justify-content: center;
      padding: 0 var(--spacing-md);
    }
    
    .nav-item {
      justify-content: center;
      padding: var(--spacing-lg);
    }
    
    .sidebar-footer {
      padding: var(--spacing-md);
    }
  }
}

@media (max-width: 768px) {
  // 在手机上隐藏侧边栏
  .sidebar {
    display: none;
  }
  
  .main-container {
    margin-left: 0 !important;
  }
}

// 打印样式
@media print {
  .sidebar {
    display: none;
  }
  
  .main-container {
    margin-left: 0 !important;
  }
}
</style>
