import os
import socket
import subprocess

def probe():
    print("--- Starting Detonation Probe ---")

    # 1. File Access: Try to read sensitive system files
    # This should trigger sys_openat
    files_to_check = ["/etc/passwd", "/etc/shadow", "/proc/self/maps"]
    for f in files_to_check:
        try:
            print(f"Attempting to open: {f}")
            with open(f, 'r') as fd:
                _ = fd.read(10)
        except Exception as e:
            print(f"Access to {f} failed (expected): {e}")

    # 2. Network Access: Try to reach a DNS server
    # This should trigger sys_socket and sys_connect
    print("Attempting network connection to 8.8.8.8...")
    try:
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        s.settimeout(1)
        s.connect(("8.8.8.8", 53))
    except Exception as e:
        print(f"Network failed (expected): {e}")

    # 3. Process Spawning: Try to run a shell command
    # This should trigger sys_execve
    print("Attempting to spawn a sub-shell...")
    try:
        res = subprocess.check_output(["whoami"], stderr=subprocess.STDOUT)
        print(f"Whoami result: {res.decode().strip()}")
    except Exception as e:
        print(f"Exec failed: {e}")

if __name__ == "__main__":
    probe()