# 向量模块权限管理手册

## 角色
- `admin`: 全量管理
- `editor`: 导入与运维操作
- `viewer`: 只读检索

## 管理入口
- 权限列表：`GET /api/admin/vector-db/permissions`
- 新增权限：`POST /api/admin/vector-db/permissions`
- 删除权限：`DELETE /api/admin/vector-db/permissions/:id`

## 最小权限建议
- 检索应用：`viewer`
- 导入机器人：`editor`
- 平台运维：`admin`
