type DefaultModel = {
  id: string
  name: string
  provider: string
  type: 'chat' | 'completion' | 'embedding' | 'image'
  enabled: boolean
  maxTokens: number
  inputPrice: number
  outputPrice: number
}

export const STORE_DEFAULT_MODELS: DefaultModel[] = [
  { id: 'gpt-4o', name: 'GPT-4o', provider: 'openai', type: 'chat', enabled: true, maxTokens: 128000, inputPrice: 2.5, outputPrice: 10 },
  { id: 'gpt-4-turbo', name: 'GPT-4 Turbo', provider: 'openai', type: 'chat', enabled: true, maxTokens: 128000, inputPrice: 10, outputPrice: 30 },
  { id: 'gpt-3.5-turbo', name: 'GPT-3.5 Turbo', provider: 'openai', type: 'chat', enabled: true, maxTokens: 16385, inputPrice: 0.5, outputPrice: 1.5 },
  { id: 'claude-3-5-sonnet', name: 'Claude 3.5 Sonnet', provider: 'anthropic', type: 'chat', enabled: true, maxTokens: 200000, inputPrice: 3, outputPrice: 15 },
  { id: 'claude-3-opus', name: 'Claude 3 Opus', provider: 'anthropic', type: 'chat', enabled: true, maxTokens: 200000, inputPrice: 15, outputPrice: 75 },
  { id: 'glm-4-plus', name: 'GLM-4 Plus', provider: 'zhipu', type: 'chat', enabled: true, maxTokens: 128000, inputPrice: 0.05, outputPrice: 0.05 },
  { id: 'glm-4-flash', name: 'GLM-4 Flash', provider: 'zhipu', type: 'chat', enabled: true, maxTokens: 128000, inputPrice: 0.001, outputPrice: 0.001 },
  { id: 'qwen-max', name: '通义千问 Max', provider: 'qwen', type: 'chat', enabled: true, maxTokens: 32000, inputPrice: 0.04, outputPrice: 0.12 },
  { id: 'qwen-plus', name: '通义千问 Plus', provider: 'qwen', type: 'chat', enabled: true, maxTokens: 128000, inputPrice: 0.0008, outputPrice: 0.002 },
  { id: 'deepseek-chat', name: 'DeepSeek Chat', provider: 'deepseek', type: 'chat', enabled: true, maxTokens: 64000, inputPrice: 0.001, outputPrice: 0.002 },
  { id: 'deepseek-coder', name: 'DeepSeek Coder', provider: 'deepseek', type: 'chat', enabled: true, maxTokens: 64000, inputPrice: 0.001, outputPrice: 0.002 },
  { id: 'doubao-pro', name: '豆包 Pro', provider: 'volcengine', type: 'chat', enabled: true, maxTokens: 32000, inputPrice: 0.0008, outputPrice: 0.002 }
]
