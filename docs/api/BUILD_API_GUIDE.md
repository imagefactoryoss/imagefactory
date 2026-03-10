# Build Execution API - Integration Guide

## Overview

The Build Execution API provides comprehensive endpoints for managing container image builds with real-time log streaming, status monitoring, and full lifecycle control.

**API Base URL:** `http://localhost:8080/api/v1` (development)

**Authentication:** Bearer Token (JWT)

## Quick Start

### 1. Authentication

All endpoints require a valid JWT token in the `Authorization` header.

Important: All *authenticated* requests MUST include an `X-Tenant-ID` header with a non‑nil tenant UUID. System administrators are not exempt — requests missing `X-Tenant-ID` will be rejected with HTTP 400.

```bash
curl -X GET http://localhost:8080/api/v1/builds \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "X-Tenant-ID: <tenant-uuid>"
```

To obtain a token, use the authentication endpoint:

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@imagefactory.local",
    "password": "__SET_BEFORE_LOGIN__"
  }'
```

### 2. Create a Build

```bash
curl -X POST http://localhost:8080/api/v1/builds \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: <tenant-uuid>" \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "550e8400-e29b-41d4-a716-446655440000",
    "git_branch": "main"
  }'
```

**Response:**
```json
{
  "id": "660e8400-e29b-41d4-a716-446655440001",
  "project_id": "550e8400-e29b-41d4-a716-446655440000",
  "build_number": 42,
  "git_branch": "main",
  "status": "queued",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

### 3. Start a Build

```bash
curl -X POST http://localhost:8080/api/v1/builds/{id}/start \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: <tenant-uuid>"
```

### 4. Monitor Build Status

```bash
curl -X GET http://localhost:8080/api/v1/builds/{id}/status \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: <tenant-uuid>"
```

**Response:**
```json
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
```

### 5. Stream Build Logs (WebSocket)

```bash
websocat 'ws://localhost:8080/api/v1/builds/{id}/logs/stream' \
  -H 'Authorization: Bearer $TOKEN' \
  -H 'X-Tenant-ID: <tenant-uuid>'
```

**Log Message Format:**
```json
{
  "build_id": "660e8400-e29b-41d4-a716-446655440001",
  "timestamp": "2024-01-15T10:35:00Z",
  "level": "INFO",
  "message": "Build step completed",
  "metadata": {
    "step": "build",
    "container": "builder"
  }
}
```

## Endpoints Reference

### Builds

| Method | Endpoint | Description | Permission |
|--------|----------|-------------|-----------|
| POST | `/builds` | Create a new build | `build:create` |
| GET | `/builds` | List all builds | `build:list` |
| GET | `/builds/{id}` | Get build details | `build:read` |
| DELETE | `/builds/{id}` | Delete a build | `build:delete` |
| POST | `/builds/{id}/start` | Start a build | `build:create` |
| POST | `/builds/{id}/cancel` | Cancel a build | `build:cancel` |
| POST | `/builds/{id}/retry` | Retry a failed build | `build:create` |

### Build Status & Logs

| Method | Endpoint | Description | Permission |
|--------|----------|-------------|-----------|
| GET | `/builds/{id}/status` | Get build status | `build:read` |
| GET | `/builds/{id}/logs` | Get build logs (HTTP) | `build:read` |
| GET | `/builds/{id}/logs/stream` | Stream logs (WebSocket) | `build:read` |

## Build Status Values

A build progresses through the following states:

- **queued**: Build has been created and is waiting to start
- **in_progress**: Build is currently running
- **success**: Build completed successfully
- **failed**: Build completed with errors
- **cancelled**: Build was cancelled by user

## Build Lifecycle

```
Create Build (queued)
    ↓
    Start Build (in_progress)
    ↓
    ┌─ Build Succeeds → success
    │
    └─ Build Fails → failed
            ↓
            Retry Build → queued (new build created)

Alternative Flow:
    Cancel Build → cancelled
```

## Filtering & Pagination

### List Builds with Filters

```bash
curl -X GET "http://localhost:8080/api/v1/builds?status=failed&project_id=UUID&page=1&limit=20" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: <tenant-uuid>"
```

**Query Parameters:**
- `status`: Filter by status (queued, in_progress, success, failed, cancelled)
- `project_id`: Filter by project UUID
- `page`: Page number (1-indexed, default: 1)
- `limit`: Items per page (default: 20, max: 100)
- `sort_by`: Sort field (created_at, updated_at, build_number)
- `sort_order`: Sort order (asc, desc, default: desc)

### Pagination Response

```json
{
  "builds": [...],
  "total_count": 100,
  "page": 1,
  "limit": 20,
  "has_more": true
}
```

## Error Handling

All errors follow a standard format:

```json
{
  "error": "Human-readable error message",
  "code": "ERROR_CODE",
  "timestamp": "2024-01-15T10:30:00Z",
  "trace_id": "abc123def456",
  "details": {
    "field": "Additional error details"
  }
}
```

### HTTP Status Codes

| Code | Meaning | Example |
|------|---------|---------|
| 200 | OK | Build retrieved successfully |
| 201 | Created | Build created successfully |
| 204 | No Content | Successful deletion |
| 400 | Bad Request | Invalid request body |
| 401 | Unauthorized | Missing/invalid token |
| 403 | Forbidden | Insufficient permissions |
| 404 | Not Found | Build doesn't exist |
| 409 | Conflict | Invalid state transition |
| 500 | Server Error | Internal server error |
| 501 | Not Implemented | Feature not yet implemented |

### Common Error Codes

| Code | Description |
|------|-------------|
| `INVALID_REQUEST` | Bad request body or parameters |
| `UNAUTHORIZED` | Missing or invalid authentication |
| `FORBIDDEN` | Insufficient permissions |
| `NOT_FOUND` | Resource doesn't exist |
| `CONFLICT` | Invalid state transition or duplicate |
| `INTERNAL_SERVER_ERROR` | Server error |

## WebSocket Streaming

### Connecting to WebSocket

```javascript
const token = 'your-jwt-token';
const buildId = 'your-build-id';
const ws = new WebSocket(
  `ws://localhost:8080/api/v1/builds/${buildId}/logs/stream`,
  [],
  { headers: { Authorization: `Bearer ${token}` } }
);

ws.onmessage = (event) => {
  const log = JSON.parse(event.data);
  console.log(`[${log.level}] ${log.message}`);
};

ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};

ws.onclose = () => {
  console.log('Connection closed');
};
```

### Log Message Format

```typescript
interface LogMessage {
  build_id: string;           // UUID
  timestamp: string;          // ISO 8601
  level: 'DEBUG'|'INFO'|'WARN'|'ERROR';
  message: string;            // Log content
  metadata?: {
    [key: string]: string;    // Additional context
  };
}
```

## Retry Logic

When implementing retry logic, consider:

1. **Idempotency**: Multiple requests should be safe
2. **Exponential backoff**: Wait longer between retries
3. **Max retries**: Limit retry attempts
4. **Timeout**: Don't wait indefinitely

Example retry implementation:

```javascript
async function createBuildWithRetry(
  projectId,
  branch,
  maxRetries = 3,
  baseDelay = 1000
) {
  for (let attempt = 0; attempt < maxRetries; attempt++) {
    try {
      return await createBuild(projectId, branch);
    } catch (error) {
      if (attempt === maxRetries - 1) throw error;
      
      const delay = baseDelay * Math.pow(2, attempt);
      await new Promise(resolve => setTimeout(resolve, delay));
    }
  }
}
```

## Rate Limiting

The API may implement rate limiting. Check the response headers:

```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1234567890
```

If you receive a `429 Too Many Requests` response, wait until the `X-RateLimit-Reset` time.

## Field Validation

### Project ID
- **Type:** UUID
- **Format:** `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`
- **Required:** Yes for build creation

### Git Branch
- **Type:** String
- **Length:** 1-255 characters
- **Required:** Yes for build creation
- **Pattern:** Must be a valid git reference

### Git Commit
- **Type:** String
- **Length:** 7-40 characters (SHA-1 hash)
- **Required:** No
- **Notes:** If provided, must match a commit in the repository

## Examples by Language

### Python

```python
import requests
import json

token = 'your-jwt-token'
headers = {'Authorization': f'Bearer {token}', 'Content-Type': 'application/json'}

# Create build
response = requests.post(
    'http://localhost:8080/api/v1/builds',
    headers=headers,
    json={
        'project_id': '550e8400-e29b-41d4-a716-446655440000',
        'git_branch': 'main'
    }
)
build = response.json()
print(f"Created build {build['id']}")

# Get build status
response = requests.get(
    f'http://localhost:8080/api/v1/builds/{build["id"]}/status',
    headers=headers
)
status = response.json()
print(f"Status: {status['status']}")
```

### Node.js

```javascript
const token = 'your-jwt-token';

// Create build
const response = await fetch('http://localhost:8080/api/v1/builds', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    project_id: '550e8400-e29b-41d4-a716-446655440000',
    git_branch: 'main'
  })
});

const build = await response.json();
console.log(`Created build ${build.id}`);

// Stream logs
const ws = new WebSocket(
  `ws://localhost:8080/api/v1/builds/${build.id}/logs/stream`,
  [],
  { headers: { Authorization: `Bearer ${token}` } }
);

ws.onmessage = (event) => {
  console.log(JSON.parse(event.data));
};
```

### cURL

See [curl-examples.sh](./curl-examples.sh) for comprehensive cURL examples.

## Best Practices

1. **Always use HTTPS in production**
2. **Store tokens securely** (don't commit to version control)
3. **Implement proper error handling** and retries
4. **Use pagination** for large datasets
5. **Monitor WebSocket connections** for disconnects
6. **Validate responses** against expected schema
7. **Log request/response** for debugging
8. **Use connection pooling** for performance
9. **Implement request timeouts** to prevent hangs
10. **Check permissions** before attempting operations

## Troubleshooting

### 401 Unauthorized
- Verify token is valid and not expired
- Check `Authorization` header format
- Ensure token is included in all requests

### 403 Forbidden
- Verify user has required permissions
- Check build belongs to user's tenant
- Ensure role has necessary permission grants

### 404 Not Found
- Verify build ID is correct (UUID format)
- Check build hasn't been deleted
- Confirm build belongs to your tenant

### WebSocket Connection Failed
- Verify endpoint includes `/stream` suffix
- Check Authorization header
- Ensure build exists and is accessible
- Try with `websocat` or browser DevTools

### Build Stuck in Progress
- Check executor logs
- Verify worker is running
- Check for resource constraints
- Manually cancel and retry

## Support

For issues or questions:
1. Check [OpenAPI specification](./build-execution-openapi.yaml)
2. Review error messages and codes
3. Check backend logs for details
4. Contact support team
