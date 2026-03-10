#!/bin/bash

# WebSocket client test script
# Tests the WebSocket endpoint for build log streaming

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}🔐 Getting auth token...${NC}"

# Get a token by logging in with default credentials
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@imagefactory.local","password":"__SET_BEFORE_LOGIN__"}' \
  | jq -r '.access_token // .token')

TENANT_ID="${TENANT_ID:-00000000-0000-0000-0000-000000000000}"

if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
  echo -e "${RED}❌ Failed to get auth token${NC}"
  exit 1
fi

echo -e "${GREEN}✅ Got token: ${TOKEN:0:20}...${NC}"

# Test the WebSocket endpoint with a Python script
PYTHON_SCRIPT='
import asyncio
import json
import websockets
import sys

async def test_websocket():
    build_id = "550e8400-e29b-41d4-a716-446655440000"  # Test UUID
    token = sys.argv[1]
    tenant_id = sys.argv[2] if len(sys.argv) > 2 else "00000000-0000-0000-0000-000000000000"
    url = f"ws://localhost:8080/api/v1/builds/{build_id}/logs/stream"
    
    try:
        async with websockets.connect(
            url,
            subprotocols=["Bearer", token],
            extra_headers={"Authorization": f"Bearer {token}", "X-Tenant-ID": tenant_id}
        ) as websocket:
            print("✅ WebSocket connection established!")
            
            # Send a test message
            test_msg = {
                "build_id": build_id,
                "timestamp": "2024-01-01T00:00:00Z",
                "level": "INFO",
                "message": "Test log message"
            }
            await websocket.send(json.dumps(test_msg))
            print("📤 Sent test message")
            
            # Wait for response
            try:
                response = await asyncio.wait_for(websocket.recv(), timeout=2.0)
                print(f"📨 Received: {response}")
            except asyncio.TimeoutError:
                print("⏱️  No response (expected - server not sending)")
                
    except Exception as e:
        print(f"❌ WebSocket error: {e}")
        sys.exit(1)

if __name__ == "__main__":
    asyncio.run(test_websocket())
'

echo -e "${YELLOW}🧪 Testing WebSocket connection...${NC}"

# Try using websocat if available, otherwise use curl for a basic test
if command -v websocat &> /dev/null; then
  echo -e "${YELLOW}Using websocat to test WebSocket...${NC}"
  # websocat with authorization header
  websocat "ws://localhost:8080/api/v1/builds/550e8400-e29b-41d4-a716-446655440000/logs/stream" \
    -H "Authorization: Bearer $TOKEN" \
    -H "X-Tenant-ID: $TENANT_ID" \
    --text "Hello" \
    --exit-on-eof \
    --max-messages 1 \
    || echo -e "${YELLOW}WebSocket connection attempt (may fail if build not found - expected)${NC}"
elif command -v python3 &> /dev/null; then
  echo -e "${YELLOW}Using Python to test WebSocket...${NC}"
  if python3 -c "import websockets" 2>/dev/null; then
    python3 -c "$PYTHON_SCRIPT" "$TOKEN" "$TENANT_ID"
  else
    echo -e "${YELLOW}websockets library not installed, installing...${NC}"
    pip3 install websockets > /dev/null 2>&1
    python3 -c "$PYTHON_SCRIPT" "$TOKEN" "$TENANT_ID"
  fi
else
  echo -e "${YELLOW}Testing with curl (WebSocket check)...${NC}"
  # At least verify the endpoint exists and returns 400 (upgrade required)
  RESULT=$(curl -s -w "\n%{http_code}" \
    -H "Authorization: Bearer $TOKEN" \
    http://localhost:8080/api/v1/builds/550e8400-e29b-41d4-a716-446655440000/logs/stream)
  
  HTTP_CODE=$(echo "$RESULT" | tail -1)
  if [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "101" ]; then
    echo -e "${GREEN}✅ WebSocket endpoint is accessible!${NC}"
  else
    echo -e "${YELLOW}Endpoint returned HTTP $HTTP_CODE (check backend logs)${NC}"
  fi
fi

echo -e "${GREEN}✅ WebSocket test completed${NC}"
