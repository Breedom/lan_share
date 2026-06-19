import asyncio
import json
import socket
import struct
import time


DISCOVERY_ADDR = "255.255.255.255"
DISCOVERY_PORT = 42069
MAGIC = b"LANSHARE_DISCOVER"
INTERVAL = 5
PEER_TIMEOUT = 15


class Discovery:
    def __init__(self, port: int, host: str):
        self.port = port
        self.host = host
        self._peers: dict[str, dict] = {}
        self._running = False

    def get_peers(self) -> list[dict]:
        now = time.time()
        alive = []
        for addr, info in list(self._peers.items()):
            if now - info["last_seen"] < PEER_TIMEOUT:
                alive.append(info)
        return alive

    async def start(self):
        self._running = True
        loop = asyncio.get_event_loop()

        transport, protocol = await loop.create_datagram_endpoint(
            lambda: DiscoveryProtocol(self),
            local_addr=("0.0.0.0", DISCOVERY_PORT),
            allow_broadcast=True,
            family=socket.AF_INET,
        )

        try:
            while self._running:
                self._broadcast()
                await asyncio.sleep(INTERVAL)
        except asyncio.CancelledError:
            pass
        finally:
            transport.close()

    def _broadcast(self):
        msg = json.dumps({
            "magic": MAGIC.decode(),
            "host": self.host,
            "port": self.port,
            "hostname": socket.gethostname(),
            "timestamp": time.time(),
        }).encode("utf-8")

        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.setsockopt(socket.SOL_SOCKET, socket.SO_BROADCAST, 1)
        try:
            sock.sendto(msg, (DISCOVERY_ADDR, DISCOVERY_PORT))
        except Exception:
            pass
        finally:
            sock.close()

    def handle_message(self, data: bytes, addr: tuple):
        try:
            msg = json.loads(data.decode("utf-8"))
        except (json.JSONDecodeError, UnicodeDecodeError):
            return

        if msg.get("magic") != MAGIC.decode():
            return

        peer_addr = msg.get("host", addr[0])
        if peer_addr == self.host:
            return

        self._peers[peer_addr] = {
            "host": peer_addr,
            "port": msg.get("port", self.port),
            "hostname": msg.get("hostname", peer_addr),
            "last_seen": time.time(),
        }


class DiscoveryProtocol(asyncio.DatagramProtocol):
    def __init__(self, discovery: Discovery):
        self.discovery = discovery

    def datagram_received(self, data: bytes, addr: tuple):
        self.discovery.handle_message(data, addr)
