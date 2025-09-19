# nclip Docker Setup

This repository includes Docker configuration for running nclip with MongoDB.

## Quick Start

1. **Start the services:**
   ```bash
   docker-compose up -d
   ```

2. **Check status:**
   ```bash
   docker-compose ps
   ```

3. **View logs:**
   ```bash
   docker-compose logs -f nclip
   docker-compose logs -f mongodb
   ```

4. **Access the application:**
   - Web UI: http://localhost:8080
   - API: http://localhost:8080

5. **Stop the services:**
   ```bash
   docker-compose down
   ```

## Services

### MongoDB
- **Image:** mongo:7.0
- **Port:** 27017
- **Database:** nclip
- **User:** nclip
- **Password:** secure_password_123
- **Features:**
  - Automatic TTL indexes for paste expiration
  - Optimized indexes for queries
  - Health checks
  - Persistent data storage

### nclip Application
- **Port:** 8080
- **Storage:** MongoDB (automatic)
- **Features:**
  - Web UI enabled
  - Prometheus metrics enabled
  - 24-hour paste expiration (default)
  - Health checks

## MongoDB Initialization

The MongoDB container automatically runs the initialization script (`scripts/mongodb-init.js`) on first startup, which:

1. Creates the `nclip` database
2. Creates a `nclip` user with appropriate permissions
3. Creates the `pastes` collection
4. Sets up essential indexes:
   - TTL index on `expires_at` for automatic cleanup
   - Unique index on `_id` for fast lookups
   - Index on `created_at` for chronological queries
   - Compound index for burn-after-read functionality

## Configuration

You can customize the application by modifying environment variables in `docker-compose.yml`:

```yaml
environment:
  NCLIP_URL: http://localhost:8080          # Base URL for paste links
  NCLIP_PORT: 8080                          # HTTP server port
  NCLIP_TTL: 24h                           # Default paste expiration
  NCLIP_ENABLE_METRICS: "true"            # Prometheus metrics
  NCLIP_ENABLE_WEBUI: "true"              # Web interface
  NCLIP_SLUG_LENGTH: 5                     # Paste ID length
  NCLIP_BUFFER_SIZE: 1048576               # Max upload size (1MB)
```

## Volumes

- `mongodb_data`: Persistent MongoDB data storage
- `./scripts/mongodb-init.js`: MongoDB initialization script

## Security Notes

**⚠️ For production use:**

1. Change the default MongoDB password in:
   - `docker-compose.yml` (MONGO_INITDB_ROOT_PASSWORD)
   - `docker-compose.yml` (NCLIP_MONGO_URL)
   - `scripts/mongodb-init.js` (user password)

2. Use Docker secrets or environment files for sensitive data
3. Configure proper network security
4. Enable MongoDB authentication and SSL/TLS
5. Set up proper backup procedures

## Troubleshooting

**MongoDB connection issues:**
```bash
# Check MongoDB logs
docker-compose logs mongodb

# Test MongoDB connection
docker-compose exec mongodb mongosh -u nclip -p secure_password_123 --authenticationDatabase admin nclip
```

**Application issues:**
```bash
# Check application logs
docker-compose logs nclip

# Check application health
curl http://localhost:8080/health
```

**Reset everything:**
```bash
# Stop and remove all containers and volumes
docker-compose down -v
docker-compose up -d
```