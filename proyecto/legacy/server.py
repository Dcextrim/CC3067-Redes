import socket
import json

# Hardcoded "database": card number -> {pin, balance}
ACCOUNTS = {
    "4111111111111111": {"pin": "1234", "balance": 500.00},
    "5500005555555559": {"pin": "0000", "balance": 1200.50},
}

HOST = "127.0.0.1"   # loopback: same machine
PORT = 9000          # arbitrary port above 1024


def send_msg(conn, action, data):
    """Serialize a dict to JSON and send it over the socket."""
    message = {"action": action, "data": data}
    conn.sendall(json.dumps(message).encode("utf-8"))


def recv_msg(conn):
    """Receive raw bytes and parse them as JSON."""
    raw = conn.recv(4096)
    if not raw:
        return None
    return json.loads(raw.decode("utf-8"))


def handle_client(conn):
    authenticated_card = None

    while True:
        message = recv_msg(conn)
        if message is None:
            print("[SERVER] Client disconnected.")
            break

        action = message.get("action")
        data = message.get("data", {})
        print(f"[SERVER] Received: {message}")

        # --- LOGIN ---
        if action == "login":
            card = data.get("card")
            pin = data.get("pin")
            account = ACCOUNTS.get(card)

            if account and account["pin"] == pin:
                authenticated_card = card
                send_msg(conn, "login_ok", {"message": "Authentication successful"})
            else:
                send_msg(conn, "login_denied", {"message": "Invalid card or PIN"})

        # --- WITHDRAW ---
        elif action == "withdraw":
            if authenticated_card is None:
                send_msg(conn, "error", {"message": "Not authenticated"})
                continue

            amount = data.get("amount", 0)
            account = ACCOUNTS[authenticated_card]

            # TODO (students): validate the amount and check for sufficient funds
            # before approving the withdrawal.
            account["balance"] -= amount
            send_msg(conn, "withdraw_ok", {
                "amount": amount,
                "balance": account["balance"]
            })

        # --- LOGOUT ---
        elif action == "logout":
            send_msg(conn, "logout_ok", {"message": "Goodbye"})
            print("[SERVER] Client logged out.")
            break

        else:
            send_msg(conn, "error", {"message": "Unknown action"})


def main():
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as server_socket:
        server_socket.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        server_socket.bind((HOST, PORT))
        server_socket.listen(1)
        print(f"[SERVER] Listening on {HOST}:{PORT} ...")

        while True:
            conn, addr = server_socket.accept()
            print(f"[SERVER] Connection from {addr}")
            with conn:
                handle_client(conn)


if __name__ == "__main__":
    main()
