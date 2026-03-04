<template>
  <div class="accounts-limit-page">
    <div class="page-hero">
      <div class="hero-content">
        <div>
          <div class="hero-title">AI服务商</div>
          <div class="hero-subtitle">按 3 步完成接入：添加账号 → 设置限额 → 回到仪表盘验证</div>
        </div>
        <div class="hero-actions">
          <el-button :type="activeSection === 'accounts' ? 'primary' : 'default'" @click="scrollToSection('accounts')">
            第1步：添加账号
          </el-button>
          <el-button :type="activeSection === 'limits' ? 'primary' : 'default'" @click="scrollToSection('limits')">
            第2步：设置限额
          </el-button>
          <el-button type="success" plain @click="goToDashboard">
            第3步：前往仪表盘验证
          </el-button>
        </div>
      </div>
      <div class="hero-tips">
        <div class="tip-item">
          <span class="tip-title">第1步：添加账号</span>
          <span class="tip-text">在下方账号管理中添加AI服务商、API Key 与服务端点</span>
        </div>
        <div class="tip-item">
          <span class="tip-title">第2步：设置限额</span>
          <span class="tip-text">切换到限额管控，为账号配置额度阈值与预警规则</span>
        </div>
        <div class="tip-item">
          <span class="tip-title">第3步：验证效果</span>
          <span class="tip-text">完成配置后前往仪表盘，确认请求与告警指标开始更新</span>
        </div>
      </div>
    </div>

    <div class="section-grid">
      <section ref="accountsRef" class="section-card">
        <div class="section-header">
          <div>
            <div class="section-title">第1步：账号管理</div>
            <div class="section-subtitle">添加AI服务商账号、API Key 与基础连接信息</div>
          </div>
          <el-button text type="primary" @click="scrollToSection('limits')">完成后进入第2步</el-button>
        </div>
        <div class="section-body">
          <ProvidersAccounts :embedded="true" />
        </div>
      </section>

      <section ref="limitsRef" class="section-card">
        <div class="section-header">
          <div>
            <div class="section-title">第2步：限额管控</div>
            <div class="section-subtitle">为已接入账号配置额度策略，并查看预警状态</div>
          </div>
          <el-button text type="success" @click="goToDashboard">完成后前往第3步验证</el-button>
        </div>
        <div class="section-body">
          <LimitManagement :embedded="true" />
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import ProvidersAccounts from '@/views/providers-accounts/index.vue'
import LimitManagement from '@/views/limit-management/index.vue'

const router = useRouter()
const accountsRef = ref<HTMLElement | null>(null)
const limitsRef = ref<HTMLElement | null>(null)
const activeSection = ref<'accounts' | 'limits'>('accounts')

const scrollToSection = (section: 'accounts' | 'limits') => {
  activeSection.value = section
  const target = section === 'accounts' ? accountsRef.value : limitsRef.value
  target?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

const goToDashboard = () => {
  router.push('/dashboard')
}
</script>

<style scoped lang="scss">
.accounts-limit-page {
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.page-hero {
  background: linear-gradient(135deg, #0f172a 0%, #1e293b 55%, #334155 100%);
  color: #fff;
  border-radius: 18px;
  padding: 20px 24px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.hero-content {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
}

.hero-title {
  font-size: 22px;
  font-weight: 600;
  letter-spacing: 0.5px;
}

.hero-subtitle {
  margin-top: 6px;
  font-size: 13px;
  color: rgba(255, 255, 255, 0.72);
}

.hero-actions {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.hero-tips {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.tip-item {
  background: rgba(15, 23, 42, 0.45);
  border: 1px solid rgba(148, 163, 184, 0.2);
  border-radius: 12px;
  padding: 12px 14px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.tip-title {
  font-size: 12px;
  color: rgba(248, 250, 252, 0.75);
}

.tip-text {
  font-size: 13px;
  color: #fff;
}

.section-grid {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.section-card {
  background: #fff;
  border-radius: 16px;
  border: 1px solid rgba(15, 23, 42, 0.06);
  padding: 14px 16px 18px;
  box-shadow: 0 12px 30px rgba(15, 23, 42, 0.06);
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
}

.section-subtitle {
  margin-top: 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.section-body {
  padding-top: 6px;
}

@media (max-width: 900px) {
  .hero-content {
    flex-direction: column;
    align-items: flex-start;
  }

  .hero-tips {
    grid-template-columns: 1fr;
  }
}
</style>
