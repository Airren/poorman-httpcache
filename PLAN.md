# Token System Design Plan

## System Overview
A token-based access control system where each API key has a fixed number of tokens (requests) they can consume. The system will be implemented as middleware that runs AFTER the cache middleware to ensure cached responses don't consume tokens.

## Architecture & Flow
```
Request → Cache Middleware → Token Middleware → Load Balancer (assign a random key and replace user key) -> Reverse Proxy → External API
                                    ↓
                            Check/Decrement Tokens
                                    ↓
                            Return 429 if exhausted
```

## Core Components

### 1. Token Manager Interface
```go
type TokenTollgate interface {
    ConsumeToken(ctx context.Context, apiKey string) (remaining int, err error)
    GetRemainingTokens(ctx context.Context, apiKey string) (int, error)
    AddTokens(ctx context.Context, apiKey string, count int) error
}
```

### 3. Token Middleware
- Extract API key using `ExtractKey()` function
- Validate API key format
- Check token availability
- Consume token on successful request
- Add `X-Remaining-Tokens` header to responses
- Return HTTP 429 when tokens exhausted

## Data Structures

### TokenInfo
```go
type TokenInfo struct {
    APIKey     string    `json:"api_key"`
    Remaining  int       `json:"remaining"`
    Total      int       `json:"total"`
    LastUsed   time.Time `json:"last_used"`
    CreatedAt  time.Time `json:"created_at"`
}
```

### TokenConfig
```go
type TokenConfig struct {
    DefaultTokens int           `json:"default_tokens"`
    TokenTTL      time.Duration `json:"token_ttl"`
    HeaderName    string        `json:"header_name"`
}
```

## Database Schema Design

### 1. API Key Requests Table (`api_key_requests`)
```sql
CREATE TABLE api_key_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL,
    company_name TEXT,
    use_case TEXT NOT NULL,
    expected_volume INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending', -- pending, approved, rejected
    requested_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reviewed_at TIMESTAMP WITH TIME ZONE,
    reviewed_by TEXT,
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_api_key_requests_email ON api_key_requests(email);
CREATE INDEX idx_api_key_requests_status ON api_key_requests(status);
CREATE INDEX idx_api_key_requests_requested_at ON api_key_requests(requested_at);
```

### 2. API Keys Info Table (`api_keys_info`)
```sql
CREATE TABLE api_keys_info (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    api_key TEXT UNIQUE NOT NULL,
    request_id UUID REFERENCES api_key_requests(id) NOT NULL,
    email TEXT NOT NULL,
    company_name TEXT,
    total_tokens INTEGER NOT NULL DEFAULT 1000,
    status TEXT NOT NULL DEFAULT 'active', -- active, suspended, expired
    expired_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by TEXT NOT NULL, -- admin email who approved
    notes TEXT
);

-- Indexes
CREATE INDEX idx_api_keys_info_api_key ON api_keys_info(api_key);
CREATE INDEX idx_api_keys_info_email ON api_keys_info(email);
CREATE INDEX idx_api_keys_info_status ON api_keys_info(status);
CREATE INDEX idx_api_keys_info_expired_at ON api_keys_info(expired_at);
```

### 3. API Key Balance Table (`api_key_balance`)
```sql
CREATE TABLE api_key_balance (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    api_key_id UUID REFERENCES api_keys_info(id) NOT NULL,
    remaining_tokens INTEGER NOT NULL DEFAULT 1000,
    last_used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_api_key_balance_api_key_id ON api_key_balance(api_key_id);
CREATE INDEX idx_api_key_balance_remaining_tokens ON api_key_balance(remaining_tokens);
CREATE INDEX idx_api_key_balance_last_used_at ON api_key_balance(last_used_at);

-- Constraints
ALTER TABLE api_key_balance ADD CONSTRAINT fk_api_key_balance_api_key_id 
    FOREIGN KEY (api_key_id) REFERENCES api_keys_info(id) ON DELETE CASCADE;
ALTER TABLE api_key_balance ADD CONSTRAINT chk_remaining_tokens_positive 
    CHECK (remaining_tokens >= 0);

### 4. Token Usage Log Table (`token_usage_log`)
```sql
CREATE TABLE token_usage_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    api_key_id UUID REFERENCES api_keys_info(id) NOT NULL,
    api_key TEXT NOT NULL,
    endpoint TEXT NOT NULL,
    method TEXT NOT NULL,
    status_code INTEGER NOT NULL,
    tokens_consumed INTEGER NOT NULL DEFAULT 1,
    ip_address INET,
    user_agent TEXT,
    request_id TEXT, -- for correlation
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_token_usage_log_api_key ON token_usage_log(api_key);
CREATE INDEX idx_token_usage_log_created_at ON token_usage_log(created_at);
CREATE INDEX idx_token_usage_log_api_key_id ON token_usage_log(api_key_id);
```

### 5. Admin Users Table (`admin_users`)
```sql
CREATE TABLE admin_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'reviewer', -- admin, reviewer
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_admin_users_email ON admin_users(email);
CREATE INDEX idx_admin_users_role ON admin_users(role);
```

## Table Split Benefits

### Performance Optimization
- **Reduced Lock Contention**: `api_keys_info` table is rarely updated, reducing lock conflicts
- **Faster Token Updates**: `api_key_balance` table can be optimized for high-frequency updates
- **Better Index Performance**: Separate indexes for different query patterns

### Scalability Improvements
- **Horizontal Partitioning**: `api_key_balance` table can be partitioned by date or API key ranges
- **Independent Scaling**: Each table can be scaled based on its specific access patterns
- **Cache Optimization**: Static info can be cached longer, balance data updated independently

### Data Management
- **Cleaner Backup Strategy**: Static data can be backed up less frequently
- **Easier Maintenance**: Balance table can be optimized/analyzed without affecting static data
- **Audit Trail**: Clear separation between configuration changes and usage tracking

### Table Relationships
- **One-to-One Relationship**: Each `api_keys_info` record has exactly one `api_key_balance` record
- **Foreign Key Constraints**: `api_key_balance.api_key_id` references `api_keys_info.id`
- **Cascade Operations**: When an API key is deleted, its balance record is also removed
- **Data Consistency**: Balance table must be updated whenever tokens are consumed or added

## Workflow & Business Logic

### API Key Request Flow
1. **User submits request** → `api_key_requests` table
2. **Admin reviews request** → Updates status to 'approved'/'rejected'
3. **If approved** → Admin creates entry in `api_keys_info` table + `api_key_balance` table
4. **Email notification** → System sends API key to user

### Token Consumption Flow
1. **Request comes in** → Extract API key from header
2. **Validate API key** → Check `api_keys_info` table
3. **Check token availability** → Verify `remaining_tokens > 0` in `api_key_balance` table
4. **Consume token** → Atomic decrement in Redis + update `api_key_balance` table
5. **Log usage** → Insert into `token_usage_log` table

## Updated Data Structures

### APIKeyRequest
```go
import (
    "time"
    "github.com/google/uuid"
)

type APIKeyRequest struct {
    ID             uuid.UUID     `json:"id" db:"id"`
    Email          string        `json:"email" db:"email"`
    CompanyName    string        `json:"company_name" db:"company_name"`
    UseCase        string        `json:"use_case" db:"use_case"`
    ExpectedVolume int           `json:"expected_volume" db:"expected_volume"`
    Status         string        `json:"status" db:"status"`
    RequestedAt    time.Time     `json:"requested_at" db:"requested_at"`
    ReviewedAt     *time.Time    `json:"reviewed_at" db:"reviewed_at"`
    ReviewedBy     string        `json:"reviewed_by" db:"reviewed_by"`
    Notes          string        `json:"notes" db:"notes"`
    CreatedAt      time.Time     `json:"created_at" db:"created_at"`
    UpdatedAt      time.Time     `json:"updated_at" db:"updated_at"`
}
```

### APIKey
```go
import (
    "time"
    "github.com/google/uuid"
)

type APIKey struct {
    ID             uuid.UUID      `json:"id" db:"id"`
    APIKey         string         `json:"api_key" db:"api_key"`
    RequestID      uuid.UUID      `json:"request_id" db:"request_id"`
    Email          string         `json:"email" db:"email"`
    CompanyName    string         `json:"company_name" db:"company_name"`
    TotalTokens    int            `json:"total_tokens" db:"total_tokens"`
    Status         string         `json:"status" db:"status"`
    ExpiredAt      *time.Time     `json:"expired_at" db:"expired_at"`
    CreatedAt      time.Time      `json:"created_at" db:"created_at"`
    UpdatedAt      time.Time      `json:"updated_at" db:"updated_at"`
    CreatedBy      string         `json:"created_by" db:"created_by"`
    Notes          string         `json:"notes" db:"notes"`
}
```

### APIKeyBalance
```go
type APIKeyBalance struct {
    ID              uuid.UUID  `json:"id" db:"id"`
    APIKeyID        uuid.UUID  `json:"api_key_id" db:"api_key_id"`
    RemainingTokens int        `json:"remaining_tokens" db:"remaining_tokens"`
    LastUsedAt      *time.Time `json:"last_used_at" db:"last_used_at"`
    CreatedAt       time.Time  `json:"created_at" db:"created_at"`
    UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}
```

## Implementation Details

### New Table Structure Considerations
- **Initial Setup**: When creating a new API key, insert into both `api_keys_info` and `api_key_balance` tables
- **Transaction Management**: Use database transactions to ensure both tables are updated atomically
- **Balance Initialization**: Set `remaining_tokens` equal to `total_tokens` from `api_keys_info`
- **Performance Monitoring**: Monitor query performance on `api_key_balance` table for optimization opportunities

### Token Consumption Strategy
- **Atomic Operation**: Use Redis `DECR` command for thread-safe token consumption
- **Check Before Consume**: Verify sufficient tokens before processing request
- **Rollback on Failure**: If request fails after token consumption, consider token refund logic
- **Dual-Table Updates**: Update both Redis and `api_key_balance` table atomically
- **Balance Table Optimization**: Use `UPDATE ... SET remaining_tokens = remaining_tokens - 1` for atomic decrement

### Error Handling
- **Invalid API Key**: Return 401 Unauthorized
- **Missing API Key**: Return 401 Unauthorized  
- **Insufficient Tokens**: Return 429 Too Many Requests
- **Storage Errors**: Log error and allow request (graceful degradation)

### Response Headers
- `X-Remaining-Tokens`: Current token count
- `X-Total-Tokens`: Original token allocation
- `X-Token-Reset`: When tokens will reset (if applicable)

## Configuration Options

### Token Options
```go
func TokenWithDefaultCount(count int) TokenOption
func TokenWithTTL(ttl time.Duration) TokenOption
func TokenWithHeaderName(name string) TokenOption
func TokenWithGracefulDegradation(enabled bool) TokenOption
```

### Integration with Cache Proxy
```go
func NewTokenProxy(host string, redisServer map[string]string, tokenConfig TokenConfig) http.Handler
```

## Admin Interface Requirements

### Request Management Endpoints
- `GET /admin/requests` - List all API key requests
- `GET /admin/requests/:id` - Get specific request details
- `PUT /admin/requests/:id/approve` - Approve request and create API key
- `PUT /admin/requests/:id/reject` - Reject request
- `POST /admin/requests/:id/notes` - Add admin notes

### API Key Management Endpoints
- `GET /admin/keys` - List all API keys
- `GET /admin/keys/:id` - Get specific API key details
- `PUT /admin/keys/:id/suspend` - Suspend API key
- `PUT /admin/keys/:id/add-tokens` - Add more tokens
- `DELETE /admin/keys/:id` - Revoke API key

## Email Notification System

### Email Templates
1. **Request Received**: Confirmation to user
2. **Request Approved**: API key details sent to user
3. **Request Rejected**: Rejection notice with feedback
4. **Low Tokens**: Warning when tokens < 10% remaining
5. **Tokens Exhausted**: Notification when tokens = 0

## Corner Cases & Solutions

### 1. Race Conditions
- **Problem**: Multiple concurrent requests with same API key
- **Solution**: Use Redis atomic operations (`DECR`, `GET`)

### 2. Token Count Accuracy
- **Problem**: Network failures during token consumption
- **Solution**: Implement idempotent operations and proper error handling

### 3. Graceful Degradation
- **Problem**: Redis unavailable
- **Solution**: Configurable fallback to allow all requests when token system is down

### 4. API Key Validation
- **Problem**: Malformed or expired keys
- **Solution**: Immediate rejection with appropriate HTTP status codes

### 5. Monitoring & Observability
- **Problem**: Lack of visibility into token usage
- **Solution**: Structured logging, metrics collection, and response headers

## Implementation Phases

### Phase 1: Core Token System (Current)
- Redis-based token storage
- Basic middleware implementation
- Simple API key validation

### Phase 2: Database Integration
- Database schema implementation
- Dual-write strategy (Redis + Database)
- Token usage logging

### Phase 3: Admin Interface
- Request management system
- API key approval workflow
- Email notification system

### Phase 4: Advanced Features
- Token analytics dashboard
- Usage pattern detection
- Automated approval rules

## Database Migration Path (Future)
- **Phase 1**: Redis-based token storage (current implementation)
- **Phase 2**: Database schema design for persistent storage
- **Phase 3**: Dual-write strategy (Redis + Database)
- **Phase 4**: Database-only with Redis as cache layer

## Security Considerations
- API key format validation
- Rate limiting on token consumption endpoints
- Audit logging for token modifications
- Secure token generation (if needed)
- Admin authentication & authorization
- Rate limiting on admin endpoints
- Audit logging for all admin actions

## Testing Strategy
- Unit tests for token manager logic
- Integration tests with Redis
- Load testing for concurrent token consumption
- Error scenario testing (Redis down, invalid keys)

## Monitoring & Metrics
- Token consumption rate per API key
- Failed token validations
- Redis performance metrics
- Response time impact of token checking
- Request approval/rejection metrics
- Token consumption patterns
- Admin action audit trail
- Email delivery success rates

## Additional Considerations

### Security
- API key generation (UUID v4 or similar)
- Admin authentication & authorization
- Rate limiting on admin endpoints
- Audit logging for all admin actions

### Monitoring
- Request approval/rejection metrics
- Token consumption patterns
- Admin action audit trail
- Email delivery success rates
