export interface ProviderCatalogItem {
  id: string
  label: string
  color: string
  logo: string
}

const PROVIDER_IDS = [
  'aingdesk',
  'alibaba',
  'amazon',
  'anthropic',
  'azure',
  'baichuan',
  'baidu',
  'bedrock',
  'bytedance',
  'cerebras',
  'chatglm',
  'claude',
  'cloudflare',
  'cohere',
  'copilot',
  'dalle',
  'deepmind',
  'deepseek',
  'doubao',
  'ernie',
  'fireworks',
  'google',
  'gradio',
  'groq',
  'huggingface',
  'hunyuan',
  'internlm',
  'langchain',
  'llamacpp',
  'lmstudio',
  'local',
  'meta',
  'microsoft',
  'midjourney',
  'minimax',
  'mistral',
  'moonshot',
  'nvidia',
  'ollama',
  'openai',
  'perplexity',
  'poe',
  'qwen',
  'replicate',
  'sambanova',
  'siliconflow',
  'sora',
  'spark',
  'stability',
  'stepfun',
  'tencent',
  'together',
  'vertexai',
  'vllm',
  'volcengine',
  'x',
  'zhipu'
] as const

const PROVIDER_LABELS: Partial<Record<(typeof PROVIDER_IDS)[number], string>> = {
  aingdesk: 'AiingDesk',
  alibaba: '阿里云',
  amazon: '亚马逊',
  anthropic: 'Anthropic Claude',
  azure: 'Azure OpenAI',
  baichuan: '百川智能',
  baidu: '百度',
  bedrock: 'Amazon Bedrock',
  bytedance: '字节跳动',
  cerebras: 'Cerebras',
  chatglm: '智谱 ChatGLM',
  claude: 'Claude',
  cloudflare: 'Cloudflare',
  cohere: 'Cohere',
  copilot: 'GitHub Copilot',
  dalle: 'DALL-E',
  deepmind: 'DeepMind',
  deepseek: 'DeepSeek',
  doubao: '豆包',
  ernie: '百度文心一言',
  fireworks: 'Fireworks AI',
  google: 'Google Gemini',
  gradio: 'Gradio',
  groq: 'Groq',
  huggingface: 'Hugging Face',
  hunyuan: '腾讯混元',
  internlm: 'InternLM',
  langchain: 'LangChain',
  llamacpp: 'llama.cpp',
  lmstudio: 'LM Studio',
  local: '本地模型',
  meta: 'Meta',
  microsoft: '微软',
  midjourney: 'Midjourney',
  minimax: 'MiniMax',
  mistral: 'Mistral AI',
  moonshot: '月之暗面 (Kimi)',
  nvidia: 'NVIDIA',
  ollama: 'Ollama',
  openai: 'OpenAI',
  perplexity: 'Perplexity',
  poe: 'Poe',
  qwen: '阿里云通义千问',
  replicate: 'Replicate',
  sambanova: 'SambaNova',
  siliconflow: 'SiliconFlow',
  sora: 'Sora',
  spark: '讯飞星火',
  stability: 'Stability AI',
  stepfun: '阶跃星辰',
  tencent: '腾讯云',
  together: 'Together AI',
  vertexai: 'Vertex AI',
  vllm: 'vLLM',
  volcengine: '火山方舟 (豆包)',
  x: 'xAI',
  zhipu: '智谱AI'
}

const PROVIDER_COLORS: Partial<Record<(typeof PROVIDER_IDS)[number], string>> = {
  openai: '#10A37F',
  anthropic: '#CC785C',
  deepseek: '#4D6BFE',
  qwen: '#FF6A00',
  zhipu: '#3657ED',
  moonshot: '#1A1A1A',
  volcengine: '#FF4D4F',
  minimax: '#615CED',
  baichuan: '#0066FF',
  ernie: '#2932E1',
  google: '#4285F4',
  hunyuan: '#00A3FF',
  spark: '#E60012',
  microsoft: '#0078D4',
  azure: '#0078D4'
}

const FALLBACK_PALETTE = ['#5B8FF9', '#5AD8A6', '#5D7092', '#F6BD16', '#E8684A', '#6DC8EC', '#9270CA', '#FF9D4D']

function fallbackColorById(id: string): string {
  let hash = 0
  for (let i = 0; i < id.length; i += 1) {
    hash = (hash << 5) - hash + id.charCodeAt(i)
    hash |= 0
  }
  return FALLBACK_PALETTE[Math.abs(hash) % FALLBACK_PALETTE.length] || '#5B8FF9'
}

export const PROVIDER_CATALOG: ProviderCatalogItem[] = PROVIDER_IDS.map((id) => ({
  id,
  label: PROVIDER_LABELS[id] || id,
  color: PROVIDER_COLORS[id] || fallbackColorById(id),
  logo: `/logos/${id}.svg`
}))
