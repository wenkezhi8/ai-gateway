<template>
  <div class="accounts-limit-page">
    <div class="page-hero">
      <div class="hero-content">
        <div>
          <div class="hero-title">账号与限额</div>
          <div class="hero-subtitle">统一管理服务商账号、限额策略与使用状态</div>
        </div>
        <div class="hero-actions">
          <el-button :type="activeSection === 'accounts' ? 'primary' : 'default'" @click="scrollToSection('accounts')">
            账号管理
          </el-button>
          <el-button :type="activeSection === 'limits' ? 'primary' : 'default'" @click="scrollToSection('limits')">
            限额管控
          </el-button>
        </div>
      </div>
      <div class="hero-tips">
        <div class="tip-item">
          <span class="tip-title">操作建议</span>
          <span class="tip-text">先配置账号与密钥，再设置限额策略</span>
        </div>
        <div class="tip-item">
          <span class="tip-title">风险提示</span>
          <span class="tip-text">限额异常可在右侧快速刷新确认</span>
        </div>
      </div>
    </div>

    <div class="section-grid">
      <section ref="accountsRef" class="section-card">
        <div class="section-header">
          <div>
            <div class="section-title">账号管理</div>
            <div class="section-subtitle">管理服务商账号与 API Key</div>
          </div>
        </div>
        <div class="section-body">
          <ProvidersAccounts :embedded="true" />
        </div>
      </section>

      <section ref="limitsRef" class="section-card">
        <div class="section-header">
          <div>
            <div class="section-title">限额管控</div>
            <div class="section-subtitle">查看账号使用与限额预警</div>
          </div>
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
import ProvidersAccounts from '@/views/providers-accounts/index.vue'
import LimitManagement from '@/views/limit-management/index.vue'

const accountsRef = ref<HTMLElement | null>(null)
const limitsRef = ref<HTMLElement | null>(null)
const activeSection = ref<'accounts' | 'limits'>('accounts')

const scrollToSection = (section: 'accounts' | 'limits') => {
  activeSection.value = section
  const target = section === 'accounts' ? accountsRef.value : limitsRef.value
  target?.scrollIntoView({ behavior: 'smooth', block: 'start' })
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
}

.hero-tips {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
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
