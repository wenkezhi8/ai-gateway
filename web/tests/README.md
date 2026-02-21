# AI Gateway Browser Automation Testing Framework

这是一个基于 Playwright 的完整浏览器自动化测试框架，专为测试 AI Gateway Web 管理后台而设计。

## 🚀 功能特性

### 核心测试场景
- ✅ **页面加载测试** - 验证所有页面的正确加载和渲染
- ✅ **导航跳转测试** - 测试页面间导航和路由功能
- ✅ **登录/登出测试** - 完整的身份认证流程测试
- ✅ **表单提交测试** - 服务商配置、账号管理等表单功能
- ✅ **API密钥管理** - 密钥生成、显示、隐藏、重新生成功能
- ✅ **流量查询/搜索** - 数据查询、筛选、搜索功能
- ✅ **异步加载测试** - 下拉刷新、加载更多、分页功能
- ✅ **配置导入/导出** - 系统配置的导入导出功能
- ✅ **弹窗交互测试** - 模态框、确认对话框等交互元素

### 性能监控
- ⏱️ **耗时埋点** - 每个测试步骤自动记录执行时间
- 📊 **性能分析** - 自动标记耗时超过3秒的操作
- 📈 **响应速度统计** - 生成详细的性能报告

### 错误捕获
- 🚨 **404/500错误** - 自动检测和记录HTTP错误
- 🔍 **资源加载失败** - 静态资源加载问题监控
- ❌ **元素交互失败** - 点击无响应、元素不存在等问题
- 📝 **断言失败** - 测试断言错误完整记录

### 测试报告
- 📊 **结构化结果** - 区分通过/失败/慢响应项
- 🔍 **详细复现步骤** - 失败测试提供完整重现步骤
- 🌐 **HTML报告** - 美观的网页版测试报告
- 📄 **JSON格式** - 机器可读的测试结果数据

## 📁 项目结构

```
tests/
├── utils/
│   ├── test-helper.ts          # 测试辅助工具类
│   └── custom-reporter.ts      # 自定义报告生成器
├── page-objects/
│   ├── login-page.ts          # 登录页面对象
│   ├── dashboard-page.ts      # 仪表盘页面对象
│   ├── providers-page.ts      # 服务商管理页面对象
│   └── accounts-page.ts        # 账号管理页面对象
├── scenarios/
│   ├── auth.spec.ts            # 认证相关测试
│   ├── dashboard.spec.ts       # 仪表盘功能测试
│   ├── providers.spec.ts       # 服务商管理测试
│   ├── accounts.spec.ts        # 账号管理测试
│   ├── navigation.spec.ts      # 导航功能测试
│   ├── data-loading.spec.ts    # 数据加载测试
│   ├── performance-errors.spec.ts # 性能和错误测试
│   └── e2e.spec.ts             # 端到端集成测试
└── results/                    # 测试结果目录
    ├── html-report/            # HTML测试报告
    ├── screenshots/            # 截图文件
    └── test-results.json       # JSON格式结果
```

## 🛠️ 安装和使用

### 1. 安装依赖
```bash
npm install
```

### 2. 安装 Playwright 浏览器
```bash
npm run test:install
```

### 3. 运行测试

#### 运行所有测试
```bash
npm run test
```

#### 运行特定测试场景
```bash
npm run test:auth          # 认证测试
npm run test:dashboard     # 仪表盘测试
npm run test:providers     # 服务商管理测试
npm run test:accounts      # 账号管理测试
npm run test:navigation    # 导航测试
npm run test:data-loading  # 数据加载测试
npm run test:performance   # 性能测试
npm run test:e2e           # 端到端测试
```

#### 有界面运行测试
```bash
npm run test:headed
```

#### 调试模式运行
```bash
npm run test:debug
```

#### 查看测试报告
```bash
npm run test:report
```

## 📊 测试配置

### Playwright 配置 (`playwright.config.ts`)
- **基础URL**: `http://localhost:8566`
- **浏览器支持**: Chromium, Firefox, Safari
- **超时设置**: 30秒测试超时, 10秒操作超时
- **报告格式**: HTML, JSON, 控制台输出
- **截图/视频**: 失败时自动截图和录制

### 性能阈值
- **慢操作阈值**: 3000毫秒 (3秒)
- **页面加载超时**: 15000毫秒 (15秒)
- **操作超时**: 10000毫秒 (10秒)

## 🎯 测试覆盖范围

### 认证测试 (auth.spec.ts)
- 登录页面正确显示
- 有效凭据登录成功
- 无效凭据显示错误信息
- 空表单提交处理
- 登出功能正常
- 未认证访问保护路由重定向
- 网络错误处理

### 仪表盘测试 (dashboard.spec.ts)
- 仪表盘页面正确加载
- 所有组件正确显示
- 导航到各个功能模块
- 实时数据更新
- 页面刷新功能
- 响应式布局
- 快捷操作功能
- 系统状态指示器

### 服务商管理测试 (providers.spec.ts)
- 页面正确加载
- 添加新服务商
- 编辑现有服务商
- 删除服务商
- 搜索功能
- 表单验证
- 网络错误处理
- 状态显示
- 分页功能
- 导入导出配置

### 账号管理测试 (accounts.spec.ts)
- 页面正确加载
- 添加新账号
- API密钥安全管理
- 重新生成API密钥
- 编辑账号信息
- 删除账号
- 搜索账号
- 表单验证
- 权限和角色管理

### 导航测试 (navigation.spec.ts)
- 主要模块间导航
- 浏览器前进后退
- 直接URL访问
- 无效路由处理
- 页面刷新状态保持
- 查询参数处理
- 键盘导航
- 加载状态导航
- 面包屑显示

### 数据加载测试 (data-loading.spec.ts)
- 无限滚动加载
- 下拉刷新功能
- 懒加载内容
- 实时数据更新
- 分页功能
- 搜索防抖
- 模态对话框
- 下拉菜单
- 加载状态显示
- 导入导出操作

### 性能和错误测试 (performance-errors.spec.ts)
- 页面加载性能监控
- 404错误处理
- 500服务器错误处理
- 网络超时处理
- 资源加载性能
- JavaScript错误处理
- 内存使用监控
- 浏览器兼容性
- 并发操作

### 端到端集成测试 (e2e.spec.ts)
- 完整用户工作流
- 跨页面数据流
- 所有交互组件测试
- 响应式设计测试
- 可访问性功能
- 错误恢复场景

## 📈 性能监控

### 自动性能指标
- **页面加载时间** - 每个页面的完整加载时间
- **操作响应时间** - 用户交互的响应速度
- **API调用耗时** - 后端接口调用时间
- **资源加载时间** - 静态资源加载性能

### 慢操作自动标记
超过3秒的操作会被自动标记为"慢响应"，包括：
- 页面加载过慢
- 表单提交响应慢
- 数据查询超时
- 网络请求延迟

## 🚨 错误捕获

### 自动错误检测
- **HTTP错误** - 404、500等状态码自动记录
- **资源加载失败** - CSS、JS、图片等资源加载问题
- **JavaScript错误** - 运行时脚本错误自动捕获
- **断言失败** - 测试断言不通过时完整记录

### 错误处理策略
- **自动截图** - 失败时自动保存完整页面截图
- **错误上下文** - 记录错误发生时的完整上下文信息
- **重试机制** - 网络相关错误支持自动重试
- **优雅降级** - 部分功能失败不影响整体测试

## 📋 测试报告

### 控制台输出
```
🧪 TEST EXECUTION REPORT
================================================================================

📊 SUMMARY:
   Total Tests: 45
   ✅ Passed: 42
   ❌ Failed: 2
   ⏭️  Skipped: 1
   ⏱️  Slow Operations (>3s): 3
   ⏰ Total Duration: 2.34s

❌ FAILED TESTS:
   ❌ Login with invalid credentials (1.23s)
      Error: Invalid username or password
      File: tests/scenarios/auth.spec.ts

⏱️  SLOW OPERATIONS:
   ⏱️  Dashboard - Load dashboard: 3.45s
   ⏱️  Providers - Add new provider: 3.12s
   ⏱️  Data loading - Test infinite scroll: 3.78s

================================================================================
🎉 42 tests passed. Success Rate: 93.3%
================================================================================
```

### HTML报告
- 🎨 **美观界面** - 现代化的报告设计
- 📱 **响应式** - 支持移动端查看
- 🔍 **详细信息** - 每个测试的详细结果
- 📊 **图表统计** - 可视化的测试统计数据
- 🖼️ **截图预览** - 失败测试的截图展示

## 🎛️ 高级配置

### 环境变量
```bash
# 设置测试基础URL
BASE_URL=http://localhost:8566

# 设置浏览器类型
BROWSER=chromium  # chromium, firefox, webkit

# 设置测试超时
TEST_TIMEOUT=30000

# 设置慢操作阈值
SLOW_THRESHOLD=3000
```

### 自定义配置
可以在 `playwright.config.ts` 中修改配置：
- 修改浏览器支持
- 调整超时设置
- 自定义报告格式
- 配置CI/CD集成

## 🔄 CI/CD 集成

### GitHub Actions 示例
```yaml
name: Playwright Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-node@v2
        with:
          node-version: '18'
      - run: npm install
      - run: npm run test:install
      - run: npm run test
      - uses: actions/upload-artifact@v2
        if: failure()
        with:
          name: playwright-report
          path: tests/results/
```

## 🛠️ 开发指南

### 添加新的测试用例
1. 在 `tests/scenarios/` 目录下创建新的测试文件
2. 使用 `test.describe()` 组织测试用例
3. 使用 `helper.measurePerformance()` 包装需要监控的操作
4. 添加适当的断言和错误处理

### 添加新的页面对象
1. 在 `tests/page-objects/` 目录下创建新的页面对象文件
2. 定义页面元素定位器
3. 实现页面操作方法
4. 在测试用例中使用页面对象

### 自定义报告器
可以在 `tests/utils/custom-reporter.ts` 中自定义报告格式：
- 修改报告样式
- 添加新的指标
- 集成外部服务
- 自定义通知机制

## 📚 最佳实践

### 测试编写
- ✅ 使用页面对象模式
- ✅ 添加性能监控点
- ✅ 包含错误处理测试
- ✅ 使用有意义的测试名称
- ✅ 添加适当的等待和断言

### 性能优化
- ⚡ 使用并行测试执行
- ⚡ 合理设置超时时间
- ⚡ 避免不必要的等待
- ⚡ 复用浏览器实例
- ⚡ 优化选择器性能

### 维护建议
- 🔧 定期更新测试用例
- 🔧 监控测试执行时间
- 🔧 定期清理测试结果
- 🔧 保持测试独立性
- 🔧 文档化复杂场景

## 🤝 贡献指南

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/new-test`)
3. 提交更改 (`git commit -am 'Add new test scenario'`)
4. 推送到分支 (`git push origin feature/new-test`)
5. 创建 Pull Request

## 📄 许可证

本项目采用 MIT 许可证。详见 [LICENSE](LICENSE) 文件。

---

**AI Gateway Testing Framework** - 让Web应用测试更简单、更可靠！ 🚀