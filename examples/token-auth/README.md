# Token-Based Authentication Example

This example demonstrates simple username/password authentication for NATS.

## Overview

Token authentication is the simplest mode:
- Username and password credentials
- Subject-based permissions
- Good for development or single-tenant applications
- Easy to understand and debug

## What's Included

1. **01-auth-config.yaml** - NatsAuthConfig in token mode
2. **02-users.yaml** - Three users with different permission levels
3. **03-deployment.yaml** - Example application deployments

## Architecture

```
┌─────────────────────────────────────────┐
│  NatsAuthConfig (token mode)            │
│  Generates: auth.conf                   │
└─────────────────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────┐
│  NatsUsers                               │
│  • publisher (write access)             │
│  • subscriber (read access)             │
│  • admin (full access)                  │
└─────────────────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────┐
│  Secrets                                 │
│  • publisher-user-creds                 │
│  • subscriber-user-creds                │
│  • admin-user-creds                     │
└─────────────────────────────────────────┘
```

## Quick Start

### 1. Deploy the authentication configuration

```bash
kubectl apply -f 01-auth-config.yaml
```

Wait for it to be ready:

```bash
kubectl wait --for=condition=Ready natsauthconfig/token-auth --timeout=60s
```

### 2. Create users

```bash
kubectl apply -f 02-users.yaml
```

Verify users are created:

```bash
kubectl get natsusers
```

Expected output:
```
NAME         AUTH TYPE   STATE   ACCOUNT   AGE
publisher    token       Ready             10s
subscriber   token       Ready             10s
admin        token       Ready             10s
```

### 3. Restart NATS to pick up auth config

```bash
kubectl rollout restart statefulset/nats
kubectl rollout status statefulset/nats
```

### 4. Deploy example applications

```bash
kubectl apply -f 03-deployment.yaml
```

Check the logs:

```bash
# Publisher app
kubectl logs -f deployment/publisher-app

# Subscriber app
kubectl logs -f deployment/subscriber-app
```

## User Permissions

### Publisher User
**Username**: `publisher`
**Password**: Auto-generated (stored in Secret)

**Permissions**:
- ✅ Can publish to: `events.>`, `metrics.>`
- ❌ Cannot subscribe (read-only publishing)

**Use case**: Services that only send events/metrics

### Subscriber User
**Username**: `subscriber`
**Password**: Auto-generated

**Permissions**:
- ✅ Can subscribe to: `events.>`, `metrics.>`, `logs.>`
- ❌ Cannot publish (read-only)

**Use case**: Monitoring, logging, analytics services

### Admin User
**Username**: `admin`
**Password**: Auto-generated

**Permissions**:
- ✅ Full access to all subjects
- ✅ Can publish and subscribe anywhere

**Use case**: Administrative tools, debugging

## Testing

### Test Publisher

```bash
# Get credentials
kubectl get secret publisher-user-creds -o jsonpath='{.data.USERNAME}' | base64 -d
kubectl get secret publisher-user-creds -o jsonpath='{.data.PASSWORD}' | base64 -d

# Publish a message
kubectl run -it --rm nats-pub --image=natsio/nats-box:latest -- \
  nats-pub -s nats://publisher:<password>@nats.default.svc:4222 \
  "events.test" "Hello from publisher"
```

### Test Subscriber

```bash
# Get credentials
kubectl get secret subscriber-user-creds -o jsonpath='{.data.USERNAME}' | base64 -d
kubectl get secret subscriber-user-creds -o jsonpath='{.data.PASSWORD}' | base64 -d

# Subscribe to events
kubectl run -it --rm nats-sub --image=natsio/nats-box:latest -- \
  nats-sub -s nats://subscriber:<password>@nats.default.svc:4222 \
  "events.>"
```

### Test Permission Denial

Try to publish with subscriber (should fail):

```bash
kubectl run -it --rm nats-test --image=natsio/nats-box:latest -- \
  nats-pub -s nats://subscriber:<password>@nats.default.svc:4222 \
  "events.test" "This should fail"
```

Expected error: `Permissions Violation for Publish`

## Retrieving Credentials

### Method 1: Using kubectl

```bash
# Get all credentials for a user
kubectl get secret publisher-user-creds -o yaml

# Get just username
kubectl get secret publisher-user-creds -o jsonpath='{.data.USERNAME}' | base64 -d

# Get just password
kubectl get secret publisher-user-creds -o jsonpath='{.data.PASSWORD}' | base64 -d

# Get NATS URL
kubectl get secret publisher-user-creds -o jsonpath='{.data.NATS_URL}' | base64 -d
```

### Method 2: Mount as environment variables

```yaml
env:
- name: NATS_URL
  valueFrom:
    secretKeyRef:
      name: publisher-user-creds
      key: NATS_URL
- name: NATS_USER
  valueFrom:
    secretKeyRef:
      name: publisher-user-creds
      key: USERNAME
- name: NATS_PASSWORD
  valueFrom:
    secretKeyRef:
      name: publisher-user-creds
      key: PASSWORD
```

## Modifying Permissions

To add or change permissions, edit the NatsUser resource:

```yaml
apiVersion: nats.example.com/v1alpha1
kind: NatsUser
metadata:
  name: publisher
spec:
  # ... existing config ...
  permissions:
    publishAllow:
      - "events.>"
      - "metrics.>"
      - "alerts.>"  # Add new subject
```

Apply the changes:

```bash
kubectl apply -f 02-users.yaml
```

The operator will automatically update the auth configuration.

## Adding New Users

Create a new user by adding to `02-users.yaml`:

```yaml
---
apiVersion: nats.example.com/v1alpha1
kind: NatsUser
metadata:
  name: my-new-user
  namespace: default
spec:
  authConfigRef:
    name: token-auth
  authType: token
  username: "my-new-user"
  passwordFrom:
    generate: true
  permissions:
    publishAllow:
      - "my.app.>"
    subscribeAllow:
      - "my.app.>"
```

## Security Best Practices

1. **Use generated passwords** - Don't specify passwords in manifests
2. **Principle of least privilege** - Only grant necessary permissions
3. **Rotate credentials** - Periodically regenerate passwords
4. **Limit subject access** - Use specific subjects, not wildcards
5. **Monitor access** - Watch NATS logs for unauthorized attempts

## Troubleshooting

### User can't connect

```bash
# Check user status
kubectl describe natsuser publisher

# Check if secret was created
kubectl get secret publisher-user-creds

# Check NATS server logs
kubectl logs statefulset/nats | grep -i auth
```

### Permission denied errors

```bash
# Verify user permissions
kubectl get natsuser publisher -o jsonpath='{.spec.permissions}' | jq .

# Check NATS auth config
kubectl get configmap nats-token-auth -o jsonpath='{.data.auth\.conf}'
```

### Password not working

```bash
# Get the actual password
kubectl get secret publisher-user-creds -o jsonpath='{.data.PASSWORD}' | base64 -d

# Check if it matches what you're using
```

## Cleanup

Remove all resources:

```bash
kubectl delete -f .
```

Or individually:

```bash
kubectl delete -f 03-deployment.yaml
kubectl delete -f 02-users.yaml
kubectl delete -f 01-auth-config.yaml
```

## Next Steps

- Try the [JWT auth example](../jwt-auth/) for multi-tenancy
- Review [DEPLOYMENT.md](../../DEPLOYMENT.md) for production setup
- Check [README.md](../../README.md) for complete documentation
