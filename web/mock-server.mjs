import { createServer } from 'http';

const mockData = {
  // Providers List - 服务商列表
  '/api/providers': {
    list: [],
    total: 0
  },

  // Accounts List - 账号列表
  '/api/accounts': {
    list: [],
    total: 0
  },

  // Routing Rules - 路由规则
  '/api/routing/rules': {
    list: [],
    total: 0,
    globalStrategy: {
      loadBalance: 'weighted',
      failover: true,
      healthCheckInterval: 30,
      timeout: 30,
      retryCount: 3
    }
  },

  // Alert Rules - 告警规则
  '/api/alerts/rules': {
    list: [],
    total: 0
  },

  // Alerts History - 告警历史
  '/api/alerts/history': {
    list: [],
    total: 0
  },

  // Dashboard Stats - 概览数据
  '/api/dashboard/stats': {
    requests_today: 0,
    success_rate: 0,
    avg_latency_ms: 0,
    active_accounts: 0,
    active_providers: 0,
    cache_hit_rate: 0,
    top_models: []
  },
  '/api/admin/dashboard/stats': {
    requests_today: 0,
    success_rate: 0,
    avg_latency_ms: 0,
    active_accounts: 0,
    active_providers: 0,
    cache_hit_rate: 0,
    top_models: []
  },

  // Request Trend - 请求趋势
  '/api/dashboard/requests': {
    period: '24h',
    interval: 'hour',
    data: []
  },
  '/api/admin/dashboard/requests': {
    period: '24h',
    interval: 'hour',
    data: []
  },
  '/api/dashboard/trend': {
    period: '24h',
    interval: 'hour',
    data: []
  },

  // Providers - 服务商分布
  '/api/dashboard/providers': {
    distribution: {},
    list: [
      { id: 'openai', name: 'OpenAI', status: 'active', requests: 0, success_rate: 0, avg_latency: 0 },
      { id: 'anthropic', name: 'Anthropic', status: 'active', requests: 0, success_rate: 0, avg_latency: 0 },
      { id: 'azure', name: 'Azure OpenAI', status: 'active', requests: 0, success_rate: 0, avg_latency: 0 },
      { id: 'google', name: 'Google AI', status: 'active', requests: 0, success_rate: 0, avg_latency: 0 },
      { id: 'volcengine', name: '火山方舟', status: 'active', requests: 0, success_rate: 0, avg_latency: 0 },
      { id: 'qwen', name: '阿里云通义千问', status: 'active', requests: 0, success_rate: 0, avg_latency: 0 },
      { id: 'ernie', name: '百度文心一言', status: 'active', requests: 0, success_rate: 0, avg_latency: 0 },
      { id: 'zhipu', name: '智谱AI', status: 'active', requests: 0, success_rate: 0, avg_latency: 0 },
      { id: 'hunyuan', name: '腾讯混元', status: 'active', requests: 0, success_rate: 0, avg_latency: 0 },
      { id: 'moonshot', name: '月之暗面', status: 'active', requests: 0, success_rate: 0, avg_latency: 0 },
      { id: 'deepseek', name: 'DeepSeek', status: 'active', requests: 0, success_rate: 0, avg_latency: 0 }
    ]
  },
  '/api/admin/providers': {
    distribution: {},
    list: [
      { id: 'openai', name: 'OpenAI', status: 'active', requests: 0, success_rate: 0, avg_latency: 0 },
      { id: 'anthropic', name: 'Anthropic', status: 'active', requests: 0, success_rate: 0, avg_latency: 0 },
      { id: 'azure', name: 'Azure OpenAI', status: 'active', requests: 0, success_rate: 0, avg_latency: 0 },
      { id: 'google', name: 'Google AI', status: 'active', requests: 0, success_rate: 0, avg_latency: 0 },
      { id: 'volcengine', name: '火山方舟', status: 'active', requests: 0, success_rate: 0, avg_latency: 0 },
      { id: 'qwen', name: '阿里云通义千问', status: 'active', requests: 0, success_rate: 0, avg_latency: 0 },
      { id: 'ernie', name: '百度文心一言', status: 'active', requests: 0, success_rate: 0, avg_latency: 0 },
      { id: 'zhipu', name: '智谱AI', status: 'active', requests: 0, success_rate: 0, avg_latency: 0 },
      { id: 'hunyuan', name: '腾讯混元', status: 'active', requests: 0, success_rate: 0, avg_latency: 0 },
      { id: 'moonshot', name: '月之暗面', status: 'active', requests: 0, success_rate: 0, avg_latency: 0 },
      { id: 'deepseek', name: 'DeepSeek', status: 'active', requests: 0, success_rate: 0, avg_latency: 0 }
    ]
  },

  // System / Realtime - 实时系统数据
  '/api/dashboard/realtime': {
    timestamp: new Date().toISOString(),
    active_connections: 0,
    requests_per_minute: 0,
    tokens_per_minute: 0,
    avg_latency_ms: 0,
    error_rate: 0,
    top_models: [],
    recent_errors: []
  },
  '/api/dashboard/system': {
    timestamp: new Date().toISOString(),
    active_connections: 0,
    requests_per_minute: 0,
    tokens_per_minute: 0,
    avg_latency_ms: 0,
    error_rate: 0,
    top_models: [],
    recent_errors: []
  },
  '/api/admin/dashboard/system': {
    timestamp: new Date().toISOString(),
    active_connections: 0,
    requests_per_minute: 0,
    tokens_per_minute: 0,
    avg_latency_ms: 0,
    error_rate: 0,
    top_models: [],
    recent_errors: []
  },

  // Cache Stats - 缓存统计
  '/api/dashboard/cache': {
    request_cache: { hits: 0, misses: 0, hit_rate: 0, size: 0, max_size: 10000, evictions: 0 },
    context_cache: { hits: 0, misses: 0, hit_rate: 0, size: 0, max_size: 5000, evictions: 0 },
    route_cache: { hits: 0, misses: 0, hit_rate: 0, size: 0, max_size: 2000, evictions: 0 },
    usage_cache: { hits: 0, misses: 0, hit_rate: 0, size: 0, max_size: 8000, evictions: 0 },
    response_cache: { hits: 0, misses: 0, hit_rate: 0, size: 0, max_size: 15000, evictions: 0 },
    token_savings: 0
  },
  '/api/cache/stats': {
    request_cache: { hits: 0, misses: 0, hit_rate: 0 },
    context_cache: { hits: 0, misses: 0, hit_rate: 0 },
    route_cache: { hits: 0, misses: 0, hit_rate: 0 },
    usage_cache: { hits: 0, misses: 0, hit_rate: 0 },
    response_cache: { hits: 0, misses: 0, hit_rate: 0 }
  },
  '/api/admin/cache/stats': {
    request_cache: { hits: 0, misses: 0, hit_rate: 0 },
    context_cache: { hits: 0, misses: 0, hit_rate: 0 },
    route_cache: { hits: 0, misses: 0, hit_rate: 0 },
    usage_cache: { hits: 0, misses: 0, hit_rate: 0 },
    response_cache: { hits: 0, misses: 0, hit_rate: 0 }
  },

  // Usage - 用量统计
  '/api/dashboard/usage': {
    prompt_tokens: 0,
    completion_tokens: 0,
    total_tokens: 0,
    by_model: {}
  },
  '/api/admin/dashboard/usage': {
    prompt_tokens: 0,
    completion_tokens: 0,
    total_tokens: 0,
    by_model: {}
  }
};

const server = createServer((req, res) => {
  console.log(`${new Date().toISOString()} ${req.method} ${req.url}`);

  // 设置 CORS 头
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, OPTIONS');
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type, Authorization');

  if (req.method === 'OPTIONS') {
    res.writeHead(200);
    res.end();
    return;
  }

  // 模拟延迟
  setTimeout(() => {
    // 精确匹配
    let endpoint = Object.keys(mockData).find(key => req.url === key || req.url.startsWith(key + '?'));

    if (endpoint) {
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify(mockData[endpoint]));
    } else {
      // 处理动态路径和写操作
      const url = req.url.split('?')[0];

      // POST/PUT/DELETE 操作模拟成功响应
      if (req.method === 'POST' || req.method === 'PUT' || req.method === 'DELETE') {
        // 提取资源 ID 的模式匹配
        const providerMatch = url.match(/^\/api\/providers\/(\d+)(\/.*)?$/);
        const accountMatch = url.match(/^\/api\/accounts\/(\d+)(\/.*)?$/);
        const ruleMatch = url.match(/^\/api\/(routing|alerts)\/rules\/(\d+)$/);

        if (providerMatch || accountMatch || ruleMatch || url === '/api/providers' || url === '/api/accounts') {
          res.writeHead(200, { 'Content-Type': 'application/json' });
          res.end(JSON.stringify({ success: true, message: '操作成功' }));
          return;
        }

        // 状态切换
        if (url.includes('/status')) {
          res.writeHead(200, { 'Content-Type': 'application/json' });
          res.end(JSON.stringify({ success: true }));
          return;
        }

        // 测试连接
        if (url.includes('/test')) {
          res.writeHead(200, { 'Content-Type': 'application/json' });
          res.end(JSON.stringify({ success: true, latency: Math.floor(Math.random() * 200) + 50 }));
          return;
        }
      }

      // GET 请求的资源详情
      const detailMatch = url.match(/^\/api\/(providers|accounts|routing\/rules|alerts\/rules)\/(\d+)$/);
      if (detailMatch && req.method === 'GET') {
        const resource = detailMatch[1];
        const id = detailMatch[2];
        const listKey = `/api/${resource.replace('/', '/')}`;
        const data = mockData[listKey];
        if (data && data.list) {
          const item = data.list.find(i => i.id === parseInt(id));
          if (item) {
            res.writeHead(200, { 'Content-Type': 'application/json' });
            res.end(JSON.stringify(item));
            return;
          }
        }
      }

      console.log(`  -> 404 Not Found`);
      res.writeHead(404, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ error: 'Not Found', path: req.url }));
    }
  }, 50);
});

const PORT = 8081;
server.listen(PORT, () => {
  console.log(`Mock server running on http://localhost:${PORT}`);
  console.log(`Available endpoints:`);
  Object.keys(mockData).forEach(endpoint => console.log(`  ${endpoint}`));
});
