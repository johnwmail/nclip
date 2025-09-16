# Docker & Kubernetes Deployment Guide

This guide shows how to deploy nclip on Docker and Kubernetes with various storage backends.

## 🗄️ Storage Backend Options

### 1. MongoDB (Recommended for self-hosted)
- **TTL Support**: Automatic document expiration
- **Mature**: Battle-tested in production
- **Flexible**: JSON-like documents
- **Performance**: Excellent for read-heavy workloads

### 2. DynamoDB (Recommended for AWS)
- **Serverless**: No infrastructure management
- **TTL Support**: Built-in expiration
- **Scalable**: Auto-scaling capabilities
- **Use case**: AWS Lambda deployments

### 3. Filesystem (Development/Simple deployments)
- **Simple**: No external dependencies
- **Limitations**: No TTL, single instance only
- **Use case**: Local development and testing

## 🐳 Docker Deployment

### Docker Compose with MongoDB

**docker-compose.yml:**
```yaml
version: '3.8'

services:
  # nclip Application
  nclip:
    build: .
    ports:
      - "8080:8080"
    environment:
      - NCLIP_STORAGE_TYPE=mongodb
      - NCLIP_MONGODB_URI=mongodb://mongo:27017
      - NCLIP_MONGODB_DATABASE=nclip
      - NCLIP_MONGODB_COLLECTION=pastes
      - NCLIP_EXPIRE_DAYS=1
      - NCLIP_DOMAIN=localhost
      - NCLIP_LOG_LEVEL=info
    depends_on:
      - mongo
    restart: unless-stopped

  # MongoDB Database
  mongo:
    image: mongo:7.0
    ports:
      - "27017:27017"
    volumes:
      - mongodb_data:/data/db
      - ./mongo-init.js:/docker-entrypoint-initdb.d/mongo-init.js:ro
    environment:
      - MONGO_INITDB_DATABASE=nclip
    restart: unless-stopped

  # MongoDB Admin UI (Optional)
  mongo-express:
    image: mongo-express:latest
    ports:
      - "8081:8081"
    environment:
      - ME_CONFIG_MONGODB_SERVER=mongo
      - ME_CONFIG_MONGODB_PORT=27017
      - ME_CONFIG_BASICAUTH_USERNAME=admin
      - ME_CONFIG_BASICAUTH_PASSWORD=admin123
    depends_on:
      - mongo
    restart: unless-stopped

volumes:
  mongodb_data:
```

**Dockerfile:**
```dockerfile
# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o nclip .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/nclip .

# Create non-root user
RUN adduser -D -s /bin/sh nclip
USER nclip

# Expose ports
EXPOSE 8080

# Run the application
CMD ["./nclip"]
```

**mongo-init.js:**
```javascript
// MongoDB initialization script
db = db.getSiblingDB('nclip');

// Create TTL index for automatic expiration (24 hours)
db.pastes.createIndex(
  { "expires_at": 1 }, 
  { expireAfterSeconds: 0 }
);

print("TTL index created for automatic paste expiration");
```

## ☸️ Kubernetes Deployment

### MongoDB on Kubernetes

**k8s/namespace.yaml:**
```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: nclip
```

**k8s/mongodb-deployment.yaml:**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mongodb
  namespace: nclip
spec:
  replicas: 1
  selector:
    matchLabels:
      app: mongodb
  template:
    metadata:
      labels:
        app: mongodb
    spec:
      containers:
      - name: mongodb
        image: mongo:7.0
        ports:
        - containerPort: 27017
        env:
        - name: MONGO_INITDB_DATABASE
          value: "nclip"
        volumeMounts:
        - name: mongodb-data
          mountPath: /data/db
        - name: mongo-init-script
          mountPath: /docker-entrypoint-initdb.d
      volumes:
      - name: mongodb-data
        persistentVolumeClaim:
          claimName: mongodb-pvc
      - name: mongo-init-script
        configMap:
          name: mongo-init-config

---
apiVersion: v1
kind: Service
metadata:
  name: mongodb-service
  namespace: nclip
spec:
  selector:
    app: mongodb
  ports:
  - port: 27017
    targetPort: 27017
  type: ClusterIP

---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: mongodb-pvc
  namespace: nclip
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi

---
apiVersion: v1
kind: ConfigMap
metadata:
  name: mongo-init-config
  namespace: nclip
data:
  mongo-init.js: |
    db = db.getSiblingDB('nclip');
    db.pastes.createIndex(
      { "expires_at": 1 }, 
      { expireAfterSeconds: 0 }
    );
    print("TTL index created for automatic paste expiration");
```

**k8s/nclip-deployment.yaml:**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nclip
  namespace: nclip
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nclip
  template:
    metadata:
      labels:
        app: nclip
    spec:
      containers:
      - name: nclip
        image: your-registry/nclip:latest
        ports:
        - containerPort: 8080
        env:
        - name: NCLIP_STORAGE_TYPE
          value: "mongodb"
        - name: NCLIP_MONGODB_URI
          value: "mongodb://mongodb-service:27017"
        - name: NCLIP_MONGODB_DATABASE
          value: "nclip"
        - name: NCLIP_MONGODB_COLLECTION
          value: "pastes"
        - name: NCLIP_EXPIRE_DAYS
          value: "1"
        - name: NCLIP_DOMAIN
          value: "your-domain.com"
        - name: NCLIP_LOG_LEVEL
          value: "info"
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5

---
apiVersion: v1
kind: Service
metadata:
  name: nclip-service
  namespace: nclip
spec:
  selector:
    app: nclip
  ports:
  - name: http
  - name: http
    port: 80
    targetPort: 8080
  type: LoadBalancer
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: nclip-ingress
  namespace: nclip
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
spec:
  tls:
  - hosts:
    - your-domain.com
    secretName: nclip-tls
  rules:
  - host: your-domain.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: nclip-service
            port:
              number: 80
```

## 🚀 Deployment Commands

### Docker Deployment

```bash
# Clone and build
git clone <your-repo>
cd nclip

# Start with MongoDB
docker-compose up -d

# View logs
docker-compose logs -f nclip

# Scale the application
docker-compose up -d --scale nclip=3
```

### Kubernetes Deployment

```bash
# Create namespace
kubectl apply -f k8s/namespace.yaml

# Deploy MongoDB
kubectl apply -f k8s/mongodb-deployment.yaml

# Deploy nclip
kubectl apply -f k8s/nclip-deployment.yaml

# Check status
kubectl get pods -n nclip
kubectl get services -n nclip

# View logs
kubectl logs -f deployment/nclip -n nclip

# Scale application
kubectl scale deployment nclip --replicas=5 -n nclip
```

## 📊 Storage Comparison

| Feature | MongoDB | DynamoDB | Filesystem |
|---------|---------|----------|------------|
| **Performance** | Fast | Fastest | Slow |
| **TTL Support** | ✅ | ✅ | Manual |
| **Persistence** | ✅ | ✅ | ✅ |
| **Scaling** | Good | Excellent | Poor |
| **Memory Usage** | Moderate | N/A | Low |
| **Complexity** | Medium | Low | Low |

## 🔧 Required Dependencies

Add to your `go.mod`:

```bash
# For MongoDB
go get go.mongodb.org/mongo-driver/mongo
go get go.mongodb.org/mongo-driver/bson

# For DynamoDB
go get github.com/aws/aws-sdk-go-v2/service/dynamodb
```

## 💰 Cost Estimation (Monthly)

### Small Scale (1000 pastes/day)
- **Container**: $10-20 (1-2 vCPU, 2-4GB RAM)
- **MongoDB**: $15-30 (managed service) or $5 (self-hosted)
- **Total**: $15-50/month

### Medium Scale (10,000 pastes/day)
- **Containers**: $30-60 (load balanced)
- **MongoDB**: $50-100 (replica set)
- **Total**: $80-160/month

This setup gives you:
✅ High availability with replica sets
✅ Automatic scaling with Kubernetes HPA
✅ Automatic paste expiration with TTL
✅ Monitoring and logging
✅ SSL/TLS termination
