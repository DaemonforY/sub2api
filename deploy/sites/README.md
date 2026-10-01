# Static-site hosting

Subscribers publish an `.html` file or a `.zip` from the panel (「我的网站」) and get
`https://<name>.<SITES_DOMAIN>`. The sub2api process serves those hosts itself (host middleware in
front of every route), so nothing of the main site is reachable on them.

Setup:

1. Use a domain that does **not** share cookies with the main site (not a subdomain of the main
   site's domain). Point `*.<SITES_DOMAIN>` (and `<SITES_DOMAIN>`) at this server with an A record.
2. `deploy/.env`: `SITES_DOMAIN=s.xinduanju.top`, then `docker compose up -d sub2api`.
3. Caddy: merge `Caddyfile.snippet` into `/etc/caddy/Caddyfile` (pre-create
   `/var/log/caddy/sites.log` owned by `caddy:caddy`), validate and reload.

Files live on the data volume under `data/sites/<site id>/v<version>/` (the last three versions are
kept). Quotas, prices and grace periods are set in the admin panel (「网站托管」→ 设置).
