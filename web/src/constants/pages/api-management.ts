export type ApiManagementStrategy = {
  value: 'auto' | 'quality' | 'speed' | 'cost' | 'custom'
  label: string
  description: string
}

export const API_MANAGEMENT_STRATEGIES: ApiManagementStrategy[] = [
  { value: 'auto', label: '智能平衡', description: '效果+速度+成本综合最优' },
  { value: 'quality', label: '效果优先', description: '优先选择效果最好的模型' },
  { value: 'speed', label: '速度优先', description: '优先选择响应最快的模型' },
  { value: 'cost', label: '成本优先', description: '优先选择成本最低的模型' },
  { value: 'custom', label: '自定义规则', description: '根据任务类型自动选择' }
]
