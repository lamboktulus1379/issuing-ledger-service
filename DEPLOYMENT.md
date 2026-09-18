# Go Project Nginx Deployment

This repository contains automated deployment scripts for setting up the Go project with nginx as a reverse proxy.

## Quick Start

```bash
# Deploy and start all services
./service.sh start

# Check service status
./service.sh status

# View logs
./service.sh logs

# Stop services
./service.sh stop
```

## Architecture

```
HTTP Client
    ↓
http://localhost:10020 → (301 redirect) → https://gra.tulus.tech
    ↓
Nginx (SSL Termination)
    ↓
https://localhost:10001 (Go Application)
```

## Services

### Go Application
- **Direct HTTPS Access**: `https://localhost:10001`
- **Features**: YouTube API integration, OAuth authentication, video management
- **Root Endpoint**: Returns `{"error":true,"message":"Not Found","path":"/"}`

### Nginx Reverse Proxy
- **HTTP Redirect**: `http://localhost:10020` → `https://gra.tulus.tech`
- **HTTPS Proxy**: `https://gra.tulus.tech` → `https://localhost:10001`
- **SSL/TLS**: Automated certificate management with mkcert

## Scripts

### `deploy-nginx.sh`
Main deployment script that handles:
- SSL certificate setup (mkcert integration)
- Nginx configuration generation
- Go application startup
- Service health checks
- Automated testing

### `service.sh`
Service management wrapper for easy operations:
- `start` - Deploy and start all services
- `stop` - Stop all services
- `restart` - Restart all services
- `status` - Check service status
- `logs` - View application logs
- `test` - Run deployment tests

## Prerequisites

1. **Homebrew** (macOS package manager)
2. **Nginx** installed via Homebrew:
   ```bash
   brew install nginx
   ```
3. **Go** programming language
4. **mkcert** for SSL certificates:
   ```bash
   brew install mkcert
   ```

## Installation

1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd go-project
   ```

2. Run the deployment script:
   ```bash
   ./service.sh start
   ```

3. Verify deployment:
   ```bash
   ./service.sh test
   ```

## Configuration

### Ports
- **Go Application**: 10001 (HTTPS)
- **HTTP Redirect**: 10020 → gra.tulus.tech
- **Nginx**: 80 (HTTP), 443 (HTTPS)

### SSL Certificates
- Location: `/opt/homebrew/etc/nginx/certs/`
- Auto-generated with mkcert for local development
- Supports `gra.tulus.tech`, `localhost`, `127.0.0.1`

### Nginx Configuration
- Main config: `/opt/homebrew/etc/nginx/nginx.conf`
- Site config: `/opt/homebrew/etc/nginx/servers/gra.tulus.tech.conf`
- Logs: `/opt/homebrew/var/log/nginx/`

## Testing Endpoints

### Root Endpoint (Should return 404)
```bash
curl -k https://localhost:10001/
# Expected: {"error":true,"message":"Not Found","path":"/"}

curl -k https://gra.tulus.tech/
# Expected: {"error":true,"message":"Not Found","path":"/"}
```

### API Endpoints
```bash
# YouTube videos (test endpoint)
curl -k https://localhost:10001/test/youtube/videos

# Health check
curl -k https://localhost:10001/healthz

# Through proxy
curl -k https://gra.tulus.tech/test/youtube/videos
```

### Redirect Test
```bash
curl -I http://localhost:10020/
# Expected: 301 redirect to https://gra.tulus.tech/
```

## Troubleshooting

### Check Service Status
```bash
./service.sh status
```

### View Logs
```bash
# Go application logs
./service.sh logs

# Nginx error logs
sudo tail -f /opt/homebrew/var/log/nginx/error.log

# Nginx access logs
sudo tail -f /opt/homebrew/var/log/nginx/access.log
```

### Port Conflicts
If ports are in use, the script will automatically kill conflicting processes. You can also manually check:

```bash
# Check what's using specific ports
lsof -i :10001
lsof -i :10020
lsof -i :443

# Kill processes on specific port
lsof -ti :10001 | xargs kill -9
```

### Certificate Issues
```bash
# Regenerate certificates
cd certs/
mkcert gra.tulus.tech localhost 127.0.0.1 ::1

# Copy to nginx directory
sudo cp gra.tulus.tech*.pem /opt/homebrew/etc/nginx/certs/
```

### Nginx Configuration Test
```bash
# Test nginx configuration
sudo nginx -t

# Reload nginx
sudo nginx -s reload
```

## Development Workflow

### Making Changes
1. Stop services: `./service.sh stop`
2. Make your changes to the Go code
3. Restart services: `./service.sh start`
4. Test: `./service.sh test`

### Viewing Logs During Development
```bash
# Watch Go application logs in real-time
./service.sh logs

# In another terminal, make requests to test
curl -k https://localhost:10001/api/some-endpoint
```

## Production Considerations

For production deployment, consider:

1. **Replace mkcert certificates** with proper SSL certificates from a CA
2. **Update domain configuration** in `deploy-nginx.sh`
3. **Configure firewall rules** for appropriate port access
4. **Set up log rotation** for application and nginx logs
5. **Configure monitoring** and health checks
6. **Set up automated backups** of configuration and data

## Security Features

The nginx configuration includes:
- **SSL/TLS termination** with modern cipher suites
- **Security headers** (X-Frame-Options, X-XSS-Protection, etc.)
- **HTTP to HTTPS redirection**
- **Proxy timeout configurations**
- **Buffer size limitations**

## Monitoring

### Health Checks
- **Go App Health**: `https://localhost:10001/healthz`
- **Proxy Health**: `https://gra.tulus.tech/health`

### Log Files
- **Go Application**: `/tmp/go-app-deployment.log`
- **Nginx Access**: `/opt/homebrew/var/log/nginx/access.log`
- **Nginx Error**: `/opt/homebrew/var/log/nginx/error.log`

## Support

If you encounter issues:

1. Check service status: `./service.sh status`
2. Run tests: `./service.sh test`
3. Check logs: `./service.sh logs`
4. Verify nginx config: `sudo nginx -t`
5. Restart services: `./service.sh restart`

For additional help, check the troubleshooting section above or review the deployment script logs.
