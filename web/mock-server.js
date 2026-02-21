const http = require('http');

const mockData = {
  '/api/dashboard/stats': {
    totalRequests: 12345,
    successRate: 98.5,
    avgLatency: 245,
    activeProviders: 3
  },
  '/api/dashboard/trend': {
    period: '24h',
    interval: 'hour',
    data: Array.from({length: 24}, (_, i) => ({
      timestamp: new Date(Date.now() - (23-i)*3600000).toISOString(),
      requests: Math.floor(Math.random() * 100) + 50,
      success: Math.floor(Math.random() * 95) + 5,
      latency: Math.floor(Math.random() * 200) + 100
    }))
  },
  '/api/dashboard/providers': [
    { name: 'OpenAI', requests: 6543, successRate: 99.2, avgLatency: 210 },
    { name: 'Anthropic', requests: 4321, successRate: 97.8, avgLatency: 280 },
    { name: 'Azure OpenAI', requests: 1481, successRate: 96.5, avgLatency: 320 }
  ]
};

const server = http.createServer((req, res) => {
  console.log(`${new Date().toISOString()} ${req.method} ${req.url}`);
  
  // 设置 CORS 头
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Methods', 'GET, POST, OPTIONS');
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type, Authorization');
  
  if (req.method === 'OPTIONS') {
    res.writeHead(200);
    res.end();
    return;
  }
  
  // 模拟延迟
  setTimeout(() => {
    const endpoint = Object.keys(mockData).find(key => req.url.startsWith(key));
    
    if (endpoint) {
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify(mockData[endpoint]));
    } else {
      res.writeHead(404, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ error: 'Not Found' }));
    }
  }, 100);
});

const PORT = 8081;
server.listen(PORT, () => {
  console.log(`Mock server running on http://localhost:${PORT}`);
});
