# Docker/Kubernetes Storage Recommendations

## 🏆 **Recommended Storage for Docker/K8s: MongoDB**

For permanent Docker/Kubernetes deployments, **MongoDB is my top recommendation** because:

### Why MongoDB?

1. **🔄 Native Kubernetes Support**
   - Official MongoDB Kubernetes Operator
   - Helm charts available
   - Built-in replica sets and sharding

2. **⚡ Performance & Scalability**
   - Horizontal scaling with sharding
   - Efficient indexing for fast queries
   - Connection pooling and caching

3. **🛡️ Data Durability**
   - ACID transactions
   - Built-in replication
   - Point-in-time recovery

4. **🕒 TTL Support**
   - Native document expiration
   - No manual cleanup needed
   - Configurable per document

5. **📊 Rich Features**
   - Complex queries for analytics
   - Aggregation pipelines
   - Full-text search capabilities

## 📋 **Deployment Options**

### Option 1: Docker Compose (Development/Small Production)

```bash
# Start with MongoDB
docker-compose up -d
```

**Access:**
- Application: http://localhost:8080
- Netcat: `echo "test" | nc localhost 9999`
- MongoDB Admin: http://localhost:8081 (admin/admin123)

### Option 2: Kubernetes (Production)

#### MongoDB Deployment:
```bash
# Deploy MongoDB-based nclip
kubectl apply -f k8s/nclip-mongodb.yaml

# Check status
kubectl get pods -n nclip
kubectl get services -n nclip
```

## 🔧 **Configuration Comparison**

| Feature | MongoDB | Filesystem | DynamoDB |
|---------|---------|------------|----------|
| **Performance** | High | Medium | High |
| **Durability** | Excellent | Good | Excellent |
| **Scalability** | Horizontal | None | Unlimited |
| **Memory Usage** | Efficient | Low | N/A |
| **TTL Support** | Native | Manual | Native |
| **Backup/Restore** | Built-in | File copy | AWS Backup |
| **Complexity** | Medium | Very Low | Medium |
| **Production Ready** | ✅ | ⚠️ | ✅ |

## 🚀 **Quick Start**

### 1. MongoDB (Recommended for self-hosted)
```bash
# Build and start
docker-compose up --build

# Test
echo "Hello MongoDB!" | curl -d @- http://localhost:8080
echo "Hello MongoDB!" | nc localhost 9999
```

### 2. Filesystem (Simple development)
```bash
# Start with filesystem storage
./nclip -storage-type filesystem -output-dir ./pastes

# Test
echo "Hello Filesystem!" | curl -d @- http://localhost:8080
```

### 3. Kubernetes Production
```bash
# Update image registry in k8s/*.yaml
sed -i 's/your-registry/your-actual-registry/' k8s/*.yaml

# Update passwords
sed -i 's/your-secure-password/actual-secure-password/' k8s/*.yaml

# Deploy
kubectl apply -f k8s/nclip-mongodb.yaml

# Get external IP
kubectl get services -n nclip
```

## 📊 **Performance Recommendations**

### For High Traffic (>1000 req/min):
- **Primary**: MongoDB with sharding and proper indexing
- **Scaling**: Horizontal pod autoscaling
- **Alternative**: DynamoDB for AWS deployments

### For General Use (100-1000 req/min):
- **Primary**: MongoDB with proper indexing
- **Scaling**: Horizontal pod autoscaling
- **Storage**: SSD storage class for performance

### For Small Deployments (<100 req/min):
- **Primary**: MongoDB single instance
- **Alternative**: Filesystem with regular backups
- **Scaling**: Single replica, manual scaling

## 🔒 **Security Considerations**

### MongoDB:
```yaml
# Enable authentication
MONGO_INITDB_ROOT_USERNAME: admin
MONGO_INITDB_ROOT_PASSWORD: secure-password

# Use dedicated user
db.createUser({
  user: "nclip",
  pwd: "app-password", 
  roles: ["readWrite"]
})
```

## 🎯 **Final Recommendation**

**For Docker/Kubernetes permanent deployments, use MongoDB** because:

1. **Best balance** of performance, durability, and features
2. **Production-proven** in containerized environments  
3. **Native Kubernetes support** with operators
4. **Automatic TTL expiration** without manual cleanup
5. **Horizontal scaling** capabilities
6. **Rich querying** for potential future features (search, analytics)

**For AWS Lambda deployments, use DynamoDB** for serverless architecture compatibility.

## 📚 **Next Steps**

1. **Development**: Start with `docker-compose up`
2. **Staging**: Deploy to Kubernetes with single replicas
3. **Production**: Enable multi-replica MongoDB with proper monitoring
4. **AWS Lambda**: Use DynamoDB for serverless deployments

Your nclip service is now ready for serious production deployment! 🎉
