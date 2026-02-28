export interface ChatProviderVisualMeta {
  label: string
  color: string
  logo: string
}

// UI metadata only. Model lists must come from backend APIs.
export const CHAT_PROVIDER_VISUALS: Record<string, ChatProviderVisualMeta> = {
  openai: { label: 'OpenAI', color: '#10A37F', logo: '/logos/openai.svg' },
  anthropic: { label: 'Anthropic Claude', color: '#CC785C', logo: '/logos/anthropic.svg' },
  deepseek: { label: 'DeepSeek', color: '#4D6BFE', logo: '/logos/deepseek.svg' },
  qwen: { label: '阿里云通义千问', color: '#FF6A00', logo: '/logos/qwen.svg' },
  zhipu: { label: '智谱AI', color: '#3657ED', logo: '/logos/zhipu.svg' },
  moonshot: { label: '月之暗面 (Kimi)', color: '#1A1A1A', logo: '/logos/moonshot.svg' },
  volcengine: { label: '火山方舟 (豆包)', color: '#FF4D4F', logo: '/logos/volcengine.svg' }
}

export const CHAT_PROVIDER_VISUAL_FALLBACK: ChatProviderVisualMeta = {
  label: 'AI Provider',
  color: '#909399',
  logo: '/logos/default.svg'
}
