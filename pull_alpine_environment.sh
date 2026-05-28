# 1. Create a log folder with broad permissions for testing
sudo mkdir -p /tmp/runsc-debug
sudo chmod 777 /tmp/runsc-debug

# 2. Run a simple command with runsc explicitly
sudo docker run --rm --runtime=runsc \
  --log-driver=json-file \
  python:3.11-alpine python -c "print('Hello from gVisor')"

# {
#   "runtimes": {
#     "runsc": {
#       "path": "/usr/local/bin/runsc",
#       "runtimeArgs": [
#         "--debug",
#         "--strace",
#         "--log-format=json"
#       ]
#     }
#   }
# }

# {
#   "runtimes": {
#     "runsc-debug": {
#       "path": "/usr/local/bin/runsc",
#       "runtimeArgs": [
#         "--debug",
#         "--strace",
#         "--log-packets"
#       ]
#     }
#   }
# }

# {
#   "runtimes": {
#     "runsc": {
#       "path": "/usr/local/bin/runsc",
#       "runtimeArgs": [
#         "--strace",
#         "--log-format=json",
#         "--debug-log=/tmp/gvisor-logs/"
#       ]
#     }
#   }
# }

# {
#   "runtimes": {
#     "runsc": {
#       "path": "/usr/local/bin/runsc",
#       "runtimeArgs": [
#         "--strace",
#         "--debug-log=/tmp/runsc-logs/",
#         "--debug-log-format=text"
#       ]
#     }
#   }
# }