#!/bin/bash

# Test script for pre-commit hook functionality
# This script creates test files to validate the hook's behavior

set -e

echo "🧪 Testing Pre-commit Hook for Large Files and Binaries"
echo "======================================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Cleanup function
cleanup() {
    echo -e "\n${BLUE}🧹 Cleaning up test files...${NC}"
    rm -f test-large-file.txt test-binary-file.exe test-image.jpg test-archive.zip test-normal.txt 2>/dev/null || true
    git reset HEAD . 2>/dev/null || true
    echo -e "${GREEN}✅ Cleanup complete${NC}"
}

# Set up cleanup trap
trap cleanup EXIT

# Test 1: Normal text file (should pass)
echo -e "\n${BLUE}Test 1: Normal text file (should PASS)${NC}"
echo "This is a normal text file for testing" > test-normal.txt
git add test-normal.txt
echo -e "${YELLOW}Running pre-commit hook...${NC}"
if .git/hooks/pre-commit; then
    echo -e "${GREEN}✅ Test 1 PASSED: Normal file allowed${NC}"
else
    echo -e "${RED}❌ Test 1 FAILED: Normal file blocked${NC}"
fi
git reset HEAD test-normal.txt

# Test 2: Large file (should fail)
echo -e "\n${BLUE}Test 2: Large file >10MB (should FAIL)${NC}"
echo -e "${YELLOW}Creating 12MB test file...${NC}"
dd if=/dev/zero of=test-large-file.txt bs=1048576 count=12 2>/dev/null
git add test-large-file.txt
echo -e "${YELLOW}Running pre-commit hook...${NC}"
if .git/hooks/pre-commit; then
    echo -e "${RED}❌ Test 2 FAILED: Large file was allowed${NC}"
else
    echo -e "${GREEN}✅ Test 2 PASSED: Large file blocked${NC}"
fi
git reset HEAD test-large-file.txt

# Test 3: Binary file (executable) (should fail)
echo -e "\n${BLUE}Test 3: Binary executable file (should FAIL)${NC}"
echo -e "${YELLOW}Creating binary file...${NC}"
# Create a simple binary file (ELF magic number)
printf '\x7fELF\x01\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00' > test-binary-file.exe
git add -f test-binary-file.exe
echo -e "${YELLOW}Running pre-commit hook...${NC}"
if .git/hooks/pre-commit; then
    echo -e "${RED}❌ Test 3 FAILED: Binary file was allowed${NC}"
else
    echo -e "${GREEN}✅ Test 3 PASSED: Binary file blocked${NC}"
fi
git reset HEAD test-binary-file.exe

# Test 4: Image file (should fail/warn)
echo -e "\n${BLUE}Test 4: Image file (should FAIL)${NC}"
echo -e "${YELLOW}Creating fake image file...${NC}"
# Create a file with JPEG magic bytes
printf '\xff\xd8\xff\xe0\x00\x10JFIF' > test-image.jpg
git add -f test-image.jpg
echo -e "${YELLOW}Running pre-commit hook...${NC}"
if .git/hooks/pre-commit; then
    echo -e "${RED}❌ Test 4 FAILED: Image file was allowed${NC}"
else
    echo -e "${GREEN}✅ Test 4 PASSED: Image file blocked${NC}"
fi
git reset HEAD test-image.jpg

# Test 5: Archive file (should fail/warn)
echo -e "\n${BLUE}Test 5: Archive file (should FAIL)${NC}"
echo -e "${YELLOW}Creating fake archive file...${NC}"
# Create a file with ZIP magic bytes
printf 'PK\x03\x04' > test-archive.zip
git add -f test-archive.zip
echo -e "${YELLOW}Running pre-commit hook...${NC}"
if .git/hooks/pre-commit; then
    echo -e "${RED}❌ Test 5 FAILED: Archive file was allowed${NC}"
else
    echo -e "${GREEN}✅ Test 5 PASSED: Archive file blocked${NC}"
fi
git reset HEAD test-archive.zip

# Test 6: Multiple files (mixed)
echo -e "\n${BLUE}Test 6: Multiple files - mixed types (should FAIL)${NC}"
echo "Small text file" > test-normal.txt
printf '\x7fELF\x01\x01\x01\x00' > test-binary-file.exe
git add test-normal.txt
git add -f test-binary-file.exe
echo -e "${YELLOW}Running pre-commit hook...${NC}"
if .git/hooks/pre-commit; then
    echo -e "${RED}❌ Test 6 FAILED: Mixed files with binary was allowed${NC}"
else
    echo -e "${GREEN}✅ Test 6 PASSED: Mixed files with binary blocked${NC}"
fi
git reset HEAD .

# Test 7: Hook bypass (should pass)
echo -e "\n${BLUE}Test 7: Hook bypass with --no-verify (should PASS)${NC}"
printf '\x7fELF\x01\x01\x01\x00' > test-binary-file.exe
git add -f test-binary-file.exe
echo -e "${YELLOW}Testing commit with --no-verify...${NC}"
if git commit --no-verify -m "Test commit with binary (bypass)" 2>/dev/null; then
    echo -e "${GREEN}✅ Test 7 PASSED: Hook bypass works${NC}"
    git reset --soft HEAD~1  # Undo the commit
else
    echo -e "${RED}❌ Test 7 FAILED: Hook bypass didn't work${NC}"
fi
git reset HEAD .

echo -e "\n${BLUE}📊 Test Summary${NC}"
echo "=============="
echo -e "${GREEN}All tests completed!${NC}"
echo -e "${YELLOW}⚠️  Note: Some tests are expected to fail (that's good!)${NC}"
echo -e "${BLUE}💡 The hook is working correctly if:${NC}"
echo -e "   • Normal files are allowed"
echo -e "   • Large files (>10MB) are blocked"  
echo -e "   • Binary files are blocked"
echo -e "   • --no-verify bypass works"

echo -e "\n${BLUE}🔧 Hook Status:${NC}"
if [[ -x .git/hooks/pre-commit ]]; then
    echo -e "${GREEN}✅ Pre-commit hook is installed and executable${NC}"
else
    echo -e "${RED}❌ Pre-commit hook is not properly installed${NC}"
fi

echo -e "\n${BLUE}📁 To check hook manually:${NC}"
echo -e "   cat .git/hooks/pre-commit"
echo -e "   ls -la .git/hooks/pre-commit"

echo -e "\n${GREEN}🎉 Pre-commit hook testing complete!${NC}"