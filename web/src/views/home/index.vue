<template>
  <main class="home-v4">
    <header class="top-nav">
      <div class="brand">
        <span class="brand-logo">AG</span>
        <div class="brand-copy">
          <strong>AI Gateway</strong>
          <small>Engineering Delivery System</small>
        </div>
      </div>
      <nav class="nav-links">
        <a href="/docs">文档</a>
        <a href="/api-management">API 管理</a>
        <a href="https://github.com/wenkezhi8/ai-gateway" target="_blank" rel="noreferrer">GitHub</a>
      </nav>
    </header>

    <section class="hero shell-card">
      <div class="hero-main">
        <p class="hero-tag">v1.6.5 · 企业级 AI Gateway</p>
        <h1>
          让 AI 接入从“能跑”
          <span>进化为可持续交付</span>
        </h1>
        <p class="hero-subtitle">
          参考 Superpowers 流程，按 TDD 执行：先定位问题、再设计方案、随后实现与验证，
          最终形成可复用的工程闭环。
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

        <div class="hero-kpis">
          <article class="kpi-card">
            <strong>6 Steps</strong>
            <span>Superpowers Workflow</span>
          </article>
          <article class="kpi-card">
            <strong>4 Stages</strong>
            <span>TDD Loop</span>
          </article>
          <article class="kpi-card">
            <strong>One Gateway</strong>
            <span>OpenAI / Anthropic Compatible</span>
          </article>
        </div>
      </div>

      <aside class="hero-side">
        <h3>Request Delivery Pipeline</h3>
        <ol>
          <li v-for="(node, index) in FLOW_NODES" :key="node">
            <span class="node-index">{{ index + 1 }}</span>
            <div class="node-copy">
              <strong>{{ node }}</strong>
              <small>Traceable · Observable · Recoverable</small>
            </div>
          </li>
        </ol>
        <div class="pipeline-note">
          <p>Fail-open + 影子观测 + 回归验证，降低策略上线风险。</p>
        </div>
      </aside>
    </section>

    <section class="proof-strip shell-card">
      <p>先计划后执行</p>
      <p>范围控制</p>
      <p>验证优先</p>
      <p>每阶段 commit</p>
      <p>输出可追溯</p>
    </section>

    <section id="workflow" class="workflow shell-card">
      <header class="section-head">
        <h2>Superpowers 标准工作流</h2>
        <p>从排查到复盘，全流程固定节奏，避免“先改后对齐”。</p>
      </header>

      <div class="workflow-timeline">
        <article v-for="(step, index) in WORKFLOW_STEPS" :key="step.title" class="timeline-item">
          <div class="timeline-mark">
            <span>{{ index + 1 }}</span>
          </div>
          <div class="timeline-body">
            <header>
              <em>{{ step.phase }}</em>
              <h3>{{ step.title }}</h3>
            </header>
            <p>{{ step.detail }}</p>
            <small>输出物：{{ step.deliverable }}</small>
          </div>
        </article>
      </div>
    </section>

    <section class="tdd shell-card">
      <header class="section-head">
        <h2>TDD 执行墙</h2>
        <p>先红后绿，再重构，最后全量验证，不跳步。</p>
      </header>

      <div class="tdd-wall">
        <article v-for="stage in TDD_STAGES" :key="stage.name" class="tdd-card">
          <strong>{{ stage.name }}</strong>
          <p>{{ stage.detail }}</p>
          <code>{{ stage.command }}</code>
        </article>
      </div>

      <div class="verify-box">
        <p>发布前统一验证</p>
        <div>
          <code>npm run test:unit</code>
          <code>npm run typecheck</code>
          <code>npm run build</code>
        </div>
      </div>
    </section>

    <section class="capability shell-card">
      <header class="section-head">
        <h2>能力矩阵</h2>
        <p>围绕质量、稳定、成本、兼容四个维度组织能力。</p>
      </header>

      <div class="cap-grid">
        <article v-for="column in CAPABILITY_COLUMNS" :key="column.title" class="cap-card">
          <h3>{{ column.title }}</h3>
          <ul>
            <li v-for="point in column.points" :key="point">{{ point }}</li>
          </ul>
        </article>
      </div>
    </section>

    <section class="quickstart shell-card">
      <header class="section-head">
        <h2>30 秒快速启动</h2>
        <p>部署、编译、调用三条路径，命令可直接复制执行。</p>
      </header>

      <el-tabs v-model="quickTab" class="quick-tabs">
        <el-tab-pane label="Docker" name="docker" />
        <el-tab-pane label="源码" name="source" />
        <el-tab-pane label="API" name="api" />
      </el-tabs>

      <div class="code-panel">
        <div class="code-panel-head">
          <span>bash</span>
          <el-button text @click="copyCurrentCommand">复制命令</el-button>
        </div>
        <pre><code>{{ currentCommand }}</code></pre>
      </div>
    </section>

    <footer class="footer shell-card">
      <p>开源免费 · MIT 许可 · 版本真相源：Git Tag</p>
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
    document.querySelector(action.route)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
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
.home-v4 {
  --bg: #eef7ff;
  --ink: #111827;
  --soft: #475569;
  --line: #d6e5f5;
  --card: rgba(255, 255, 255, 0.88);
  --accent: #0f766e;
  --accent-strong: #0c4a6e;
  --signal: #ea580c;

  min-height: 100vh;
  padding: 24px;
  color: var(--ink);
  background:
    radial-gradient(circle at 14% 0%, rgba(15, 118, 110, 0.18) 0%, transparent 36%),
    radial-gradient(circle at 82% 8%, rgba(14, 116, 144, 0.14) 0%, transparent 30%),
    linear-gradient(180deg, #f6fbff 0%, #eaf3fc 56%, #edf6ff 100%);
  font-family: 'Avenir Next', 'Manrope', 'SF Pro Display', 'PingFang SC', 'Segoe UI', sans-serif;
}

.shell-card {
  background: var(--card);
  border: 1px solid var(--line);
  border-radius: 22px;
  box-shadow: 0 12px 30px rgba(15, 23, 42, 0.07);
  backdrop-filter: blur(6px);
}

.top-nav {
  max-width: 1220px;
  margin: 0 auto 18px;
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

.brand-logo {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  display: grid;
  place-items: center;
  font-weight: 900;
  letter-spacing: 0.04em;
  color: #fff;
  background: linear-gradient(145deg, var(--accent-strong), var(--accent));
}

.brand-copy {
  display: grid;

  strong {
    line-height: 1;
    font-size: 17px;
  }

  small {
    margin-top: 4px;
    color: var(--soft);
    font-size: 12px;
  }
}

.nav-links {
  display: flex;
  gap: 16px;

  a {
    color: var(--soft);
    text-decoration: none;
    font-size: 14px;

    &:hover {
      color: var(--accent-strong);
    }
  }
}

.hero {
  max-width: 1220px;
  margin: 0 auto 18px;
  padding: 28px;
  display: grid;
  grid-template-columns: 1.35fr 1fr;
  gap: 14px;
}

.hero-main {
  animation: rise 0.45s ease-out both;

  .hero-tag {
    width: fit-content;
    margin: 0;
    padding: 7px 11px;
    border-radius: 999px;
    font-size: 12px;
    font-weight: 700;
    color: #0f766e;
    background: rgba(15, 118, 110, 0.15);
  }

  h1 {
    margin: 12px 0 0;
    font-size: 46px;
    line-height: 1.05;
    letter-spacing: -0.025em;

    span {
      margin-top: 7px;
      display: block;
      color: var(--accent-strong);
    }
  }

  .hero-subtitle {
    margin: 14px 0 0;
    color: var(--soft);
    line-height: 1.7;
    max-width: 680px;
  }
}

.hero-actions {
  margin-top: 18px;
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.hero-kpis {
  margin-top: 16px;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}

.kpi-card {
  border: 1px solid var(--line);
  border-radius: 14px;
  padding: 10px;
  background: #fff;
  display: grid;
  gap: 3px;

  strong {
    font-size: 18px;
    color: #0c4a6e;
  }

  span {
    font-size: 12px;
    color: var(--soft);
  }
}

.hero-side {
  border: 1px solid var(--line);
  border-radius: 18px;
  background: #fff;
  padding: 18px;
  animation: rise 0.56s ease-out both;

  h3 {
    margin: 0;
    font-size: 20px;
  }

  ol {
    list-style: none;
    margin: 14px 0 0;
    padding: 0;
    display: grid;
    gap: 9px;
  }

  li {
    display: flex;
    align-items: flex-start;
    gap: 10px;
    border-radius: 14px;
    border: 1px solid var(--line);
    padding: 10px;
    background: #f9fcff;
  }

  .node-index {
    width: 26px;
    height: 26px;
    border-radius: 999px;
    display: grid;
    place-items: center;
    color: #fff;
    font-size: 12px;
    font-weight: 700;
    background: linear-gradient(145deg, var(--signal), #fb923c);
  }

  .node-copy {
    display: grid;
    gap: 3px;

    strong {
      font-size: 14px;
    }

    small {
      color: var(--soft);
      font-size: 12px;
    }
  }

  .pipeline-note {
    margin-top: 12px;
    border-radius: 12px;
    padding: 10px;
    border: 1px dashed #b6d2ea;
    background: #f3faff;

    p {
      margin: 0;
      color: #0c4a6e;
      font-size: 13px;
      line-height: 1.55;
    }
  }
}

.proof-strip {
  max-width: 1220px;
  margin: 0 auto 18px;
  padding: 12px;
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 8px;

  p {
    margin: 0;
    border-radius: 10px;
    padding: 8px;
    text-align: center;
    font-size: 13px;
    font-weight: 700;
    color: #14532d;
    border: 1px solid #c7e8d0;
    background: #f0fff4;
  }
}

.workflow,
.tdd,
.capability,
.quickstart,
.footer {
  max-width: 1220px;
  margin: 0 auto 18px;
  padding: 24px;
}

.section-head {
  h2 {
    margin: 0;
    font-size: 31px;
    letter-spacing: -0.01em;
  }

  p {
    margin: 8px 0 0;
    color: var(--soft);
  }
}

.workflow-timeline {
  margin-top: 18px;
  display: grid;
  gap: 10px;
}

.timeline-item {
  display: grid;
  grid-template-columns: 38px 1fr;
  gap: 10px;
  align-items: stretch;
}

.timeline-mark {
  display: grid;
  align-content: start;

  span {
    width: 30px;
    height: 30px;
    border-radius: 10px;
    display: grid;
    place-items: center;
    color: #fff;
    background: linear-gradient(145deg, var(--accent), var(--accent-strong));
    font-size: 13px;
    font-weight: 800;
  }
}

.timeline-body {
  border: 1px solid var(--line);
  border-radius: 16px;
  background: #fff;
  padding: 12px;

  header {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;

    em {
      font-style: normal;
      padding: 3px 7px;
      border-radius: 999px;
      font-size: 11px;
      font-weight: 800;
      color: #7c2d12;
      border: 1px solid #fed7aa;
      background: #fff7ed;
      letter-spacing: 0.04em;
    }

    h3 {
      margin: 0;
      font-size: 17px;
    }
  }

  p {
    margin: 8px 0 6px;
    color: var(--soft);
    font-size: 14px;
    line-height: 1.55;
  }

  small {
    color: #0c4a6e;
    font-weight: 700;
    font-size: 12px;
  }
}

.tdd-wall {
  margin-top: 18px;
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}

.tdd-card {
  border: 1px solid var(--line);
  border-radius: 16px;
  padding: 12px;
  background: #fff;
  display: grid;
  gap: 10px;

  strong {
    font-size: 21px;
    letter-spacing: 0.03em;
  }

  p {
    margin: 0;
    color: var(--soft);
    font-size: 13px;
    line-height: 1.55;
  }

  code {
    display: block;
    border: 1px dashed #c0d8ee;
    border-radius: 10px;
    background: #f5faff;
    color: #0c4a6e;
    padding: 10px;
    font-size: 12px;
    line-height: 1.5;
    white-space: pre-wrap;
    word-break: break-word;
  }
}

.verify-box {
  margin-top: 12px;
  border: 1px solid #fdd4a6;
  border-radius: 12px;
  background: #fff8ef;
  padding: 10px;
  display: grid;
  gap: 8px;

  p {
    margin: 0;
    font-size: 13px;
    color: #7c2d12;
    font-weight: 700;
  }

  div {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }

  code {
    display: inline-block;
    padding: 4px 8px;
    border-radius: 8px;
    border: 1px solid #fdd4a6;
    background: #fff;
    color: #7c2d12;
    font-size: 12px;
  }
}

.cap-grid {
  margin-top: 16px;
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}

.cap-card {
  border: 1px solid var(--line);
  border-radius: 16px;
  background: #fff;
  padding: 12px;

  h3 {
    margin: 0;
    font-size: 17px;
  }

  ul {
    margin: 10px 0 0;
    padding: 0;
    list-style: none;
    display: grid;
    gap: 7px;
  }

  li {
    font-size: 14px;
    color: var(--soft);
    position: relative;
    padding-left: 16px;

    &::before {
      content: '';
      width: 7px;
      height: 7px;
      border-radius: 999px;
      background: var(--accent);
      position: absolute;
      left: 0;
      top: 8px;
    }
  }
}

.quick-tabs {
  margin-top: 14px;
}

.code-panel {
  margin-top: 10px;
  border: 1px solid var(--line);
  border-radius: 16px;
  overflow: hidden;
  background: #fff;

  pre {
    margin: 0;
    padding: 14px;
    overflow-x: auto;
  }

  code {
    font-family: 'JetBrains Mono', 'SF Mono', 'Menlo', 'Consolas', monospace;
    font-size: 13px;
    line-height: 1.65;
    color: var(--ink);
  }
}

.code-panel-head {
  border-bottom: 1px solid var(--line);
  background: #f8fcff;
  color: var(--soft);
  font-size: 13px;
  padding: 10px 12px;
  display: flex;
  justify-content: space-between;
}

.footer {
  p {
    margin: 0;
    color: var(--soft);
    font-size: 14px;
  }
}

.footer-links {
  margin-top: 10px;
  display: flex;
  flex-wrap: wrap;
  gap: 14px;

  a {
    text-decoration: none;
    color: #0c4a6e;
    font-size: 14px;

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

  .proof-strip {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .tdd-wall {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .cap-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 768px) {
  .home-v4 {
    padding: 16px;
  }

  .top-nav {
    flex-direction: column;
    align-items: flex-start;
  }

  .hero {
    padding: 20px;
  }

  .hero-main h1 {
    font-size: 34px;
  }

  .hero-kpis,
  .proof-strip,
  .tdd-wall,
  .cap-grid {
    grid-template-columns: 1fr;
  }

  .workflow,
  .tdd,
  .capability,
  .quickstart,
  .footer {
    padding: 18px;
  }

  .section-head h2 {
    font-size: 24px;
  }
}
</style>
