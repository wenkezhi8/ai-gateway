<template>
  <div class="settings-page">
    <el-row :gutter="24">
      <!-- 左侧设置菜单 -->
      <el-col :span="6">
        <el-card class="settings-nav" shadow="never">
          <div class="nav-list">
            <div
              v-for="item in settingsMenu"
              :key="item.key"
              class="nav-item"
              :class="{ active: activeSection === item.key }"
              @click="activeSection = item.key"
            >
              <el-icon :size="18"><component :is="item.icon" /></el-icon>
              <span>{{ item.label }}</span>
            </div>
          </div>
        </el-card>
      </el-col>

      <!-- 右侧设置内容 -->
      <el-col :span="18">
        <!-- 外观设置 -->
        <el-card v-show="activeSection === 'appearance'" class="settings-card" shadow="never">
          <template #header>
            <div class="card-header">
              <el-icon :size="20"><Brush /></el-icon>
              <span>外观设置</span>
            </div>
          </template>

          <el-form label-width="120px" class="settings-form">
            <el-form-item label="主题风格">
              <el-radio-group v-model="settings.themeVariant" @change="handleThemeVariantChange">
                <el-radio-button value="apple">Apple</el-radio-button>
                <el-radio-button value="dashboard">仪表盘</el-radio-button>
              </el-radio-group>
            </el-form-item>
            <el-form-item label="主题模式">
              <el-radio-group v-model="settings.theme" @change="handleThemeChange">
                <el-radio-button value="light">亮色</el-radio-button>
                <el-radio-button value="dark">暗色</el-radio-button>
                <el-radio-button value="auto">跟随系统</el-radio-button>
              </el-radio-group>
            </el-form-item>

            <el-form-item label="主题色">
              <div class="color-picker-group">
                <div
                  v-for="color in themeColors"
                  :key="color"
                  class="color-item"
                  :class="{ active: settings.primaryColor === color }"
                  :style="{ backgroundColor: color }"
                  @click="handleColorChange(color)"
                />
              </div>
            </el-form-item>

            <el-form-item label="圆角大小">
              <el-slider v-model="settings.borderRadius" :min="4" :max="24" :step="2" show-input />
            </el-form-item>

            <el-form-item label="启用动画">
              <el-switch v-model="settings.enableAnimation" />
            </el-form-item>
          </el-form>
        </el-card>

        <!-- 网关配置 -->
        <el-card v-show="activeSection === 'gateway'" class="settings-card" shadow="never">
          <template #header>
            <div class="card-header">
              <el-icon :size="20"><Connection /></el-icon>
              <span>网关配置</span>
            </div>
          </template>

          <el-form label-width="120px" class="settings-form">
            <el-form-item label="监听地址">
              <el-input v-model="settings.gateway.host" placeholder="0.0.0.0" />
            </el-form-item>

            <el-form-item label="监听端口">
              <el-input-number v-model="settings.gateway.port" :min="1" :max="65535" />
            </el-form-item>

            <el-form-item label="请求超时">
              <el-input-number v-model="settings.gateway.timeout" :min="1" :max="300" />
              <span class="form-hint">秒</span>
            </el-form-item>

            <el-form-item label="最大连接数">
              <el-input-number v-model="settings.gateway.maxConnections" :min="1" :max="10000" />
            </el-form-item>

            <el-form-item label="启用CORS">
              <el-switch v-model="settings.gateway.enableCors" />
            </el-form-item>

            <el-form-item label="CORS域名">
              <el-input
                v-model="settings.gateway.corsOrigins"
                placeholder="* 或域名列表，逗号分隔"
                :disabled="!settings.gateway.enableCors"
              />
            </el-form-item>
          </el-form>
        </el-card>

        <!-- 缓存配置 -->
        <el-card v-show="activeSection === 'cache'" class="settings-card" shadow="never">
          <template #header>
            <div class="card-header">
              <el-icon :size="20"><Box /></el-icon>
              <span>缓存配置</span>
            </div>
          </template>

          <el-form label-width="120px" class="settings-form">
            <el-form-item label="启用缓存">
              <el-switch v-model="settings.cache.enabled" />
            </el-form-item>

            <el-form-item label="缓存类型">
              <el-select v-model="settings.cache.type" :disabled="!settings.cache.enabled">
                <el-option label="内存缓存" value="memory" />
                <el-option label="Redis缓存" value="redis" />
              </el-select>
            </el-form-item>

            <el-form-item label="默认TTL">
              <el-input-number v-model="settings.cache.defaultTTL" :min="60" :max="86400" />
              <span class="form-hint">秒</span>
            </el-form-item>

            <el-form-item label="最大缓存大小">
              <el-input-number v-model="settings.cache.maxSize" :min="100" :max="10000" />
              <span class="form-hint">MB</span>
            </el-form-item>

            <el-divider content-position="left">Redis配置</el-divider>

            <el-form-item label="Redis地址">
              <el-input
                v-model="settings.cache.redis.host"
                placeholder="localhost:6379"
                :disabled="settings.cache.type !== 'redis'"
              />
            </el-form-item>

            <el-form-item label="Redis密码">
              <el-input
                v-model="settings.cache.redis.password"
                type="password"
                placeholder="可选"
                :disabled="settings.cache.type !== 'redis'"
                show-password
              />
            </el-form-item>

            <el-form-item label="Redis数据库">
              <el-input-number v-model="settings.cache.redis.db" :min="0" :max="15" :disabled="settings.cache.type !== 'redis'" />
            </el-form-item>
          </el-form>
        </el-card>

        <!-- 日志配置 -->
        <el-card v-show="activeSection === 'logging'" class="settings-card" shadow="never">
          <template #header>
            <div class="card-header">
              <el-icon :size="20"><Document /></el-icon>
              <span>日志配置</span>
            </div>
          </template>

          <el-form label-width="120px" class="settings-form">
            <el-form-item label="日志级别">
              <el-select v-model="settings.logging.level">
                <el-option label="Debug" value="debug" />
                <el-option label="Info" value="info" />
                <el-option label="Warning" value="warn" />
                <el-option label="Error" value="error" />
              </el-select>
            </el-form-item>

            <el-form-item label="日志格式">
              <el-select v-model="settings.logging.format">
                <el-option label="JSON" value="json" />
                <el-option label="文本" value="text" />
              </el-select>
            </el-form-item>

            <el-form-item label="日志输出">
              <el-checkbox-group v-model="settings.logging.outputs">
                <el-checkbox value="console">控制台</el-checkbox>
                <el-checkbox value="file">文件</el-checkbox>
              </el-checkbox-group>
            </el-form-item>

            <el-form-item label="日志路径" v-if="settings.logging.outputs.includes('file')">
              <el-input v-model="settings.logging.filePath" placeholder="/var/log/ai-gateway" />
            </el-form-item>

            <el-form-item label="最大文件大小" v-if="settings.logging.outputs.includes('file')">
              <el-input-number v-model="settings.logging.maxFileSize" :min="1" :max="100" />
              <span class="form-hint">MB</span>
            </el-form-item>

            <el-form-item label="保留天数" v-if="settings.logging.outputs.includes('file')">
              <el-input-number v-model="settings.logging.maxBackups" :min="1" :max="90" />
              <span class="form-hint">天</span>
            </el-form-item>
          </el-form>
        </el-card>

        <!-- 安全配置 -->
        <el-card v-show="activeSection === 'security'" class="settings-card" shadow="never">
          <template #header>
            <div class="card-header">
              <el-icon :size="20"><Lock /></el-icon>
              <span>安全配置</span>
            </div>
          </template>

          <el-form label-width="120px" class="settings-form">
            <el-form-item label="启用认证">
              <el-switch v-model="settings.security.enabled" />
            </el-form-item>

            <el-form-item label="认证方式">
              <el-select v-model="settings.security.type" :disabled="!settings.security.enabled">
                <el-option label="API Key" value="apikey" />
                <el-option label="JWT" value="jwt" />
                <el-option label="OAuth2" value="oauth2" />
              </el-select>
            </el-form-item>

            <el-form-item label="启用限流">
              <el-switch v-model="settings.security.rateLimit" />
            </el-form-item>

            <el-form-item label="每分钟限制" v-if="settings.security.rateLimit">
              <el-input-number v-model="settings.security.rateLimitRPM" :min="1" :max="10000" />
              <span class="form-hint">请求</span>
            </el-form-item>

            <el-form-item label="IP白名单">
              <el-input
                v-model="settings.security.ipWhitelist"
                type="textarea"
                :rows="3"
                placeholder="每行一个IP或CIDR"
              />
            </el-form-item>
          </el-form>
        </el-card>

        <!-- 关于 -->
        <el-card v-show="activeSection === 'about'" class="settings-card" shadow="never">
          <template #header>
            <div class="card-header">
              <el-icon :size="20"><InfoFilled /></el-icon>
              <span>关于</span>
            </div>
          </template>

          <div class="about-content">
            <div class="app-info">
              <div class="app-logo">
                <el-icon :size="48"><Platform /></el-icon>
              </div>
              <h2>AI Gateway</h2>
              <p class="version">版本 1.0.0</p>
            </div>

            <el-descriptions :column="1" border class="system-info">
              <el-descriptions-item label="系统名称">AI智能网关</el-descriptions-item>
              <el-descriptions-item label="运行环境">Production</el-descriptions-item>
              <el-descriptions-item label="Go版本">1.21+</el-descriptions-item>
              <el-descriptions-item label="前端框架">Vue 3 + Element Plus</el-descriptions-item>
              <el-descriptions-item label="后端框架">Go + Gin</el-descriptions-item>
            </el-descriptions>

            <div class="links">
              <el-button type="primary" link>
                <el-icon><Document /></el-icon>
                查看文档
              </el-button>
              <el-button type="primary" link>
                <el-icon><Link /></el-icon>
                GitHub
              </el-button>
            </div>
          </div>
        </el-card>

        <!-- 保存按钮 -->
        <div class="settings-actions">
          <el-button @click="resetSettings">重置</el-button>
          <el-button type="primary" @click="saveSettings">保存设置</el-button>
        </div>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { useTheme } from '@/composables/useTheme'
import { SETTINGS_MENU_ITEMS, THEME_COLOR_OPTIONS, createSettingsDefaults } from '@/constants/pages/settings'

const { setTheme, setVariant, currentTheme } = useTheme()

const activeSection = ref('appearance')

const settingsMenu = [...SETTINGS_MENU_ITEMS]

const themeColors = [...THEME_COLOR_OPTIONS]

const settings = reactive(createSettingsDefaults())

const handleThemeChange = (theme: string) => {
  setTheme(theme as 'light' | 'dark' | 'auto')
}

const handleThemeVariantChange = (variant: string) => {
  setVariant(variant as 'apple' | 'dashboard')
}

// 应用主题色到 CSS 变量
const applyPrimaryColor = (color: string) => {
  document.documentElement.style.setProperty('--color-primary', color)
  // 生成浅色版本
  const r = parseInt(color.slice(1, 3), 16)
  const g = parseInt(color.slice(3, 5), 16)
  const b = parseInt(color.slice(5, 7), 16)
  const lightColor = `rgba(${r}, ${g}, ${b}, 0.8)`
  document.documentElement.style.setProperty('--color-primary-light', lightColor)
}

// 处理主题色变更
const handleColorChange = (color: string) => {
  settings.primaryColor = color
  applyPrimaryColor(color)
  localStorage.setItem('ai-gateway-primary-color', color)
}

// 页面加载时从 localStorage 加载设置
onMounted(() => {
  // 同步主题设置
  settings.theme = currentTheme.value.mode
  settings.themeVariant = currentTheme.value.variant

  // 加载保存的主题色
  const savedPrimaryColor = localStorage.getItem('ai-gateway-primary-color')
  if (savedPrimaryColor) {
    settings.primaryColor = savedPrimaryColor
    applyPrimaryColor(savedPrimaryColor)
  }

  // 加载其他设置
  const savedSettings = localStorage.getItem('ai-gateway-settings')
  if (savedSettings) {
    try {
      const parsed = JSON.parse(savedSettings)
      // 只加载非主题设置（主题由 useTheme 管理）
      if (parsed.borderRadius) settings.borderRadius = parsed.borderRadius
      if (parsed.enableAnimation !== undefined) settings.enableAnimation = parsed.enableAnimation
      if (parsed.gateway) Object.assign(settings.gateway, parsed.gateway)
      if (parsed.cache) Object.assign(settings.cache, parsed.cache)
      if (parsed.logging) Object.assign(settings.logging, parsed.logging)
      if (parsed.security) Object.assign(settings.security, parsed.security)
    } catch (e) {
      console.error('Failed to load settings:', e)
    }
  }
})

const resetSettings = () => {
  Object.assign(settings, createSettingsDefaults())
  localStorage.removeItem('ai-gateway-settings')
  ElMessage.success('设置已重置为默认值')
}

const saveSettings = () => {
  try {
    localStorage.setItem('ai-gateway-settings', JSON.stringify(settings))
    ElMessage.success('设置保存成功')
  } catch (error) {
    console.error('Failed to save settings:', error)
    ElMessage.error('设置保存失败')
  }
}
</script>

<style scoped lang="scss">
.settings-page {
  .settings-nav {
    border-radius: var(--border-radius-lg);

    .nav-list {
      .nav-item {
        display: flex;
        align-items: center;
        gap: var(--spacing-md);
        padding: var(--spacing-md) var(--spacing-lg);
        margin-bottom: 4px;
        border-radius: var(--border-radius-md);
        cursor: pointer;
        color: var(--text-secondary);
        transition: all var(--transition-fast);

        &:hover {
          background: var(--bg-tertiary);
          color: var(--text-primary);
        }

        &.active {
          background: rgba(0, 122, 255, 0.1);
          color: var(--color-primary);
          font-weight: var(--font-weight-medium);
        }
      }
    }
  }

  .settings-card {
    border-radius: var(--border-radius-lg);
    margin-bottom: var(--spacing-xl);

    .card-header {
      display: flex;
      align-items: center;
      gap: var(--spacing-sm);
      font-weight: var(--font-weight-semibold);
      font-size: var(--font-size-lg);
    }
  }

  .settings-form {
    max-width: 600px;

    .form-hint {
      margin-left: var(--spacing-sm);
      color: var(--text-tertiary);
    }

    .color-picker-group {
      display: flex;
      gap: var(--spacing-sm);

      .color-item {
        width: 32px;
        height: 32px;
        border-radius: var(--border-radius-md);
        cursor: pointer;
        transition: all var(--transition-fast);
        border: 2px solid transparent;

        &:hover {
          transform: scale(1.1);
        }

        &.active {
          border-color: var(--text-primary);
          box-shadow: 0 0 0 2px rgba(0, 0, 0, 0.1);
        }
      }
    }
  }

  .about-content {
    .app-info {
      text-align: center;
      padding: var(--spacing-xl) 0;

      .app-logo {
        width: 80px;
        height: 80px;
        margin: 0 auto var(--spacing-lg);
        background: linear-gradient(135deg, var(--color-primary), var(--color-primary-light));
        border-radius: var(--border-radius-xl);
        display: flex;
        align-items: center;
        justify-content: center;
        color: white;
      }

      h2 {
        margin: 0;
        font-size: var(--font-size-3xl);
        font-weight: var(--font-weight-bold);
      }

      .version {
        color: var(--text-tertiary);
        margin-top: var(--spacing-sm);
      }
    }

    .system-info {
      margin: var(--spacing-xl) 0;
    }

    .links {
      display: flex;
      justify-content: center;
      gap: var(--spacing-xl);
      margin-top: var(--spacing-xl);
    }
  }

  .settings-actions {
    display: flex;
    justify-content: flex-end;
    gap: var(--spacing-md);
    padding-top: var(--spacing-xl);
  }
}
</style>
