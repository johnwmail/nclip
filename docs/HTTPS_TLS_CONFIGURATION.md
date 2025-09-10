# 🔒 HTTPS/TLS Configuration for nclip

## 🎯 **Best Practice: Reverse Proxy Handles HTTPS**

**Important**: nclip does **NOT** handle HTTPS/TLS directly. This follows security best practices where the application focuses on business logic while dedicated tools handle TLS termination.

## ✅ **Why No Built-in HTTPS?**

1. **🛡️ Security**: Dedicated reverse proxies are hardened for TLS
2. **🔧 Separation of Concerns**: App handles business logic, proxy handles protocol
3. **📈 Performance**: Optimized TLS implementations in nginx/HAProxy
4. **🔄 Flexibility**: Easy certificate management and renewal
5. **⚖️ Load Balancing**: Reverse proxies provide load balancing + TLS

## 🚀 **Recommended Architecture**

```
Internet → Reverse Proxy (HTTPS) → nclip (HTTP)
          (nginx/HAProxy)          (port 8080)
```

## 🔧 **Configuration Examples**

### **nginx Configuration**
```nginx
server {
    listen 443 ssl http2;
    server_name paste.example.com;
    
    # SSL Configuration
    ssl_certificate /etc/ssl/certs/paste.example.com.crt;
    ssl_certificate_key /etc/ssl/private/paste.example.com.key;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES256-GCM-SHA384:ECDHE-RSA-CHACHA20-POLY1305;
    
    # Proxy to nclip
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # For netcat uploads via HTTP
        client_max_body_size 10M;
    }
}

# Redirect HTTP to HTTPS
server {
    listen 80;
    server_name paste.example.com;
    return 301 https://$server_name$request_uri;
}
```

### **HAProxy Configuration**
```haproxy
global
    ssl-default-bind-ciphers ECDHE+AESGCM:ECDHE+CHACHA20:DHE+AESGCM:DHE+CHACHA20:!aNULL:!SHA1:!AESCCM
    ssl-default-bind-options ssl-min-ver TLSv1.2 no-tls-tickets

frontend https_frontend
    bind *:443 ssl crt /etc/ssl/certs/paste.example.com.pem
    default_backend nclip_backend

frontend http_frontend
    bind *:80
    redirect scheme https code 301

backend nclip_backend
    server nclip1 localhost:8080 check
```

### **Traefik Configuration** (Docker)
```yaml
# docker-compose.yml
version: '3.8'
services:
  nclip:
    image: nclip:latest
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.nclip.rule=Host(`paste.example.com`)"
      - "traefik.http.routers.nclip.entrypoints=websecure"
      - "traefik.http.routers.nclip.tls.certresolver=letsencrypt"
      - "traefik.http.services.nclip.loadbalancer.server.port=8080"
    networks:
      - traefik

  traefik:
    image: traefik:v2.9
    command:
      - --entrypoints.websecure.address=:443
      - --entrypoints.web.address=:80
      - --certificatesresolvers.letsencrypt.acme.email=admin@example.com
      - --certificatesresolvers.letsencrypt.acme.storage=/letsencrypt/acme.json
      - --certificatesresolvers.letsencrypt.acme.httpchallenge.entrypoint=web
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - "/var/run/docker.sock:/var/run/docker.sock:ro"
      - "./letsencrypt:/letsencrypt"
    networks:
      - traefik

networks:
  traefik:
    external: true
```

## 🎛️ **nclip Configuration**

Start nclip with your domain name (reverse proxy will handle HTTPS):

```bash
# nclip serves HTTP only - reverse proxy handles HTTPS
./nclip -domain paste.example.com -http-port 8080

# Environment variable approach
export NCLIP_DOMAIN=paste.example.com
export NCLIP_HTTP_PORT=8080
./nclip
```

## 🌐 **Domain Configuration**

When using a reverse proxy, configure nclip with your public domain:

```bash
# ✅ Correct: Use public domain name
./nclip -domain paste.example.com

# ❌ Wrong: Don't use localhost in production
./nclip -domain localhost
```

nclip will generate URLs like `http://paste.example.com/abc123`, but users will access `https://paste.example.com/abc123` through the reverse proxy.

## 🔒 **Security Considerations**

### **Reverse Proxy Security**
- Use strong TLS ciphers (TLSv1.2+)
- Enable HSTS headers
- Configure proper SSL certificates
- Regular security updates

### **nclip Security**
- Bind to localhost only: `./nclip -http-port 8080`
- Use rate limiting: `./nclip -rate-limit 20/min`
- Configure proper log levels
- Regular application updates

## 🐳 **Docker Example**

```yaml
# docker-compose.yml
version: '3.8'
services:
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
      - ./ssl:/etc/ssl/certs
    depends_on:
      - nclip
  
  nclip:
    image: nclip:latest
    environment:
      NCLIP_DOMAIN: paste.example.com
      NCLIP_HTTP_PORT: 8080
    expose:
      - "8080"
    # Don't expose port 8080 to host - only nginx accesses it
```

## ☸️ **Kubernetes Example**

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: nclip-ingress
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
spec:
  tls:
  - hosts:
    - paste.example.com
    secretName: nclip-tls
  rules:
  - host: paste.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: nclip-service
            port:
              number: 8080
```

## 🎯 **Summary**

1. **nclip serves HTTP only** - no built-in HTTPS
2. **Reverse proxy handles TLS** - nginx, HAProxy, Traefik, etc.
3. **Configure domain properly** - use public domain name
4. **Security at proxy level** - certificates, ciphers, headers
5. **Keep it simple** - separation of concerns

This approach follows industry best practices and makes your deployment more secure, scalable, and maintainable! 🚀
