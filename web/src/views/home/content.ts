export interface HeroAction {
  id: 'quick-start' | 'workflow' | 'github' | string
  label: string
  kind: 'primary' | 'secondary' | 'link'
  route?: string
  href?: string
}

export interface WorkflowStep {
  title: string
  detail: string
  deliverable: string
}

export interface TddStage {
  name: 'RED' | 'GREEN' | 'REFACTOR' | 'VERIFY'
  detail: string
  command: string
}

export const HERO_ACTIONS: HeroAction[] = [
  {
    id: 'quick-start',
    label: '快速开始',
    kind: 'primary',
    route: '/docs'
  },
  {
    id: 'workflow',
    label: '查看工作流',
    kind: 'secondary',
    route: '#workflow'
  },
  {
    id: 'github',
    label: 'GitHub',
    kind: 'link',
    href: 'https://github.com/wenkezhi8/ai-gateway'
  }
]

export const WORKFLOW_STEPS: WorkflowStep[] = [
  {
    title: '问题排查测试',
    detail: '读取现有代码并执行相关测试，完整列出缺陷与影响范围。',
    deliverable: '问题清单'
  },
  {
    title: '修复方案讨论',
    detail: '确定修复边界、技术方案、排期节点与验收标准。',
    deliverable: '修复方案'
  },
  {
    title: '代码修复',
    detail: '只修改需求相关逻辑，避免无关变更引入新风险。',
    deliverable: '变更补丁'
  },
  {
    title: '回归验证',
    detail: '执行类型检查、单测与构建验证，确认修复有效。',
    deliverable: '验证结果'
  },
  {
    title: '合规审计',
    detail: '检查编码规范、安全规范与接口一致性。',
    deliverable: '审计结论'
  },
  {
    title: '复盘归档',
    detail: '输出根因分析与规避方案，形成团队可复用经验。',
    deliverable: '复盘报告'
  }
]

export const TDD_STAGES: TddStage[] = [
  {
    name: 'RED',
    detail: '先写失败测试，定义预期行为。',
    command: 'npm run test:unit -- src/views/home/content.test.ts'
  },
  {
    name: 'GREEN',
    detail: '最小实现让测试通过，不引入额外功能。',
    command: 'npm run test:unit -- src/views/home/content.test.ts'
  },
  {
    name: 'REFACTOR',
    detail: '在测试仍然通过的前提下优化结构与可读性。',
    command: 'npm run test:unit'
  },
  {
    name: 'VERIFY',
    detail: '做完整验证，防止“看起来完成”的假阳性。',
    command: 'npm run typecheck && npm run build'
  }
]

export const FLOW_NODES = [
  'Client / SDK',
  'AI Gateway Guardrails',
  'Router + Cache + Limit',
  'Provider Mesh'
] as const

export const CAPABILITY_COLUMNS = [
  {
    title: '质量保障',
    points: ['TDD 执行', '回归验证', '根因复盘']
  },
  {
    title: '稳定运行',
    points: ['多账号容灾', '限流与熔断', '实时监控']
  },
  {
    title: '成本控制',
    points: ['智能路由', '响应缓存', '按任务 TTL']
  },
  {
    title: '生态兼容',
    points: ['OpenAI 兼容', 'Anthropic 兼容', '多服务商接入']
  }
] as const

export const QUICK_START_COMMANDS = {
  docker: `git clone https://github.com/wenkezhi8/ai-gateway.git
cd ai-gateway

docker-compose up -d
docker-compose logs -f`,
  source: `git clone https://github.com/wenkezhi8/ai-gateway.git
cd ai-gateway

make build
cd web && npm install && npm run build
./bin/ai-gateway`,
  api: `curl http://localhost:8566/api/v1/chat/completions \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -d '{"model":"auto","messages":[{"role":"user","content":"你好"}]}'`
} as const
