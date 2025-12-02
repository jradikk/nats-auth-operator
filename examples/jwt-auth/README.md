# JWT-Based Authentication Example

This example demonstrates full NATS JWT authentication with operator/account/user hierarchy and fine-grained permissions.

## Overview

JWT authentication provides:
- **Multi-tenancy** - Isolated accounts for different teams/applications
- **Account-level resource limits** - Connections, subscriptions, payload size
- **Fine-grained permissions** - Subject-based publish/subscribe control
- **JetStream support** - Stream and consumer permissions per account
- **Credential files** - Secure `.creds` files for easy client authentication

## What's Included

1. **01-auth-config.yaml** - NatsAuthConfig in JWT mode with operator
2. **02-accounts.yaml** - Three accounts (production, staging, shared-services)
3. **03-users.yaml** - Multiple users across accounts with different permissions
4. **04-deployments.yaml** - Example microservices using JWT authentication
5. **stream-based/** - JetStream examples with stream permissions

## Architecture

```
┌─────────────────────────────────────────┐
│  NATS Operator                           │
│  (Auto-generated keypair)               │
└─────────────────────────────────────────┘
              │
              ├─────────────────────────┬─────────────────────────┐
              ▼                         ▼                         ▼
┌───────────────────────┐  ┌───────────────────────┐  ┌───────────────────────┐
│  Production Account   │  │  Staging Account      │  │  Shared Services      │
│  Limits: 100 conn     │  │  Limits: 50 conn      │  │  Limits: 200 conn     │
└───────────────────────┘  └───────────────────────┘  └───────────────────────┘
    │                          │                          │
    ├──────┬──────┐            ├──────┐                  ├──────┐
    ▼      ▼      ▼            ▼      ▼                  ▼      ▼
  api-   order- payment-     api-  test-            logging- monitoring-
  service service service   service user            service   service
```

## Quick Start

### 1. Deploy JWT authentication configuration

```bash
kubectl apply -f 01-auth-config.yaml
```

Wait for the operator to be created:

```bash
kubectl wait --for=condition=Ready natsauthconfig/main --timeout=60s
```

Check operator public key:

```bash
kubectl get natsauthconfig main -o jsonpath='{.status.operatorPubKey}'
```

### 2. Create accounts

```bash
kubectl apply -f 02-accounts.yaml
```

Verify accounts:

```bash
kubectl get natsaccounts
```

Expected output:
```
NAME               ACCOUNT ID                                              READY   AGE
production         ACVXYZ...                                              True    10s
staging            ACVABC...                                              True    10s
shared-services    ACVDEF...                                              True    10s
```

### 3. Create users

```bash
kubectl apply -f 03-users.yaml
```

Verify users:

```bash
kubectl get natsusers -o wide
```

### 4. Restart NATS

```bash
kubectl rollout restart statefulset/nats
kubectl rollout status statefulset/nats
```

### 5. Deploy example applications

```bash
kubectl apply -f 04-deployments.yaml
```

Watch the logs:

```bash
kubectl logs -f deployment/api-service
kubectl logs -f deployment/order-service
```

## Accounts

### Production Account

**Purpose**: Live production services
**Limits**:
- Max connections: 100
- Max subscriptions: 1,000
- Max payload: 1 MB
- Data: Unlimited

**Users**:
- `api-service` - API gateway with full permissions
- `order-service` - Order processing (specific subjects)
- `payment-service` - Payment processing (specific subjects)

### Staging Account

**Purpose**: Testing and development
**Limits**:
- Max connections: 50
- Max subscriptions: 500
- Max payload: 1 MB

**Users**:
- `api-service-staging` - Staging API
- `test-user` - For manual testing

### Shared Services Account

**Purpose**: Cross-environment services (logging, monitoring)
**Limits**:
- Max connections: 200
- Max subscriptions: 2,000
- Max payload: 10 MB (for logs)

**Users**:
- `logging-service` - Centralized logging
- `monitoring-service` - Metrics and monitoring

## User Permissions

### API Service (Production)
**Account**: production
**Permissions**:
- ✅ Publish: `api.>`, `events.>`
- ✅ Subscribe: `orders.>`, `payments.>`, `inventory.>`

**Use case**: API gateway that routes requests to backend services

### Order Service (Production)
**Account**: production
**Permissions**:
- ✅ Publish: `orders.created`, `orders.updated`, `orders.completed`
- ✅ Subscribe: `orders.commands.>`, `payments.completed`

**Use case**: Order processing microservice

### Payment Service (Production)
**Account**: production
**Permissions**:
- ✅ Publish: `payments.>`, `notifications.payment.>`
- ✅ Subscribe: `orders.>`, `payments.requests.>`

**Use case**: Payment processing service

### Logging Service (Shared)
**Account**: shared-services
**Permissions**:
- ✅ Subscribe: `logs.>`, `events.>`, `errors.>`
- ❌ Cannot publish (read-only)

**Use case**: Centralized log aggregation

## Testing

### Test with .creds file

```bash
# Get the credentials file
kubectl get secret api-service-user-creds -o jsonpath='{.data.user\.creds}' | base64 -d > /tmp/api.creds

# Test connection
kubectl run -it --rm nats-test --image=natsio/nats-box:latest -- \
  sh -c "echo '...' > /tmp/api.creds && nats-pub --creds=/tmp/api.creds -s nats://nats.default.svc:4222 'api.test' 'Hello'"
```

### Test subject permissions

Try to publish to a denied subject:

```bash
# This should fail - order-service can't publish to api.* subjects
kubectl run -it --rm nats-test --image=natsio/nats-box:latest -- \
  nats-pub --creds=/tmp/order.creds -s nats://nats.default.svc:4222 'api.unauthorized' 'Should fail'
```

Expected error: `Permissions Violation`

### Test account isolation

Users in different accounts cannot communicate:

```bash
# Production user publishes
nats-pub --creds=/tmp/prod-api.creds "api.test" "Hello"

# Staging user tries to subscribe (will not receive message - different account)
nats-sub --creds=/tmp/staging-api.creds "api.test"
```

## Retrieving Credentials

### Get .creds file

```bash
# Get credentials for a specific user
kubectl get secret api-service-user-creds -o jsonpath='{.data.user\.creds}' | base64 -d

# Save to file
kubectl get secret api-service-user-creds -o jsonpath='{.data.user\.creds}' | base64 -d > api-service.creds
```

### Mount in Pod

```yaml
spec:
  containers:
  - name: app
    env:
    - name: NATS_CREDS_FILE
      value: /nats/user.creds
    volumeMounts:
    - name: nats-creds
      mountPath: /nats
      readOnly: true
  volumes:
  - name: nats-creds
    secret:
      secretName: api-service-user-creds
      items:
      - key: user.creds
        path: user.creds
```

## JetStream Examples

See [stream-based/README.md](stream-based/README.md) for examples of:
- Creating streams with account-level isolation
- Stream-specific publish/subscribe permissions
- Consumer permissions
- Stream limits per account

## Modifying Permissions

### Add new subject to user

```bash
kubectl edit natsuser api-service
```

Add to `publishAllow`:

```yaml
permissions:
  publishAllow:
    - "api.>"
    - "events.>"
    - "notifications.>"  # New subject
```

The operator will automatically update the user JWT.

### Change account limits

```bash
kubectl edit natsaccount production
```

Update limits:

```yaml
limits:
  conn: 200  # Increase from 100
  subs: 2000 # Increase from 1000
```

## Adding Users to Existing Accounts

Create a new user in the production account:

```yaml
apiVersion: nats.example.com/v1alpha1
kind: NatsUser
metadata:
  name: notification-service
  namespace: default
spec:
  authConfigRef:
    name: main
  authType: jwt
  accountRef:
    name: production  # Use existing account
  permissions:
    publishAllow:
      - "notifications.>"
    subscribeAllow:
      - "orders.>"
      - "payments.>"
```

## Security Best Practices

1. **Account Isolation** - Use separate accounts for different environments
2. **Least Privilege** - Grant minimum required permissions
3. **Subject Namespacing** - Use hierarchical subjects (e.g., `service.action.resource`)
4. **Credential Rotation** - Regularly regenerate user JWTs
5. **Monitor Limits** - Watch for accounts hitting connection/subscription limits
6. **Secure .creds files** - Store in Kubernetes Secrets, never in code

## Troubleshooting

### User can't connect

```bash
# Check user status
kubectl describe natsuser api-service

# Verify account is ready
kubectl describe natsaccount production

# Check credentials secret
kubectl get secret api-service-user-creds -o yaml

# Verify JWT is valid
kubectl get secret api-service-user-creds -o jsonpath='{.data.user\.jwt}' | base64 -d
```

### Account limit reached

```bash
# Check account status
kubectl describe natsaccount production

# Check current limits
kubectl get natsaccount production -o jsonpath='{.spec.limits}'

# Increase limits if needed
kubectl edit natsaccount production
```

### Permission errors

```bash
# Verify user permissions
kubectl get natsuser api-service -o jsonpath='{.spec.permissions}' | jq .

# Check NATS server logs
kubectl logs statefulset/nats | grep -i permission
```

## Cleanup

```bash
kubectl delete -f 04-deployments.yaml
kubectl delete -f 03-users.yaml
kubectl delete -f 02-accounts.yaml
kubectl delete -f 01-auth-config.yaml
```

## Next Steps

- Explore [stream-based examples](stream-based/README.md) for JetStream
- Review [DEPLOYMENT.md](../../DEPLOYMENT.md) for production setup
- Check [README.md](../../README.md) for complete documentation
