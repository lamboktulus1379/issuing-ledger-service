# Local HTTPS for gra.tulus.tech (go-project)

## Prereqs
- Homebrew Nginx (nginx service)
- mkcert installed and trust store set up
- Add to /etc/hosts:
  - `127.0.0.1 gra.tulus.tech`

## Certs
Generate local certs for gra.tulus.tech inside the repo (ignored by git):

```zsh
mkdir -p ./certs
mkcert -key-file ./certs/gra.tulus.tech-key.pem -cert-file ./certs/gra.tulus.tech-cert.pem gra.tulus.tech localhost 127.0.0.1 ::1
```

## Install Nginx site

```zsh
# copy site file
sudo cp ./local-dev/nginx/gra.tulus.tech.conf /opt/homebrew/etc/nginx/servers/gra.tulus.tech.conf

# check and reload nginx
sudo nginx -t -c /opt/homebrew/etc/nginx/nginx.conf
sudo brew services restart nginx
```

## Run the Go app
- Default port: 10001 (from `config.json`).
- TLS in the app config is enabled but Nginx already terminates TLS; you can set `app.tlsEnabled` to false for local reverse proxy, or keep as-is (Nginx to HTTPS → upstream HTTP is fine if app runs HTTP).

Run dev:
```zsh
# optional: force port, and disable in-app TLS since Nginx handles HTTPS
APP_PORT=10001 TLS_ENABLED=0 go run ./...
```

## Test
```zsh
curl -kI https://gra.tulus.tech
```

If you want the app reachable at HTTPS directly (no Nginx), set `app.tlsEnabled` true and ensure cert paths are valid; Nginx can still proxy to `https://127.0.0.1:10001` by adjusting `proxy_pass` accordingly.

## Auto-run with launchd (optional)
You can have the app auto-start on login (port 10001, TLS disabled behind Nginx):

```zsh
# one-time: load/update the job and start
chmod +x ./local-dev/deploy-run.sh
./local-dev/deploy-run.sh

# tail logs
tail -f ~/Library/Logs/go-project.out.log
```

## Nginx operations cheat sheet

### macOS (Homebrew Nginx)

- Test configuration
  - `sudo nginx -t -c /opt/homebrew/etc/nginx/nginx.conf`

- Reload without full restart
  - `sudo nginx -s reload -c /opt/homebrew/etc/nginx/nginx.conf`

- Restart service
  - `brew services restart nginx`

- List Nginx processes / listening ports
  - `pgrep -a nginx`
  - `sudo lsof -nP -iTCP -sTCP:LISTEN | grep nginx || true`

- Logs
  - Access/error logs: `/opt/homebrew/var/log/nginx/access.log`, `/opt/homebrew/var/log/nginx/error.log`
  - Tail: `tail -f /opt/homebrew/var/log/nginx/{access,error}.log`

- Enable/disable a specific site and reload
  - Homebrew keeps server files under `/opt/homebrew/etc/nginx/servers/`
  - Disable: move the file out of `servers/` then `sudo nginx -t -c /opt/homebrew/etc/nginx/nginx.conf && sudo nginx -s reload -c /opt/homebrew/etc/nginx/nginx.conf`
  - Enable: place the `*.conf` back into `servers/` and reload as above

### Ubuntu (remote VM)

- Test configuration
  - `sudo nginx -t`

- Reload without dropping connections
  - `sudo systemctl reload nginx`

- Restart service
  - `sudo systemctl restart nginx`

- Status / processes / ports
  - `systemctl status --no-pager nginx`
  - `pgrep -a nginx || ps aux | grep '[n]ginx'`
  - `ss -tulpn 2>/dev/null | grep -E ':(80|443)\b' || netstat -tulpn 2>/dev/null | grep -E ':(80|443)' || true`

- Logs
  - Journal: `sudo journalctl -u nginx -n 100 --no-pager`
  - Files: `/var/log/nginx/access.log`, `/var/log/nginx/error.log`
  - Tail: `sudo tail -f /var/log/nginx/{access,error}.log`

- Enable/disable a specific site and reload
  - Sites live in `/etc/nginx/sites-available/` with symlinks in `/etc/nginx/sites-enabled/`
  - Disable site: `sudo rm /etc/nginx/sites-enabled/example.conf && sudo nginx -t && sudo systemctl reload nginx`
  - Enable site: `sudo ln -s /etc/nginx/sites-available/example.conf /etc/nginx/sites-enabled/example.conf && sudo nginx -t && sudo systemctl reload nginx`

- List everything (configs, sites, modules, processes)
  - Dump full effective config: `sudo nginx -T`
  - Version/options/modules: `nginx -V 2>&1`
  - Sites: `ls -la /etc/nginx/sites-available && ls -la /etc/nginx/sites-enabled`
  - Extra configs: `ls -la /etc/nginx/conf.d || true; ls -la /etc/nginx/snippets || true`
  - All conf files: `sudo find /etc/nginx -type f -name '*.conf' | sort`
  - All server_name entries: `sudo nginx -T 2>/dev/null | grep -E "server_name\s" | sed 's/^\s*//' | sort -u`
  - Processes: `ps -o pid,ppid,cmd -C nginx || pgrep -a nginx`
  - Ports: `ss -tulpn 2>/dev/null | grep -E ':(80|443)\b' || netstat -tulpn 2>/dev/null | grep -E ':(80|443)' || true`
  - Systemd units: `systemctl list-units --type=service | grep -i nginx || true; systemctl list-unit-files | grep -i nginx || true`
