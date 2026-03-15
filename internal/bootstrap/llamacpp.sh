#!/bin/bash
set -e
exec > /var/log/haven-bootstrap.log 2>&1

echo "Writing TLS certificate..."
mkdir -p /etc/haven
echo '{{HAVEN_TLS_CERT_B64}}' | base64 -d > /etc/haven/server.crt
echo '{{HAVEN_TLS_KEY_B64}}' | base64 -d > /etc/haven/server.key
chmod 600 /etc/haven/server.key

echo "Installing llama-server..."
LLAMA_CPP_VERSION="b5200"
# Ubuntu prebuilt binary is glibc-based and compatible with Amazon Linux 2023,
# which also uses glibc. It is the most commonly available prebuilt release.
for i in $(seq 1 5); do
    curl -fsSL "https://github.com/ggerganov/llama.cpp/releases/download/${LLAMA_CPP_VERSION}/llama-${LLAMA_CPP_VERSION}-bin-ubuntu-x64.zip" -o /tmp/llama.zip && break
    echo "Download attempt $i failed, retrying in 10s..."
    if [ "$i" -eq 5 ]; then
        echo "ERROR: llama-server download failed after 5 attempts, aborting."
        exit 1
    fi
    sleep 10
done
dnf install -y unzip
unzip -o /tmp/llama.zip -d /opt/llama-cpp/
chmod +x /opt/llama-cpp/build/bin/llama-server

echo "Configuring llama-server service..."
cat > /etc/systemd/system/llama-server.service << 'UNIT'
[Unit]
Description=llama.cpp Server
After=network.target

[Service]
ExecStart=/opt/llama-cpp/build/bin/llama-server \
    --host 127.0.0.1 --port 8080 \
    --hf-repo {{HAVEN_HF_REPO}} \
    --hf-file {{HAVEN_HF_FILE}} \
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

echo "Bootstrap complete. Model will be downloaded on first request."
