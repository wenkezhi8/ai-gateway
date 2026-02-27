export const PROVIDERS_ACCOUNTS_BASE_TYPES: Array<{ label: string; value: string }> = [
  { label: 'OpenAI', value: 'openai' },
  { label: 'Anthropic Claude', value: 'anthropic' },
  { label: 'Azure OpenAI', value: 'azure-openai' },
  { label: 'Google Gemini', value: 'google' },
  { label: 'DeepSeek', value: 'deepseek' },
  { label: '阿里云通义千问', value: 'qwen' },
  { label: '智谱AI', value: 'zhipu' },
  { label: '月之暗面 (Kimi)', value: 'moonshot' },
  { label: 'MiniMax', value: 'minimax' },
  { label: '百川智能', value: 'baichuan' },
  { label: '火山方舟 (豆包)', value: 'volcengine' },
  { label: '百度文心一言', value: 'ernie' },
  { label: '腾讯混元', value: 'hunyuan' },
  { label: '讯飞星火', value: 'spark' },
  { label: 'llama.cpp', value: 'llamacpp' },
  { label: 'vLLM', value: 'vllm' },
  { label: 'Ollama', value: 'ollama' },
  { label: 'LM Studio', value: 'lmstudio' },
  { label: 'AingDesk', value: 'aingdesk' }
]

export const INTERNATIONAL_PROVIDER_SET = new Set(['openai', 'anthropic', 'azure-openai', 'google'])
export const CHINESE_PROVIDER_SET = new Set(['deepseek', 'qwen', 'zhipu', 'moonshot', 'minimax', 'baichuan', 'volcengine', 'ernie', 'hunyuan', 'spark'])
export const LOCAL_PROVIDER_SET = new Set(['llamacpp', 'vllm', 'ollama', 'lmstudio', 'aingdesk'])

export const PROVIDERS_ACCOUNTS_DEFAULT_ENDPOINTS: Record<string, string> = {
  openai: 'https://api.openai.com/v1',
  anthropic: 'https://api.anthropic.com/v1',
  'azure-openai': 'https://your-resource.openai.azure.com',
  google: 'https://generativelanguage.googleapis.com/v1',
  deepseek: 'https://api.deepseek.com/v1',
  qwen: 'https://dashscope.aliyuncs.com/compatible-mode/v1',
  zhipu: 'https://open.bigmodel.cn/api/paas/v4',
  moonshot: 'https://api.moonshot.cn/v1',
  minimax: 'https://api.minimax.chat/v1',
  baichuan: 'https://api.baichuan-ai.com/v1',
  volcengine: 'https://ark.cn-beijing.volces.com/api/v3',
  ernie: 'https://aip.baidubce.com/rpc/2.0/ai_custom/v1/wenxinworkshop/chat',
  hunyuan: 'https://api.hunyuan.cloud.tencent.com/v1',
  spark: 'https://spark-api-open.xfyun.com/v1',
  llamacpp: 'http://localhost:8080/v1',
  vllm: 'http://localhost:8000/v1',
  ollama: 'http://localhost:11434/v1',
  lmstudio: 'http://localhost:1234/v1',
  aingdesk: 'http://localhost:5678/v1'
}

export const PROVIDERS_ACCOUNTS_CODING_PLAN_ENDPOINTS: Record<string, string> = {
  openai: 'https://api.openai.com/v1',
  anthropic: 'https://api.anthropic.com/v1',
  deepseek: 'https://api.deepseek.com/v1',
  qwen: 'https://coding.dashscope.aliyuncs.com/v1',
  zhipu: 'https://open.bigmodel.cn/api/coding/paas/v4',
  moonshot: 'https://api.kimi.com/coding/v1',
  kimi: 'https://api.kimi.com/coding/v1',
  minimax: 'https://api.minimaxi.com/anthropic/v1',
  volcengine: 'https://ark.cn-beijing.volces.com/api/coding/v3'
}

export const PROVIDERS_ACCOUNTS_PROVIDER_COLORS: Record<string, string> = {
  openai: '#10A37F',
  anthropic: '#CC785C',
  'azure-openai': '#0078D4',
  google: '#4285F4',
  deepseek: '#4D6BFE',
  qwen: '#FF6A00',
  zhipu: '#3657ED',
  moonshot: '#1A1A1A',
  minimax: '#615CED',
  baichuan: '#0066FF',
  volcengine: '#FF4D4F',
  ernie: '#2932E1',
  hunyuan: '#00A3FF',
  spark: '#E60012',
  llamacpp: '#4A90D9',
  vllm: '#FF6B6B',
  ollama: '#6B7280',
  lmstudio: '#3B82F6',
  aingdesk: '#8B5CF6'
}

export const PROVIDERS_ACCOUNTS_PROVIDER_LOGOS: Record<string, string> = {
  openai: '/logos/openai.svg',
  anthropic: '/logos/anthropic.svg',
  'azure-openai': '/logos/azure.svg',
  google: '/logos/google.svg',
  deepseek: '/logos/deepseek.svg',
  qwen: '/logos/qwen.svg',
  zhipu: '/logos/zhipu.svg',
  moonshot: '/logos/moonshot.svg',
  minimax: '/logos/minimax.svg',
  baichuan: '/logos/baichuan.svg',
  volcengine: '/logos/volcengine.svg',
  ernie: '/logos/ernie.svg',
  hunyuan: '/logos/hunyuan.svg',
  spark: '/logos/spark.svg',
  llamacpp: '/logos/llamacpp.svg',
  vllm: '/logos/vllm.svg',
  ollama: '/logos/ollama.svg',
  lmstudio: '/logos/lmstudio.svg',
  aingdesk: '/logos/aingdesk.svg'
}
