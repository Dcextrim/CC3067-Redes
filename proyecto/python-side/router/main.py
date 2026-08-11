"""Punto de entrada CLI de un nodo router (ver docs/protocol.md)."""

from __future__ import annotations

import argparse
import logging

from router.config import AttachedHost, Neighbor, NodeConfig, load
from router.node import Node


def _prompt_config(name: str) -> NodeConfig:
    print(f"[ROUTER {name}] No routing table found ({name}_tabla_enrutamiento.csv). Let's configure this node.")
    ip = input("This router's IP: ").strip() or "127.0.0.1"
    port = int(input("This router's listen port: ").strip())
    neighbors: list[Neighbor] = []
    print("Enter neighbors (blank IP to finish):")
    while True:
        neighbor_ip = input("  Neighbor IP: ").strip()
        if not neighbor_ip:
            break
        neighbor_port = int(input("  Neighbor port: ").strip())
        neighbor_cost = int(input("  Link cost [1]: ").strip() or "1")
        neighbors.append(Neighbor(neighbor_ip, neighbor_port, neighbor_cost))
    attached_host = None
    role = input("Attached host? (C=Client, S=Server, N=None): ").strip().upper()
    if role in ("C", "S"):
        host_ip = input(f"  {role} host IP: ").strip()
        host_port = int(input(f"  {role} host port: ").strip())
        attached_host = AttachedHost("client" if role == "C" else "server", host_ip, host_port)
    return NodeConfig(name=name, ip=ip, port=port, neighbors=tuple(neighbors), attached_host=attached_host)


def main() -> None:
    logging.basicConfig(level=logging.INFO, format="%(message)s")
    parser = argparse.ArgumentParser(description="Nodo router Link State (CC3067 Lab 3)")
    parser.add_argument("--config", help="Ruta a un config.json; si se omite, se pide interactivamente")
    args = parser.parse_args()

    if args.config:
        config = load(args.config)
    else:
        name = input("Node name (a single letter, e.g. A): ").strip()
        config = _prompt_config(name)

    Node(config).run_forever()


if __name__ == "__main__":
    main()
