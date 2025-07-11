#!/bin/bash

# Script to run benchmarks comparing getTransactionsByWallet and getTransactionsByWalletV2
# This script runs the benchmarks multiple times and generates a comparison report

set -e

BENCHMARK_DIR="internal/walletwatch"
OUTPUT_DIR="benchmark_results"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

# Create output directory
mkdir -p "$OUTPUT_DIR"

echo "🚀 Running benchmarks for getTransactionsByWallet vs getTransactionsByWalletV2"
echo "Results will be saved to: $OUTPUT_DIR"
echo ""

# Function to run a specific benchmark
run_benchmark() {
    local benchmark_name="$1"
    local output_file="$2"
    
    echo "Running $benchmark_name..."
    go test -bench="$benchmark_name" -benchmem -count=5 -timeout=30m "./$BENCHMARK_DIR" > "$output_file"
    echo "✅ $benchmark_name completed"
}

# Run all benchmark categories
echo "📊 Running basic size comparison benchmarks..."
run_benchmark "BenchmarkGetTransactionsByWallet.*_(Small|Medium|Large)$" "$OUTPUT_DIR/size_comparison_${TIMESTAMP}.txt"

echo ""
echo "📈 Running transaction count variation benchmarks..."
run_benchmark "BenchmarkGetTransactionsByWallet.*_VaryingTxCount" "$OUTPUT_DIR/tx_count_variation_${TIMESTAMP}.txt"

echo ""
echo "🎯 Running watched ratio variation benchmarks..."
run_benchmark "BenchmarkGetTransactionsByWallet.*_VaryingWatchedRatio" "$OUTPUT_DIR/watched_ratio_variation_${TIMESTAMP}.txt"

echo ""
echo "💾 Running memory allocation benchmarks..."
run_benchmark "BenchmarkGetTransactionsByWallet.*_MemAlloc" "$OUTPUT_DIR/memory_allocation_${TIMESTAMP}.txt"

echo ""
echo "📋 Generating summary report..."

# Create a comprehensive summary report
cat > "$OUTPUT_DIR/benchmark_summary_${TIMESTAMP}.md" << EOF
# Benchmark Comparison: getTransactionsByWallet vs getTransactionsByWalletV2

Generated on: $(date)

## Overview

This report compares the performance of two implementations:
- \`getTransactionsByWallet\` (original implementation)
- \`getTransactionsByWalletV2\` (optimized implementation)

## Key Differences

### getTransactionsByWallet (V1)
- Uses \`types.Set[string]\` for transaction hashes per wallet
- Creates intermediate sets that are converted to slices
- Uses \`DefaultMap\` with \`Set\` values

### getTransactionsByWalletV2 (V2)
- Uses \`*types.Set[string]\` (pointer to set) for transaction hashes per wallet
- Uses \`*[]Transaction\` (pointer to slice) for final results
- Includes capacity hints for slice allocation
- More explicit about memory management

## Benchmark Results

### Size Comparison (Small/Medium/Large datasets)
\`\`\`
$(cat "$OUTPUT_DIR/size_comparison_${TIMESTAMP}.txt")
\`\`\`

### Transaction Count Variation
\`\`\`
$(cat "$OUTPUT_DIR/tx_count_variation_${TIMESTAMP}.txt")
\`\`\`

### Watched Wallet Ratio Variation
\`\`\`
$(cat "$OUTPUT_DIR/watched_ratio_variation_${TIMESTAMP}.txt")
\`\`\`

### Memory Allocation Analysis
\`\`\`
$(cat "$OUTPUT_DIR/memory_allocation_${TIMESTAMP}.txt")
\`\`\`

## Analysis

### Performance Metrics to Compare:
1. **Execution Time**: ns/op (nanoseconds per operation)
2. **Memory Allocations**: allocs/op (number of allocations per operation)
3. **Memory Usage**: B/op (bytes allocated per operation)

### Expected Improvements in V2:
- Reduced memory allocations due to pointer usage
- Better memory locality with capacity hints
- Potentially faster execution due to fewer intermediate allocations

### Recommendations:
- If V2 shows significant improvements in memory usage and/or speed, consider adopting it
- Pay attention to scenarios with high watched wallet ratios and large transaction counts
- Monitor for any regressions in edge cases

EOF

echo "✅ Benchmark summary generated: $OUTPUT_DIR/benchmark_summary_${TIMESTAMP}.md"
echo ""
echo "🎉 All benchmarks completed successfully!"
echo ""
echo "📁 Results location: $OUTPUT_DIR/"
echo "📄 Summary report: $OUTPUT_DIR/benchmark_summary_${TIMESTAMP}.md"
echo ""
echo "To view the summary:"
echo "cat $OUTPUT_DIR/benchmark_summary_${TIMESTAMP}.md"
