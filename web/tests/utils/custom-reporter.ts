import type { FullConfig, Reporter, Suite, TestCase, TestResult } from '@playwright/test/reporter'

// Minimal custom reporter placeholder.
// Keeps playwright.config.ts reporter reference valid.
// 改动点: no-op reporter (future: add project-specific summary output)
class CustomReporter implements Reporter {
  onBegin(_config: FullConfig, _suite: Suite) {}
  onTestBegin(_test: TestCase, _result: TestResult) {}
  onTestEnd(_test: TestCase, _result: TestResult) {}
  onEnd() {}
}

export default CustomReporter
