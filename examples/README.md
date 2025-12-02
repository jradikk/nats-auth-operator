# NATS Auth Operator Examples

This directory contains complete, working examples for both token-based and JWT-based authentication modes.

## Directory Structure

```
examples/
├── token-auth/          # Simple token-based authentication examples
│   ├── README.md        # Token auth guide
│   ├── 01-auth-config.yaml
│   ├── 02-users.yaml
│   └── 03-deployment.yaml
│
└── jwt-auth/            # JWT-based authentication with accounts
    ├── README.md        # JWT auth guide
    ├── 01-auth-config.yaml
    ├── 02-accounts.yaml
    ├── 03-users.yaml
    ├── 04-deployments.yaml
    └── stream-based/    # Examples with JetStream permissions
        ├── README.md
        ├── accounts.yaml
        └── users.yaml
```

## Quick Start

### Token-Based Authentication

Simple username/password authentication, best for development or single-tenant deployments.

```bash
cd examples/token-auth
kubectl apply -f .
```

See [token-auth/README.md](token-auth/README.md) for details.

### JWT-Based Authentication

Full NATS operator/account/user hierarchy with fine-grained permissions and JetStream support.

```bash
cd examples/jwt-auth
kubectl apply -f .
```

See [jwt-auth/README.md](jwt-auth/README.md) for details.

## Choosing an Authentication Mode

| Feature | Token Auth | JWT Auth |
|---------|-----------|----------|
| **Complexity** | Low | Medium |
| **Multi-tenancy** | No | Yes |
| **Account isolation** | No | Yes |
| **Resource limits** | No | Yes (per account) |
| **Subject permissions** | Yes | Yes |
| **JetStream streams** | Global | Per account |
| **Credential rotation** | Manual | Supported |
| **Best for** | Dev/test, simple apps | Production, multi-tenant |

## Example Scenarios

### Token Auth Examples
1. **Simple application** - Single user with basic permissions
2. **Multi-service** - Multiple users with different permissions
3. **Read-only user** - Monitoring/auditing access

### JWT Auth Examples
1. **Multi-tenant SaaS** - Multiple accounts with isolated users
2. **Microservices** - Service mesh with account-level isolation
3. **JetStream workflows** - Stream-based permissions and limits
4. **Development teams** - Team accounts with shared resources

## Prerequisites

Before running these examples, ensure:

1. **NATS Auth Operator is installed**:
   ```bash
   kubectl get pods -n nats-auth-operator-system
   ```

2. **NATS server is deployed** (see [DEPLOYMENT.md](../DEPLOYMENT.md))

3. **kubectl access** to your cluster

## Testing Your Setup

After deploying an example, test the connection:

```bash
# Get the credentials secret name
kubectl get natsusers -o wide

# For JWT mode, test with credentials
kubectl run nats-test --rm -it --image=natsio/nats-box:latest -- \
  nats-sub --creds=/tmp/user.creds -s nats://nats.default.svc:4222 "test.>"

# For token mode, test with username/password
kubectl run nats-test --rm -it --image=natsio/nats-box:latest -- \
  nats-sub -s nats://user:password@nats.default.svc:4222 "test.>"
```

## Cleanup

Remove all example resources:

```bash
# Token auth
kubectl delete -f examples/token-auth/

# JWT auth
kubectl delete -f examples/jwt-auth/
```

## Next Steps

- Review the specific README in each example directory
- Modify permissions to match your use case
- Check [DEPLOYMENT.md](../DEPLOYMENT.md) for production guidance
- See [README.md](../README.md) for complete documentation
