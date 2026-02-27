export const PROVIDERS_AVAILABLE_MODELS = [
  'gpt-4o', 'gpt-4o-mini', 'gpt-4-turbo', 'gpt-4', 'gpt-3.5-turbo', 'o1', 'o1-mini', 'o1-preview',
  'claude-3-5-sonnet-20241022', 'claude-3-opus-20240229', 'claude-3-sonnet-20240229', 'claude-3-haiku-20240307',
  'gemini-2.0-flash-exp', 'gemini-1.5-pro', 'gemini-1.5-flash', 'gemini-pro',
  'gpt-4o', 'gpt-4', 'gpt-35-turbo',
  'doubao-pro-256k', 'doubao-pro-128k', 'doubao-pro-32k', 'doubao-lite-128k', 'doubao-lite-32k',
  'qwen-max', 'qwen-max-longcontext', 'qwen-plus', 'qwen-turbo', 'qwen-long',
  'ernie-4.0-8k', 'ernie-4.0', 'ernie-3.5-8k', 'ernie-3.5', 'ernie-speed-8k', 'ernie-speed',
  'glm-4-plus', 'glm-4-0520', 'glm-4-air', 'glm-4-airx', 'glm-4-long', 'glm-4-flash',
  'hunyuan-lite', 'hunyuan-standard', 'hunyuan-pro', 'hunyuan-turbo',
  'moonshot-v1-8k', 'moonshot-v1-32k', 'moonshot-v1-128k',
  'abab6.5-chat', 'abab6.5s-chat', 'abab5.5-chat', 'abab5.5s-chat',
  'Baichuan4', 'Baichuan3-Turbo', 'Baichuan3-Turbo-128k', 'Baichuan2-Turbo',
  'spark-v3.5', 'spark-v3.0', 'spark-v2.0', 'spark-v1.5',
  'pangu-natural-language-10b', 'pangu-nlg-2b',
  'nova-ptc-xl-v1', 'nova-ptc-large-v1',
  '360gpt2-pro', '360gpt-turbo',
  'deepseek-chat', 'deepseek-reasoner'
] as const

export const PROVIDERS_COLOR_MAP: Record<string, string> = {
  openai: '#10A37F',
  azure: '#0078D4',
  anthropic: '#CC785C',
  google: '#4285F4',
  volcengine: '#FF4D4F',
  qwen: '#FF6A00',
  ernie: '#2932E1',
  zhipu: '#3657ED',
  hunyuan: '#00A3FF',
  moonshot: '#1A1A1A',
  minimax: '#615CED',
  baichuan: '#0066FF',
  spark: '#E60012',
  deepseek: '#4D6BFE',
  custom: '#8B5CF6'
}

export const PROVIDERS_ICON_MAP: Record<string, string> = {
  openai: 'ChatDotRound',
  azure: 'Platform',
  anthropic: 'ChatLineRound',
  google: 'Star',
  volcengine: 'Lightning',
  qwen: 'Sunny',
  ernie: 'Reading',
  zhipu: 'MagicStick',
  hunyuan: 'Connection',
  moonshot: 'Moon',
  minimax: 'Cpu',
  baichuan: 'TrendCharts',
  spark: 'Promotion',
  deepseek: 'Search',
  custom: 'Setting'
}

export const PROVIDERS_ENDPOINT_MAP: Record<string, string> = {
  openai: 'https://api.openai.com/v1',
  azure: 'https://your-resource.openai.azure.com',
  anthropic: 'https://api.anthropic.com/v1',
  google: 'https://generativelanguage.googleapis.com/v1beta',
  volcengine: 'https://ark.cn-beijing.volces.com/api/v3',
  qwen: 'https://dashscope.aliyuncs.com/api/v1',
  ernie: 'https://aip.baidubce.com/rpc/2.0/ai_custom/v1',
  zhipu: 'https://open.bigmodel.cn/api/paas/v4',
  hunyuan: 'https://hunyuan.tencentcloudapi.com',
  moonshot: 'https://api.moonshot.cn/v1',
  minimax: 'https://api.minimax.chat/v1',
  baichuan: 'https://api.baichuan-ai.com/v1',
  spark: 'https://spark-api-open.xf-yun.com/v1',
  deepseek: 'https://api.deepseek.com/v1'
}
