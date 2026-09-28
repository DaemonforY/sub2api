# 无限画布子站部署

把 [infinite-canvas](https://github.com/basketikun/infinite-canvas) 作为 Sub2API 的配套生图工作台，部署在子域名（默认 `canvas.hivegpt.cn`）。

镜像来自本站定制分支 [DaemonforY/infinite-canvas](https://github.com/DaemonforY/infinite-canvas)（push 到 main 自动构建 `ghcr.io/daemonfory/infinite-canvas:latest`）。相对上游的改动：渠道默认接口指向主站网关；顶栏和移动端菜单有回主站的链接；去掉了 GitHub 链接；填写非主站接口时提示前往主站注册购买。主站地址、名称、默认接口通过 `MAIN_SITE_URL` / `MAIN_SITE_NAME` / `MAIN_SITE_API_BASE_URL` 环境变量配置，可复用到其他站点。

它是纯静态前端：用户在画布里填入本网关地址和自己的 API Key，浏览器直接调用 `https://<主站>/v1/images/generations`、`/v1/chat/completions` 等接口。主站的「使用密钥」弹窗和首页都带有跳转入口，跳转时会用 `?baseUrl=` 预填网关地址。

## 步骤

1. **DNS**：在 DNSPod 给 `canvas.<domain>` 添加 A 记录，指向主站服务器。
2. **启动容器**：

   ```bash
   cd /home/ubuntu/sub2api/deploy/canvas
   sudo docker compose pull && sudo docker compose up -d
   ```

   容器只监听 `127.0.0.1:3000`。
3. **Caddy**：把 `Caddyfile.snippet` 里的块追加到 `/etc/caddy/Caddyfile`，然后：

   ```bash
   sudo caddy validate --config /etc/caddy/Caddyfile && sudo systemctl reload caddy
   ```

4. **CORS**：画布是跨域调用主站，主站必须放行该来源。在主站 `deploy/.env` 里设置：

   ```
   CORS_ALLOWED_ORIGINS=https://canvas.<domain>
   ```

   多个来源用英文逗号分隔。然后 `cd deploy && sudo docker compose up -d` 重建 sub2api 容器。

## 升级

```bash
cd /home/ubuntu/sub2api/deploy/canvas && sudo docker compose pull && sudo docker compose up -d
```

## 修改子域名

前端入口链接在 `frontend/src/constants/crossSites.ts` 的 `CANVAS_SITE_URL`，改完需要重新构建镜像。
