# 📋 nclip - Complete Parameter Reference

## 🚀 **Configuration Methods**

nclip supports configuration via:
- **Command-line flags** (with `-` or `--` prefix)
- **Environment variables** (with `NCLIP_` prefix)
- **Default values** (built-in)

**Precedence Order**: CLI flags > Environment variables > Default values

## 🚀 **All Supported Parameters**

### **Server Configuration**

| Parameter | Type | Default | Environment Variable | Description |
|-----------|------|---------|---------------------|-------------|
| `--url` | string | `http://localhost:8080/` | `NCLIP_URL` | Base URL template for generated paste URLs |
| `--http-port` | int | `8080` | `NCLIP_HTTP_PORT` | HTTP port for web interface |

### **Storage Configuration**

| Parameter | Type | Default | Environment Variable | Description |
|-----------|------|---------|---------------------|-------------|
| `--storage-type` | string | `filesystem` | `NCLIP_STORAGE_TYPE` | Storage backend: `filesystem`, `mongodb`, `dynamodb` |
| `--output-dir` | string | `./pastes` | `NCLIP_OUTPUT_DIR` | Directory to store paste files (filesystem only) |
| `--slug-length` | int | `8` | `NCLIP_SLUG_LENGTH` | Length of generated slug IDs (1-32) |
| `--buffer-size-mb` | int | `1` | - | Maximum paste size in MB (1-100) |

### **MongoDB Configuration**

| Parameter | Type | Default | Environment Variable | Description |
|-----------|------|---------|---------------------|-------------|
| `--mongodb-uri` | string | `mongodb://localhost:27017` | `NCLIP_MONGODB_URI` | MongoDB connection URI |
| `--mongodb-database` | string | `nclip` | `NCLIP_MONGODB_DATABASE` | MongoDB database name |
| `--mongodb-collection` | string | `pastes` | `NCLIP_MONGODB_COLLECTION` | MongoDB collection name |

### **DynamoDB Configuration**

| Parameter | Type | Default | Environment Variable | Description |
|-----------|------|---------|---------------------|-------------|
| `--dynamodb-table` | string | `nclip-pastes` | `NCLIP_DYNAMODB_TABLE` | DynamoDB table name |
| - | string | - | `AWS_REGION` | AWS region for DynamoDB |
| - | string | - | `AWS_ACCESS_KEY_ID` | AWS access key (or use IAM roles) |
| - | string | - | `AWS_SECRET_ACCESS_KEY` | AWS secret key (or use IAM roles) |

### **Paste Management**

| Parameter | Type | Default | Environment Variable | Description |
|-----------|------|---------|---------------------|-------------|
| `--expire-days` | int | `1` | `NCLIP_EXPIRE_DAYS` | Auto-delete pastes after N days (0 = no expiration) |
| `--rate-limit` | string | `10/min` | `NCLIP_RATE_LIMIT` | Rate limit per IP (e.g., `10/min`, `100/hour`) |

### **Logging & Operations**

| Parameter | Type | Default | Environment Variable | Description |
|-----------|------|---------|---------------------|-------------|
| `--log-level` | string | `info` | `NCLIP_LOG_LEVEL` | Log level: `debug`, `info`, `warn`, `error` |
| `--log-file` | string | `""` | `NCLIP_LOG_FILE` | Path to log file (empty = stdout) |
| `--user` | string | `""` | `NCLIP_USER` | User to run as (requires root) |

### **Feature Flags**

| Parameter | Type | Default | Environment Variable | Description |
|-----------|------|---------|---------------------|-------------|
| `--enable-webui` | bool | `true` | `NCLIP_ENABLE_WEBUI` | Enable web UI interface |
| `--enable-metrics` | bool | `true` | `NCLIP_ENABLE_METRICS` | Enable metrics endpoint |

## 💡 **Usage Examples**

### **Basic Usage**
```bash
# Start with defaults (filesystem storage)
./nclip
# Returns URLs like: http://localhost:8080/abc12345

# Custom URL with HTTPS and custom path
./nclip --url https://paste.example.com/clips/
# Returns URLs like: https://paste.example.com/clips/abc12345

# Custom URL with port and path
./nclip --url https://nclip.app:8443/paste/
# Returns URLs like: https://nclip.app:8443/paste/abc12345
```

### **Storage Backends**

#### **Filesystem Storage** (Default)
```bash
./nclip --storage-type filesystem --output-dir /var/lib/pastes
```

#### **MongoDB Storage**
```bash
./nclip --storage-type mongodb \
        --mongodb-uri mongodb://localhost:27017 \
        --mongodb-database nclip \
        --mongodb-collection pastes
```

#### **DynamoDB Storage** (AWS)
```bash
./nclip --storage-type dynamodb \
        --dynamodb-table my-nclip-table
# Requires AWS credentials via environment variables:
# export AWS_REGION=us-east-1
# export AWS_ACCESS_KEY_ID=AKIA...
# export AWS_SECRET_ACCESS_KEY=...
```

### **Production Configuration**
```bash
./nclip --url https://paste.company.com/clips/ \
        --storage-type mongodb \
        --mongodb-uri mongodb://mongo-cluster:27017 \
        --expire-days 30 \
        --rate-limit 50/min \
        --buffer-size-mb 5 \
        --log-level info \
        --log-file /var/log/nclip.log
# Returns URLs like: https://paste.company.com/clips/abc12345
```

### **Development Setup**
```bash
./nclip --url http://localhost:3000/dev/ \
        --storage-type filesystem \
        --output-dir ./dev-pastes \
        --expire-days 1 \
        --log-level debug
# Returns URLs like: http://localhost:3000/dev/abc12345
```

### **Mixed Configuration** (CLI overrides Environment)
```bash
# Set base config with environment variables
export NCLIP_STORAGE_TYPE=mongodb
export NCLIP_MONGODB_URI=mongodb://localhost:27017
export NCLIP_LOG_LEVEL=info

# Override specific settings with CLI flags
./nclip --url https://production.example.com/paste/ \
        --expire-days 30 \
        --rate-limit 200/hour \
        --log-level warn
# Returns URLs like: https://production.example.com/paste/abc12345
```

## 🌍 **Environment Variables**

All parameters can be set via environment variables with the `NCLIP_` prefix:

```bash
# Server configuration
export NCLIP_URL=https://paste.example.com/clips/
export NCLIP_HTTP_PORT=8080

# Storage configuration
export NCLIP_STORAGE_TYPE=mongodb
export NCLIP_MONGODB_URI=mongodb://localhost:27017
export NCLIP_EXPIRE_DAYS=30

# Feature flags
export NCLIP_ENABLE_WEBUI=true
export NCLIP_ENABLE_METRICS=true

# Start nclip (will use environment variables)
./nclip
# Returns URLs like: https://paste.example.com/clips/abc12345
```

## ⚙️ **Configuration Validation**

nclip validates all configuration on startup:

- **URL**: Must be a valid URL with protocol (http/https), can include port and path
- **Ports**: Must be 1-65535
- **Slug Length**: Must be 1-32 characters
- **Buffer Size**: Must be 1KB-100MB
- **Expire Days**: Cannot be negative (0 = no expiration)
- **Storage Type**: Must be one of: `filesystem`, `mongodb`, `dynamodb`

### **URL Format Examples**
```bash
# Valid URL formats
--url http://localhost:8080/                    # Default
--url https://paste.example.com/                # Simple HTTPS
--url https://nclip.app:8443/paste/             # Custom port and path
--url http://192.168.1.100:3000/clips/          # IP address with path
--url https://subdomain.example.com/api/v1/     # Complex path

# Invalid formats (will show validation errors)
--url paste.example.com                         # Missing protocol
--url https://                                  # Incomplete URL
--url ftp://example.com/                        # Invalid protocol
```

## 🔧 **Advanced Configuration**

### **URL Template Configuration**

The `--url` parameter defines the base URL template for generated paste URLs. The slug will be appended to create the final URL.

**Format**: `protocol://domain[:port][/path/]`

**Examples**:
```bash
# Default - Local development
--url http://localhost:8080/
# Creates: http://localhost:8080/abc12345

# Production with HTTPS and custom path
--url https://nclip.app/paste/
# Creates: https://nclip.app/paste/abc12345

# Custom port and path
--url https://paste.company.com:8443/clips/
# Creates: https://paste.company.com:8443/clips/abc12345

# Behind reverse proxy with subpath
--url https://api.example.com/v1/nclip/
# Creates: https://api.example.com/v1/nclip/abc12345
```

**Important Notes**:
- Always include trailing slash for paths
- Protocol is required (http or https)
- Port is optional (will use protocol default if omitted)
- Path is optional but recommended for organization
```bash
--rate-limit 10/min     # 10 requests per minute
--rate-limit 100/hour   # 100 requests per hour
--rate-limit 1000/day   # 1000 requests per day
--rate-limit 5/sec      # 5 requests per second
```

### **MongoDB URI Examples**
```bash
# Local MongoDB
--mongodb-uri mongodb://localhost:27017

# MongoDB with authentication
--mongodb-uri mongodb://user:pass@localhost:27017/database

# MongoDB cluster
--mongodb-uri mongodb://host1:27017,host2:27017,host3:27017/database?replicaSet=myReplicaSet

# MongoDB Atlas
--mongodb-uri mongodb+srv://user:pass@cluster.mongodb.net/database
```

### **AWS DynamoDB Setup**
```bash
# Method 1: Environment variables (development)
export AWS_REGION=us-east-1
export AWS_ACCESS_KEY_ID=AKIA...
export AWS_SECRET_ACCESS_KEY=...
./nclip --storage-type dynamodb --dynamodb-table nclip-prod

# Method 2: IAM roles (recommended for production)
# No credentials needed - uses instance/container role
./nclip --storage-type dynamodb --dynamodb-table nclip-prod

# Method 3: AWS credentials file
# Uses ~/.aws/credentials automatically
./nclip --storage-type dynamodb --dynamodb-table nclip-prod
```

## 🚀 **Quick Reference**

### **Minimal Start**
```bash
./nclip
# Starts on localhost:8080 (HTTP only)
# Returns URLs like: http://localhost:8080/abc12345
```

### **Production Ready**
```bash
./nclip --url https://paste.company.com/clips/ --storage-type mongodb --mongodb-uri mongodb://cluster:27017
# Returns URLs like: https://paste.company.com/clips/abc12345
```

### **High Performance**
```bash
./nclip --url https://nclip.app/paste/ --storage-type mongodb --mongodb-uri mongodb://cluster:27017 --rate-limit 100/min --buffer-size-mb 10
# Returns URLs like: https://nclip.app/paste/abc12345
```

### **Help**
```bash
./nclip --help
# Shows all parameters with descriptions and examples
```

## 📊 **Default Values Summary**

| Setting | Default Value | Purpose |
|---------|---------------|---------|
| **Base URL** | `http://localhost:8080/` | Local development with HTTP |
| **Port** | `8080` (HTTP) | Standard non-privileged port |
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
