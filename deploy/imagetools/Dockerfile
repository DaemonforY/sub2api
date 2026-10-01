# Image tools for HiveGPT: background removal (isnet / u2net) and 4x super-resolution
# (Real-ESRGAN x4v3) on CPU. Built on the server: docker compose build imagetools
FROM python:3.12-slim AS export
RUN pip install --no-cache-dir torch==2.8.0 --index-url https://download.pytorch.org/whl/cpu && pip install --no-cache-dir onnx==1.23.1
COPY fetch.py export_esrgan.py /m/
RUN python /m/fetch.py https://github.com/xinntao/Real-ESRGAN/releases/download/v0.2.5.0/realesr-general-x4v3.pth /m/realesr-general-x4v3.pth \
        8dc7edb9ac80ccdc30c3a5dca6616509367f05fbc184ad95b731f05bece96292 \
    && python /m/export_esrgan.py /m/realesr-general-x4v3.pth /m/realesr-general-x4v3.onnx

FROM python:3.12-slim
RUN pip install --no-cache-dir onnxruntime==1.30.0 numpy==2.5.3 pillow==12.3.0
COPY fetch.py /tmp/fetch.py
RUN mkdir /models \
    && python /tmp/fetch.py https://github.com/danielgatis/rembg/releases/download/v0.0.0/isnet-general-use.onnx /models/isnet-general-use.onnx \
        60920e99c45464f2ba57bee2ad08c919a52bbf852739e96947fbb4358c0d964a \
    && python /tmp/fetch.py https://github.com/danielgatis/rembg/releases/download/v0.0.0/u2net.onnx /models/u2net.onnx \
        8d10d2f3bb75ae3b6d527c77944fc5e7dcd94b29809d47a739a7a728a912b491 \
    && rm /tmp/fetch.py
COPY --from=export /m/realesr-general-x4v3.onnx /models/
COPY server.py /app/server.py
RUN chmod -R a+r /models /app
USER nobody
EXPOSE 7000
HEALTHCHECK --interval=30s --timeout=5s --start-period=60s CMD python -c "import urllib.request; urllib.request.urlopen('http://127.0.0.1:7000/health', timeout=4)"
CMD ["python", "/app/server.py"]
