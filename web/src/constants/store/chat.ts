import type { ProviderConfig } from '@/types/chat'

export const CHAT_DEFAULT_PROVIDER = 'openai'
export const CHAT_DEFAULT_MODEL = 'gpt-4o'

export const CHAT_DEFAULT_PROVIDERS: ProviderConfig[] = [
  { label: 'OpenAI', value: 'openai', color: '#10A37F', models: ['gpt-4o', 'gpt-4o-mini', 'gpt-4-turbo'], logo: '/logos/openai.svg' },
  { label: 'Anthropic Claude', value: 'anthropic', color: '#CC785C', models: ['claude-3-5-sonnet-20241022', 'claude-3-5-haiku-20241022'], logo: '/logos/anthropic.svg' },
  { label: 'DeepSeek', value: 'deepseek', color: '#4D6BFE', models: ['deepseek-chat', 'deepseek-reasoner', 'deepseek-coder'], logo: '/logos/deepseek.svg' },
  { label: '阿里云通义千问', value: 'qwen', color: '#FF6A00', models: ['qwen-max', 'qwen-plus', 'qwen-turbo'], logo: '/logos/qwen.svg' },
  { label: '智谱AI', value: 'zhipu', color: '#3657ED', models: ['glm-4-plus', 'glm-4', 'glm-4-flash'], logo: '/logos/zhipu.svg' },
  { label: '月之暗面 (Kimi)', value: 'moonshot', color: '#1A1A1A', models: ['moonshot-v1-8k', 'moonshot-v1-32k'], logo: '/logos/moonshot.svg' },
  { label: '火山方舟 (豆包)', value: 'volcengine', color: '#FF4D4F', models: ['doubao-pro-128k', 'doubao-lite-128k'], logo: '/logos/volcengine.svg' }
]
