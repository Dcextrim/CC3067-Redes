"""Interoperabilidad Python <-> Go sobre el envoltorio DATA real.

Go arma un frame DATA (Hamming(7,4) sobre {from,to,msg} completo, ver
docs/protocol.md) y Python debe decodificarlo igual. Es la contraparte de
go-side/internal/router/interop_test.go (que prueba lo inverso). Se salta
si no hay un binario `go` en PATH.
"""

from __future__ import annotations

import json
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "python-side"))

from router.forwarding.network import decode_payload  # noqa: E402

GO_SNIPPET = """
package main

import (
	"encoding/json"
	"fmt"

	"cc3067/lab3/go-side/internal/router/control"
	"cc3067/lab3/go-side/internal/router/forwarding"
)

func main() {
	envelope, err := forwarding.EncodeEnvelope(control.Payload{From: "A", To: "B", Msg: "interop"})
	if err != nil {
		panic(err)
	}
	raw, _ := json.Marshal(envelope)
	fmt.Println(string(raw))
}
"""


def main() -> int:
    go_bin = shutil.which("go")
    if go_bin is None:
        print("SKIP: no hay binario 'go' en PATH; se omite la prueba de interoperabilidad")
        return 0

    go_side = ROOT / "go-side"
    with tempfile.TemporaryDirectory(dir=go_side / "cmd") as tmp_dir:
        snippet_path = Path(tmp_dir) / "main.go"
        snippet_path.write_text(GO_SNIPPET, encoding="utf-8")
        result = subprocess.run(
            [go_bin, "run", str(snippet_path)], cwd=go_side, capture_output=True, text=True
        )
        if result.returncode != 0:
            print(result.stderr, file=sys.stderr)
            return 1

    envelope = json.loads(result.stdout.strip())
    payload = decode_payload(envelope)
    assert payload == {"from": "A", "to": "B", "msg": "interop"}, payload
    print("OK: Python decodifico correctamente un frame DATA armado por Go.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
