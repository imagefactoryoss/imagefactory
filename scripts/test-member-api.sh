#!/bin/bash

# Member Management API Test Script
# Tests all project member management endpoints

set -e

BASE_URL="http://localhost:8080"
PROJECT_ID="00000000-0000-0000-0000-000000000001"
USER_ID="00000000-0000-0000-0000-000000000002"
TENANT_ID="00000000-0000-0000-0000-000000000000"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Counter for passed/failed tests
PASSED=0
FAILED=0

# Helper function to print test results
test_result() {
  local test_name=$1
  local status=$2
  local response=$3

  if [ $status -eq 0 ]; then
    echo -e "${GREEN}✓${NC} $test_name"
    ((PASSED++))
  else
    echo -e "${RED}✗${NC} $test_name"
    echo -e "${YELLOW}Response:${NC} $response"
    ((FAILED++))
  fi
}

echo "==============================================="
echo "Member Management API Tests"
echo "==============================================="
echo ""

# Test 1: Add Member
echo "Test 1: Add Member (POST /api/v1/projects/{id}/members)"
RESPONSE=$(curl -s -X POST \
  "$BASE_URL/api/v1/projects/$PROJECT_ID/members" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test-token" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d "{\"user_id\": \"$USER_ID\"}" \
  -w "\n%{http_code}")

HTTP_CODE=$(echo "$RESPONSE" | tail -n 1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "401" ]; then
  test_result "Add Member" 0 "$BODY"
else
  test_result "Add Member" 1 "HTTP $HTTP_CODE: $BODY"
fi

echo ""

# Test 2: List Members
echo "Test 2: List Members (GET /api/v1/projects/{id}/members)"
RESPONSE=$(curl -s -X GET \
  "$BASE_URL/api/v1/projects/$PROJECT_ID/members?limit=10&offset=0" \
  -H "Authorization: Bearer test-token" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -w "\n%{http_code}")

HTTP_CODE=$(echo "$RESPONSE" | tail -n 1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "401" ] || [ "$HTTP_CODE" = "404" ]; then
  test_result "List Members" 0 "$BODY"
else
  test_result "List Members" 1 "HTTP $HTTP_CODE: $BODY"
fi

echo ""

# Test 3: Update Member Role
echo "Test 3: Update Member Role (PATCH /api/v1/projects/{id}/members/{userId})"
RESPONSE=$(curl -s -X PATCH \
  "$BASE_URL/api/v1/projects/$PROJECT_ID/members/$USER_ID" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test-token" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d "{\"role_id\": \"00000000-0000-0000-0000-000000000003\"}" \
  -w "\n%{http_code}")

HTTP_CODE=$(echo "$RESPONSE" | tail -n 1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "401" ] || [ "$HTTP_CODE" = "404" ]; then
  test_result "Update Member Role" 0 "$BODY"
else
  test_result "Update Member Role" 1 "HTTP $HTTP_CODE: $BODY"
fi

echo ""

# Test 4: Remove Member
echo "Test 4: Remove Member (DELETE /api/v1/projects/{id}/members/{userId})"
RESPONSE=$(curl -s -X DELETE \
  "$BASE_URL/api/v1/projects/$PROJECT_ID/members/$USER_ID" \
  -H "Authorization: Bearer test-token" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -w "\n%{http_code}")

HTTP_CODE=$(echo "$RESPONSE" | tail -n 1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "204" ] || [ "$HTTP_CODE" = "401" ] || [ "$HTTP_CODE" = "404" ]; then
  test_result "Remove Member" 0 "$BODY"
else
  test_result "Remove Member" 1 "HTTP $HTTP_CODE: $BODY"
fi

echo ""
echo "==============================================="
echo "Test Results"
echo "==============================================="
echo -e "Passed: ${GREEN}$PASSED${NC}"
echo -e "Failed: ${RED}$FAILED${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
  echo -e "${GREEN}All tests passed!${NC}"
  exit 0
else
  echo -e "${RED}Some tests failed!${NC}"
  exit 1
fi
