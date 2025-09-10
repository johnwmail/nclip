# 📋 nclip - Complete Parameter Reference

## 🚀 **All Supported Parameters**

### **Server Configuration**

| Parameter | Type | Default | Environment Variable | Description |
|-----------|------|---------|---------------------|-------------|
| `-domain` | string | `localhost` | `NCLIP_DOMAIN` | Domain name for generated URLs |
| `-tcp-port` | int | `9999` | `NCLIP_TCP_PORT` | TCP port for netcat connections |
| `-http-port` | int | `8080` | `NCLIP_HTTP_PORT` | HTTP port for web interface |

### **Storage Configuration**

| Parameter | Type | Default | Environment Variable | Description |
|-----------|------|---------|---------------------|-------------|
| `-storage-type` | string | `filesystem` | `NCLIP_STORAGE_TYPE` | Storage backend: `filesystem`, `mongodb`, `dynamodb` |
| `-output-dir` | string | `./pastes` | `NCLIP_OUTPUT_DIR` | Directory to store paste files (filesystem only) |
| `-slug-length` | int | `8` | `NCLIP_SLUG_LENGTH` | Length of generated slug IDs (1-32) |
| `-buffer-size-mb` | int | `1` | `NCLIP_BUFFER_SIZE_MB` | Maximum paste size in MB (1-100) |

### **MongoDB Configuration**

| Parameter | Type | Default | Environment Variable | Description |
|-----------|------|---------|---------------------|-------------|
| `-mongodb-uri` | string | `mongodb://localhost:27017` | `NCLIP_MONGODB_URI` | MongoDB connection URI |
| `-mongodb-database` | string | `nclip` | `NCLIP_MONGODB_DATABASE` | MongoDB database name |
| `-mongodb-collection` | string | `pastes` | `NCLIP_MONGODB_COLLECTION` | MongoDB collection name |


### **DynamoDB Configuration**

| Parameter | Type | Default | Environment Variable | Description |
|-----------|------|---------|---------------------|-------------|
| `-dynamodb-table` | string | `nclip-pastes` | `NCLIP_DYNAMODB_TABLE` | DynamoDB table name |

### **Paste Management**

| Parameter | Type | Default | Environment Variable | Description |
|-----------|------|---------|---------------------|-------------|
| `-expire-days` | int | `1` | `NCLIP_EXPIRE_DAYS` | Auto-delete pastes after N days (0 = no expiration) |
| `-rate-limit` | string | `10/min` | `NCLIP_RATE_LIMIT` | Rate limit per IP (e.g., `10/min`, `100/hour`) |

### **Logging & Operations**

| Parameter | Type | Default | Environment Variable | Description |
|-----------|------|---------|---------------------|-------------|
| `-log-level` | string | `info` | `NCLIP_LOG_LEVEL` | Log level: `debug`, `info`, `warn`, `error` |
| `-log-file` | string | `""` | `NCLIP_LOG_FILE` | Path to log file (empty = stdout) |
| `-user` | string | `""` | `NCLIP_USER` | User to run as (requires root) |

### **Feature Flags**

| Parameter | Type | Default | Environment Variable | Description |
|-----------|------|---------|---------------------|-------------|
| `-enable-webui` | bool | `true` | `NCLIP_ENABLE_WEBUI` | Enable web UI interface |
| `-enable-metrics` | bool | `true` | `NCLIP_ENABLE_METRICS` | Enable metrics endpoint |

## 💡 **Usage Examples**

### **Basic Usage**
```bash
# Start with defaults (filesystem storage)
./nclip

# Custom domain and ports
./nclip -domain paste.example.com -tcp-port 9999 -http-port 8080

# Custom domain (HTTPS handled by reverse proxy)
./nclip -domain paste.example.com
```

### **Storage Backends**

#### **Filesystem Storage** (Default)
```bash
./nclip -storage-type filesystem -output-dir /var/lib/pastes
```

#### **MongoDB Storage**
```bash
./nclip -storage-type mongodb \
        -mongodb-uri mongodb://localhost:27017 \
        -mongodb-database nclip \
        -mongodb-collection pastes
```

#### **DynamoDB Storage** (AWS)
```bash
./nclip -storage-type dynamodb \
        -dynamodb-table my-nclip-table
# Requires AWS credentials via environment variables
```

### **Production Configuration**
```bash
./nclip -domain paste.company.com \
        -storage-type mongodb \
        -mongodb-uri mongodb://mongo-cluster:27017 \
        -expire-days 30 \
        -rate-limit 50/min \
        -buffer-size-mb 5 \
        -log-level info \
        -log-file /var/log/nclip.log
# Note: HTTPS handled by reverse proxy (nginx, HAProxy, etc.)
```

### **Development Setup**
```bash
./nclip -domain localhost:8080 \
        -storage-type filesystem \
        -output-dir ./dev-pastes \
        -expire-days 1 \
        -log-level debug
```

## 🌍 **Environment Variables**

All parameters can be set via environment variables with the `NCLIP_` prefix:

```bash
# Server configuration
export NCLIP_DOMAIN=paste.example.com
export NCLIP_HTTP_PORT=8080
export NCLIP_TCP_PORT=9999

# Storage configuration
export NCLIP_STORAGE_TYPE=mongodb
export NCLIP_MONGODB_URI=mongodb://localhost:27017
export NCLIP_EXPIRE_DAYS=30

# Feature flags
export NCLIP_ENABLE_WEBUI=true
export NCLIP_ENABLE_METRICS=true

# Start nclip (will use environment variables)
./nclip
```

## ⚙️ **Configuration Validation**

nclip validates all configuration on startup:

- **Domain**: Cannot be empty
- **Ports**: Must be 1-65535, TCP and HTTP ports must be different
- **Slug Length**: Must be 1-32 characters
- **Buffer Size**: Must be 1KB-100MB
- **Expire Days**: Cannot be negative (0 = no expiration)
- **Storage Type**: Must be one of: `filesystem`, `mongodb`, `dynamodb`

## 🔧 **Advanced Configuration**

### **Rate Limiting Formats**
```bash
-rate-limit 10/min     # 10 requests per minute
-rate-limit 100/hour   # 100 requests per hour
-rate-limit 1000/day   # 1000 requests per day
```

### **MongoDB URI Examples**
```bash
# Local MongoDB
-mongodb-uri mongodb://localhost:27017

# MongoDB with authentication
-mongodb-uri mongodb://user:pass@localhost:27017/database

# MongoDB cluster
-mongodb-uri mongodb://host1:27017,host2:27017,host3:27017/database?replicaSet=myReplicaSet

# MongoDB Atlas
-mongodb-uri mongodb+srv://user:pass@cluster.mongodb.net/database
```

## 🚀 **Quick Reference**

### **Minimal Start**
```bash
./nclip
# Starts on localhost:8080 (HTTP) and localhost:9999 (TCP)
```

### **Production Ready**
```bash
./nclip -domain paste.company.com -storage-type mongodb -mongodb-uri mongodb://cluster:27017
# HTTPS handled by reverse proxy
```

### **High Performance**
```bash
./nclip -storage-type mongodb -mongodb-uri mongodb://cluster:27017 -rate-limit 100/min -buffer-size-mb 10
```

### **Help**
```bash
./nclip --help
# Shows all parameters with descriptions and examples
```

## 📊 **Default Values Summary**

| Setting | Default Value | Purpose |
|---------|---------------|---------|
| **Domain** | `localhost` | Local development |
| **Ports** | `8080` (HTTP), `9999` (TCP) | Standard non-privileged ports |
| **Storage** | `filesystem` | No dependencies |
| **Expiration** | `1 day` | Serverless-friendly |
| **Rate Limit** | `10/min` | Conservative rate limiting |
| **Buffer Size** | `1 MB` | Reasonable paste size limit |
| **Log Level** | `info` | Balanced logging |
| **Features** | All enabled | Full functionality |

Perfect for getting started quickly while being production-ready! 🎯

## 🔒 **HTTPS/TLS Configuration**

**Important**: nclip does **NOT** handle HTTPS/TLS directly. This follows security best practices:

- **✅ Best Practice**: Use reverse proxy (nginx, HAProxy, Traefik) for HTTPS termination
- **✅ Security**: Dedicated tools handle TLS properly
- **✅ Performance**: Optimized TLS implementations
- **✅ Flexibility**: Easy certificate management and renewal

See `docs/HTTPS_TLS_CONFIGURATION.md` for complete setup examples with nginx, HAProxy, and Kubernetes ingress.
