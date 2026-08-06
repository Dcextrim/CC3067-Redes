import socket
import json

HOST = "127.0.0.1"
PORT = 9000


def send_msg(sock, action, data):
    message = {"action": action, "data": data}
    sock.sendall(json.dumps(message).encode("utf-8"))


def recv_msg(sock):
    raw = sock.recv(4096)
    if not raw:
        return None
    return json.loads(raw.decode("utf-8"))


def login(sock):
    """Ask for card + PIN, retry until authenticated."""
    while True:
        card = input("Enter card number: ").strip()
        pin = input("Enter PIN: ").strip()

        send_msg(sock, "login", {"card": card, "pin": pin})
        response = recv_msg(sock)

        if response["action"] == "login_ok":
            print(">> " + response["data"]["message"])
            return True
        else:
            print(">> " + response["data"]["message"])
            print("   Please try again.\n")


def menu(sock):
    """Show the menu and handle the user's choices."""
    while True:
        print("\n--- MENU ---")
        print("1) Withdraw money")
        print("2) Logout")
        choice = input("Choose an option: ").strip()

        if choice == "1":
            amount = float(input("Amount to withdraw: ").strip())

            send_msg(sock, "withdraw", {"amount": amount})
            response = recv_msg(sock)

            if response["action"] == "withdraw_ok":
                d = response["data"]
                print(f">> Please take your ${d['amount']:.2f}")
                print(f">> Remaining balance: ${d['balance']:.2f}")
            else:
                print(">> " + response["data"]["message"])

        elif choice == "2":
            send_msg(sock, "logout", {})
            response = recv_msg(sock)
            print(">> " + response["data"]["message"])
            break

        else:
            print(">> Invalid option.")


def main():
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.connect((HOST, PORT))
        print(f"[CLIENT] Connected to {HOST}:{PORT}")

        if login(sock):
            menu(sock)


if __name__ == "__main__":
    main()
