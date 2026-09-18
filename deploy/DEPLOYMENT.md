# Deployment: gra.tulus.tech (Nginx TLS termination)

This guide runs the Go app on HTTP port `10010` and terminates TLS at Nginx for `gra.tulus.tech`.

## 1) Build and stage the app on server

```bash
# On your build machine or server
cd /opt/go-project
# Copy your built binary here as `server` or build
# Example: scp server user@server:/opt/go-project/server
chmod +x server
```

## 2) Install systemd service

```bash
sudo cp /opt/go-project/deploy/systemd/go-project.service /etc/systemd/system/go-project.service
sudo systemctl daemon-reload
sudo systemctl enable go-project
sudo systemctl start go-project
sudo systemctl status go-project
```

The service exports:
- APP_PORT=10010 (HTTP)
- TLS_ENABLED=0 (TLS off; handled by Nginx)
- YOUTUBE_REDIRECT_URL, FACEBOOK_REDIRECT_URL pointing to https://gra.tulus.tech

## 3) Install Nginx vhost

```bash
sudo cp /opt/go-project/deploy/nginx.gra.tulus.tech.conf /etc/nginx/conf.d/gra.tulus.tech.conf
# Ensure certs exist (e.g., certbot):
# sudo certbot --nginx -d gra.tulus.tech
sudo nginx -t
sudo systemctl reload nginx
```

File points to:
- ssl_certificate /etc/letsencrypt/live/gra.tulus.tech/fullchain.pem
- ssl_certificate_key /etc/letsencrypt/live/gra.tulus.tech/privkey.pem
- proxy_pass http://127.0.0.1:10010

## 4) DNS

Create an A/AAAA record for gra.tulus.tech to your server IP.

## 5) Smoke test

- Check app health (if you have one):
  - curl -k https://gra.tulus.tech/health
- Verify logs:
  - sudo journalctl -u go-project -f
  - sudo tail -f /var/log/nginx/access.log /var/log/nginx/error.log

## 6) Rolling updates

```bash
sudo systemctl restart go-project
```

## Tips
- If you need HTTP locally, run with `TLS_ENABLED=0` and different `APP_PORT`.
- Keep OAuth callbacks matching the public domain and scheme.
