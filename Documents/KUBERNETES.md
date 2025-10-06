# Kubernetes Deployment Guide for nclip

This guide provides detailed instructions for deploying nclip on Kubernetes, including manifest customization, scaling, ingress, and troubleshooting.

## Table of Contents

- [Quick Start](#quick-start)
- [Prerequisites](#prerequisites)
- [Manifest Overview](#manifest-overview)
- [Configuration](#configuration)
- [Ingress & TLS](#ingress--tls)
- [Scaling & High Availability](#scaling--high-availability)
- [Storage](#storage)
- [Monitoring & Health Checks](#monitoring--health-checks)
- [Troubleshooting](#troubleshooting)
- [Security](#security)

## Quick Start

### Deploy with kubectl

```bash
# Clone the repository
git clone https://github.com/johnwmail/nclip.git
cd nclip

# Create namespace (optional)
kubectl apply -f k8s/namespace.yaml

# Deploy nclip
kubectl apply -f k8s/

# Check deployment status
kubectl get pods -n nclip
kubectl get svc -n nclip
```

### Access the Application

```bash
# Port forward for local access
kubectl port-forward -n nclip svc/nclip 8080:8080

# Visit http://localhost:8080
```

## Prerequisites

- Kubernetes cluster (1.19+)
- kubectl configured to access your cluster
- (Optional) Ingress controller for external access
- (Optional) Persistent volume provisioner for data persistence

## Manifest Overview

| File | Purpose | Type |
|------|---------|------|
| `k8s/namespace.yaml` | Namespace for isolation | Namespace |
| `k8s/deployment.yaml` | nclip application deployment | Deployment |
| `k8s/service.yaml` | Service to expose nclip | Service |
| `k8s/ingress.yaml` | Ingress for external access | Ingress |
| `k8s/pvc.yaml` | Persistent volume claim for data | PersistentVolumeClaim |
| `k8s/kustomization.yaml` | Kustomize configuration | Kustomization |

## Configuration

### Environment Variables

Configure nclip through environment variables in the deployment:

```yaml
env:
- name: NCLIP_URL
  value: "https://demo.nclip.app"
- name: NCLIP_PORT
  value: "8080"
- name: NCLIP_TTL
  value: "24h"
```

### Upload Auth (API Keys) in Kubernetes

To enable upload authentication use `NCLIP_UPLOAD_AUTH=true` and provide the API keys via a Kubernetes Secret. Store keys as a single comma-separated string in the secret.

Example: create a secret containing API keys:

```bash
kubectl create secret generic nclip-api-keys \
  --from-literal=api-keys="secret1,secret2" -n nclip
```

Then reference it in your `deployment.yaml`:

```yaml
env:
  - name: NCLIP_UPLOAD_AUTH
    value: "true"
  - name: NCLIP_API_KEYS
    valueFrom:
      secretKeyRef:
        name: nclip-api-keys
        key: api-keys
```

Notes:

- Rotate keys by updating the secret and restarting pods (or use rolling update).
- Do not put keys directly in manifests or repo. Use sealed secrets or external secret managers for production.


### Resource Limits

Set appropriate resource requests and limits for production:

```yaml
resources:
  requests:
    memory: "128Mi"
    cpu: "100m"
  limits:
    memory: "512Mi"
    cpu: "500m"
```

### Service Types

Choose the appropriate service type:

- **ClusterIP**: Internal cluster access only
- **LoadBalancer**: Cloud provider load balancer
- **NodePort**: Direct node port access

## Storage

nclip uses filesystem storage by default. For production deployments with persistent data:

```bash
# Apply the persistent volume claim
kubectl apply -f k8s/pvc.yaml
```

Update `deployment.yaml` to mount the persistent volume:

```yaml
volumeMounts:
- name: nclip-data
  mountPath: /data

volumes:
- name: nclip-data
  persistentVolumeClaim:
    claimName: nclip-pvc
```

## Monitoring & Health Checks

nclip includes built-in health checks accessible at `/health`:

```bash
# Check health endpoint
curl http://your-nclip-service/health

# View application logs
kubectl logs -n nclip deployment/nclip

# Monitor resource usage
kubectl top pods -n nclip
```

## Ingress & TLS

Configure ingress for external access:

```bash
# Install ingress controller (nginx example)
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.8.1/deploy/static/provider/cloud/deploy.yaml

# Apply nclip ingress
kubectl apply -f k8s/ingress.yaml
```

For HTTPS, create and configure TLS secrets:

```bash
# Create TLS secret from certificate files
kubectl create secret tls nclip-tls --cert=tls.crt --key=tls.key -n nclip

# Or use cert-manager for automatic certificates
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml
```

Update `ingress.yaml` to reference the TLS secret:

```yaml
tls:
- hosts:
  - demo.nclip.app
  secretName: nclip-tls
```

## Scaling & High Availability

### Horizontal Scaling

```bash
# Scale deployment to multiple replicas
kubectl scale deployment nclip --replicas=3 -n nclip
```

### Pod Anti-Affinity

Configure anti-affinity to spread pods across nodes:

```yaml
affinity:
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
    - weight: 100
      podAffinityTerm:
        labelSelector:
          matchExpressions:
          - key: app
            operator: In
            values:
            - nclip
        topologyKey: kubernetes.io/hostname
```

### Pod Disruption Budget

Ensure high availability during cluster maintenance:

```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: nclip-pdb
spec:
  minAvailable: 1
  selector:
    matchLabels:
      app: nclip
```

## Troubleshooting

- Check pod logs: `kubectl logs deployment/nclip`
- Describe resources for error details: `kubectl describe pod <pod>`
- Use `kubectl get all -A` to see all resources and their status.
- For local testing, port-forward: `kubectl port-forward svc/nclip 8080:8080`

---

## Advanced: Kustomize Overlays

- Use overlays for staging/production differences (e.g., resources, domains, secrets).
- See [Kustomize documentation](https://kubectl.docs.kubernetes.io/references/kustomize/) for more.

---

## Security Notes

- Change all default passwords and secrets before production use.
- Restrict ingress to trusted IPs if possible.
- Use network policies for pod-level security.

---

## References

- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [Kustomize Documentation](https://kubectl.docs.kubernetes.io/references/kustomize/)

---

For questions or contributions, see the main [README.md](../README.md) or open an issue.
