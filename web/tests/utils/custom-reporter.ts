import type { FullConfig, FullResult, TestCase, TestResult } from '@playwright/test/reporter';
import * as fs from 'fs';
import * as path from 'path';

interface TestReport {
  summary: {
    total: number;
    passed: number;
    failed: number;
    skipped: number;
    slowOperations: number;
    totalDuration: number;
  };
  tests: TestDetail[];
  performance: PerformanceDetail[];
  errors: ErrorDetail[];
}

interface TestDetail {
  title: string;
  status: 'passed' | 'failed' | 'skipped';
  duration: number;
  file: string;
  error?: string;
  retry?: number;
}

interface PerformanceDetail {
  testName: string;
  action: string;
  duration: number;
  url: string;
  status: 'pass' | 'fail' | 'slow';
  error?: string;
}

interface ErrorDetail {
  testName: string;
  action: string;
  error: string;
  url: string;
  screenshot?: string;
}

class CustomReporter {
  private report: TestReport = {
    summary: {
      total: 0,
      passed: 0,
      failed: 0,
      skipped: 0,
      slowOperations: 0,
      totalDuration: 0
    },
    tests: [],
    performance: [],
    errors: []
  };

  async onBegin(config: FullConfig, result: FullResult) {
    console.log('Starting test execution...');
    console.log(`Test files: ${config.projects?.[0]?.testDir ? 'configured' : 'default'}`);
  }

  async onTestBegin(test: TestCase, result: TestResult) {
    console.log(`Starting test: ${test.title}`);
  }

  async onTestEnd(test: TestCase, result: TestResult) {
    this.report.summary.total++;
    this.report.summary.totalDuration += result.duration;

    const testDetail: TestDetail = {
      title: test.title,
      status: result.status as 'passed' | 'failed' | 'skipped',
      duration: result.duration,
      file: test.location.file,
      retry: result.retry
    };

    if (result.status === 'passed') {
      this.report.summary.passed++;
    } else if (result.status === 'failed') {
      this.report.summary.failed++;
      testDetail.error = result.error?.message;
    } else if (result.status === 'skipped') {
      this.report.summary.skipped++;
    }

    this.report.tests.push(testDetail);

    // Extract performance data from test annotations
    const perfData = result.annotations?.filter(a => a.type === 'performance');
    if (perfData) {
      perfData.forEach(perf => {
        const perfDetail: PerformanceDetail = {
          testName: test.title,
          action: perf.description || '',
          duration: parseInt(perf.description?.split(':')[1] || '0'),
          url: test.titlePath().join('/') || '',
          status: 'pass'
        };
        
        if (perfDetail.duration > 3000) {
          perfDetail.status = 'slow';
          this.report.summary.slowOperations++;
        }
        
        this.report.performance.push(perfDetail);
      });
    }

    if (result.status === 'failed' && result.error) {
      const errorDetail: ErrorDetail = {
        testName: test.title,
        action: 'Test execution',
        error: result.error?.message || '',
        url: test.titlePath().join('/') || '',
        screenshot: result.attachments?.find(a => a.name.endsWith('png'))?.path
      };
      this.report.errors.push(errorDetail);
    }
  }

  async onEnd(result: FullResult) {
    this.generateReport();
    this.saveReport();
  }

  private generateReport() {
    console.log('\n' + '='.repeat(80));
    console.log('🧪 TEST EXECUTION REPORT');
    console.log('='.repeat(80));
    
    const { summary } = this.report;
    console.log(`\n📊 SUMMARY:`);
    console.log(`   Total Tests: ${summary.total}`);
    console.log(`   ✅ Passed: ${summary.passed}`);
    console.log(`   ❌ Failed: ${summary.failed}`);
    console.log(`   ⏭️  Skipped: ${summary.skipped}`);
    console.log(`   ⏱️  Slow Operations (>3s): ${summary.slowOperations}`);
    console.log(`   ⏰ Total Duration: ${(summary.totalDuration / 1000).toFixed(2)}s`);

    if (summary.failed > 0) {
      console.log(`\n❌ FAILED TESTS:`);
      this.report.tests.filter(t => t.status === 'failed').forEach(test => {
        console.log(`   ❌ ${test.title} (${(test.duration / 1000).toFixed(2)}s)`);
        console.log(`      Error: ${test.error}`);
        console.log(`      File: ${test.file}`);
      });
    }

    if (summary.slowOperations > 0) {
      console.log(`\n⏱️  SLOW OPERATIONS:`);
      this.report.performance.filter(p => p.status === 'slow').forEach(perf => {
        console.log(`   ⏱️  ${perf.testName} - ${perf.action}: ${(perf.duration / 1000).toFixed(2)}s`);
      });
    }

    console.log('\n' + '='.repeat(80));
    
    const successRate = ((summary.passed / summary.total) * 100).toFixed(1);
    if (summary.failed === 0) {
      console.log(`🎉 ALL TESTS PASSED! Success Rate: ${successRate}%`);
    } else {
      console.log(`⚠️  ${summary.failed} test(s) failed. Success Rate: ${successRate}%`);
    }
    console.log('='.repeat(80) + '\n');
  }

  private saveReport() {
    const reportDir = path.join(process.cwd(), 'tests', 'results');
    if (!fs.existsSync(reportDir)) {
      fs.mkdirSync(reportDir, { recursive: true });
    }

    const reportPath = path.join(reportDir, `test-report-${Date.now()}.json`);
    fs.writeFileSync(reportPath, JSON.stringify(this.report, null, 2));
    
    const htmlPath = path.join(reportDir, 'test-report-latest.html');
    this.generateHtmlReport(htmlPath);
    
    console.log(`\n📁 Reports saved:`);
    console.log(`   📊 JSON: ${reportPath}`);
    console.log(`   🌐 HTML: ${htmlPath}`);
  }

  private generateHtmlReport(outputPath: string) {
    const html = `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>AI Gateway Test Report</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; margin: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 30px; border-radius: 8px 8px 0 0; }
        .summary { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 20px; margin: 20px; }
        .summary-card { background: white; padding: 20px; border-radius: 8px; text-align: center; box-shadow: 0 2px 8px rgba(0,0,0,0.1); }
        .summary-card h3 { margin: 0; color: #333; font-size: 24px; }
        .summary-card p { margin: 5px 0; color: #666; }
        .content { padding: 0 20px 20px; }
        .test-item { border-left: 4px solid #ddd; padding: 15px; margin: 10px 0; background: #f9f9f9; border-radius: 0 4px 4px 0; }
        .test-passed { border-left-color: #4caf50; }
        .test-failed { border-left-color: #f44336; }
        .test-skipped { border-left-color: #ff9800; }
        .error { background: #ffebee; color: #c62828; padding: 10px; border-radius: 4px; margin: 5px 0; }
        .slow { background: #fff3e0; color: #e65100; padding: 10px; border-radius: 4px; margin: 5px 0; }
        .performance { background: #e8f5e8; padding: 15px; border-radius: 8px; margin: 10px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🧪 AI Gateway Test Report</h1>
            <p>Generated on ${new Date().toLocaleString('zh-CN')}</p>
        </div>
        
        <div class="summary">
            <div class="summary-card">
                <h3>${this.report.summary.total}</h3>
                <p>Total Tests</p>
            </div>
            <div class="summary-card">
                <h3 style="color: #4caf50;">${this.report.summary.passed}</h3>
                <p>Passed</p>
            </div>
            <div class="summary-card">
                <h3 style="color: #f44336;">${this.report.summary.failed}</h3>
                <p>Failed</p>
            </div>
            <div class="summary-card">
                <h3 style="color: #ff9800;">${this.report.summary.skipped}</h3>
                <p>Skipped</p>
            </div>
            <div class="summary-card">
                <h3 style="color: #e65100;">${this.report.summary.slowOperations}</h3>
                <p>Slow Operations</p>
            </div>
        </div>
        
        <div class="content">
            ${this.report.summary.failed > 0 ? `
            <h2>❌ Failed Tests</h2>
            ${this.report.tests.filter(t => t.status === 'failed').map(test => `
                <div class="test-item test-failed">
                    <h4>${test.title}</h4>
                    <p>Duration: ${(test.duration / 1000).toFixed(2)}s</p>
                    <p>File: ${test.file}</p>
                    ${test.error ? `<div class="error">${test.error}</div>` : ''}
                </div>
            `).join('')}
            ` : ''}
            
            ${this.report.summary.slowOperations > 0 ? `
            <h2>⏱️ Slow Operations (>3s)</h2>
            ${this.report.performance.filter(p => p.status === 'slow').map(perf => `
                <div class="slow">
                    <strong>${perf.testName}</strong> - ${perf.action}: ${(perf.duration / 1000).toFixed(2)}s
                </div>
            `).join('')}
            ` : ''}
            
            <h2>📊 All Test Results</h2>
            ${this.report.tests.map(test => `
                <div class="test-item test-${test.status}">
                    <h4>${test.title}</h4>
                    <p>Duration: ${(test.duration / 1000).toFixed(2)}s | Status: ${test.status.toUpperCase()}</p>
                </div>
            `).join('')}
        </div>
    </div>
</body>
</html>`;

    fs.writeFileSync(outputPath, html);
  }
}

export default CustomReporter;