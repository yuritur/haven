#!/bin/bash
set -e
exec > /var/log/haven-bootstrap.log 2>&1

echo "Writing TLS certificate..."
mkdir -p /etc/haven
echo '{{HAVEN_TLS_CERT_B64}}' | base64 -d > /etc/haven/server.crt
echo '{{HAVEN_TLS_KEY_B64}}' | base64 -d > /etc/haven/server.key
chmod 600 /etc/haven/server.key

echo "Downloading model from HuggingFace..."
mkdir -p /opt/haven/models
HF_URL="https://huggingface.co/{{HAVEN_HF_REPO}}/resolve/main/{{HAVEN_HF_FILE}}"
for i in $(seq 1 5); do
    curl -fsSL "${HF_URL}" -o /opt/haven/models/{{HAVEN_HF_FILE}} && break
    echo "Model download attempt $i failed, retrying in 10s..."
    if [ "$i" -eq 5 ]; then
        echo "ERROR: model download failed after 5 attempts, aborting."
        exit 1
    fi
    sleep 10
done

echo "Building llama-server from source..."
dnf install -y cmake gcc-c++ git
LLAMA_CPP_VERSION="b5200"
git clone --depth 1 --branch "${LLAMA_CPP_VERSION}" https://github.com/ggerganov/llama.cpp /tmp/llama-cpp

CMAKE_FLAGS="-DLLAMA_CURL=OFF -DCMAKE_BUILD_TYPE=Release"
if command -v nvcc &>/dev/null; then
    echo "CUDA detected, building with GPU support..."
    CMAKE_FLAGS="${CMAKE_FLAGS} -DGGML_CUDA=ON"
fi

cmake -B /tmp/llama-cpp/build /tmp/llama-cpp ${CMAKE_FLAGS}
cmake --build /tmp/llama-cpp/build --target llama-server -j$(nproc)
mkdir -p /opt/llama-cpp/bin
cp /tmp/llama-cpp/build/bin/llama-server /opt/llama-cpp/bin/
cp /tmp/llama-cpp/build/bin/*.so* /opt/llama-cpp/bin/ 2>/dev/null || true
rm -rf /tmp/llama-cpp

echo "Configuring llama-server service..."
cat > /etc/systemd/system/llama-server.service << 'UNIT'
[Unit]
Description=llama.cpp Server
After=network.target

[Service]
Environment=LD_LIBRARY_PATH=/opt/llama-cpp/bin
ExecStart=/opt/llama-cpp/bin/llama-server \
    --host 127.0.0.1 --port 8080 \
    --model /opt/haven/models/{{HAVEN_HF_FILE}} \
    --api-key {{HAVEN_API_KEY}} \
    --ctx-size 4096 {{HAVEN_GPU_LAYERS}}
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
UNIT

systemctl daemon-reload
systemctl enable llama-server
systemctl start llama-server

echo "Installing nginx..."
dnf install -y nginx
chown root:nginx /etc/haven/server.key
chmod 640 /etc/haven/server.key

cat > /etc/nginx/conf.d/haven.conf << 'NGINX'
server {
    listen 11434 ssl;
    ssl_certificate /etc/haven/server.crt;
    ssl_certificate_key /etc/haven/server.key;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host localhost;
        proxy_read_timeout 600s;
        proxy_buffering off;
    }
}
NGINX

systemctl enable nginx
systemctl start nginx

echo "Bootstrap complete."
