#!/bin/bash
set -e
exec > /var/log/haven-bootstrap.log 2>&1

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
Environment="OLLAMA_HOST=0.0.0.0:11434"
CONF

systemctl daemon-reload
systemctl enable ollama
systemctl restart ollama

echo "Waiting for Ollama to start..."
for i in $(seq 1 30); do
    curl -sf http://localhost:11434/ > /dev/null 2>&1 && break
    sleep 2
done

echo "Pulling model {{HAVEN_MODEL}}..."
curl -sf -X POST http://localhost:11434/api/pull \
    -H 'Content-Type: application/json' \
    -d '{"name":"{{HAVEN_MODEL}}","stream":false}'

echo "Enabling API key auth..."
cat > /etc/systemd/system/ollama.service.d/override.conf << 'CONF'
[Service]
Environment="OLLAMA_HOST=0.0.0.0:11434"
Environment="OLLAMA_API_KEY={{HAVEN_API_KEY}}"
CONF

systemctl daemon-reload
systemctl restart ollama
echo "Bootstrap complete."
