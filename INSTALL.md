# Packet Repackage System

A layer-2 packet modification system that intercepts network packets via NFQueue, applies user-defined rules for packet transformation, and repackages modified packets.

## Features

- **Network Configuration**: Web-based VLAN and bridge configuration
- **Rule Engine**: Field-based packet filtering with complex condition support
- **Packet Modification**: Arithmetic operations, shell functions, field substitution
- **Testing Mode**: Test rules against sample packets before deployment
- **Logging**: Comprehensive packet processing logs with before/after comparison
- **Auto Checksum**: Automatic IP/TCP/UDP checksum recalculation

## Architecture

- **Backend**: Go with NFQueue integration, SQLite database
- **Frontend**: Vue 3 with Element Plus UI components
- **Processing**: Real-time packet interception and modification

## Prerequisites

- Linux (tested on Ubuntu 24.04)
- Root privileges (for NFQueue access)
- Go 1.21 or higher
- Node.js 18 or higher
- nftables installed

## Installation

### Backend Setup

```bash
cd server
go mod download
go build -o packet-repackage main.go
```

### Frontend Setup

```bash
cd web
npm install
npm run build
```

## Usage

### Start Backend Server

```bash
# Run with NFQueue (requires root)
sudo ./server/packet-repackage -db ./data/packet.db -port 8080 -queue 0

# Run in API-only mode (no root required, for configuration only)
./server/packet-repackage -db ./data/packet.db -port 8080 -no-queue
```

### Start Frontend Development Server

```bash
cd web
npm run dev
```

Access the web interface at `http://localhost:3000`

### Configure NFTables

Create nftables rules to send packets to queue:

```bash
sudo nft add table ip netvine-table
sudo nft add chain ip netvine-table base-rule-chain { type filter hook forward priority 0\; policy drop\; }
sudo nft add rule ip netvine-table base-rule-chain queue num 0-3 bypass
```

## Configuration Example

### 1. Configure Network (Bridge & VLAN)

- Create bridge
- Add interfaces to bridge with VLAN configuration
- Assign IP addresses to VLAN interfaces

### 2. Define Fields

Example fields for the sample packet:
- **tagName**: Offset 0x58, Length 16, Type string
- **option**: Offset 0x69, Length 5, Type string

### 3. Create Rules

Match condition:
```
tagName == "BHB10A01YP01_pmt" && option == "opset"
```

Actions (JSON):
```json
[
  {"field": "tagName", "op": "set", "value": "BHB10A01YP01"},
  {"field": "option", "op": "set", "value": "opreset"}
]
```

Output template:
```
tagName + 0x2e + option
```

### 4. Test Rules

Use test mode to verify rules work correctly with sample packets before enabling them in production.

### 5. Monitor Logs

View processing logs to see packet modifications in real-time.

## API Endpoints

### Network
- `GET /api/interfaces` - List network interfaces
- `POST /api/vlan` - Configure VLAN
- `POST /api/vlan/ip` - Add IP to VLAN interface

### Fields
- `GET /api/fields` - List all fields
- `POST /api/fields` - Create field
- `PUT /api/fields/:id` - Update field
- `DELETE /api/fields/:id` - Delete field

### Rules
- `GET /api/rules` - List all rules
- `POST /api/rules` - Create rule
- `PUT /api/rules/:id` - Update rule
- `DELETE /api/rules/:id` - Delete rule
- `POST /api/rules/:id/toggle` - Enable/disable rule

### Test & Logs
- `POST /api/test` - Test rule against packet
- `GET /api/logs` - Query processing logs

## Sample Packet

```
000400010006002381672e81000008004500005e4ba640004011f49eac10a0edac10013c18d018d0004ac047b2c20a00fcf26469010000004b3c030058480500115104000017000b5c5df2f109000000000001000082304248423130413031595030315f706d742e6f7073657400
```

## License

MIT

## Support

For issues and questions, please open an issue on the project repository.
