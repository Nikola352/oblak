# 1. Create the dedicated gVisor log directory with open permissions
sudo mkdir -p /tmp/gvisor-logs/
sudo chmod 777 /tmp/gvisor-logs/

# 2. Write the 3rd runtime configuration directly into the Docker daemon file
sudo tee /etc/docker/daemon.json << 'EOF'
{
  "runtimes": {
    "runsc": {
      "path": "/usr/local/bin/runsc",
      "runtimeArgs": [
        "--strace",
        "--log-format=json",
        "--debug-log=/tmp/gvisor-logs/"
      ]
    }
  }
}
EOF
curl -LO https://storage.googleapis.com/gvisor/releases/release/latest/x86_64/runsc

# 2. Make it executable
chmod +x runsc

# 3. Move it to /usr/local/bin/
sudo mv runsc /usr/local/bin/
# 3. Restart the Docker daemon to apply the new runtime engine args
sudo systemctl restart docker

echo "Done! gVisor runtime configured and Docker restarted successfully."