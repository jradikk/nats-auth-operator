# JetStream Stream-Based Permissions Example

This example demonstrates how to use JWT authentication with NATS JetStream for stream-based messaging with fine-grained permissions.

## Overview

JetStream provides:
- **Persistent messaging** - Messages stored and replayed
- **Stream isolation** - Streams scoped to accounts
- **Consumer permissions** - Control who can consume from streams
- **Delivery guarantees** - At-least-once, exactly-once delivery
- **Subject-based filtering** - Consume specific subjects from streams

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│  Production Account                                      │
│                                                          │
│  Streams:                                               │
│  ├─ ORDERS (subjects: orders.>)                        │
│  ├─ PAYMENTS (subjects: payments.>)                    │
│  └─ EVENTS (subjects: events.>)                        │
│                                                          │
│  Users:                                                 │
│  ├─ stream-publisher (write to streams)                │
│  ├─ stream-consumer (read from streams)                │
│  └─ stream-admin (manage streams)                      │
└─────────────────────────────────────────────────────────┘
```

## What's Included

1. **accounts.yaml** - Account with JetStream limits
2. **users.yaml** - Users with stream-specific permissions
3. **stream-setup.sh** - Script to create JetStream streams
4. **examples.sh** - Usage examples

## Account Configuration with JetStream Limits

The account is configured with JetStream-specific limits:

```yaml
apiVersion: nats.example.com/v1alpha1
kind: NatsAccount
metadata:
  name: production-streams
spec:
  limits:
    # Standard limits
    conn: 100
    subs: 1000
    payload: 1048576

    # JetStream limits (set via account JWT)
    # These would need to be added to the AccountLimits type
    # For now, configure in NATS server directly
```

## User Permissions

### Stream Publisher

**Can**:
- ✅ Publish to stream subjects: `orders.>`, `payments.>`, `events.>`
- ✅ Create ephemeral consumers (for acknowledgments)

**Cannot**:
- ❌ Create streams
- ❌ Create durable consumers
- ❌ Subscribe to non-stream subjects

### Stream Consumer

**Can**:
- ✅ Subscribe to stream subjects
- ✅ Create consumers on existing streams
- ✅ Acknowledge messages

**Cannot**:
- ❌ Publish to streams
- ❌ Create or delete streams
- ❌ Modify stream configuration

### Stream Admin

**Can**:
- ✅ Create, modify, delete streams
- ✅ Create, modify, delete consumers
- ✅ Publish and subscribe
- ✅ View stream info and metrics

**Cannot**:
- ❌ Access other accounts' streams

## Quick Start

### 1. Deploy Account and Users

```bash
kubectl apply -f accounts.yaml
kubectl apply -f users.yaml
```

### 2. Wait for Resources

```bash
kubectl wait --for=condition=Ready natsaccount/production-streams --timeout=60s
kubectl get natsusers -l account=production-streams
```

### 3. Create Streams

Get the admin credentials and create streams:

```bash
# Get admin credentials
kubectl get secret stream-admin-user-creds -o jsonpath='{.data.user\.creds}' | base64 -d > /tmp/admin.creds

# Create ORDERS stream
nats stream add ORDERS \
  --creds=/tmp/admin.creds \
  --subjects="orders.>" \
  --storage=file \
  --retention=limits \
  --max-msgs=1000000 \
  --max-age=7d

# Create PAYMENTS stream
nats stream add PAYMENTS \
  --creds=/tmp/admin.creds \
  --subjects="payments.>" \
  --storage=file \
  --retention=limits \
  --max-msgs=500000 \
  --max-age=30d

# Create EVENTS stream
nats stream add EVENTS \
  --creds=/tmp/admin.creds \
  --subjects="events.>" \
  --storage=file \
  --retention=interest \
  --max-msgs=10000000 \
  --max-age=1d
```

### 4. Publish to Streams

```bash
# Get publisher credentials
kubectl get secret stream-publisher-user-creds -o jsonpath='{.data.user\.creds}' | base64 -d > /tmp/publisher.creds

# Publish order
nats pub orders.created \
  --creds=/tmp/publisher.creds \
  '{"order_id":"123","customer":"John Doe","amount":99.99}'

# Publish payment
nats pub payments.processed \
  --creds=/tmp/publisher.creds \
  '{"payment_id":"PAY-456","order_id":"123","status":"completed"}'

# Verify messages in stream
nats stream info ORDERS --creds=/tmp/admin.creds
```

### 5. Consume from Streams

```bash
# Get consumer credentials
kubectl get secret stream-consumer-user-creds -o jsonpath='{.data.user\.creds}' | base64 -d > /tmp/consumer.creds

# Create consumer on ORDERS stream
nats consumer add ORDERS ORDER_PROCESSOR \
  --creds=/tmp/consumer.creds \
  --filter="orders.>" \
  --ack=explicit \
  --pull \
  --deliver=all \
  --max-deliver=3

# Consume messages
nats consumer next ORDERS ORDER_PROCESSOR \
  --creds=/tmp/consumer.creds \
  --count=10 \
  --ack
```

## Subject-Based Stream Filtering

### Wildcard Subscriptions

Users can filter stream messages by subject:

```bash
# Only consume orders.created messages
nats consumer add ORDERS CREATED_ORDERS \
  --filter="orders.created" \
  --creds=/tmp/consumer.creds

# Only consume payment failures
nats consumer add PAYMENTS FAILED_PAYMENTS \
  --filter="payments.failed" \
  --creds=/tmp/consumer.creds
```

### Subject Permissions

The NatsUser permissions control which subjects a user can:

1. **Publish to stream**: `publishAllow` must include the subject
2. **Subscribe from stream**: `subscribeAllow` must include the subject
3. **Create consumers**: Requires subscribe permission on filtered subject

Example:

```yaml
permissions:
  publishAllow:
    - "orders.created"      # Can only publish orders.created
    - "orders.updated"      # Can only publish orders.updated
  subscribeAllow:
    - "orders.>"            # Can consume all order messages
```

## Stream Retention Policies

### Limits-Based Retention

Messages kept until limits reached:

```bash
nats stream add ORDERS \
  --retention=limits \
  --max-msgs=1000000 \
  --max-bytes=1GB \
  --max-age=7d
```

### Interest-Based Retention

Messages kept only while consumers exist:

```bash
nats stream add EVENTS \
  --retention=interest \
  --max-age=1d
```

### Work Queue Retention

Messages deleted after acknowledgment:

```bash
nats stream add TASKS \
  --retention=workqueue \
  --max-msgs=100000
```

## Consumer Configurations

### Pull Consumer (Manual Control)

```bash
nats consumer add ORDERS MANUAL_PROCESSOR \
  --pull \
  --ack=explicit \
  --max-deliver=3 \
  --max-waiting=10
```

### Push Consumer (Auto Delivery)

```bash
nats consumer add ORDERS AUTO_PROCESSOR \
  --deliver-subject="process.orders" \
  --ack=explicit \
  --max-deliver=3
```

### Ephemeral vs Durable

```bash
# Durable consumer (survives restarts)
nats consumer add ORDERS DURABLE_PROC --durable

# Ephemeral consumer (deleted when inactive)
nats consumer add ORDERS --ephemeral
```

## Examples

### Example 1: Order Processing Pipeline

```bash
# Publisher publishes orders
nats pub orders.created \
  --creds=/tmp/publisher.creds \
  '{"order_id":"ORD-001","items":[...]}'

# Consumer processes orders
nats consumer next ORDERS ORDER_PROCESSOR \
  --creds=/tmp/consumer.creds \
  --count=1 \
  --ack

# Publisher publishes update
nats pub orders.completed \
  --creds=/tmp/publisher.creds \
  '{"order_id":"ORD-001","status":"shipped"}'
```

### Example 2: Failed Message Retry

```bash
# Message fails processing (not acked)
nats consumer next ORDERS ORDER_PROCESSOR --no-ack

# Message redelivered automatically after AckWait timeout
# (up to MaxDeliver times)
```

### Example 3: Filtered Consumption

```bash
# Create consumer for specific order types
nats consumer add ORDERS HIGH_PRIORITY_ORDERS \
  --filter="orders.priority.high" \
  --creds=/tmp/consumer.creds

# Only high-priority orders delivered
nats consumer next ORDERS HIGH_PRIORITY_ORDERS --count=10
```

## Permission Testing

### Test Publisher Permissions

```bash
# Should succeed - allowed subject
nats pub orders.created '{"test":true}' --creds=/tmp/publisher.creds

# Should fail - denied subject
nats pub admin.commands '{"test":true}' --creds=/tmp/publisher.creds
# Expected: Permissions Violation
```

### Test Consumer Permissions

```bash
# Should succeed - can create consumer
nats consumer add ORDERS TEST_CONSUMER --creds=/tmp/consumer.creds

# Should fail - cannot create stream
nats stream add TEST_STREAM --creds=/tmp/consumer.creds
# Expected: Permission denied
```

### Test Stream Admin Permissions

```bash
# Should succeed - full admin access
nats stream add NEW_STREAM --creds=/tmp/admin.creds
nats stream rm NEW_STREAM --creds=/tmp/admin.creds
```

## Monitoring

### View Stream Info

```bash
# Stream statistics
nats stream info ORDERS --creds=/tmp/admin.creds

# Consumer statistics
nats consumer info ORDERS ORDER_PROCESSOR --creds=/tmp/admin.creds

# List all consumers
nats consumer ls ORDERS --creds=/tmp/admin.creds
```

### Check Message Counts

```bash
# Total messages in stream
nats stream info ORDERS --creds=/tmp/admin.creds | grep "Messages:"

# Consumer lag
nats consumer info ORDERS ORDER_PROCESSOR --creds=/tmp/admin.creds | grep "Pending:"
```

## Troubleshooting

### Cannot Create Stream

```bash
# Check user permissions
kubectl get natsuser stream-admin -o jsonpath='{.spec.permissions}'

# Verify account limits
kubectl get natsaccount production-streams -o jsonpath='{.spec.limits}'
```

### Cannot Publish to Stream

```bash
# Verify subject permissions
kubectl get natsuser stream-publisher -o jsonpath='{.spec.permissions.publishAllow}'

# Check if stream exists
nats stream ls --creds=/tmp/admin.creds
```

### Messages Not Being Consumed

```bash
# Check consumer exists
nats consumer ls ORDERS --creds=/tmp/admin.creds

# Check consumer state
nats consumer info ORDERS ORDER_PROCESSOR --creds=/tmp/admin.creds

# Check pending messages
nats stream info ORDERS --creds=/tmp/admin.creds | grep "Pending"
```

## Best Practices

1. **Use Pull Consumers** for controlled message processing
2. **Set MaxDeliver** to prevent infinite retries
3. **Use Subject Filters** to reduce consumer load
4. **Monitor Consumer Lag** to detect processing issues
5. **Set Appropriate Retention** based on use case
6. **Use Durable Consumers** for important workloads
7. **Implement Dead Letter Queues** for failed messages
8. **Set Account Limits** to prevent resource exhaustion

## Cleanup

```bash
# Delete consumers
nats consumer rm ORDERS ORDER_PROCESSOR --creds=/tmp/admin.creds

# Delete streams
nats stream rm ORDERS --creds=/tmp/admin.creds
nats stream rm PAYMENTS --creds=/tmp/admin.creds
nats stream rm EVENTS --creds=/tmp/admin.creds

# Delete Kubernetes resources
kubectl delete -f users.yaml
kubectl delete -f accounts.yaml
```

## Next Steps

- Review [JetStream documentation](https://docs.nats.io/nats-concepts/jetstream)
- Explore [consumer patterns](https://docs.nats.io/nats-concepts/jetstream/consumers)
- Learn about [stream retention](https://docs.nats.io/nats-concepts/jetstream/streams#retention-policies)
