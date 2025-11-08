# API Usage Guide

## POST / - Create Schedule Configuration

This endpoint accepts a JSON payload to create a schedule configuration with validation middleware.

### Request

**Method:** `POST`  
**URL:** `/`  
**Content-Type:** `application/json`

**Body:**
```json
{
  "userId": "user123",
  "campaignId": "campaign456",
  "marketplace": "US",
  "interval": 30
}
```

**Fields:**
- `userId` (string, required): The user identifier
- `campaignId` (string, required): The campaign identifier
- `marketplace` (string, required): The marketplace code
- `interval` (integer, required): Time interval in minutes, must be a multiple of 15 (e.g., 15, 30, 45, 60, etc.)

### Validation Rules

The middleware automatically validates:
1. All fields are required (non-empty)
2. `interval` must be greater than 0
3. `interval` must be a multiple of 15 minutes

### Response

**Success (201 Created):**
```json
{
  "status": "success",
  "message": "Schedule configuration created successfully",
  "data": {
    "userId": "user123",
    "campaignId": "campaign456",
    "marketplace": "US",
    "interval": 30,
    "dueAt": "2025-11-08T15:30:00Z"
  }
}
```

Note: `dueAt` is automatically calculated by adding the interval to the current time.

**Validation Error (400 Bad Request):**
```json
{
  "error": "Validation failed",
  "fields": {
    "userId": "userId is required",
    "interval": "interval must be a multiple of 15 minutes (e.g., 15, 30, 45, 60, etc.)"
  }
}
```

**Invalid JSON (400 Bad Request):**
```json
{
  "error": "Invalid JSON body: unexpected end of JSON input"
}
```

**Wrong Content-Type (400 Bad Request):**
```json
{
  "error": "Content-Type must be application/json"
}
```

### Example cURL Commands

**Valid request:**
```bash
curl -X POST http://localhost:8080/ \
  -H "Content-Type: application/json" \
  -d '{
    "userId": "user123",
    "campaignId": "campaign456",
    "marketplace": "US",
    "interval": 30
  }'
```

**Invalid interval (not a multiple of 15):**
```bash
curl -X POST http://localhost:8080/ \
  -H "Content-Type: application/json" \
  -d '{
    "userId": "user123",
    "campaignId": "campaign456",
    "marketplace": "US",
    "interval": 20
  }'
```

**Missing required fields:**
```bash
curl -X POST http://localhost:8080/ \
  -H "Content-Type: application/json" \
  -d '{
    "userId": "user123"
  }'
```

## GET /ping - Heartbeat

Health check endpoint.

**Response (200 OK):**
```
.
```

### Example:
```bash
curl http://localhost:8080/ping
```

## How the Middleware Works

1. **Request Reception**: The POST request is received by the server
2. **Middleware Chain**: The request passes through:
   - Logger middleware (logs the request)
   - Recoverer middleware (handles panics)
   - ValidateBody middleware (validates JSON payload)
3. **Validation**: The ValidateBody middleware:
   - Checks Content-Type is `application/json`
   - Parses the JSON body into `ScheduleConfigRequest`
   - Calls `Validate()` to check all required fields
   - If validation fails, returns 400 with error details
   - If validation succeeds, stores the validated config in request context
4. **Handler Execution**: The handler retrieves the validated config from context and processes it
5. **Response**: Returns success or error response

### Code Flow

```
Client Request
    ↓
Logger Middleware
    ↓
Recoverer Middleware
    ↓
ValidateBody Middleware
    ├─ Parse JSON
    ├─ Validate Fields
    ├─ Store in Context
    ↓
ConfigHandler.HandleScheduleUpdate
    ├─ Retrieve from Context
    ├─ Convert to Domain Entity
    ├─ Process Business Logic
    ↓
JSON Response
```

