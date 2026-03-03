# AI Gateway 版本管理指南

## 版本说明

AI Gateway 支持三个版本，系统设置中可直接切换：

- 默认版本：`standard`

- 基础版（`basic`）
  - 纯网关能力
  - 依赖：Redis
- 标准版（`standard`）
  - 含基础版全部能力
  - 增加语义缓存（Ollama 向量能力）
  - 依赖：Redis + Ollama
- 企业版（`enterprise`）
  - 含标准版全部能力
  - 增加向量管理与知识库入口
  - 依赖：Redis + Ollama + Qdrant

## 切换方式

1. 打开系统设置 -> 版本管理。
2. 选择目标版本。
3. 检查依赖状态（未满足时不可切换）。
4. 点击“保存配置”。

后端接口：

- `GET /api/admin/edition`
- `PUT /api/admin/edition`
- `GET /api/admin/edition/definitions`
- `GET /api/admin/edition/dependencies`

## UI 可见性规则

- 基础版
  - 不显示 Ollama 管理菜单
  - 不显示向量管理入口
  - 不显示知识库入口
  - 缓存页不显示语义签名与向量索引区块
- 标准版
  - 显示侧边栏 `Ollama 管理`
  - 不显示向量管理入口
  - 不显示知识库入口
  - 显示缓存页语义签名与向量索引区块
- 企业版
  - 显示侧边栏 `Ollama 管理`
  - 显示向量管理入口
  - 显示知识库入口
  - 显示缓存页语义签名与向量索引区块
