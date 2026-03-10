#!/bin/bash

# Coverage Analysis Script for Image Factory Backend
# Analyzes test coverage by package and generates actionable insights

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BACKEND_DIR="$PROJECT_ROOT/backend"
COVERAGE_FILE="$BACKEND_DIR/coverage.out"
OUTPUT_HTML="$BACKEND_DIR/coverage.html"
ANALYSIS_FILE="$BACKEND_DIR/coverage_analysis.txt"

echo "🔍 Image Factory Test Coverage Analysis"
echo "========================================"
echo ""

# Run tests with coverage (allow failures for integration tests)
echo "📊 Running tests with coverage..."
cd "$BACKEND_DIR"
go test -coverprofile="$COVERAGE_FILE" ./... || true

# Generate HTML report
echo "📄 Generating HTML coverage report..."
go tool cover -html="$COVERAGE_FILE" -o "$OUTPUT_HTML"

# Analyze by package
echo "📦 Coverage by Package:"
echo "======================"
echo ""

{
    echo "# Test Coverage Analysis"
    echo ""
    echo "Generated: $(date)"
    echo ""
    echo "## Package Coverage Summary"
    echo ""
    echo "| Package | Coverage | Status |"
    echo "|---------|----------|--------|"

    go tool cover -func="$COVERAGE_FILE" | awk '
    BEGIN { 
        excellent = 0
        good = 0
        fair = 0
        poor = 0
        untested = 0
    }
    !/total/ {
        split($NF, a, "%")
        coverage = a[1]
        package_name = $1
        
        if (coverage >= 80) {
            status = "✅ Excellent"
            excellent++
        } else if (coverage >= 60) {
            status = "✅ Good"
            good++
        } else if (coverage >= 40) {
            status = "⚠️  Fair"
            fair++
        } else if (coverage > 0) {
            status = "❌ Poor"
            poor++
        } else {
            status = "⭕ Untested"
            untested++
        }
        
        # Only show file-level summary (not function level)
        if (package_name ~ /\.go:/) {
            printf "| `%s` | %.1f%% | %s |\n", package_name, coverage, status
        }
    }
    END {
        print ""
        print "## Summary"
        print ""
        print "- **Excellent (80%+)**: " excellent
        print "- **Good (60-79%)**: " good
        print "- **Fair (40-59%)**: " fair
        print "- **Poor (1-39%)**: " poor
        print "- **Untested (0%)**: " untested
        print ""
        print "## Recommendations"
        print ""
        print "### High Priority (Needs Coverage)"
        print "- Focus on untested packages first"
        print "- Then improve poor coverage (1-39%)"
        print ""
        print "### Medium Priority (Improvement Opportunity)"
        print "- Increase fair coverage (40-59%) to good"
        print ""
        print "### Long Term"
        print "- Target 80%+ overall coverage"
        print "- Maintain excellent coverage in critical paths"
    }
    ' >> "$ANALYSIS_FILE"
} || true

# Print summary
echo ""
go tool cover -func="$COVERAGE_FILE" | awk '
/total/ {
    split($NF, a, "%")
    printf "📊 Overall Coverage: %.1f%%\n", a[1]
}'

# Show top packages by coverage
echo ""
echo "🏆 Top 10 Best Covered Packages:"
go tool cover -func="$COVERAGE_FILE" | grep -v "total" | sort -t':' -k3 -rn | head -10 | awk '{printf "  %-60s %.1f%%\n", $1, substr($NF,1,length($NF)-1)}'

echo ""
echo "📉 Top 10 Worst Covered Packages:"
go tool cover -func="$COVERAGE_FILE" | grep -v "total" | sort -t':' -k3 -n | head -10 | awk '{printf "  %-60s %.1f%%\n", $1, substr($NF,1,length($NF)-1)}'

echo ""
echo "✅ Reports generated:"
echo "   - HTML: $OUTPUT_HTML"
echo "   - Analysis: $ANALYSIS_FILE"
echo ""
echo "💡 View HTML report: open $OUTPUT_HTML"
