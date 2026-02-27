<template>
  <main class="home-v3">
    <header class="topbar">
      <div class="brand">
        <span class="brand-mark">AG</span>
        <div class="brand-text">
          <strong>AI Gateway</strong>
          <small>Superpowers + TDD</small>
        </div>
      </div>
      <div class="top-links">
        <a href="/docs">文档</a>
        <a href="/api-management">API 管理</a>
        <a href="https://github.com/wenkezhi8/ai-gateway" target="_blank" rel="noreferrer">GitHub</a>
      </div>
    </header>

    <section class="hero">
      <div class="hero-copy">
        <p class="badge">v1.6.5 · 企业级 AI Gateway</p>
        <h1>
          把 AI 接入做成
          <span>可持续交付的工程系统</span>
        </h1>
        <p class="subtitle">
          不是只跑通接口，而是用 Superpowers 工作流和 TDD 模式，
          把排查、修复、验证、审计变成可复用的团队能力。
        </p>

        <div class="hero-actions">
          <el-button
            v-for="action in HERO_ACTIONS"
            :key="action.id"
            :type="action.kind === 'primary' ? 'primary' : undefined"
            size="large"
            @click="handleAction(action)"
          >
            {{ action.label }}
          </el-button>
        </div>

        <div class="hero-stats">
          <div class="stat-card">
            <strong>6 步</strong>
            <span>工程工作流</span>
          </div>
          <div class="stat-card">
            <strong>4 阶段</strong>
            <span>TDD 闭环</span>
          </div>
          <div class="stat-card">
            <strong>2 套协议</strong>
            <span>OpenAI / Anthropic</span>
          </div>
        </div>
      </div>

      <div class="hero-flow">
        <h3>请求流可视化</h3>
        <ol>
          <li v-for="(node, index) in FLOW_NODES" :key="node">
            <span class="node-index">{{ index + 1 }}</span>
            <span class="node-title">{{ node }}</span>
          </li>
        </ol>
        <p class="flow-tip">Fail-open 策略 + 影子观测 + 回归验证</p>
      </div>
    </section>

    <section id="workflow" class="workflow section-card">
      <div class="section-head">
        <h2>Superpowers 标准工作流</h2>
        <p>每次需求都沿同一条路径推进，避免“先改后对齐”的返工。</p>
      </div>

      <div class="workflow-grid">
        <article v-for="(step, index) in WORKFLOW_STEPS" :key="step.title" class="workflow-item">
          <div class="workflow-title">
            <span class="step-no">{{ index + 1 }}</span>
            <h3>{{ step.title }}</h3>
          </div>
          <p>{{ step.detail }}</p>
          <em>输出：{{ step.deliverable }}</em>
        </article>
      </div>
    </section>

    <section class="tdd section-card">
      <div class="section-head">
        <h2>TDD 执行模式</h2>
        <p>先测试后实现，证据优先，拒绝“看起来修好了”。</p>
      </div>

      <div class="tdd-grid">
        <article v-for="stage in TDD_STAGES" :key="stage.name" class="tdd-item">
          <header>
            <strong>{{ stage.name }}</strong>
            <span>{{ stage.detail }}</span>
          </header>
          <code>{{ stage.command }}</code>
        </article>
      </div>

      <div class="verify-banner">
        <p>
          发布前统一验证：
          <code>npm run test:unit</code>
          <code>npm run typecheck</code>
          <code>npm run build</code>
        </p>
      </div>
    </section>

    <section class="capability section-card">
      <div class="section-head">
        <h2>能力矩阵</h2>
        <p>按质量、稳定性、成本、兼容性组织能力，而不是堆功能点。</p>
      </div>

      <div class="cap-grid">
        <article v-for="column in CAPABILITY_COLUMNS" :key="column.title" class="cap-item">
          <h3>{{ column.title }}</h3>
          <ul>
            <li v-for="point in column.points" :key="point">{{ point }}</li>
          </ul>
        </article>
      </div>
    </section>

    <section class="quickstart section-card">
      <div class="section-head">
        <h2>30 秒快速启动</h2>
        <p>部署命令、源码路径、API 调用三条通路，立即可验证。</p>
      </div>

      <el-tabs v-model="quickTab" class="quick-tabs">
        <el-tab-pane label="Docker" name="docker" />
        <el-tab-pane label="源码" name="source" />
        <el-tab-pane label="API" name="api" />
      </el-tabs>

      <div class="code-box">
        <div class="code-head">
          <span>bash</span>
          <el-button text @click="copyCurrentCommand">复制命令</el-button>
        </div>
        <pre><code>{{ currentCommand }}</code></pre>
      </div>
    </section>

    <footer class="footer">
      <p>开源免费 · MIT 许可 · 版本真相源为 Git Tag</p>
      <div class="footer-links">
        <a href="/docs">文档中心</a>
        <a href="/api-management">API 管理</a>
        <a href="https://github.com/wenkezhi8/ai-gateway" target="_blank" rel="noreferrer">GitHub</a>
      </div>
    </footer>
  </main>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  CAPABILITY_COLUMNS,
  FLOW_NODES,
  HERO_ACTIONS,
  QUICK_START_COMMANDS,
  TDD_STAGES,
  type HeroAction,
  WORKFLOW_STEPS
} from './content'

const router = useRouter()
const quickTab = ref<keyof typeof QUICK_START_COMMANDS>('docker')

const currentCommand = computed(() => QUICK_START_COMMANDS[quickTab.value])

const copyCurrentCommand = async () => {
  await navigator.clipboard.writeText(currentCommand.value)
  ElMessage.success('命令已复制')
}

const handleAction = async (action: HeroAction) => {
  if (action.route?.startsWith('#')) {
    const target = document.querySelector(action.route)
    target?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    return
  }

  if (action.route) {
    await router.push(action.route)
    return
  }

  if (action.href) {
    window.open(action.href, '_blank', 'noopener')
  }
}
</script>

<style scoped lang="scss">
.home-v3 {
  --ink: #0f172a;
  --ink-soft: #475569;
  --paper: #f7fbff;
  --panel: #ffffff;
  --line: #d6e2f0;
  --accent: #0ea5a4;
  --accent-strong: #0369a1;
  --signal: #f97316;
  --radius: 20px;

  min-height: 100vh;
  padding: 28px;
  color: var(--ink);
  background:
    radial-gradient(circle at 12% 8%, rgba(14, 165, 164, 0.14) 0%, transparent 42%),
    radial-gradient(circle at 88% 16%, rgba(249, 115, 22, 0.12) 0%, transparent 36%),
    linear-gradient(180deg, #f8fbff 0%, #edf4fb 58%, #eef6ff 100%);
  font-family: 'Avenir Next', 'Manrope', 'SF Pro Display', 'PingFang SC', 'Segoe UI', sans-serif;
}

.topbar {
  max-width: 1200px;
  margin: 0 auto 20px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
}

.brand {
  display: flex;
  align-items: center;
  gap: 12px;
}

.brand-mark {
  width: 42px;
  height: 42px;
  border-radius: 12px;
  display: grid;
  place-items: center;
  font-weight: 800;
  letter-spacing: 0.04em;
  color: #fff;
  background: linear-gradient(145deg, var(--accent-strong), var(--accent));
}

.brand-text {
  display: grid;
  line-height: 1.1;

  strong {
    font-size: 16px;
    font-weight: 800;
  }

  small {
    font-size: 12px;
    color: var(--ink-soft);
  }
}

.top-links {
  display: flex;
  gap: 18px;

  a {
    font-size: 14px;
    color: var(--ink-soft);
    text-decoration: none;

    &:hover {
      color: var(--accent-strong);
    }
  }
}

.hero {
  max-width: 1200px;
  margin: 0 auto 22px;
  display: grid;
  grid-template-columns: 1.35fr 1fr;
  gap: 18px;
}

.hero-copy,
.hero-flow,
.section-card,
.footer {
  background: rgba(255, 255, 255, 0.82);
  border: 1px solid var(--line);
  border-radius: var(--radius);
  backdrop-filter: blur(6px);
  box-shadow: 0 10px 28px rgba(15, 23, 42, 0.06);
}

.hero-copy {
  padding: 34px;
  animation: rise 0.45s ease-out both;

  .badge {
    width: fit-content;
    padding: 8px 12px;
    border-radius: 999px;
    margin: 0 0 12px;
    font-size: 12px;
    font-weight: 700;
    color: #075985;
    background: rgba(14, 165, 164, 0.14);
  }

  h1 {
    margin: 0;
    font-size: 44px;
    line-height: 1.08;
    letter-spacing: -0.02em;

    span {
      display: block;
      margin-top: 6px;
      color: var(--accent-strong);
    }
  }

  .subtitle {
    margin: 14px 0 0;
    color: var(--ink-soft);
    line-height: 1.65;
    max-width: 640px;
  }
}

.hero-actions {
  margin-top: 18px;
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.hero-stats {
  margin-top: 18px;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}

.stat-card {
  padding: 12px;
  border-radius: 14px;
  border: 1px solid var(--line);
  background: #ffffff;
  display: grid;

  strong {
    font-size: 18px;
    color: #0c4a6e;
  }

  span {
    margin-top: 2px;
    font-size: 12px;
    color: var(--ink-soft);
  }
}

.hero-flow {
  padding: 26px;
  animation: rise 0.55s ease-out both;

  h3 {
    margin: 0;
    font-size: 20px;
  }

  ol {
    margin: 16px 0 0;
    padding: 0;
    list-style: none;
    display: grid;
    gap: 10px;
  }

  li {
    padding: 10px 12px;
    border-radius: 14px;
    border: 1px solid var(--line);
    background: #fff;
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .node-index {
    width: 26px;
    height: 26px;
    border-radius: 999px;
    display: grid;
    place-items: center;
    font-size: 12px;
    font-weight: 700;
    color: #fff;
    background: var(--signal);
  }

  .node-title {
    font-size: 14px;
    font-weight: 600;
  }

  .flow-tip {
    margin: 14px 0 0;
    font-size: 13px;
    color: var(--ink-soft);
  }
}

.section-card {
  max-width: 1200px;
  margin: 0 auto 22px;
  padding: 28px;
}

.section-head {
  h2 {
    margin: 0;
    font-size: 30px;
    letter-spacing: -0.01em;
  }

  p {
    margin: 8px 0 0;
    color: var(--ink-soft);
  }
}

.workflow-grid {
  margin-top: 18px;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.workflow-item {
  border: 1px solid var(--line);
  border-radius: 16px;
  padding: 14px;
  background: #fff;

  p {
    margin: 10px 0 8px;
    color: var(--ink-soft);
    line-height: 1.6;
    font-size: 14px;
  }

  em {
    font-style: normal;
    color: #0c4a6e;
    font-size: 13px;
    font-weight: 700;
  }
}

.workflow-title {
  display: flex;
  align-items: center;
  gap: 8px;

  h3 {
    margin: 0;
    font-size: 16px;
  }
}

.step-no {
  width: 24px;
  height: 24px;
  border-radius: 8px;
  display: grid;
  place-items: center;
  font-size: 12px;
  font-weight: 800;
  color: #fff;
  background: linear-gradient(140deg, var(--accent), var(--accent-strong));
}

.tdd-grid {
  margin-top: 18px;
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}

.tdd-item {
  border: 1px solid var(--line);
  border-radius: 16px;
  padding: 12px;
  background: #fff;
  display: grid;
  gap: 10px;

  header {
    display: grid;
    gap: 6px;
  }

  strong {
    font-size: 20px;
    color: #0f172a;
    letter-spacing: 0.03em;
  }

  span {
    color: var(--ink-soft);
    font-size: 13px;
    line-height: 1.5;
  }

  code {
    display: block;
    padding: 10px;
    border-radius: 10px;
    font-size: 12px;
    line-height: 1.5;
    border: 1px dashed #bfd6ea;
    background: #f5faff;
    color: #0c4a6e;
    white-space: pre-wrap;
    word-break: break-word;
  }
}

.verify-banner {
  margin-top: 12px;
  border-radius: 14px;
  border: 1px solid #ffd9b0;
  background: #fff8f0;
  padding: 12px;

  p {
    margin: 0;
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
    align-items: center;
  }

  code {
    display: inline-block;
    padding: 4px 8px;
    border-radius: 8px;
    background: #fff;
    border: 1px solid #ffd9b0;
    color: #7c2d12;
    font-size: 12px;
  }
}

.cap-grid {
  margin-top: 18px;
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}

.cap-item {
  border: 1px solid var(--line);
  border-radius: 16px;
  background: #fff;
  padding: 12px;

  h3 {
    margin: 0;
    font-size: 16px;
  }

  ul {
    margin: 10px 0 0;
    padding: 0;
    list-style: none;
    display: grid;
    gap: 8px;

    li {
      font-size: 14px;
      color: var(--ink-soft);
      padding-left: 16px;
      position: relative;

      &::before {
        content: '';
        width: 7px;
        height: 7px;
        border-radius: 999px;
        background: var(--accent);
        position: absolute;
        left: 0;
        top: 7px;
      }
    }
  }
}

.quick-tabs {
  margin-top: 16px;
}

.code-box {
  margin-top: 10px;
  border: 1px solid var(--line);
  border-radius: 16px;
  overflow: hidden;
  background: #fff;

  .code-head {
    padding: 10px 12px;
    display: flex;
    justify-content: space-between;
    border-bottom: 1px solid var(--line);
    background: #f8fcff;
    font-size: 13px;
    color: var(--ink-soft);
  }

  pre {
    margin: 0;
    padding: 14px;
    overflow-x: auto;
  }

  code {
    font-family: 'JetBrains Mono', 'SF Mono', 'Menlo', 'Consolas', monospace;
    font-size: 13px;
    line-height: 1.65;
    color: #0f172a;
  }
}

.footer {
  max-width: 1200px;
  margin: 0 auto;
  padding: 22px;

  p {
    margin: 0;
    color: var(--ink-soft);
    font-size: 14px;
  }
}

.footer-links {
  margin-top: 10px;
  display: flex;
  gap: 14px;
  flex-wrap: wrap;

  a {
    color: #0c4a6e;
    font-size: 14px;
    text-decoration: none;

    &:hover {
      color: var(--accent-strong);
    }
  }
}

@keyframes rise {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@media (max-width: 1024px) {
  .hero {
    grid-template-columns: 1fr;
  }

  .workflow-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .tdd-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .cap-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 768px) {
  .home-v3 {
    padding: 16px;
  }

  .topbar {
    flex-direction: column;
    align-items: flex-start;
  }

  .hero-copy {
    padding: 22px;

    h1 {
      font-size: 34px;
    }
  }

  .hero-stats {
    grid-template-columns: 1fr;
  }

  .workflow-grid,
  .tdd-grid,
  .cap-grid {
    grid-template-columns: 1fr;
  }

  .section-card {
    padding: 20px;
  }

  .section-head h2 {
    font-size: 24px;
  }
}
</style>
