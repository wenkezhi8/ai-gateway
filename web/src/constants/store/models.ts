export const MODEL_FORM_DEFAULTS = {
  type: 'chat' as const,
  enabled: true,
  maxTokens: 4096,
  inputPrice: 0,
  outputPrice: 0
}

export const MODEL_TYPE_LABELS: Record<string, string> = {
  chat: '对话',
  completion: '补全',
  embedding: '向量',
  image: '图像'
}
