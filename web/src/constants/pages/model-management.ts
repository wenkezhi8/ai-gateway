export type ModelManagementProvider = {
  id: string
  label: string
  color: string
  logo: string
  defaultModel: string
  models: string[]
  custom: boolean
}

export const MODEL_MANAGEMENT_DEFAULT_COLOR = '#409EFF'
export const MODEL_MANAGEMENT_FALLBACK_COLOR = '#909399'
export const MODEL_MANAGEMENT_DEFAULT_SCORE = 80

export const MODEL_MANAGEMENT_DEFAULT_PROVIDERS: ModelManagementProvider[] = [
  { id: 'deepseek', label: 'DeepSeek', color: '#4D6BFE', logo: '/logos/deepseek.svg', defaultModel: 'deepseek-chat', models: ['deepseek-chat', 'deepseek-reasoner', 'deepseek-coder'], custom: false },
  { id: 'openai', label: 'OpenAI', color: '#10A37F', logo: '/logos/openai.svg', defaultModel: 'gpt-4o', models: ['gpt-4o', 'gpt-4o-mini', 'gpt-4-turbo', 'gpt-3.5-turbo', 'o1', 'o1-mini'], custom: false },
  { id: 'anthropic', label: 'Anthropic', color: '#CC785C', logo: '/logos/anthropic.svg', defaultModel: 'claude-3-5-sonnet-20241022', models: ['claude-3-5-sonnet-20241022', 'claude-3-5-haiku-20241022', 'claude-3-opus-20240229'], custom: false },
  { id: 'qwen', label: '阿里云通义千问', color: '#FF6A00', logo: '/logos/qwen.svg', defaultModel: 'qwen-max', models: ['qwen-max', 'qwen-plus', 'qwen-turbo', 'qwen-long', 'qwen-vl-max'], custom: false },
  { id: 'zhipu', label: '智谱AI', color: '#3657ED', logo: '/logos/zhipu.svg', defaultModel: 'glm-4-plus', models: ['glm-4-plus', 'glm-4', 'glm-4-air', 'glm-4-flash', 'glm-4-long'], custom: false },
  { id: 'moonshot', label: '月之暗面 (Kimi)', color: '#1A1A1A', logo: '/logos/moonshot.svg', defaultModel: 'moonshot-v1-8k', models: ['kimi-k2.5', 'moonshot-v1-8k', 'moonshot-v1-32k', 'moonshot-v1-128k'], custom: false },
  { id: 'minimax', label: 'MiniMax', color: '#615CED', logo: '/logos/minimax.svg', defaultModel: 'abab6.5s-chat', models: ['abab6.5s-chat', 'abab6.5g-chat', 'abab6.5t-chat', 'abab5.5-chat'], custom: false },
  { id: 'baichuan', label: '百川智能', color: '#0066FF', logo: '/logos/baichuan.svg', defaultModel: 'Baichuan4', models: ['Baichuan4', 'Baichuan3-Turbo', 'Baichuan3-Turbo-128k'], custom: false },
  { id: 'volcengine', label: '火山方舟 (豆包)', color: '#FF4D4F', logo: '/logos/volcengine.svg', defaultModel: 'doubao-pro-128k', models: ['doubao-pro-256k', 'doubao-pro-128k', 'doubao-pro-32k', 'doubao-lite-128k'], custom: false },
  { id: 'google', label: 'Google Gemini', color: '#4285F4', logo: '/logos/google.svg', defaultModel: 'gemini-2.0-flash', models: ['gemini-2.0-flash', 'gemini-1.5-pro', 'gemini-1.5-flash'], custom: false }
]
