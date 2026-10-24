# MP4 renderer for HiveGPT 视频: headless Chromium (CPU, SwiftShader) + ffmpeg, driven by server.mjs.
# Built on the server: docker compose build videorender
FROM node:22-bookworm-slim
RUN apt-get update \
    && apt-get install -y --no-install-recommends chromium ffmpeg fonts-noto-cjk fonts-noto-color-emoji ca-certificates \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY package.json package-lock.json /app/
RUN npm ci --omit=dev --no-audit --no-fund && npm cache clean --force
COPY server.mjs /app/
# uid 1000 = the sub2api user that owns /app/data on the shared volume.
USER node
ENV CHROME_PATH=/usr/bin/chromium PORT=7100 VIDEO_ROOT=/app/data/video
EXPOSE 7100
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s CMD node -e "fetch('http://127.0.0.1:7100/health').then(r=>process.exit(r.ok?0:1)).catch(()=>process.exit(1))"
CMD ["node", "/app/server.mjs"]
