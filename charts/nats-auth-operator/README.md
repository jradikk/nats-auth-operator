# NATS Auth Operator Helm Chart

This Helm chart deploys the NATS Auth Operator to your Kubernetes cluster. The operator manages NATS authentication using JWT and token-based authentication modes.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+

## Installation

### Install the CRDs

First, install the Custom Resource Definitions:

```bash
kubectl apply -f https://raw.githubusercontent.com/jradikk/nats-auth-operator/main/config/crd/bases/nats.jradikk_natsauthconfigs.yaml
kubectl apply -f https://raw.githubusercontent.com/jradikk/nats-auth-operator/main/config/crd/bases/nats.jradikk_natsaccounts.yaml
kubectl apply -f https://raw.githubusercontent.com/jradikk/nats-auth-operator/main/config/crd/bases/nats.jradikk_natsusers.yaml
```

### Install the Chart

Add the Helm repository (once published):

```bash
helm repo add nats-auth-operator https://jradikk.github.io/nats-auth-operator
helm repo update
```

Install the chart:

```bash
helm install nats-auth-operator nats-auth-operator/nats-auth-operator \
  --namespace nats-system \
  --create-namespace
```

Or install from local directory:

```bash
helm install nats-auth-operator ./charts/nats-auth-operator \
  --namespace nats-system \
  --create-namespace
```

## Configuration

### Values

The following table lists the configurable parameters of the NATS Auth Operator chart and their default values.

#### Image Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `image.repository` | Image repository | `jradikk/nats-auth-operator` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `image.tag` | Image tag (overrides chart appVersion) | `""` |
| `imagePullSecrets` | Image pull secrets | `[]` |

#### Controller Manager Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `controllerManager.replicas` | Number of controller manager replicas | `1` |
| `controllerManager.manager.image.repository` | Manager image repository | `jradikk/nats-auth-operator` |
| `controllerManager.manager.image.tag` | Manager image tag | `latest` |
| `controllerManager.manager.resources.limits.cpu` | CPU limit | `500m` |
| `controllerManager.manager.resources.limits.memory` | Memory limit | `128Mi` |
| `controllerManager.manager.resources.requests.cpu` | CPU request | `10m` |
| `controllerManager.manager.resources.requests.memory` | Memory request | `64Mi` |
| `controllerManager.manager.containerSecurityContext` | Security context for manager container | See values.yaml |
| `controllerManager.serviceAccount.annotations` | Annotations for controller manager service account | `{}` |

#### Kube-RBAC-Proxy Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `kubeRbacProxy.image.repository` | Kube-RBAC-Proxy image repository | `gcr.io/kubebuilder/kube-rbac-proxy` |
| `kubeRbacProxy.image.tag` | Kube-RBAC-Proxy image tag | `v0.15.0` |
| `kubeRbacProxy.resources.limits.cpu` | CPU limit | `500m` |
| `kubeRbacProxy.resources.limits.memory` | Memory limit | `128Mi` |
| `kubeRbacProxy.resources.requests.cpu` | CPU request | `5m` |
| `kubeRbacProxy.resources.requests.memory` | Memory request | `64Mi` |
| `kubeRbacProxy.containerSecurityContext` | Security context for kube-rbac-proxy | See values.yaml |

#### Service Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `metricsService.type` | Metrics service type | `ClusterIP` |
| `metricsService.annotations` | Annotations for metrics service | `{}` |
| `metricsService.ports` | Metrics service ports | See values.yaml |

#### Service Account

| Parameter | Description | Default |
|-----------|-------------|---------|
| `serviceAccount.create` | Create service account | `true` |
| `serviceAccount.automount` | Automount service account token | `true` |
| `serviceAccount.annotations` | Service account annotations | `{}` |
| `serviceAccount.name` | Service account name | `""` (generated) |

#### Pod Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `podAnnotations` | Pod annotations | `{}` |
| `podLabels` | Pod labels | `{}` |
| `podSecurityContext` | Pod security context | See values.yaml |
| `securityContext` | Container security context | See values.yaml |
| `nodeSelector` | Node selector | `{}` |
| `tolerations` | Tolerations | `[]` |
| `affinity` | Affinity rules | `{}` |

#### Health Probes

| Parameter | Description | Default |
|-----------|-------------|---------|
| `livenessProbe.httpGet.path` | Liveness probe path | `/healthz` |
| `livenessProbe.httpGet.port` | Liveness probe port | `8081` |
| `livenessProbe.initialDelaySeconds` | Initial delay | `15` |
| `livenessProbe.periodSeconds` | Period | `20` |
| `readinessProbe.httpGet.path` | Readiness probe path | `/readyz` |
| `readinessProbe.httpGet.port` | Readiness probe port | `8081` |
| `readinessProbe.initialDelaySeconds` | Initial delay | `5` |
| `readinessProbe.periodSeconds` | Period | `10` |

#### Other Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `leaderElection.enabled` | Enable leader election | `true` |
| `nameOverride` | Override chart name | `""` |
| `fullnameOverride` | Override full name | `""` |

### Example Custom Values

Create a `values.yaml` file with your custom values:

```yaml
# Custom image tag
controllerManager:
  manager:
    image:
      tag: "v0.1.0"

  # Increase resources for production
  manager:
    resources:
      limits:
        cpu: 1000m
        memory: 256Mi
      requests:
        cpu: 100m
        memory: 128Mi

  # Run multiple replicas
  replicas: 2

# Add node affinity for specific node pool
affinity:
  nodeAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      nodeSelectorTerms:
      - matchExpressions:
        - key: workload
          operator: In
          values:
          - operators

# Add tolerations for tainted nodes
tolerations:
- key: "workload"
  operator: "Equal"
  value: "operators"
  effect: "NoSchedule"
```

Install with custom values:

```bash
helm install nats-auth-operator ./charts/nats-auth-operator \
  --namespace nats-system \
  --create-namespace \
  --values values.yaml
```

## Usage

After installing the chart, create your NATS authentication resources:

### 1. Create NatsAuthConfig

```yaml
apiVersion: nats.jradikk/v1alpha1
kind: NatsAuthConfig
metadata:
  name: main
  namespace: default
spec:
  natsURL: "nats://nats.default.svc.cluster.local:4222"
  mode: jwt
  serverAuthConfig:
    name: "nats-auth-jwts"
    namespace: "default"
    type: "Secret"
  jwt:
    operatorName: "MyOperator"
```

### 2. Create NatsAccount

```yaml
apiVersion: nats.jradikk/v1alpha1
kind: NatsAccount
metadata:
  name: myapp-account
  namespace: default
spec:
  authConfigRef:
    name: main
    namespace: default
  description: "Application account with JetStream"
  limits:
    conn: 100
    subs: 1000
    payload: 1073741824
    data: -1
    exports: -1
    imports: -1
    wildcardExports: true
    jetstream:
      memoryStorage: -1
      diskStorage: -1
      streams: -1
      consumer: -1
```

### 3. Create NatsUser

```yaml
apiVersion: nats.jradikk/v1alpha1
kind: NatsUser
metadata:
  name: app-user
  namespace: default
spec:
  authConfigRef:
    name: main
  authType: jwt
  accountRef:
    name: myapp-account
  username: "app-user"
  permissions:
    publishAllow:
      - "app.>"
      - "$JS.ACK.>"
      - "$JS.API.>"
      - "_INBOX.>"
    subscribeAllow:
      - "app.>"
      - "$JS.API.>"
      - "_INBOX.>"
```

## Integrating with NATS Helm Chart

The operator creates a Secret with JWT credentials. Reference these in your NATS Helm chart values:

```yaml
nats:
  jetstream:
    enabled: true

container:
  env:
    # Operator JWT
    NATS_OPERATOR_JWT:
      valueFrom:
        secretKeyRef:
          name: nats-auth-jwts
          key: operator

    # Account JWTs
    NATS_MYAPP_ACCOUNT_JWT:
      valueFrom:
        secretKeyRef:
          name: nats-auth-jwts
          key: myapp-account

config:
  cluster:
    enabled: true

  # Use resolver_preload to load account JWTs
  resolver_preload: |
    $(NATS_MYAPP_ACCOUNT_JWT)

  # Configure operator
  operator: $(NATS_OPERATOR_JWT)
```

## Upgrading

### Upgrade the Chart

```bash
helm upgrade nats-auth-operator nats-auth-operator/nats-auth-operator \
  --namespace nats-system
```

### Upgrade CRDs

CRDs are not automatically upgraded by Helm. Upgrade them manually:

```bash
kubectl apply -f https://raw.githubusercontent.com/jradikk/nats-auth-operator/main/config/crd/bases/nats.jradikk_natsauthconfigs.yaml
kubectl apply -f https://raw.githubusercontent.com/jradikk/nats-auth-operator/main/config/crd/bases/nats.jradikk_natsaccounts.yaml
kubectl apply -f https://raw.githubusercontent.com/jradikk/nats-auth-operator/main/config/crd/bases/nats.jradikk_natsusers.yaml
```

## Uninstallation

### Uninstall the Chart

```bash
helm uninstall nats-auth-operator --namespace nats-system
```

### Clean up CRDs

**Warning:** This will delete all NatsAuthConfig, NatsAccount, and NatsUser resources.

```bash
kubectl delete crd natsauthconfigs.nats.jradikk
kubectl delete crd natsaccounts.nats.jradikk
kubectl delete crd natsusers.nats.jradikk
```

## Troubleshooting

### Check Operator Logs

```bash
kubectl logs -n nats-system deployment/nats-auth-operator-controller-manager -c manager
```

### Check Metrics

```bash
kubectl port-forward -n nats-system svc/nats-auth-operator-metrics-service 8443:8443
curl -k https://localhost:8443/metrics
```

### Common Issues

See the main [README.md](../../README.md#troubleshooting) for detailed troubleshooting guidance.

## Development

### Local Testing

Test the chart locally:

```bash
helm install test ./charts/nats-auth-operator \
  --namespace test \
  --create-namespace \
  --dry-run --debug
```

### Linting

```bash
helm lint ./charts/nats-auth-operator
```

## Contributing

Contributions are welcome! Please see the main repository [CONTRIBUTING.md](../../CONTRIBUTING.md) for details.

## License

See [LICENSE](../../LICENSE) for license information.
