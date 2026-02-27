<template>
  <section class="docs-route-page">
    <header class="route-head">
      <p class="route-badge">Install Wizard</p>
      <h1>安装向导</h1>
      <p>按顺序完成环境准备、服务配置、请求验证与上线前检查。</p>
    </header>

    <div class="wizard-layout">
      <aside class="step-nav">
        <button
          v-for="(step, index) in steps"
          :key="step.id"
          type="button"
          class="step-nav-item"
          :class="{ active: currentStep === step.id }"
          @click="scrollTo(step.id)"
        >
          <span class="step-no">{{ index + 1 }}</span>
          <span class="step-meta">
            <strong>{{ step.title }}</strong>
            <small>{{ step.summary }}</small>
          </span>
        </button>
      </aside>

      <div class="step-content">
        <article v-for="step in steps" :id="step.id" :key="`${step.id}-panel`" class="step-panel">
          <h2>{{ step.title }}</h2>
          <p>{{ step.description }}</p>

          <div class="code-card" v-if="step.command">
            <div class="code-head">
              <span>命令示例</span>
              <button type="button" @click="copyCode(step.command)">复制</button>
            </div>
            <pre><code>{{ step.command }}</code></pre>
          </div>
        </article>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { ElMessage } from 'element-plus'

interface WizardStep {
  id: string
  title: string
  summary: string
  description: string
  command?: string
}

const steps: WizardStep[] = [
  {
    id: 'wizard-step-prepare',
    title: '准备运行环境',
    summary: '依赖与端口检查',
    description: '确保 Node、Go、Redis 与端口配置满足网关运行要求。',
    command: `node -v\ngo version\nredis-cli ping\nlsof -i :8566`
  },
  {
    id: 'wizard-step-launch',
    title: '启动网关服务',
    summary: '统一启动脚本',
    description: '优先使用项目统一脚本，避免前后端构建与进程状态不一致。',
    command: `make build\n./scripts/dev-restart.sh\ncurl http://localhost:8566/health`
  },
  {
    id: 'wizard-step-provider',
    title: '配置服务商账号',
    summary: '至少双上游',
    description: '在控制台录入至少两家服务商账号，启用故障切换和限额保护。'
  },
  {
    id: 'wizard-step-request',
    title: '发送首个请求',
    summary: 'OpenAI 兼容入口',
    description: '统一调用 /api/v1/chat/completions，验证路由、缓存和返回格式。',
    command: `curl -X POST http://localhost:8566/api/v1/chat/completions \\\n  -H "Content-Type: application/json" \\\n  -H "Authorization: Bearer YOUR_API_KEY" \\\n  -d '{"model":"auto","messages":[{"role":"user","content":"hello"}]}'`
  },
  {
    id: 'wizard-step-release',
    title: '上线前检查',
    summary: '安全与可观测',
    description: '确认 JWT、告警、日志脱敏、监控指标与回滚方案均准备完成。',
    command: `curl http://localhost:8566/api/admin/dashboard/realtime\ncurl http://localhost:8566/api/admin/cache/stats`
  }
]

const currentStep = ref(steps[0]?.id ?? 'wizard-step-prepare')

const scrollTo = (id: string) => {
  const el = document.getElementById(id)
  if (!el) return
  el.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

const copyCode = async (value: string) => {
  try {
    await navigator.clipboard.writeText(value)
    ElMessage.success('命令已复制')
  } catch {
    ElMessage.error('复制失败')
  }
}

const updateActiveStep = () => {
  const checkpoint = window.scrollY + 160
  let active = steps[0]?.id ?? ''

  for (const step of steps) {
    const el = document.getElementById(step.id)
    if (!el) continue
    if (el.offsetTop <= checkpoint) {
      active = step.id
    }
  }

  currentStep.value = active
}

onMounted(() => {
  updateActiveStep()
  window.addEventListener('scroll', updateActiveStep, { passive: true })
})

onUnmounted(() => {
  window.removeEventListener('scroll', updateActiveStep)
})
</script>

<style scoped>
.docs-route-page {
  border: 1px solid rgba(98, 136, 164, 0.24);
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.86);
  padding: 0.95rem;
}

.route-head h1 {
  margin: 0.35rem 0 0;
  font-size: 1.55rem;
}

.route-head p {
  margin: 0.45rem 0 0;
  color: #4f667b;
}

.route-badge {
  margin: 0;
  display: inline-flex;
  align-items: center;
  padding: 0.22rem 0.58rem;
  border-radius: 999px;
  font-size: 0.78rem;
  color: #0f766e;
  background: rgba(14, 165, 160, 0.12);
}

.wizard-layout {
  margin-top: 0.8rem;
  display: grid;
  grid-template-columns: 280px 1fr;
  gap: 0.8rem;
}

.step-nav {
  position: sticky;
  top: 1rem;
  align-self: start;
}

.step-nav-item {
  width: 100%;
  border: 1px solid rgba(98, 136, 164, 0.24);
  border-radius: 11px;
  background: rgba(255, 255, 255, 0.88);
  display: flex;
  align-items: flex-start;
  gap: 0.6rem;
  text-align: left;
  padding: 0.55rem;
  cursor: pointer;
}

.step-nav-item + .step-nav-item {
  margin-top: 0.5rem;
}

.step-nav-item.active {
  border-color: rgba(14, 165, 160, 0.62);
  box-shadow: 0 8px 18px rgba(19, 136, 201, 0.14);
}

.step-no {
  display: grid;
  place-items: center;
  width: 1.5rem;
  height: 1.5rem;
  border-radius: 999px;
  background: linear-gradient(135deg, #0ea5a0, #46b7e5);
  color: #f7feff;
  font-size: 0.78rem;
  font-weight: 700;
}

.step-meta {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}

.step-meta strong {
  font-size: 0.9rem;
}

.step-meta small {
  color: #4f667b;
}

.step-panel {
  border: 1px solid rgba(98, 136, 164, 0.24);
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.9);
  padding: 0.8rem;
}

.step-panel + .step-panel {
  margin-top: 0.75rem;
}

.step-panel h2 {
  margin: 0;
  font-size: 1.1rem;
}

.step-panel p {
  margin: 0.45rem 0 0;
  color: #4f667b;
  line-height: 1.6;
}

.code-card {
  margin-top: 0.7rem;
  border: 1px solid rgba(98, 136, 164, 0.24);
  border-radius: 11px;
  background: rgba(255, 255, 255, 0.92);
  padding: 0.6rem;
}

.code-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 0.82rem;
  color: #4f667b;
}

.code-head button {
  border: 1px solid rgba(98, 136, 164, 0.26);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.96);
  color: #4f667b;
  padding: 0.2rem 0.55rem;
  cursor: pointer;
}

.code-card pre {
  margin: 0.55rem 0 0;
  white-space: pre-wrap;
  line-height: 1.55;
  font-size: 0.82rem;
  color: #1a3850;
}

@media (max-width: 1080px) {
  .wizard-layout {
    grid-template-columns: 1fr;
  }

  .step-nav {
    position: static;
  }
}
</style>
