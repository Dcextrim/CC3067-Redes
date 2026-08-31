# KAESER Compressor Technical Assistant - Local MCP Server

**Author:** Daniel Chet

**Course:** CC3067 Redes

**Protocol:** Model Context Protocol over stdio, manually implemented with JSON-RPC 2.0

## Overview

This project implements a local Model Context Protocol (MCP) server for a simulated industrial compressor maintenance assistant. An MCP host can use the server to consult KAESER compressor specifications, interpret documented SIGMA CONTROL 2 messages, check a simulated maintenance plan and spare-parts inventory, and record simulated sensor readings.

The MCP message exchange is implemented manually. The project does not use an MCP SDK or framework.

## Features

- Newline-delimited JSON-RPC 2.0 communication through `stdin` and `stdout`.
- MCP initialization, ping, tool discovery, and tool execution.
- Five tools with JSON Schema input and output definitions.
- Argument validation and MCP/JSON-RPC error handling.
- Structured and text tool results.
- Source information and a safety warning in every successful tool response.
- Persistent sensor readings in a local JSON file.
- Automated unit and stdio integration tests using only Node.js built-in modules.

## Safety and data disclaimer

This is an academic prototype. It does not connect to physical machinery and must not be used to control, start, stop, or modify compressor equipment. Maintenance history, serial numbers, inventory, sensor ranges, and sensor readings are simulated. Always consult the equipment manual and the plant's authorized safety procedures.

## Requirements

- Node.js 18 or newer
- npm, included with Node.js
- An MCP-compatible host for interactive use, or a terminal for direct JSON-RPC testing

No third-party runtime dependencies are required.

## Installation

```bash
git clone https://github.com/Dcextrim/CC3067-Redes.git
cd CC3067-Redes/mcp-kaeser-server
npm install
```

Verify the installation:

```bash
npm test
```

Start the server manually:

```bash
npm start
```

The process waits for newline-delimited JSON-RPC messages on standard input. It is normal for it to display no prompt while waiting.

## MCP host configuration

MCP hosts launch this server as a subprocess. Use the absolute path to `server.js` in the host's MCP configuration.

Windows example:

```json
{
  "mcpServers": {
    "kaeser-assistant": {
      "command": "C:\\Program Files\\nodejs\\node.exe",
      "args": [
        "C:\\absolute\\path\\to\\CC3067-Redes\\mcp-kaeser-server\\server.js"
      ]
    }
  }
}
```

macOS/Linux example:

```json
{
  "mcpServers": {
    "kaeser-assistant": {
      "command": "/usr/bin/node",
      "args": [
        "/absolute/path/to/CC3067-Redes/mcp-kaeser-server/server.js"
      ]
    }
  }
}
```

Replace the example paths with the values returned by `where.exe node` on Windows or `which node` on macOS/Linux. Restart the MCP host after changing its configuration. The host should discover a server named `kaeser-assistant` with five tools.

## Available tools

| Tool | Required arguments | Purpose |
| --- | --- | --- |
| `consultar_specs` | `modelo: string` | Returns family, configuration, pressure, flow, power, dimensions, and source for a model. |
| `diagnosticar_falla` | `codigo_mensaje: string`, `modelo: string` | Interprets a documented four-digit SIGMA CONTROL 2 alarm or warning code. |
| `calcular_mantenimiento_pendiente` | `numero_serie: string`, `horas_operacion: number >= 0` | Compares operating hours with the simulated maintenance history. |
| `consultar_disponibilidad_repuesto` | `numero_parte: string` | Queries the simulated local spare-parts inventory. |
| `registrar_lectura_sensor` | `numero_serie: string`, `tipo: presion\|temperatura`, `valor: number`, `unidad: bar\|degC` | Persists a simulated reading and evaluates its configured range. |

Model and identifier lookups are case-insensitive. Supported demonstration serial numbers are `KC-2291` and `KC-4402`. Demonstration part numbers include `FA-330` and `FS-10`.

## Example MCP session

Each message must be a single JSON object followed by a newline.

Initialize the server:

```json
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"manual-test","version":"1.0.0"}}}
```

Notify that initialization completed:

```json
{"jsonrpc":"2.0","method":"notifications/initialized"}
```

List the tools:

```json
{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}
```

Diagnose the proposal's sample message:

```json
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"diagnosticar_falla","arguments":{"codigo_mensaje":"0044 A","modelo":"SM 13"}}}
```

The response includes both human-readable text and structured content:

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "..."
      }
    ],
    "structuredContent": {
      "codigo_mensaje": "0044 A",
      "modelo_evaluado": "SM 13",
      "descripcion": "No pressure buildup",
      "requiere_servicio_autorizado": true
    },
    "isError": false
  }
}
```

Check the simulated maintenance plan:

```json
{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"calcular_mantenimiento_pendiente","arguments":{"numero_serie":"KC-2291","horas_operacion":4200}}}
```

Check a spare part:

```json
{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"consultar_disponibilidad_repuesto","arguments":{"numero_parte":"FA-330"}}}
```

Record a pressure reading:

```json
{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"registrar_lectura_sensor","arguments":{"numero_serie":"KC-2291","tipo":"presion","valor":7.2,"unidad":"bar"}}}
```

## Direct PowerShell smoke test

From `mcp-kaeser-server`, run:

```powershell
@(
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"powershell-test","version":"1.0.0"}}}',
  '{"jsonrpc":"2.0","method":"notifications/initialized"}',
  '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'
) | node server.js
```

## Testing with the official MCP Inspector

```bash
npx -y @modelcontextprotocol/inspector --cli node server.js --method tools/list --format json
```

Example tool call:

```bash
npx -y @modelcontextprotocol/inspector --cli node server.js --method tools/call --tool-name diagnosticar_falla --tool-args-json '{"codigo_mensaje":"0044 A","modelo":"SM 13"}' --format json
```

PowerShell users should keep the JSON argument inside single quotes, as shown above.

## Local persistence

`registrar_lectura_sensor` creates `data/lecturas.json` on the first successful call and appends subsequent readings. This runtime file is intentionally excluded from Git.

For isolated tests or another storage location, set `KAESER_LECTURAS_PATH` to an absolute JSON file path before starting the server.

## Error behavior

- Malformed JSON returns JSON-RPC error `-32700`.
- Invalid JSON-RPC requests return `-32600`.
- Unknown methods return `-32601`, including requests whose ID is `0`.
- Malformed MCP method parameters and unknown tools return `-32602`.
- Tool input and business-rule failures return an MCP tool result with `isError: true`, allowing the host or language model to correct the request.

## Data sources

The technical catalog was transcribed from the following KAESER documents supplied for this project:

| Data | Reference |
| --- | --- |
| SXC models | `SXC-Series.pdf`, Technical specifications |
| SX and AIRCENTER 3-8 models | `SX-Series.pdf`, Technical specifications |
| SM, SM SFC, and AIRCENTER 10-16 models | `SM-Series.pdf`, Technical specifications |
| SK, SK SFC, and AIRCENTER 22-25 models | `SK-Series.pdf`, Technical data |
| ASK, ASK SFC, and ASK T SFC models | `ASK-Series.pdf`, Technical data |
| Fault codes | `SIGMA CONTROL 2 SCREW FLUID >=4.5.X`, section 10.2, pp. 195-197 |

The maintenance plan, inventory, sensor limits, serial numbers, and recorded readings are explicitly simulated and are not official KAESER data.

## Project structure

```text
mcp-kaeser-server/
|-- data/
|   |-- diagnosticos.json
|   |-- especificaciones.json
|   |-- inventario.json
|   |-- mantenimiento.json
|   `-- rangos-sensores.json
|-- test/
|   `-- server.test.js
|-- herramientas.js
|-- package.json
`-- server.js
```

## License

ISC
