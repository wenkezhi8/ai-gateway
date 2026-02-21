const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');

console.log("=== AI Gateway 网站全面功能测试 ===\n");

// 1. 检查项目结构
console.log("1. 项目结构检查:");
const requiredDirs = ['src', 'src/components', 'src/views', 'src/api', 'src/styles'];
const requiredFiles = [
  'src/main.ts',
  'src/App.vue', 
  'src/router/index.ts',
  'src/components/Layout/index.vue',
  'src/views/dashboard/index.vue',
  'src/views/settings/index.vue',
  'src/views/login/index.vue'
];

let structureOk = true;
requiredDirs.forEach(dir => {
  if (fs.existsSync(dir)) {
    console.log(`  ✓ ${dir}`);
  } else {
    console.log(`  ✗ ${dir} (缺失)`);
    structureOk = false;
  }
});

requiredFiles.forEach(file => {
  if (fs.existsSync(file)) {
    console.log(`  ✓ ${file}`);
  } else {
    console.log(`  ✗ ${file} (缺失)`);
    structureOk = false;
  }
});

// 2. 检查路由配置
console.log("\n2. 路由配置检查:");
try {
  const routerContent = fs.readFileSync('src/router/index.ts', 'utf8');
  const routes = [
    { path: '/dashboard', name: 'Dashboard' },
    { path: '/providers', name: 'Providers' },
    { path: '/accounts', name: 'Accounts' },
    { path: '/routing', name: 'Routing' },
    { path: '/cache', name: 'Cache' },
    { path: '/alerts', name: 'Alerts' },
    { path: '/test-center', name: 'TestCenter' },
    { path: '/settings', name: 'Settings' },
    { path: '/login', name: 'Login' },
    { path: '/404', name: 'NotFound' }
  ];

  routes.forEach(route => {
    if (routerContent.includes(`path: '${route.path}'`) || routerContent.includes(`name: '${route.name}'`)) {
      console.log(`  ✓ ${route.path} (${route.name})`);
    } else {
      console.log(`  ✗ ${route.path} (${route.name}) - 路由缺失`);
    }
  });
} catch (error) {
  console.log(`  ✗ 无法读取路由文件: ${error.message}`);
}

// 3. 检查API端点
console.log("\n3. API端点检查:");
const apiEndpoints = [
  '/api/dashboard/stats',
  '/api/dashboard/trend',
  '/api/dashboard/providers',
  '/api/dashboard/realtime',
  '/api/admin/dashboard/system'
];

apiEndpoints.forEach(endpoint => {
  try {
    const result = execSync(`curl -s -o /dev/null -w "%{http_code}" "http://localhost:8566${endpoint}"`, { encoding: 'utf8' }).trim();
    if (result === '200') {
      console.log(`  ✓ ${endpoint} (HTTP ${result})`);
    } else {
      console.log(`  ⚠ ${endpoint} (HTTP ${result}) - 非200响应`);
    }
  } catch (error) {
    console.log(`  ✗ ${endpoint} - 请求失败: ${error.message}`);
  }
});

// 4. 检查页面组件
console.log("\n4. 页面组件检查:");
const pages = [
  { file: 'src/views/dashboard/index.vue', name: 'Dashboard' },
  { file: 'src/views/providers/index.vue', name: 'Providers' },
  { file: 'src/views/settings/index.vue', name: 'Settings' },
  { file: 'src/views/login/index.vue', name: 'Login' },
  { file: 'src/views/accounts/index.vue', name: 'Accounts' },
  { file: 'src/views/routing/index.vue', name: 'Routing' },
  { file: 'src/views/cache/index.vue', name: 'Cache' },
  { file: 'src/views/alerts/index.vue', name: 'Alerts' },
  { file: 'src/views/test-center/index.vue', name: 'TestCenter' },
  { file: 'src/views/error/404.vue', name: '404页面' }
];

pages.forEach(page => {
  if (fs.existsSync(page.file)) {
    try {
      const content = fs.readFileSync(page.file, 'utf8');
      const hasTemplate = content.includes('<template>');
      const hasScript = content.includes('<script') || content.includes('export');
      const hasStyle = content.includes('<style') || content.includes('lang="scss"');
      
      console.log(`  ✓ ${page.name}`);
      if (!hasTemplate) console.log(`    ⚠ 缺少template部分`);
      if (!hasScript) console.log(`    ⚠ 缺少script部分`);
      if (!hasStyle) console.log(`    ⚠ 缺少style部分`);
    } catch (error) {
      console.log(`  ⚠ ${page.name} - 读取失败: ${error.message}`);
    }
  } else {
    console.log(`  ✗ ${page.name} - 文件不存在`);
  }
});

// 5. 检查JavaScript错误
console.log("\n5. JavaScript代码检查:");
const jsFiles = [
  'src/api/request.ts',
  'src/api/metrics.ts',
  'src/api/provider.ts',
  'src/composables/useTheme.ts',
  'src/utils/errorHandler.ts'
];

jsFiles.forEach(file => {
  if (fs.existsSync(file)) {
    try {
      const content = fs.readFileSync(file, 'utf8');
      // 检查常见的代码问题
      const issues = [];
      
      if (content.includes('console.log(') && !content.includes('// console.log')) {
        issues.push('存在console.log语句（生产环境应移除）');
      }
      
      if (content.includes('any)')) {
        issues.push('使用any类型（应使用具体类型）');
      }
      
      if (content.includes('catch (error) {') && !content.includes('error: any')) {
        issues.push('catch块未指定错误类型');
      }
      
      if (content.includes('setTimeout(') && !content.includes('clearTimeout')) {
        issues.push('使用setTimeout但未清理');
      }
      
      if (issues.length === 0) {
        console.log(`  ✓ ${file}`);
      } else {
        console.log(`  ⚠ ${file}`);
        issues.forEach(issue => console.log(`    - ${issue}`));
      }
    } catch (error) {
      console.log(`  ⚠ ${file} - 读取失败`);
    }
  } else {
    console.log(`  ⚠ ${file} - 文件不存在`);
  }
});

// 6. 检查CSS/样式问题
console.log("\n6. 样式文件检查:");
const styleFiles = [
  'src/styles/variables.scss',
  'src/styles/index.scss',
  'src/styles/apple.scss'
];

styleFiles.forEach(file => {
  if (fs.existsSync(file)) {
    try {
      const content = fs.readFileSync(file, 'utf8');
      const issues = [];
      
      if (content.includes('@import') && file.endsWith('.scss')) {
        issues.push('使用已弃用的@import语法（应使用@use）');
      }
      
      if (content.includes('!important')) {
        issues.push('使用!important（应避免）');
      }
      
      if (content.includes('px') && !content.includes('calc') && !content.includes('border-radius')) {
        issues.push('使用固定像素值（应考虑使用rem/em）');
      }
      
      if (issues.length === 0) {
        console.log(`  ✓ ${file}`);
      } else {
        console.log(`  ⚠ ${file}`);
        issues.forEach(issue => console.log(`    - ${issue}`));
      }
    } catch (error) {
      console.log(`  ⚠ ${file} - 读取失败`);
    }
  } else {
    console.log(`  ⚠ ${file} - 文件不存在`);
  }
});

// 7. 检查依赖包
console.log("\n7. 依赖包检查:");
try {
  const packageJson = JSON.parse(fs.readFileSync('package.json', 'utf8'));
  
  console.log(`  ✓ Vue: ${packageJson.dependencies?.vue || '未找到'}`);
  console.log(`  ✓ Element Plus: ${packageJson.dependencies?.['element-plus'] || '未找到'}`);
  console.log(`  ✓ Vue Router: ${packageJson.dependencies?.['vue-router'] || '未找到'}`);
  console.log(`  ✓ ECharts: ${packageJson.dependencies?.echarts || '未找到'}`);
  console.log(`  ✓ Axios: ${packageJson.dependencies?.axios || '未找到'}`);
  
  // 检查是否有安全漏洞的版本
  const vulnerableVersions = {
    'axios': ['<1.0.0'],
    'vue': ['<3.0.0']
  };
  
  Object.entries(vulnerableVersions).forEach(([pkg, versions]) => {
    if (packageJson.dependencies?.[pkg]) {
      console.log(`    ⚠ ${pkg}: ${packageJson.dependencies[pkg]} - 检查版本安全性`);
    }
  });
} catch (error) {
  console.log(`  ✗ 无法读取package.json: ${error.message}`);
}

// 8. 检查构建配置
console.log("\n8. 构建配置检查:");
const configFiles = ['vite.config.ts', 'tsconfig.json', 'tsconfig.app.json', 'tsconfig.node.json'];

configFiles.forEach(file => {
  if (fs.existsSync(file)) {
    try {
      const content = fs.readFileSync(file, 'utf8');
      console.log(`  ✓ ${file}`);
      
      if (file === 'vite.config.ts') {
        if (!content.includes('port:')) {
          console.log(`    ⚠ 未指定端口配置`);
        }
        if (!content.includes('proxy:')) {
          console.log(`    ⚠ 未配置API代理`);
        }
      }
    } catch (error) {
      console.log(`  ⚠ ${file} - 读取失败`);
    }
  } else {
    console.log(`  ⚠ ${file} - 文件不存在`);
  }
});

console.log("\n=== 测试完成 ===");
console.log("\n建议的手动测试:");
console.log("1. 访问所有页面，检查路由是否正确");
console.log("2. 测试侧边栏折叠/展开功能");
console.log("3. 测试主题切换功能");
console.log("4. 测试表单提交和验证");
console.log("5. 测试对话框打开/关闭");
console.log("6. 测试移动端响应式布局");
console.log("7. 测试浏览器前进/后退导航");
console.log("8. 测试API错误处理");
console.log("9. 测试加载状态和骨架屏");
console.log("10. 测试图表交互和响应式");
