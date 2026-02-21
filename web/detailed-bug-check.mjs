import fs from 'fs';

console.log("=== AI Gateway 详细BUG检查 ===\n");

// 检查常见的代码问题
console.log("1. 检查未处理的Promise:");
const filesToCheck = [
  'src/views/dashboard/index.vue',
  'src/views/providers/index.vue',
  'src/views/settings/index.vue',
  'src/views/login/index.vue'
];

filesToCheck.forEach(file => {
  if (fs.existsSync(file)) {
    try {
      const content = fs.readFileSync(file, 'utf8');
      const lines = content.split('\n');
      let hasUnhandledPromise = false;
      
      lines.forEach((line, index) => {
        if (line.includes('await') && !line.includes('try') && !line.includes('catch')) {
          // 检查前几行是否有try
          const prevLines = lines.slice(Math.max(0, index - 3), index);
          const hasTryInPrevLines = prevLines.some(l => l.includes('try {'));
          
          if (!hasTryInPrevLines) {
            console.log(`  ⚠ ${file}:${index + 1} - 未处理的Promise: ${line.trim()}`);
            hasUnhandledPromise = true;
          }
        }
      });
      
      if (!hasUnhandledPromise) {
        console.log(`  ✓ ${file} - 无未处理Promise`);
      }
    } catch (error) {
      console.log(`  ⚠ ${file} - 读取失败`);
    }
  }
});

console.log("\n2. 检查未定义的变量:");
const checkUndefinedVariables = (content, fileName) => {
  const issues = [];
  const lines = content.split('\n');
  
  // 检查常见的未定义变量模式
  lines.forEach((line, index) => {
    if (line.includes('this.') && line.includes('=') && !line.includes('const') && !line.includes('let') && !line.includes('var')) {
      issues.push(`第${index + 1}行: 可能使用未定义的this属性`);
    }
    
    if (line.includes('window.') && !line.includes('window.location') && !line.includes('window.localStorage')) {
      issues.push(`第${index + 1}行: 直接访问window对象属性`);
    }
  });
  
  return issues;
};

filesToCheck.forEach(file => {
  if (fs.existsSync(file)) {
    try {
      const content = fs.readFileSync(file, 'utf8');
      const issues = checkUndefinedVariables(content, file);
      
      if (issues.length === 0) {
        console.log(`  ✓ ${file} - 无未定义变量问题`);
      } else {
        console.log(`  ⚠ ${file}:`);
        issues.forEach(issue => console.log(`    - ${issue}`));
      }
    } catch (error) {
      console.log(`  ⚠ ${file} - 读取失败`);
    }
  }
});

console.log("\n3. 检查内存泄漏问题:");
const checkMemoryLeaks = (content, fileName) => {
  const issues = [];
  const lines = content.split('\n');
  
  lines.forEach((line, index) => {
    if (line.includes('addEventListener') && !line.includes('removeEventListener')) {
      issues.push(`第${index + 1}行: 添加事件监听器但未移除`);
    }
    
    if (line.includes('setInterval') && !line.includes('clearInterval')) {
      issues.push(`第${index + 1}行: 设置定时器但未清理`);
    }
    
    if (line.includes('setTimeout') && line.includes('=>') && !line.includes('clearTimeout')) {
      issues.push(`第${index + 1}行: 设置超时但未清理`);
    }
  });
  
  return issues;
};

filesToCheck.forEach(file => {
  if (fs.existsSync(file)) {
    try {
      const content = fs.readFileSync(file, 'utf8');
      const issues = checkMemoryLeaks(content, file);
      
      if (issues.length === 0) {
        console.log(`  ✓ ${file} - 无内存泄漏问题`);
      } else {
        console.log(`  ⚠ ${file}:`);
        issues.forEach(issue => console.log(`    - ${issue}`));
      }
    } catch (error) {
      console.log(`  ⚠ ${file} - 读取失败`);
    }
  }
});

console.log("\n4. 检查表单验证问题:");
const checkFormValidation = () => {
  console.log("  检查登录页面表单验证:");
  try {
    const loginContent = fs.readFileSync('src/views/login/index.vue', 'utf8');
    
    if (loginContent.includes('loginRules')) {
      console.log(`    ✓ 登录页面有表单验证规则`);
    } else {
      console.log(`    ⚠ 登录页面缺少表单验证规则`);
    }
    
    if (loginContent.includes('required: true')) {
      console.log(`    ✓ 登录页面有必填字段验证`);
    } else {
      console.log(`    ⚠ 登录页面缺少必填字段验证`);
    }
  } catch (error) {
    console.log(`    ⚠ 无法检查登录页面: ${error.message}`);
  }
  
  console.log("\n  检查Providers页面表单验证:");
  try {
    const providersContent = fs.readFileSync('src/views/providers/index.vue', 'utf8');
    
    if (providersContent.includes('formRules') || providersContent.includes('rules')) {
      console.log(`    ✓ Providers页面有表单验证规则`);
    } else {
      console.log(`    ⚠ Providers页面缺少表单验证规则`);
    }
  } catch (error) {
    console.log(`    ⚠ 无法检查Providers页面: ${error.message}`);
  }
};

checkFormValidation();

console.log("\n5. 检查响应式数据问题:");
const checkReactiveData = (content, fileName) => {
  const issues = [];
  const lines = content.split('\n');
  
  lines.forEach((line, index) => {
    // 检查直接修改props
    if (line.includes('props.') && line.includes('=')) {
      issues.push(`第${index + 1}行: 可能直接修改props`);
    }
    
    // 检查在setup外使用ref
    if (line.includes('ref(') && !line.includes('const') && !line.includes('let') && !line.includes('var')) {
      issues.push(`第${index + 1}行: 可能错误使用ref`);
    }
  });
  
  return issues;
};

filesToCheck.forEach(file => {
  if (fs.existsSync(file)) {
    try {
      const content = fs.readFileSync(file, 'utf8');
      const issues = checkReactiveData(content, file);
      
      if (issues.length === 0) {
        console.log(`  ✓ ${file} - 无响应式数据问题`);
      } else {
        console.log(`  ⚠ ${file}:`);
        issues.forEach(issue => console.log(`    - ${issue}`));
      }
    } catch (error) {
      console.log(`  ⚠ ${file} - 读取失败`);
    }
  }
});

console.log("\n6. 检查类型安全问题:");
const checkTypeSafety = () => {
  console.log("  检查TypeScript类型使用:");
  
  const tsFiles = [
    'src/api/metrics.ts',
    'src/api/provider.ts',
    'src/composables/useTheme.ts'
  ];
  
  tsFiles.forEach(file => {
    if (fs.existsSync(file)) {
      try {
        const content = fs.readFileSync(file, 'utf8');
        const anyCount = (content.match(/any/g) || []).length;
        
        if (anyCount === 0) {
          console.log(`    ✓ ${file} - 无any类型使用`);
        } else {
          console.log(`    ⚠ ${file} - 使用${anyCount}次any类型`);
        }
      } catch (error) {
        console.log(`    ⚠ ${file} - 读取失败`);
      }
    }
  });
};

checkTypeSafety();

console.log("\n=== BUG检查完成 ===");
