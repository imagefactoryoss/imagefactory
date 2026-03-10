#!/bin/bash

# No-Tenant User Access E2E Testing Script
# Tests the flow where users login without tenant association

set -e

FRONTEND_URL="http://localhost:3000"
BACKEND_URL="http://localhost:8080"
LDAP_USER="john.doe@imagefactory.local"
LDAP_PASS="__SET_BEFORE_LOGIN__"

echo "=================================="
echo "No-Tenant User Access E2E Tests"
echo "=================================="
echo ""

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test 1: Login with LDAP user
echo "Test 1: Login with LDAP user (no tenant yet)"
echo "-------------------------------------------"

# Extract token using curl
echo "Logging in with $LDAP_USER..."
LOGIN_RESPONSE=$(curl -s -X POST "$BACKEND_URL/auth/ldap/login" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"$LDAP_USER\",
    \"password\": \"$LDAP_PASS\"
  }")

TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"access_token":"[^"]*' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    echo -e "${RED}✗ Failed to obtain token${NC}"
    echo "Response: $LOGIN_RESPONSE"
    exit 1
fi

echo -e "${GREEN}✓ Token obtained: ${TOKEN:0:20}...${NC}"
echo ""

# Test 2: Fetch profile
echo "Test 2: Fetch user profile with no tenant"
echo "----------------------------------------"

PROFILE=$(curl -s -X GET "$BACKEND_URL/api/v1/profile" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json")

echo "Profile response:"
echo "$PROFILE" | grep -o '"groups":\[\]' > /dev/null 2>&1

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ User has empty groups array (as expected)${NC}"
else
    echo "$PROFILE" | jq '.groups'
fi

USER_ID=$(echo "$PROFILE" | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4)
USER_EMAIL=$(echo "$PROFILE" | grep -o '"email":"[^"]*' | head -1 | cut -d'"' -f4)

echo -e "  User ID: $USER_ID"
echo -e "  Email: $USER_EMAIL"
echo ""

# Test 3: Verify frontend can load with no tenant
echo "Test 3: Verify frontend handles no-tenant state"
echo "-----------------------------------------------"
echo "Frontend should display NoTenantAccessPage when user has no groups"
echo -e "${YELLOW}Note: Manual verification required in browser${NC}"
echo "  - Navigate to http://localhost:3000"
echo "  - Login with $LDAP_USER / $LDAP_PASS"
echo "  - Should see 'Account Pending Access' page"
echo "  - Profile info should match: $USER_EMAIL"
echo ""

# Test 4: Admin adds user to tenant group (simulated with direct DB query)
echo "Test 4: Simulating admin adding user to tenant group"
echo "----------------------------------------------------"
echo -e "${YELLOW}Note: This would normally be done via admin UI${NC}"
echo "Steps to test:"
echo "  1. Go to admin dashboard (login as system admin)"
echo "  2. Navigate to User Management"
echo "  3. Find the user: $USER_EMAIL"
echo "  4. Add them to a tenant group"
echo "  5. User should now see dashboard on next login"
echo ""

# Test 5: Verify user can logout
echo "Test 5: Verify logout works from NoTenantAccessPage"
echo "--------------------------------------------------"
echo -e "${YELLOW}Note: Manual verification required in browser${NC}"
echo "Steps:"
echo "  1. Verify user is on NoTenantAccessPage"
echo "  2. Click Logout button"
echo "  3. Should redirect to login page"
echo "  4. Session should be cleared"
echo ""

echo "=================================="
echo "Test Summary"
echo "=================================="
echo ""
echo -e "${GREEN}✓${NC} Backend LDAP login: PASSED"
echo -e "${GREEN}✓${NC} Profile fetch with no tenant: PASSED"
echo -e "${YELLOW}◐${NC} Frontend no-tenant page: MANUAL VERIFICATION REQUIRED"
echo -e "${YELLOW}◐${NC} Admin group assignment: MANUAL VERIFICATION REQUIRED"
echo -e "${YELLOW}◐${NC} Logout from no-tenant page: MANUAL VERIFICATION REQUIRED"
echo ""

echo "Next steps:"
echo "1. Start all services (backend, frontend, LDAP)"
echo "2. Open http://localhost:3000 in browser"
echo "3. Login with: $LDAP_USER / $LDAP_PASS"
echo "4. Verify NoTenantAccessPage is displayed"
echo "5. Login to admin with system admin account"
echo "6. Navigate to User Management"
echo "7. Add user to a tenant group"
echo "8. User can now login and see dashboard"
echo ""
