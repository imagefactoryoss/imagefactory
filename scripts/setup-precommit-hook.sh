#!/bin/bash

# Setup script for installing pre-commit hooks
# This script helps team members quickly set up the repository's pre-commit hooks

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${BLUE}🔧 Image Factory - Pre-commit Hook Setup${NC}"
echo "========================================"

# Check if we're in a git repository
if [[ ! -d .git ]]; then
    echo -e "${RED}❌ Error: This doesn't appear to be a git repository${NC}"
    echo -e "   Make sure you're in the project root directory"
    exit 1
fi

# Check if the hook already exists
if [[ -f .git/hooks/pre-commit ]]; then
    echo -e "${YELLOW}⚠️  Pre-commit hook already exists${NC}"
    echo -e "   Checking if it's executable..."
    
    if [[ -x .git/hooks/pre-commit ]]; then
        echo -e "${GREEN}✅ Pre-commit hook is already installed and executable${NC}"
    else
        echo -e "${YELLOW}🔧 Making existing hook executable...${NC}"
        chmod +x .git/hooks/pre-commit
        echo -e "${GREEN}✅ Pre-commit hook is now executable${NC}"
    fi
else
    echo -e "${RED}❌ Error: Pre-commit hook not found at .git/hooks/pre-commit${NC}"
    echo -e "   This script is for repositories that already have the hook file"
    echo -e "   The hook should be committed to the repository and copied manually"
    exit 1
fi

# Test the hook
echo -e "\n${BLUE}🧪 Testing pre-commit hook...${NC}"
if [[ -x scripts/test-precommit-hook.sh ]]; then
    echo -e "${YELLOW}Running test suite...${NC}"
    if ./scripts/test-precommit-hook.sh >/dev/null 2>&1; then
        echo -e "${GREEN}✅ Pre-commit hook tests passed${NC}"
    else
        echo -e "${YELLOW}⚠️  Some tests failed (this may be expected)${NC}"
        echo -e "   Run './scripts/test-precommit-hook.sh' manually to see details"
    fi
else
    echo -e "${YELLOW}⚠️  Test script not found, creating simple test...${NC}"
    echo "Test file" > .test-precommit-setup.txt
    git add .test-precommit-setup.txt
    
    if .git/hooks/pre-commit >/dev/null 2>&1; then
        echo -e "${GREEN}✅ Pre-commit hook is working${NC}"
    else
        echo -e "${RED}❌ Pre-commit hook test failed${NC}"
    fi
    
    git reset HEAD .test-precommit-setup.txt >/dev/null 2>&1
    rm -f .test-precommit-setup.txt
fi

# Display configuration information
echo -e "\n${BLUE}📋 Configuration Summary:${NC}"
echo -e "   📁 Hook location: .git/hooks/pre-commit"
echo -e "   📏 File size limit: 10 MB"
echo -e "   🔧 Binary detection: Enabled"
echo -e "   ⚡ Suspicious extensions: Monitored"

echo -e "\n${BLUE}💡 Usage Tips:${NC}"
echo -e "   🚀 Normal commits: git commit -m \"your message\""
echo -e "   🚫 Bypass hook: git commit --no-verify -m \"emergency commit\""
echo -e "   📖 Full documentation: docs/Pre-commit-Hook-Guide.md"
echo -e "   🧪 Run tests: ./scripts/test-precommit-hook.sh"

echo -e "\n${BLUE}🔗 For large files, consider:${NC}"
echo -e "   • Git LFS: git lfs track \"*.ext\" && git add .gitattributes"
echo -e "   • External storage (S3, CDN, artifact repositories)"
echo -e "   • Adding patterns to .gitignore"

echo -e "\n${GREEN}🎉 Pre-commit hook setup complete!${NC}"
echo -e "${GREEN}   Your commits will now be automatically checked for large files and binaries${NC}"

# Optional: Show recent hook activity
if command -v git &> /dev/null; then
    echo -e "\n${BLUE}📊 Recent commits (for reference):${NC}"
    git log --oneline -5 || true
fi

echo -e "\n${BLUE}❓ Need help?${NC}"
echo -e "   • Read: docs/Pre-commit-Hook-Guide.md"
echo -e "   • Test: ./scripts/test-precommit-hook.sh"
echo -e "   • Issues: Check if files are staged and hook is executable"