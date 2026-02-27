export type DocsProvider = {
  name: string
  enabled: boolean
  models: string[]
  endpoint: string
}

export const DOCS_PROVIDERS: DocsProvider[] = [
  {
    name: 'OpenAI',
    enabled: true,
    models: ['gpt-4o', 'gpt-4o-mini', 'gpt-4-turbo', 'gpt-3.5-turbo', 'o1', 'o1-mini'],
    endpoint: 'https://api.openai.com/v1'
  },
  {
    name: 'Anthropic',
    enabled: true,
    models: ['claude-3-5-sonnet', 'claude-3-5-haiku', 'claude-3-opus'],
    endpoint: 'https://api.anthropic.com/v1'
  },
  {
    name: '智谱 AI',
    enabled: true,
    models: ['glm-4-plus', 'glm-4-air', 'glm-4-flash', 'glm-4-long'],
    endpoint: 'https://open.bigmodel.cn/api/paas/v4'
  },
  {
    name: '通义千问',
    enabled: true,
    models: ['qwen-max', 'qwen-plus', 'qwen-turbo', 'qwen-long'],
    endpoint: 'https://dashscope.aliyuncs.com/api/v1'
  },
  {
    name: 'DeepSeek',
    enabled: true,
    models: ['deepseek-chat', 'deepseek-coder', 'deepseek-reasoner'],
    endpoint: 'https://api.deepseek.com/v1'
  },
  {
    name: '火山方舟',
    enabled: true,
    models: ['doubao-pro-32k', 'doubao-pro-128k', 'doubao-lite-32k'],
    endpoint: 'https://ark.cn-beijing.volces.com/api/v3'
  },
  {
    name: '文心一言',
    enabled: false,
    models: ['ernie-4.0', 'ernie-3.5', 'ernie-speed'],
    endpoint: 'https://aip.baidubce.com/rpc/2.0/ai_custom/v1'
  }
]
