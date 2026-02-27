<template>
  <div class="docs-wizard-page">
    <div class="ambient-grid"></div>
    <div class="ambient-orb orb-left"></div>
    <div class="ambient-orb orb-right"></div>

    <header class="docs-topbar">
      <router-link to="/" class="brand-link">
        <span class="brand-mark">AG</span>
        <span class="brand-text">AI Gateway Docs</span>
      </router-link>

      <nav class="top-nav">
        <a href="#wizard">接入向导</a>
        <a href="#reference">接口示例</a>
        <a href="#release">上线检查</a>
      </nav>

      <div class="top-actions">
        <router-link to="/login" class="btn btn-outline">控制台登录</router-link>
      </div>
    </header>

    <main class="docs-main">
      <section class="docs-hero">
        <p class="hero-badge">Start Wizard</p>
        <h1>文档中心 · 独立向导页</h1>
        <p class="hero-subtitle">
          参考 Clawd 向导页的信息组织方式，重构为「步骤导航 + 分步执行 + 可复制示例」，
          同时保持 AI Gateway 当前浅色科技风视觉语言。
        </p>

        <div class="hero-chips">
          <span>OpenAI Compatible</span>
          <span>Anthropic Compatible</span>
          <span>Provider Routing</span>
          <span>Usage & Alerts</span>
        </div>
      </section>

      <section id="wizard" class="wizard-layout">
        <aside class="step-nav">
          <div class="step-nav-head">
            <h2>接入步骤</h2>
            <p>{{ Math.round(progress) }}% 完成路径</p>
          </div>

          <button
            v-for="(step, index) in steps"
            :key="step.id"
            type="button"
            class="step-nav-item"
            :class="{ active: currentStep === step.id }"
            @click="scrollToStep(step.id)"
          >
            <span class="step-no">{{ index + 1 }}</span>
            <span class="step-meta">
              <strong>{{ step.title }}</strong>
              <small>{{ step.summary }}</small>
            </span>
          </button>

          <div class="progress-track" aria-hidden="true">
            <span class="progress-fill" :style="{ width: `${progress}%` }"></span>
          </div>
        </aside>

        <div class="step-content">
          <article id="step-setup" class="step-panel">
            <header class="panel-head">
              <span class="panel-index">步骤 1</span>
              <h3>部署网关服务</h3>
              <p>先确认基础服务端口和运行环境，保证 API 网关可用。</p>
            </header>

            <div class="panel-grid two-cols">
              <div class="info-card">
                <h4>推荐启动方式</h4>
                <p>开发环境优先使用统一脚本，避免前端缓存与后端进程不一致。</p>
                <ul>
                  <li>前后端统一端口：<code>8566</code></li>
                  <li>Metrics：<code>9090</code></li>
                  <li>Redis：<code>6379</code></li>
                </ul>
              </div>

              <div class="code-card">
                <div class="code-head">
                  <span>bash</span>
                  <button type="button" @click="copyCode(bootstrapCommand, '部署命令')">复制</button>
                </div>
                <pre><code>{{ bootstrapCommand }}</code></pre>
              </div>
            </div>
          </article>

          <article id="step-provider" class="step-panel">
            <header class="panel-head">
              <span class="panel-index">步骤 2</span>
              <h3>配置服务商与模型池</h3>
              <p>配置 API Key、端点与模型集，路由层才能进行稳定调度。</p>
            </header>

            <div class="panel-grid">
              <div class="info-card">
                <h4>服务商建议</h4>
                <p>至少准备 2 家上游，确保失败切换与成本弹性。</p>
              </div>
            </div>

            <div class="provider-table-wrap">
              <table class="provider-table">
                <thead>
                  <tr>
                    <th>服务商</th>
                    <th>端点</th>
                    <th>示例模型</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="provider in providers" :key="provider.name">
                    <td>{{ provider.name }}</td>
                    <td><code>{{ provider.endpoint }}</code></td>
                    <td>{{ provider.models.slice(0, 2).join(' / ') }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </article>

          <article id="step-call" class="step-panel">
            <header class="panel-head">
              <span class="panel-index">步骤 3</span>
              <h3>切换业务调用到统一入口</h3>
              <p>客户端只需调用一个标准接口，后端完成模型路由与账号切换。</p>
            </header>

            <div id="reference" class="panel-grid two-cols">
              <div class="code-card">
                <div class="code-head">
                  <span>OpenAI Curl</span>
                  <button type="button" @click="copyCode(openaiCurl, 'OpenAI 示例')">复制</button>
                </div>
                <pre><code>{{ openaiCurl }}</code></pre>
              </div>

              <div class="code-card">
                <div class="code-head">
                  <span>Anthropic Curl</span>
                  <button type="button" @click="copyCode(anthropicCurl, 'Anthropic 示例')">复制</button>
                </div>
                <pre><code>{{ anthropicCurl }}</code></pre>
              </div>
            </div>
          </article>

          <article id="step-validate" class="step-panel">
            <header class="panel-head">
              <span class="panel-index">步骤 4</span>
              <h3>验证与回归检查</h3>
              <p>接入完成后先验证接口可用，再观察缓存命中与成功率。</p>
            </header>

            <div class="panel-grid two-cols">
              <div class="code-card">
                <div class="code-head">
                  <span>健康检查</span>
                  <button type="button" @click="copyCode(checkCommands, '检查命令')">复制</button>
                </div>
                <pre><code>{{ checkCommands }}</code></pre>
              </div>

              <div class="checklist-card">
                <h4>上线前检查清单</h4>
                <ul>
                  <li>JWT 秘钥已替换生产值</li>
                  <li>至少 2 个可用服务商账号</li>
                  <li>告警渠道已启用并测试通过</li>
                  <li>日志中无明文 API Key</li>
                </ul>
              </div>
            </div>
          </article>

          <article id="step-release" class="step-panel">
            <header class="panel-head">
              <span id="release" class="panel-index">步骤 5</span>
              <h3>进入控制台持续优化</h3>
              <p>上线后在控制台根据流量与成本实时调优路由和缓存策略。</p>
            </header>

            <div class="panel-grid three-cols">
              <router-link to="/dashboard" class="jump-card">
                <h4>监控仪表盘</h4>
                <p>看请求量、成功率、TTFT 趋势。</p>
              </router-link>

              <router-link to="/api-management" class="jump-card">
                <h4>API 管理</h4>
                <p>调整网关参数与访问策略。</p>
              </router-link>

              <router-link to="/routing" class="jump-card">
                <h4>路由策略</h4>
                <p>按任务类型设置模型偏好。</p>
              </router-link>
            </div>
          </article>
        </div>
      </section>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { DOCS_PROVIDERS } from '@/constants/pages/docs'

interface WizardStep {
  id: string
  title: string
  summary: string
}

const steps: WizardStep[] = [
  { id: 'step-setup', title: '部署服务', summary: '启动网关进程' },
  { id: 'step-provider', title: '配置服务商', summary: '录入密钥与模型' },
  { id: 'step-call', title: '切换调用', summary: '统一 API 入口' },
  { id: 'step-validate', title: '运行验证', summary: '健康检查与回归' },
  { id: 'step-release', title: '持续优化', summary: '运营闭环调优' }
]

const providers = DOCS_PROVIDERS.slice(0, 6)
const currentStep = ref<string>(steps[0]?.id ?? 'step-setup')

const bootstrapCommand = `# 1) 启动依赖与网关
make build
./scripts/dev-restart.sh

# 2) 健康检查
curl http://localhost:8566/health`

const openaiCurl = `curl -X POST http://localhost:8566/api/v1/chat/completions \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -d '{
    "model": "auto",
    "messages": [{"role": "user", "content": "介绍一下 AI Gateway"}],
    "stream": false
  }'`

const anthropicCurl = `curl -X POST http://localhost:8566/api/anthropic/v1/messages \\
  -H "Content-Type: application/json" \\
  -H "x-api-key: YOUR_API_KEY" \\
  -H "anthropic-version: 2023-06-01" \\
  -d '{
    "model": "claude-3-5-sonnet-20241022",
    "max_tokens": 512,
    "messages": [{"role": "user", "content": "帮我生成发布说明"}]
  }'`

const checkCommands = `# 健康探针
curl http://localhost:8566/health

# 实时看板
curl http://localhost:8566/api/admin/dashboard/realtime

# 缓存统计
curl http://localhost:8566/api/admin/cache/stats`

const copyCode = async (value: string, label: string) => {
  try {
    await navigator.clipboard.writeText(value)
    ElMessage.success(`${label}已复制`) 
  } catch (error) {
    ElMessage.error('复制失败，请手动复制')
  }
}

const scrollToStep = (id: string) => {
  const element = document.getElementById(id)
  if (!element) return
  element.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

const updateCurrentStep = () => {
  const checkpoint = window.scrollY + 180
  let activeId = steps[0]?.id ?? ''

  for (const step of steps) {
    const element = document.getElementById(step.id)
    if (!element) continue
    if (element.offsetTop <= checkpoint) {
      activeId = step.id
    }
  }

  currentStep.value = activeId
}

const progress = computed(() => {
  const index = steps.findIndex((step) => step.id === currentStep.value)
  if (index < 0) return 0
  return ((index + 1) / steps.length) * 100
})

onMounted(() => {
  updateCurrentStep()
  window.addEventListener('scroll', updateCurrentStep, { passive: true })
})

onUnmounted(() => {
  window.removeEventListener('scroll', updateCurrentStep)
})
</script>

<style scoped>
.docs-wizard-page {
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
.docs-main {
  position: relative;
  z-index: 2;
  max-width: 1220px;
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

.top-nav a:hover {
  color: var(--text-primary);
}

.docs-main {
  margin-top: 1rem;
}

.docs-hero,
.wizard-layout {
  border: 1px solid var(--line);
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.78);
  backdrop-filter: blur(8px);
}

.docs-hero {
  padding: 1.35rem;
}

.hero-badge {
  margin: 0;
  display: inline-flex;
  align-items: center;
  padding: 0.24rem 0.65rem;
  border-radius: 999px;
  font-size: 0.82rem;
  color: #0f766e;
  background: rgba(14, 165, 160, 0.12);
}

.docs-hero h1 {
  margin: 0.7rem 0 0;
  font-size: clamp(1.7rem, 3.6vw, 2.7rem);
}

.hero-subtitle {
  margin: 0.75rem 0 0;
  max-width: 68ch;
  color: var(--text-secondary);
  line-height: 1.65;
}

.hero-chips {
  margin-top: 0.9rem;
  display: flex;
  flex-wrap: wrap;
  gap: 0.45rem;
}

.hero-chips span {
  padding: 0.3rem 0.6rem;
  border-radius: 999px;
  border: 1px solid rgba(14, 165, 160, 0.24);
  background: rgba(255, 255, 255, 0.8);
  color: #3f6079;
  font-size: 0.78rem;
}

.wizard-layout {
  margin-top: 1rem;
  padding: 1rem;
  display: grid;
  grid-template-columns: 300px 1fr;
  gap: 1rem;
}

.step-nav {
  position: sticky;
  top: 1rem;
  align-self: start;
  border: 1px solid var(--line);
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.84);
  padding: 0.9rem;
}

.step-nav-head h2 {
  margin: 0;
  font-size: 1.05rem;
}

.step-nav-head p {
  margin: 0.3rem 0 0.6rem;
  color: var(--text-secondary);
  font-size: 0.86rem;
}

.step-nav-item {
  width: 100%;
  border: 1px solid var(--line);
  background: rgba(255, 255, 255, 0.82);
  border-radius: 12px;
  display: flex;
  align-items: flex-start;
  gap: 0.65rem;
  text-align: left;
  padding: 0.6rem;
  cursor: pointer;
  transition: border-color 0.2s ease, transform 0.2s ease, box-shadow 0.2s ease;
}

.step-nav-item + .step-nav-item {
  margin-top: 0.55rem;
}

.step-nav-item:hover {
  transform: translateY(-1px);
}

.step-nav-item.active {
  border-color: rgba(14, 165, 160, 0.55);
  box-shadow: 0 8px 22px rgba(19, 136, 201, 0.16);
}

.step-no {
  display: grid;
  place-items: center;
  flex-shrink: 0;
  width: 1.55rem;
  height: 1.55rem;
  border-radius: 999px;
  background: linear-gradient(135deg, var(--accent), #46b7e5);
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
  font-size: 0.92rem;
}

.step-meta small {
  color: var(--text-secondary);
}

.progress-track {
  margin-top: 0.8rem;
  width: 100%;
  height: 0.46rem;
  border-radius: 999px;
  background: rgba(98, 136, 164, 0.2);
  overflow: hidden;
}

.progress-fill {
  display: block;
  height: 100%;
  background: linear-gradient(90deg, var(--accent), var(--accent-strong));
  border-radius: inherit;
  transition: width 0.25s ease;
}

.step-content {
  min-width: 0;
}

.step-panel {
  border: 1px solid var(--line);
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.86);
  padding: 1rem;
}

.step-panel + .step-panel {
  margin-top: 0.9rem;
}

.panel-head h3 {
  margin: 0.25rem 0 0;
  font-size: 1.25rem;
}

.panel-head p {
  margin: 0.45rem 0 0;
  color: var(--text-secondary);
}

.panel-index {
  display: inline-flex;
  align-items: center;
  padding: 0.22rem 0.58rem;
  border-radius: 999px;
  font-size: 0.78rem;
  color: #0f766e;
  background: rgba(14, 165, 160, 0.12);
}

.panel-grid {
  margin-top: 0.85rem;
  display: grid;
  gap: 0.8rem;
}

.two-cols {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.three-cols {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.info-card,
.checklist-card,
.code-card,
.jump-card {
  border: 1px solid var(--line);
  border-radius: 12px;
  background: linear-gradient(165deg, rgba(251, 254, 255, 0.94), rgba(236, 246, 255, 0.88));
  padding: 0.85rem;
}

.info-card h4,
.checklist-card h4,
.jump-card h4 {
  margin: 0;
}

.info-card p,
.jump-card p {
  margin: 0.45rem 0 0;
  color: var(--text-secondary);
  line-height: 1.6;
}

.info-card ul,
.checklist-card ul {
  margin: 0.5rem 0 0;
  padding-left: 1rem;
  color: var(--text-secondary);
  line-height: 1.7;
}

.code-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.6rem;
  font-size: 0.82rem;
  color: var(--text-secondary);
}

.code-head button {
  border: 1px solid var(--line);
  background: rgba(255, 255, 255, 0.92);
  color: var(--text-secondary);
  border-radius: 9px;
  padding: 0.28rem 0.58rem;
  cursor: pointer;
}

.code-card pre {
  margin: 0.65rem 0 0;
  overflow-x: auto;
  white-space: pre;
  font-family: 'IBM Plex Mono', ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.82rem;
  line-height: 1.55;
  color: #1a3850;
}

.provider-table-wrap {
  margin-top: 0.8rem;
  overflow-x: auto;
}

.provider-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.9rem;
}

.provider-table th,
.provider-table td {
  padding: 0.62rem 0.68rem;
  border-bottom: 1px solid rgba(98, 136, 164, 0.2);
  text-align: left;
}

.provider-table th {
  color: var(--text-secondary);
  font-weight: 600;
}

.provider-table code,
.info-card code {
  background: rgba(255, 255, 255, 0.88);
  border: 1px solid rgba(98, 136, 164, 0.24);
  padding: 0.14rem 0.34rem;
  border-radius: 6px;
}

.jump-card {
  text-decoration: none;
  color: inherit;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}

.jump-card:hover {
  transform: translateY(-1px);
  box-shadow: 0 10px 24px rgba(19, 136, 201, 0.14);
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

@media (max-width: 1120px) {
  .wizard-layout {
    grid-template-columns: 1fr;
  }

  .step-nav {
    position: static;
  }

  .three-cols {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 820px) {
  .docs-wizard-page {
    padding: 0.8rem 0.8rem 1.5rem;
  }

  .top-nav {
    display: none;
  }

  .two-cols,
  .three-cols {
    grid-template-columns: 1fr;
  }

  .brand-text {
    font-size: 0.95rem;
  }
}
</style>
