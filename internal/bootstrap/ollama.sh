#!/bin/bash
set -e
exec > /var/log/haven-bootstrap.log 2>&1

echo "Writing TLS certificate..."
mkdir -p /etc/haven
echo '{{HAVEN_TLS_CERT_B64}}' | base64 -d > /etc/haven/server.crt
echo '{{HAVEN_TLS_KEY_B64}}' | base64 -d > /etc/haven/server.key
chmod 600 /etc/haven/server.key

if [ "{{HAVEN_GPU}}" = "true" ]; then
    echo "Installing NVIDIA drivers and CUDA toolkit..."
    dnf install -y kernel-devel-$(uname -r) kernel-headers-$(uname -r)
    dnf config-manager --add-repo \
        https://developer.download.nvidia.com/compute/cuda/repos/amzn2023/x86_64/cuda-amzn2023.repo
    dnf install -y cuda-toolkit nvidia-driver-latest-dkms
    nvidia-smi || { echo "ERROR: nvidia-smi failed"; exit 1; }
    echo "NVIDIA drivers installed successfully."
fi

echo "Installing Ollama..."
for i in $(seq 1 5); do
    curl -fsSL https://ollama.com/install.sh | sh && break
    echo "Install attempt $i failed, retrying in 10s..."
    if [ "$i" -eq 5 ]; then
        echo "ERROR: Ollama installation failed after 5 attempts, aborting."
        exit 1
    fi
    sleep 10
done

echo "Configuring Ollama..."
mkdir -p /etc/systemd/system/ollama.service.d
cat > /etc/systemd/system/ollama.service.d/override.conf << 'CONF'
[Service]
Environment="OLLAMA_HOST=127.0.0.1:11435"
Environment="OLLAMA_ORIGINS=*"
CONF

systemctl daemon-reload
systemctl enable ollama
systemctl restart ollama

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
        proxy_pass http://127.0.0.1:11435;
        proxy_set_header Host localhost;
        proxy_read_timeout 600s;
        proxy_buffering off;
    }
}
NGINX

systemctl enable nginx
systemctl start nginx

echo "Waiting for Ollama to start..."
for i in $(seq 1 30); do
    curl -sf http://127.0.0.1:11435/ > /dev/null 2>&1 && break
    sleep 2
done

echo "Pulling model {{HAVEN_MODEL}}..."
curl -sf -X POST http://127.0.0.1:11435/api/pull \
    -H 'Content-Type: application/json' \
    -d '{"name":"{{HAVEN_MODEL}}","stream":false}'

echo "Enabling API key auth..."
cat > /etc/systemd/system/ollama.service.d/override.conf << 'CONF'
[Service]
Environment="OLLAMA_HOST=127.0.0.1:11435"
Environment="OLLAMA_ORIGINS=*"
Environment="OLLAMA_API_KEY={{HAVEN_API_KEY}}"
CONF

systemctl daemon-reload
systemctl restart ollama
echo "Bootstrap complete."
