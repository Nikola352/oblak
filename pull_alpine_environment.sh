# 1. Create a log folder with broad permissions for testing
sudo mkdir -p /tmp/runsc-debug
sudo chmod 777 /tmp/runsc-debug

# 2. Run a simple command with runsc explicitly
sudo docker run --rm --runtime=runsc \
  --log-driver=json-file \
  python:3.11-alpine python -c "print('Hello from gVisor')"
