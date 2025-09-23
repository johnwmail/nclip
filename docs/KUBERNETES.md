# Kubernetes Deployment Guide for nclip

This guide provides detailed instructions for deploying nclip on Kubernetes, including manifest customization, scaling, ingress, and troubleshooting.

---

## Quick Start

1. **Clone the repository:**
   ```bash
   git clone https://github.com/johnwmail/nclip.git
   cd nclip
   ```

2. **Create the namespace (optional):**
   ```bash
   kubectl apply -f k8s/namespace.yaml
   ```

3. **Deploy MongoDB:**
   ```bash
   kubectl apply -f k8s/mongodb/
   ```

4. **Deploy nclip app and service:**
   ```bash
   kubectl apply -f k8s/deployment.yaml
   kubectl apply -f k8s/service.yaml
   ```

5. **(Optional) Deploy ingress:**
   ```bash
   kubectl apply -f k8s/ingress.yaml
   ```

6. **(Optional) Use kustomize for overlays:**
   ```bash
   kubectl apply -k k8s/
   ```

---

## Manifest Overview

- `k8s/namespace.yaml`: Namespace for isolation
- `k8s/mongodb/`: MongoDB deployment, service, PVC, secret, configmap
- `k8s/deployment.yaml`: nclip Deployment
- `k8s/service.yaml`: nclip Service (ClusterIP/LoadBalancer)
- `k8s/ingress.yaml`: Ingress for external HTTP(S) access
- `k8s/kustomization.yaml`: Kustomize support

---

## Customization

- **Secrets:** Edit `k8s/mongodb/secret.yaml` for MongoDB credentials.
- **Config:** Adjust `k8s/mongodb/configmap.yaml` and environment variables in `deployment.yaml` as needed.
- **Storage:** Edit `k8s/mongodb/pvc.yaml` for persistent volume size/class.
- **Resources:** Set CPU/memory requests/limits in deployments for production.
- **Service Type:** Change `service.yaml` to `LoadBalancer` for cloud, or use `NodePort` for local testing.

---

## Ingress & TLS

- Edit `k8s/ingress.yaml` to set your domain and TLS settings.
- Ensure an ingress controller (e.g., nginx, traefik) is installed in your cluster.
- For HTTPS, configure TLS secrets and reference them in the ingress manifest.

---

## Scaling & High Availability

- Increase `replicas` in `deployment.yaml` for nclip.
- Use a managed MongoDB or a StatefulSet for MongoDB in production.
- Consider anti-affinity and pod disruption budgets for resilience.

---

## Troubleshooting

- Check pod logs: `kubectl logs deployment/nclip`
- Check MongoDB pod logs: `kubectl logs deployment/nclip-mongodb -n <namespace>`
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