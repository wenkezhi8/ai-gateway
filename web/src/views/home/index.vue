<template>
  <div class="home-page">
    <div class="ambient-grid"></div>
    <div class="ambient-orb orb-left"></div>
    <div class="ambient-orb orb-right"></div>

    <header class="home-header reveal reveal-1">
      <div class="brand">
        <span class="brand-mark">AG</span>
        <span class="brand-text">AI Gateway</span>
      </div>
      <nav class="top-nav">
        <a href="#capabilities">能力</a>
        <a href="#providers">服务商</a>
        <a href="#quickstart">接入</a>
      </nav>
      <div class="header-actions">
        <router-link to="/login" class="btn btn-ghost">控制台</router-link>
      </div>
    </header>

    <main class="home-main">
      <section class="hero reveal reveal-2">
        <div class="hero-copy">
          <p class="eyebrow">统一网关 · 智能路由 · 运维可观测</p>
          <h1>一个 API 入口，
            <span>把多模型能力稳定交付到生产</span>
          </h1>
          <p class="hero-subtitle">
            兼容 OpenAI / Anthropic 调用格式，内置账号切换、限流、缓存、路由策略、Usage 统计与告警，
            让团队从“多家模型对接成本”切换到“业务价值交付”。
          </p>
          <div class="hero-actions">
            <router-link to="/login" class="btn btn-primary">进入控制台</router-link>
            <router-link to="/docs" class="btn btn-outline">查看文档中心</router-link>
            <a href="https://github.com/wenkezhi8/ai-gateway" target="_blank" rel="noreferrer" class="btn btn-ghost">
              GitHub
            </a>
          </div>

          <div class="hero-pills">
            <span>OpenAI Compatible</span>
            <span>Anthropic Compatible</span>
            <span>Redis + SQLite</span>
            <span>Prometheus Metrics</span>
          </div>
        </div>

        <aside class="terminal-card">
          <div class="terminal-head">
            <span class="dot dot-red"></span>
            <span class="dot dot-amber"></span>
            <span class="dot dot-green"></span>
            <strong>gateway-demo.sh</strong>
          </div>
          <pre><code>$ curl -X POST /api/v1/chat/completions
  -H "Authorization: Bearer ***"
  -H "Content-Type: application/json"
  -d '{"model":"auto","messages":[...]} '

# AI Gateway
→ task: code, difficulty: medium
→ route: qwen / qwen-plus
→ cache: miss
← 200 OK (ttft: 438ms)

$ curl /api/admin/dashboard/realtime
{"requests_per_minute":192,
 "cache_hit_rate":0.37,
 "success_rate":0.996}</code></pre>
        </aside>
      </section>

      <section id="capabilities" class="section reveal reveal-3">
        <div class="section-head">
          <h2>面向生产的核心能力</h2>
          <p>参考 OpenClaw 生态站点的“信息密度 + 模块节奏”，并聚焦 ai-gateway 的真实运维场景。</p>
        </div>
        <div class="capability-grid">
          <article v-for="item in capabilities" :key="item.title" class="capability-card">
            <div class="capability-icon">{{ item.icon }}</div>
            <h3>{{ item.title }}</h3>
            <p>{{ item.description }}</p>
          </article>
        </div>
      </section>

      <section id="providers" class="section providers reveal reveal-4">
        <div class="section-head">
          <h2>服务商与模型统一接入</h2>
          <p>一套鉴权、一套监控、一套路由策略，覆盖主流模型服务商。</p>
        </div>
        <div class="provider-matrix">
          <article v-for="provider in providers" :key="provider.name" class="provider-item">
            <img :src="provider.logo" :alt="provider.name" />
            <div>
              <h3>{{ provider.name }}</h3>
              <p>{{ provider.note }}</p>
            </div>
          </article>
        </div>
      </section>

      <section id="quickstart" class="section quickstart reveal reveal-5">
        <div class="section-head">
          <h2>3 步接入你的业务系统</h2>
        </div>
        <ol class="steps">
          <li>
            <h3>部署网关与配置账号</h3>
            <p>本地 `make run` 或 Docker 启动，填入上游服务商密钥并开启账号策略。</p>
          </li>
          <li>
            <h3>把业务调用切到统一入口</h3>
            <p>客户端仅调用 `/api/v1/chat/completions`，模型参数可用 `auto/default/latest`。</p>
          </li>
          <li>
            <h3>在控制台持续优化</h3>
            <p>通过路由评分、缓存策略、Usage 与告警闭环，按成本/质量/速度持续调优。</p>
          </li>
        </ol>
      </section>
    </main>

    <footer class="home-footer reveal reveal-5">
      <p>AI Gateway · Unified Model Access Layer</p>
      <div class="footer-links">
        <router-link to="/docs">文档</router-link>
        <router-link to="/login">控制台</router-link>
        <a href="/health" target="_blank" rel="noreferrer">健康检查</a>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
const capabilities = [
  {
    icon: '⇄',
    title: '智能路由与模型选择',
    description: '按任务类型、难度与评分在多模型间自动选择，支持 default/latest/auto 模式。'
  },
  {
    icon: '⚡',
    title: '多层缓存与请求去重',
    description: '请求缓存、语义缓存、路由缓存协同，降低成本并提升高并发下的响应稳定性。'
  },
  {
    icon: '🧭',
    title: '账号切换与限额保护',
    description: '账号级 token/rpm 限额与自动切换机制，避免单账号打满导致服务中断。'
  },
  {
    icon: '📈',
    title: '可观测性与运营看板',
    description: '实时趋势、请求日志、TTFT 指标、告警规则与健康诊断，定位问题更直接。'
  },
  {
    icon: '🔐',
    title: '统一认证与管理 API',
    description: 'JWT 登录、API Key 管理、权限分层与审计日志，便于团队协作治理。'
  },
  {
    icon: '🧩',
    title: 'OpenAI/Anthropic 兼容',
    description: '尽量保持上层业务调用协议不变，把复杂度收敛在网关层。'
  }
]

const providers = [
  { name: 'OpenAI', note: 'GPT 系列', logo: '/logos/openai.svg' },
  { name: 'Anthropic', note: 'Claude 系列', logo: '/logos/anthropic.svg' },
  { name: 'DeepSeek', note: '通用与代码模型', logo: '/logos/deepseek.svg' },
  { name: 'Qwen', note: '通义系列', logo: '/logos/qwen.svg' },
  { name: 'Zhipu', note: 'GLM 系列', logo: '/logos/zhipu.svg' },
  { name: 'Volcengine', note: '豆包系列', logo: '/logos/volcengine.svg' }
]
</script>

<style scoped>
@import url('https://fonts.googleapis.com/css2?family=Space+Grotesk:wght@400;500;700&family=IBM+Plex+Mono:wght@400;500&display=swap');

.home-page {
  --bg-0: #f7fbff;
  --bg-1: #eef6ff;
  --bg-2: #e4f0ff;
  --text-primary: #0f2438;
  --text-secondary: #4f667b;
  --accent: #0ea5a0;
  --accent-strong: #1388c9;
  --warm: #f4b740;
  --line: rgba(98, 136, 164, 0.24);
  position: relative;
  min-height: 100vh;
  overflow: hidden;
  padding: 1.2rem 1.2rem 2.2rem;
  background:
    radial-gradient(circle at 8% 12%, rgba(14, 165, 160, 0.2), transparent 36%),
    radial-gradient(circle at 88% 10%, rgba(244, 183, 64, 0.16), transparent 32%),
    linear-gradient(145deg, var(--bg-0) 0%, var(--bg-1) 46%, var(--bg-2) 100%);
  color: var(--text-primary);
  font-family: 'Space Grotesk', 'PingFang SC', 'Microsoft YaHei', sans-serif;
}

.ambient-grid {
  position: absolute;
  inset: 0;
  pointer-events: none;
  background-image:
    linear-gradient(rgba(114, 152, 181, 0.12) 1px, transparent 1px),
    linear-gradient(90deg, rgba(114, 152, 181, 0.12) 1px, transparent 1px);
  background-size: 36px 36px;
  mask-image: radial-gradient(circle at center, black 32%, transparent 80%);
}

.ambient-orb {
  position: absolute;
  width: 24rem;
  height: 24rem;
  border-radius: 999px;
  filter: blur(64px);
  opacity: 0.52;
  pointer-events: none;
}

.orb-left {
  left: -10rem;
  top: 22rem;
  background: rgba(14, 165, 160, 0.26);
}

.orb-right {
  right: -8rem;
  top: 14rem;
  background: rgba(244, 183, 64, 0.24);
}

.home-header,
.home-main,
.home-footer {
  position: relative;
  z-index: 2;
  max-width: 1180px;
  margin: 0 auto;
}

.home-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.7rem 1rem;
  border: 1px solid var(--line);
  border-radius: 14px;
  backdrop-filter: blur(10px);
  background: rgba(255, 255, 255, 0.74);
}

.brand {
  display: flex;
  align-items: center;
  gap: 0.65rem;
  font-weight: 700;
  letter-spacing: 0.2px;
}

.brand-mark {
  display: grid;
  place-items: center;
  width: 2rem;
  height: 2rem;
  border-radius: 10px;
  background: linear-gradient(135deg, var(--accent), #5bc0eb);
  color: #041017;
  font-weight: 700;
  font-size: 0.78rem;
}

.top-nav {
  display: flex;
  gap: 1rem;
}

.top-nav a,
.footer-links a {
  color: var(--text-secondary);
  text-decoration: none;
  transition: color 0.18s ease;
}

.top-nav a:hover,
.footer-links a:hover {
  color: var(--text-primary);
}

.home-main {
  margin-top: 1.3rem;
}

.hero {
  display: grid;
  grid-template-columns: 1.1fr 0.9fr;
  gap: 1.2rem;
  align-items: stretch;
}

.hero-copy,
.terminal-card,
.section,
.home-footer {
  border: 1px solid var(--line);
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.78);
  backdrop-filter: blur(8px);
}

.hero-copy {
  padding: 1.5rem;
}

.eyebrow {
  display: inline-block;
  margin: 0;
  padding: 0.2rem 0.65rem;
  border-radius: 999px;
  background: rgba(14, 165, 160, 0.12);
  color: #0f766e;
  font-size: 0.84rem;
  letter-spacing: 0.04em;
}

.hero h1 {
  margin: 0.85rem 0 0;
  line-height: 1.1;
  font-size: clamp(2rem, 4vw, 3.1rem);
}

.hero h1 span {
  display: block;
  margin-top: 0.25rem;
  color: #0f766e;
}

.hero-subtitle {
  margin-top: 0.95rem;
  color: var(--text-secondary);
  line-height: 1.65;
}

.hero-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.65rem;
  margin-top: 1.15rem;
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 2.55rem;
  padding: 0 1rem;
  border-radius: 12px;
  border: 1px solid transparent;
  text-decoration: none;
  font-weight: 600;
  transition: transform 0.2s ease, box-shadow 0.2s ease, border-color 0.2s ease;
}

.btn:hover {
  transform: translateY(-1px);
}

.btn-primary {
  background: linear-gradient(135deg, var(--accent), var(--accent-strong));
  color: #f8fffd;
  box-shadow: 0 10px 24px rgba(19, 136, 201, 0.25);
}

.btn-outline {
  border-color: rgba(14, 165, 160, 0.38);
  color: #0f5f66;
}

.btn-ghost {
  border-color: var(--line);
  color: var(--text-secondary);
  background: rgba(255, 255, 255, 0.72);
}

.hero-pills {
  margin-top: 1rem;
  display: flex;
  flex-wrap: wrap;
  gap: 0.45rem;
}

.hero-pills span {
  padding: 0.32rem 0.62rem;
  border-radius: 999px;
  border: 1px solid rgba(14, 165, 160, 0.24);
  color: #3f6079;
  background: rgba(255, 255, 255, 0.76);
  font-size: 0.78rem;
}

.terminal-card {
  padding: 1rem;
  font-family: 'IBM Plex Mono', ui-monospace, SFMono-Regular, Menlo, monospace;
  background: linear-gradient(165deg, rgba(251, 254, 255, 0.92), rgba(240, 248, 255, 0.9));
  box-shadow: inset 0 0 0 1px rgba(19, 136, 201, 0.12);
}

.terminal-head {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  color: #4a6479;
  font-size: 0.82rem;
}

.dot {
  width: 0.62rem;
  height: 0.62rem;
  border-radius: 999px;
}

.dot-red { background: #fb7185; }
.dot-amber { background: #f59e0b; }
.dot-green { background: #34d399; }

.terminal-card pre {
  margin: 0.8rem 0 0;
  white-space: pre-wrap;
  line-height: 1.6;
  color: #18344c;
  font-size: 0.82rem;
}

.section {
  margin-top: 1.2rem;
  padding: 1.25rem;
}

.section-head h2 {
  margin: 0;
  font-size: clamp(1.35rem, 2.6vw, 2rem);
}

.section-head p {
  margin: 0.55rem 0 0;
  color: var(--text-secondary);
}

.capability-grid {
  margin-top: 1rem;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.8rem;
}

.capability-card {
  border: 1px solid rgba(98, 136, 164, 0.26);
  border-radius: 14px;
  padding: 0.95rem;
  background: linear-gradient(165deg, rgba(250, 253, 255, 0.92), rgba(236, 246, 255, 0.86));
}

.capability-icon {
  font-size: 1.22rem;
  width: 2rem;
  height: 2rem;
  border-radius: 10px;
  display: grid;
  place-items: center;
  color: #f4feff;
  background: linear-gradient(135deg, var(--accent), #46b7e5);
}

.capability-card h3 {
  margin: 0.75rem 0 0;
  font-size: 1rem;
}

.capability-card p {
  margin: 0.5rem 0 0;
  color: var(--text-secondary);
  line-height: 1.5;
  font-size: 0.92rem;
}

.provider-matrix {
  margin-top: 0.95rem;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.7rem;
}

.provider-item {
  display: flex;
  align-items: center;
  gap: 0.8rem;
  border: 1px solid rgba(98, 136, 164, 0.24);
  border-radius: 12px;
  padding: 0.75rem;
  background: rgba(255, 255, 255, 0.84);
}

.provider-item img {
  width: 2rem;
  height: 2rem;
  object-fit: contain;
}

.provider-item h3 {
  margin: 0;
  font-size: 1rem;
}

.provider-item p {
  margin: 0.2rem 0 0;
  color: var(--text-secondary);
  font-size: 0.88rem;
}

.steps {
  margin: 0.95rem 0 0;
  padding-left: 1.2rem;
  display: grid;
  gap: 0.95rem;
}

.steps h3 {
  margin: 0;
  color: #12324b;
}

.steps p {
  margin: 0.35rem 0 0;
  color: var(--text-secondary);
}

.home-footer {
  margin-top: 1.2rem;
  padding: 1rem 1.1rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 0.8rem;
}

.home-footer p {
  margin: 0;
  color: var(--text-secondary);
}

.footer-links {
  display: flex;
  gap: 0.9rem;
}

.reveal {
  opacity: 0;
  transform: translateY(12px);
  animation: reveal-up 0.6s ease forwards;
}

.reveal-1 { animation-delay: 0.05s; }
.reveal-2 { animation-delay: 0.12s; }
.reveal-3 { animation-delay: 0.2s; }
.reveal-4 { animation-delay: 0.28s; }
.reveal-5 { animation-delay: 0.36s; }

@keyframes reveal-up {
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@media (max-width: 1080px) {
  .hero {
    grid-template-columns: 1fr;
  }

  .capability-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 760px) {
  .home-page {
    padding: 0.8rem 0.8rem 1.5rem;
  }

  .top-nav {
    display: none;
  }

  .brand-text {
    font-size: 0.95rem;
  }

  .hero-copy,
  .section,
  .home-footer,
  .terminal-card {
    border-radius: 14px;
  }

  .capability-grid,
  .provider-matrix {
    grid-template-columns: 1fr;
  }

  .home-footer {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
