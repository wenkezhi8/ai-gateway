#!/bin/bash

# AI Gateway Benchmark Test Suite
# Uses vegeta for HTTP load testing
# Targets: 99.9% availability, P99 latency ≤ 50ms

set -e

# Configuration
GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
DURATION="${DURATION:-30s}"
RATE="${RATE:-100}"
OUTPUT_DIR="${OUTPUT_DIR:-./benchmark-results}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Create output directory
mkdir -p "$OUTPUT_DIR"

echo -e "${GREEN}=== AI Gateway Benchmark Test Suite ===${NC}"
echo "Gateway URL: $GATEWAY_URL"
echo "Duration: $DURATION"
echo "Rate: $RATE requests/second"
echo "Output: $OUTPUT_DIR"
echo ""

# Check if vegeta is installed
if ! command -v vegeta &> /dev/null; then
    echo -e "${RED}Error: vegeta is not installed${NC}"
    echo "Install with: brew install vegeta or go install github.com/tsenart/vegeta@latest"
    exit 1
fi

# Check if gateway is running
echo -e "${YELLOW}Checking gateway health...${NC}"
if ! curl -s -f "$GATEWAY_URL/health" > /dev/null; then
    echo -e "${RED}Error: Gateway is not running at $GATEWAY_URL${NC}"
    exit 1
fi
echo -e "${GREEN}Gateway is healthy${NC}"
echo ""

# Test 1: Health Check Endpoint
test_health_endpoint() {
    echo -e "${GREEN}Test 1: Health Check Endpoint${NC}"
    echo "GET $GATEWAY_URL/health" | vegeta attack -duration=$DURATION -rate=$RATE | \
        vegeta encode -to json > "$OUTPUT_DIR/health-results.json"

    # Generate report
    cat "$OUTPUT_DIR/health-results.json" | vegeta report > "$OUTPUT_DIR/health-report.txt"
    cat "$OUTPUT_DIR/health-report.txt"

    # Check SLA
    LATENCY=$(cat "$OUTPUT_DIR/health-results.json" | jq -r '.latencies.p99 // 0')
    LATENCY_MS=$((LATENCY / 1000000))

    if [ "$LATENCY_MS" -le 50 ]; then
        echo -e "${GREEN}✓ P99 latency: ${LATENCY_MS}ms (target: ≤50ms)${NC}"
    else
        echo -e "${RED}✗ P99 latency: ${LATENCY_MS}ms (target: ≤50ms)${NC}"
    fi
    echo ""
}

# Test 2: List Providers Endpoint
test_list_providers() {
    echo -e "${GREEN}Test 2: List Providers Endpoint${NC}"
    echo "GET $GATEWAY_URL/api/v1/providers" | vegeta attack -duration=$DURATION -rate=$RATE | \
        vegeta encode -to json > "$OUTPUT_DIR/providers-results.json"

    cat "$OUTPUT_DIR/providers-results.json" | vegeta report > "$OUTPUT_DIR/providers-report.txt"
    cat "$OUTPUT_DIR/providers-report.txt"
    echo ""
}

# Test 3: Chat Completions Endpoint
test_chat_completions() {
    echo -e "${GREEN}Test 3: Chat Completions Endpoint${NC}"

    # Create request body
    cat > /tmp/chat-request.json <<EOF
{
    "model": "gpt-4",
    "messages": [
        {"role": "user", "content": "Hello, this is a benchmark test"}
    ],
    "temperature": 0.7,
    "max_tokens": 100
}
EOF

    echo "POST $GATEWAY_URL/api/v1/chat/completions" | \
        vegeta attack -duration=$DURATION -rate=$RATE \
        -body=/tmp/chat-request.json \
        -header="Content-Type: application/json" | \
        vegeta encode -to json > "$OUTPUT_DIR/chat-results.json"

    cat "$OUTPUT_DIR/chat-results.json" | vegeta report > "$OUTPUT_DIR/chat-report.txt"
    cat "$OUTPUT_DIR/chat-report.txt"
    echo ""
}

# Test 4: Normal Load Test
test_normal_load() {
    echo -e "${GREEN}Test 4: Normal Load Test (100 RPS)${NC}"

    # Mixed workload
    (
        echo "GET $GATEWAY_URL/health"
        echo "GET $GATEWAY_URL/api/v1/providers"
    ) | vegeta attack -duration=60s -rate=100 | vegeta encode -to json > "$OUTPUT_DIR/normal-load.json"

    cat "$OUTPUT_DIR/normal-load.json" | vegeta report > "$OUTPUT_DIR/normal-load-report.txt"
    cat "$OUTPUT_DIR/normal-load-report.txt"
    echo ""
}

# Test 5: Peak Load Test
test_peak_load() {
    echo -e "${GREEN}Test 5: Peak Load Test (500 RPS)${NC}"

    (
        echo "GET $GATEWAY_URL/health"
        echo "GET $GATEWAY_URL/api/v1/providers"
    ) | vegeta attack -duration=30s -rate=500 | vegeta encode -to json > "$OUTPUT_DIR/peak-load.json"

    cat "$OUTPUT_DIR/peak-load.json" | vegeta report > "$OUTPUT_DIR/peak-load-report.txt"
    cat "$OUTPUT_DIR/peak-load-report.txt"

    # Check availability
    SUCCESS_RATE=$(cat "$OUTPUT_DIR/peak-load.json" | jq -r '.success_ratio // 0')
    SUCCESS_PERCENT=$(echo "$SUCCESS_RATE * 100" | bc -l)

    if (( $(echo "$SUCCESS_PERCENT >= 99.9" | bc -l) )); then
        echo -e "${GREEN}✓ Availability: ${SUCCESS_PERCENT}% (target: ≥99.9%)${NC}"
    else
        echo -e "${RED}✗ Availability: ${SUCCESS_PERCENT}% (target: ≥99.9%)${NC}"
    fi
    echo ""
}

# Test 6: Sustained Load Test
test_sustained_load() {
    echo -e "${GREEN}Test 6: Sustained Load Test (5 minutes at 200 RPS)${NC}"

    (
        echo "GET $GATEWAY_URL/health"
        echo "GET $GATEWAY_URL/api/v1/providers"
    ) | vegeta attack -duration=5m -rate=200 | vegeta encode -to json > "$OUTPUT_DIR/sustained-load.json"

    cat "$OUTPUT_DIR/sustained-load.json" | vegeta report > "$OUTPUT_DIR/sustained-load-report.txt"
    cat "$OUTPUT_DIR/sustained-load-report.txt"
    echo ""
}

# Test 7: Failover Test
test_failover() {
    echo -e "${GREEN}Test 7: Failover Test${NC}"
    echo "This test requires manual intervention to simulate provider failure"
    echo "Running basic availability check during simulated failure..."

    # Monitor availability for 60 seconds
    START_TIME=$(date +%s)
    TOTAL_REQUESTS=0
    SUCCESSFUL_REQUESTS=0

    for i in {1..60}; do
        RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" "$GATEWAY_URL/health")
        TOTAL_REQUESTS=$((TOTAL_REQUESTS + 1))
        if [ "$RESPONSE" = "200" ]; then
            SUCCESSFUL_REQUESTS=$((SUCCESSFUL_REQUESTS + 1))
        fi
        sleep 1
    done

    AVAILABILITY=$(echo "scale=2; $SUCCESSFUL_REQUESTS * 100 / $TOTAL_REQUESTS" | bc)
    echo "Availability during test: ${AVAILABILITY}%"
    echo "Successful requests: $SUCCESSFUL_REQUESTS / $TOTAL_REQUESTS"
    echo ""
}

# Generate summary report
generate_summary() {
    echo -e "${GREEN}=== Benchmark Summary ===${NC}"

    SUMMARY_FILE="$OUTPUT_DIR/summary.md"

    cat > "$SUMMARY_FILE" <<EOF
# AI Gateway Benchmark Results

**Test Date:** $(date)
**Gateway URL:** $GATEWAY_URL
**Duration per test:** $DURATION
**Target Rate:** $RATE RPS

## Test Results

### 1. Health Check Endpoint
\`\`\`
$(cat "$OUTPUT_DIR/health-report.txt" 2>/dev/null || echo "No results")
\`\`\`

### 2. List Providers Endpoint
\`\`\`
$(cat "$OUTPUT_DIR/providers-report.txt" 2>/dev/null || echo "No results")
\`\`\`

### 3. Chat Completions Endpoint
\`\`\`
$(cat "$OUTPUT_DIR/chat-report.txt" 2>/dev/null || echo "No results")
\`\`\`

### 4. Normal Load Test
\`\`\`
$(cat "$OUTPUT_DIR/normal-load-report.txt" 2>/dev/null || echo "No results")
\`\`\`

### 5. Peak Load Test
\`\`\`
$(cat "$OUTPUT_DIR/peak-load-report.txt" 2>/dev/null || echo "No results")
\`\`\`

### 6. Sustained Load Test
\`\`\`
$(cat "$OUTPUT_DIR/sustained-load-report.txt" 2>/dev/null || echo "No results")
\`\`\`

## SLA Compliance

| Metric | Target | Status |
|--------|--------|--------|
| Availability | 99.9% | Check individual reports |
| P99 Latency | ≤50ms | Check individual reports |
| Failover Time | ≤3s | Manual verification required |

EOF

    echo "Summary report saved to: $SUMMARY_FILE"
}

# Run all tests
main() {
    case "${1:-all}" in
        health)
            test_health_endpoint
            ;;
        providers)
            test_list_providers
            ;;
        chat)
            test_chat_completions
            ;;
        normal)
            test_normal_load
            ;;
        peak)
            test_peak_load
            ;;
        sustained)
            test_sustained_load
            ;;
        failover)
            test_failover
            ;;
        all)
            test_health_endpoint
            test_list_providers
            test_chat_completions
            test_normal_load
            test_peak_load
            # test_sustained_load  # Uncomment for full test
            # test_failover        # Uncomment for full test
            generate_summary
            ;;
        *)
            echo "Usage: $0 {health|providers|chat|normal|peak|sustained|failover|all}"
            exit 1
            ;;
    esac
}

main "$@"
