
# Security Checklist

## PostgreSQL Security & Performance

- [ ] Enable Row-Level Security (RLS) where appropriate
- [x] Use parameterized SQL queries
- [x] Apply least-privilege database roles
- [ ] Enforce NOT NULL, UNIQUE, CHECK, and foreign key constraints
- [x] Use transactions for multi-step operations
- [ ] Prevent overlapping property bookings
- [ ] Avoid SELECT * queries
- [ ] Prevent N+1 query problems
- [x] Set query execution timeouts
- [x] Configure database connection pooling
- [ ] Create indexes for frequently queried columns
- [ ] Implement pagination and query limits
- [ ] Configure automated database backups
- [ ] Test database restoration procedures
- [x] Use versioned database migrations
- [x] Never expose database credentials in logs
- [ ] Keep PostgreSQL inaccessible from the public internet
- [ ] Use separate development, staging, and production databases
- [x] Test RLS policies and database permissions
- [ ] Monitor slow queries and connection usage

## Go API Security

- [ ] Enforce HTTPS for all production requests
- [ ] Implement secure user authentication
- [ ] Enforce authorization for every protected resource
- [ ] Validate and sanitize inputs where appropriate
- [ ] Limit HTTP request body sizes
- [ ] Implement rate limiting
- [x] Configure a strict CORS origin allowlist
- [ ] Implement CSRF protection for cookie-based authentication
- [x] Configure HTTP server timeouts
- [ ] Secure session cookies and access tokens
- [ ] Return safe error messages without exposing internals
- [ ] Use idempotency keys for payment operations
- [ ] Verify incoming webhook signatures
- [ ] Log authentication failures and sensitive operations
- [ ] Avoid logging passwords, tokens, or sensitive user data
- [ ] Store production secrets in Railway environment variables
- [x] Protect Railway and GitHub accounts with MFA
- [ ] Restrict production deployment permissions
- [x] Keep Go dependencies updated
- [x] Run go vet and govulncheck
- [ ] Write tests for unauthorized and forbidden requests
- [ ] Test invalid inputs and oversized requests
- [ ] Configure health checks and service monitoring

## API Authentication Guidelines

- [ ] Use sessions or access tokens for authenticated guests
- [ ] Use API keys or service credentials for trusted integrations
- [ ] Never embed secret API keys in public clients
- [ ] Keep genuinely public endpoints accessible without authentication
- [ ] Verify ownership when accessing bookings and properties
- [ ] Use role-based or permission-based authorization where needed
- [ ] Implement secure token expiration and revocation
- [ ] Use short-lived credentials where practical

## Testing & Deployment

- [ ] Write failing tests before implementing critical business rules
- [ ] Test database transactions and rollbacks
- [ ] Test concurrent booking attempts
- [ ] Test authentication and authorization boundaries
- [x] Test database connection failures
- [ ] Test HTTP handler error responses
- [ ] Run automated tests before deployment
- [ ] Run migrations through a controlled deployment process
- [x] Keep production and testing credentials separate in application code
- [ ] Configure logging and alerts for critical failures
- [ ] Test backup recovery and incident response
