#!/bin/bash
# Build Execution API - cURL Examples
# This script demonstrates how to use the Build Execution API endpoints

set -e

# Configuration
API_BASE_URL="http://localhost:8080/api/v1"
TOKEN="your-jwt-token-here"  # Replace with actual token
TENANT_ID="00000000-0000-0000-0000-000000000000"  # Replace with tenant UUID (X-Tenant-ID)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper function to print section headers
print_section() {
    echo -e "\n${BLUE}======================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}======================================${NC}"
}

# Helper function to print curl commands
print_command() {
    echo -e "${YELLOW}$ $1${NC}"
}

# ============================================================================
# 1. CREATE A BUILD
# ============================================================================
print_section "1. CREATE A BUILD"

print_command "curl -X POST $API_BASE_URL/builds \
  -H 'Authorization: Bearer $TOKEN' \
  -H 'X-Tenant-ID: $TENANT_ID' \
  -H 'Content-Type: application/json' \
  -d '{ \
    \"project_id\": \"550e8400-e29b-41d4-a716-446655440000\", \
    \"git_branch\": \"main\" \
  }'"

echo -e "${GREEN}Example Response:${NC}"
cat << 'EOF'
{
  "id": "660e8400-e29b-41d4-a716-446655440001",
  "project_id": "550e8400-e29b-41d4-a716-446655440000",
  "build_number": 42,
  "git_branch": "main",
  "status": "queued",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
EOF

# ============================================================================
# 2. LIST ALL BUILDS
# ============================================================================
print_section "2. LIST ALL BUILDS"

print_command "curl -X GET '$API_BASE_URL/builds?page=1&limit=20&sort_by=created_at&sort_order=desc' \
  -H 'Authorization: Bearer $TOKEN' \
  -H 'X-Tenant-ID: $TENANT_ID'"

echo -e "${GREEN}Example Response:${NC}"
cat << 'EOF'
{
  "builds": [
    {
      "id": "660e8400-e29b-41d4-a716-446655440001",
      "project_id": "550e8400-e29b-41d4-a716-446655440000",
      "build_number": 42,
      "git_branch": "main",
      "status": "success",
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T11:30:00Z"
    }
  ],
  "total_count": 42,
  "page": 1,
  "limit": 20,
  "has_more": true
}
EOF

# ============================================================================
# 3. GET BUILD DETAILS
# ============================================================================
print_section "3. GET BUILD DETAILS"

print_command "curl -X GET $API_BASE_URL/builds/660e8400-e29b-41d4-a716-446655440001 \
  -H 'Authorization: Bearer $TOKEN' \
  -H 'X-Tenant-ID: $TENANT_ID'"

echo -e "${GREEN}Example Response:${NC}"
cat << 'EOF'
{
  "id": "660e8400-e29b-41d4-a716-446655440001",
  "project_id": "550e8400-e29b-41d4-a716-446655440000",
  "build_number": 42,
  "git_branch": "main",
  "git_commit": "abc123def456",
  "git_author_name": "John Doe",
  "git_author_email": "john@example.com",
  "status": "success",
  "started_at": "2024-01-15T10:35:00Z",
  "completed_at": "2024-01-15T11:30:00Z",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T11:30:00Z"
}
EOF

# ============================================================================
# 4. START A BUILD
# ============================================================================
print_section "4. START A BUILD"

print_command "curl -X POST $API_BASE_URL/builds/660e8400-e29b-41d4-a716-446655440001/start \
  -H 'Authorization: Bearer $TOKEN' \
  -H 'X-Tenant-ID: $TENANT_ID' \
  -H 'Content-Type: application/json'"

echo -e "${GREEN}Example Response:${NC}"
cat << 'EOF'
{
  "id": "660e8400-e29b-41d4-a716-446655440001",
  "project_id": "550e8400-e29b-41d4-a716-446655440000",
  "build_number": 42,
  "git_branch": "main",
  "status": "in_progress",
  "started_at": "2024-01-15T10:35:00Z",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:35:00Z"
}
EOF

# ============================================================================
# 5. GET BUILD STATUS
# ============================================================================
print_section "5. GET BUILD STATUS"

print_command "curl -X GET $API_BASE_URL/builds/660e8400-e29b-41d4-a716-446655440001/status \
  -H 'Authorization: Bearer $TOKEN' \
  -H 'X-Tenant-ID: $TENANT_ID'"

echo -e "${GREEN}Example Response:${NC}"
cat << 'EOF'
{
  "build_id": "660e8400-e29b-41d4-a716-446655440001",
  "status": "in_progress",
  "progress": {
    "current_step": 5,
    "total_steps": 10,
    "percentage": 50
  },
  "started_at": "2024-01-15T10:35:00Z",
  "estimated_completion": "2024-01-15T11:05:00Z"
}
EOF

# ============================================================================
# 6. GET BUILD LOGS
# ============================================================================
print_section "6. GET BUILD LOGS"

print_command "curl -X GET '$API_BASE_URL/builds/660e8400-e29b-41d4-a716-446655440001/logs?offset=0&limit=100' \
  -H 'Authorization: Bearer $TOKEN' \
  -H 'X-Tenant-ID: $TENANT_ID'"

echo -e "${GREEN}Example Response:${NC}"
cat << 'EOF'
{
  "build_id": "660e8400-e29b-41d4-a716-446655440001",
  "logs": [
    {
      "timestamp": "2024-01-15T10:35:00Z",
      "level": "INFO",
      "message": "Build started"
    },
    {
      "timestamp": "2024-01-15T10:36:00Z",
      "level": "INFO",
      "message": "Pulling base image..."
    },
    {
      "timestamp": "2024-01-15T10:37:00Z",
      "level": "INFO",
      "message": "Installing dependencies"
    },
    {
      "timestamp": "2024-01-15T10:38:00Z",
      "level": "WARN",
      "message": "Some deprecation warnings in dependencies"
    }
  ],
  "total_lines": 245,
  "offset": 0,
  "limit": 100
}
EOF

# ============================================================================
# 7. STREAM BUILD LOGS (WebSocket)
# ============================================================================
print_section "7. STREAM BUILD LOGS (WebSocket)"

echo -e "${YELLOW}Note: WebSocket connections require special tools like wscat or websocat${NC}"

print_command "websocat 'ws://localhost:8080/api/v1/builds/660e8400-e29b-41d4-a716-446655440001/logs/stream' \
  -H 'Authorization: Bearer $TOKEN' \
  -H 'X-Tenant-ID: $TENANT_ID'"

echo -e "${GREEN}Example Messages (received from server):${NC}"
cat << 'EOF'
{"build_id":"660e8400-e29b-41d4-a716-446655440001","timestamp":"2024-01-15T10:35:00Z","level":"INFO","message":"Build started"}
{"build_id":"660e8400-e29b-41d4-a716-446655440001","timestamp":"2024-01-15T10:36:00Z","level":"INFO","message":"Pulling base image..."}
{"build_id":"660e8400-e29b-41d4-a716-446655440001","timestamp":"2024-01-15T10:37:00Z","level":"INFO","message":"Installing dependencies"}
EOF

# ============================================================================
# 8. CANCEL A BUILD
# ============================================================================
print_section "8. CANCEL A BUILD"

print_command "curl -X POST $API_BASE_URL/builds/660e8400-e29b-41d4-a716-446655440001/cancel \
  -H 'Authorization: Bearer $TOKEN' \
  -H 'X-Tenant-ID: $TENANT_ID' \
  -H 'Content-Type: application/json' \
  -d '{ \
    \"reason\": \"User requested cancellation\" \
  }'"

echo -e "${GREEN}Example Response:${NC}"
cat << 'EOF'
{
  "id": "660e8400-e29b-41d4-a716-446655440001",
  "project_id": "550e8400-e29b-41d4-a716-446655440000",
  "build_number": 42,
  "git_branch": "main",
  "status": "cancelled",
  "started_at": "2024-01-15T10:35:00Z",
  "completed_at": "2024-01-15T10:45:00Z",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:45:00Z"
}
EOF

# ============================================================================
# 9. RETRY A BUILD
# ============================================================================
print_section "9. RETRY A FAILED BUILD"

print_command "curl -X POST $API_BASE_URL/builds/660e8400-e29b-41d4-a716-446655440001/retry \
  -H 'Authorization: Bearer $TOKEN' \
  -H 'X-Tenant-ID: $TENANT_ID' \
  -H 'Content-Type: application/json'"

echo -e "${GREEN}Example Response:${NC}"
cat << 'EOF'
{
  "id": "770e8400-e29b-41d4-a716-446655440002",
  "project_id": "550e8400-e29b-41d4-a716-446655440000",
  "build_number": 43,
  "git_branch": "main",
  "status": "queued",
  "created_at": "2024-01-15T11:30:00Z",
  "updated_at": "2024-01-15T11:30:00Z"
}
EOF

# ============================================================================
# ERROR EXAMPLES
# ============================================================================
print_section "ERROR EXAMPLES"

echo -e "${YELLOW}1. Missing Authentication:${NC}"
print_command "curl -X GET $API_BASE_URL/builds/660e8400-e29b-41d4-a716-446655440001"

echo -e "${GREEN}Response:${NC}"
cat << 'EOF'
HTTP/1.1 401 Unauthorized
{
  "error": "Missing or invalid authentication token",
  "code": "UNAUTHORIZED",
  "timestamp": "2024-01-15T10:30:00Z"
}
EOF

echo -e "\n${YELLOW}2. Build Not Found:${NC}"
print_command "curl -X GET $API_BASE_URL/builds/000e8400-e29b-41d4-a716-446655440000 \
  -H 'Authorization: Bearer $TOKEN'"

echo -e "${GREEN}Response:${NC}"
cat << 'EOF'
HTTP/1.1 404 Not Found
{
  "error": "Build not found",
  "code": "NOT_FOUND",
  "timestamp": "2024-01-15T10:30:00Z"
}
EOF

echo -e "\n${YELLOW}3. Invalid Request Body:${NC}"
print_command "curl -X POST $API_BASE_URL/builds \
  -H 'Authorization: Bearer $TOKEN' \
  -H 'Content-Type: application/json' \
  -d '{ \"git_branch\": \"main\" }'"

echo -e "${GREEN}Response:${NC}"
cat << 'EOF'
HTTP/1.1 400 Bad Request
{
  "error": "project_id is required",
  "code": "INVALID_REQUEST",
  "timestamp": "2024-01-15T10:30:00Z"
}
EOF

echo -e "\n${YELLOW}4. Permission Denied:${NC}"
print_command "curl -X POST $API_BASE_URL/builds/660e8400-e29b-41d4-a716-446655440001/cancel \
  -H 'Authorization: Bearer $OTHER_USER_TOKEN' \
  -H 'X-Tenant-ID: $TENANT_ID'"

echo -e "${GREEN}Response:${NC}"
cat << 'EOF'
HTTP/1.1 403 Forbidden
{
  "error": "You do not have permission to access this resource",
  "code": "FORBIDDEN",
  "timestamp": "2024-01-15T10:30:00Z"
}
EOF

# ============================================================================
# FILTERING AND PAGINATION
# ============================================================================
print_section "FILTERING AND PAGINATION"

echo -e "${YELLOW}List builds with filters:${NC}"
print_command "curl -X GET '$API_BASE_URL/builds?status=failed&project_id=550e8400-e29b-41d4-a716-446655440000&page=2&limit=10' \
  -H 'Authorization: Bearer $TOKEN' \
  -H 'X-Tenant-ID: $TENANT_ID' \
  -H 'X-Tenant-ID: $TENANT_ID'"

echo -e "\n${YELLOW}Sort by different fields:${NC}"
print_command "curl -X GET '$API_BASE_URL/builds?sort_by=build_number&sort_order=asc' \
  -H 'Authorization: Bearer $TOKEN' \
  -H 'X-Tenant-ID: $TENANT_ID'"

echo -e "\n${YELLOW}Get specific log offset:${NC}"
print_command "curl -X GET '$API_BASE_URL/builds/660e8400-e29b-41d4-a716-446655440001/logs?offset=100&limit=50' \
  -H 'Authorization: Bearer $TOKEN' \
  -H 'X-Tenant-ID: $TENANT_ID'"

# ============================================================================
# PERMISSIONS REFERENCE
# ============================================================================
print_section "PERMISSIONS REFERENCE"

cat << 'EOF'
The following permissions are required for each endpoint:

Endpoint                              Method  Required Permission
────────────────────────────────────────────────────────────────
/api/v1/builds                        POST    build:create
/api/v1/builds                        GET     build:list
/api/v1/builds/{id}                   GET     build:read
/api/v1/builds/{id}                   DELETE  build:delete
/api/v1/builds/{id}/start             POST    build:create
/api/v1/builds/{id}/cancel            POST    build:cancel
/api/v1/builds/{id}/retry             POST    build:create
/api/v1/builds/{id}/status            GET     build:read
/api/v1/builds/{id}/logs              GET     build:read
/api/v1/builds/{id}/logs/stream       GET     build:read (WebSocket)
EOF

echo -e "\n${GREEN}✅ cURL Examples Complete${NC}\n"
