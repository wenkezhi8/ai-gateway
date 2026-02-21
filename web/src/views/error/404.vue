<template>
  <div class="error-page">
    <div class="error-content">
      <div class="error-icon">
        <el-icon :size="80"><WarningFilled /></el-icon>
      </div>
      <h1 class="error-code">404</h1>
      <h2 class="error-title">页面未找到</h2>
      <p class="error-desc">抱歉，您访问的页面不存在或已被移除</p>
      <div class="error-actions">
        <el-button type="primary" size="large" @click="goHome">
          <el-icon><HomeFilled /></el-icon>
          返回首页
        </el-button>
        <el-button size="large" @click="goBack">
          <el-icon><Back /></el-icon>
          返回上页
        </el-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()

const goHome = () => {
  router.push('/')
}

const goBack = () => {
  router.back()
}
</script>

<style scoped lang="scss">
.error-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-secondary);

  .error-content {
    text-align: center;
    padding: var(--spacing-2xl);

    .error-icon {
      margin-bottom: var(--spacing-xl);
      color: var(--color-primary);
      animation: float 3s ease-in-out infinite;
    }

    .error-code {
      font-size: 120px;
      font-weight: var(--font-weight-bold);
      color: var(--color-primary);
      margin: 0;
      line-height: 1;
      background: linear-gradient(135deg, var(--color-primary), #5856D6);
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
      background-clip: text;
    }

    .error-title {
      font-size: var(--font-size-2xl);
      color: var(--text-primary);
      margin: var(--spacing-lg) 0 var(--spacing-md);
      font-weight: var(--font-weight-semibold);
    }

    .error-desc {
      color: var(--text-secondary);
      margin-bottom: var(--spacing-2xl);
      font-size: var(--font-size-lg);
      max-width: 400px;
      margin-left: auto;
      margin-right: auto;
    }

    .error-actions {
      display: flex;
      gap: var(--spacing-md);
      justify-content: center;
    }
  }
}

@keyframes float {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-10px); }
}

// 暗色模式适配
[data-theme="dark"] {
  .error-page {
    .error-code {
      background: linear-gradient(135deg, #0A84FF, #5E5CE6);
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
      background-clip: text;
    }
  }
}
</style>
