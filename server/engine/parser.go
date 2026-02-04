package engine

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"packet-repackage/models"
	"strconv"
	"strings"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// PacketContext holds the packet and extracted field values
type PacketContext struct {
	RawPacket  []byte
	Fields     map[string]interface{} // Field name -> value
	Packet     gopacket.Packet
	EtherLayer *layers.Ethernet
	IPv4Layer  *layers.IPv4
	TCPLayer   *layers.TCP
	UDPLayer   *layers.UDP
}

// Get5Tuple returns a string representation of the 5-tuple
func (ctx *PacketContext) Get5Tuple() string {
	var srcIP, dstIP string
	var srcPort, dstPort int
	proto := "unknown"

	if ctx.IPv4Layer != nil {
		srcIP = ctx.IPv4Layer.SrcIP.String()
		dstIP = ctx.IPv4Layer.DstIP.String()
		proto = ctx.IPv4Layer.Protocol.String()
	}

	if ctx.TCPLayer != nil {
		srcPort = int(ctx.TCPLayer.SrcPort)
		dstPort = int(ctx.TCPLayer.DstPort)
		proto = "TCP"
	} else if ctx.UDPLayer != nil {
		srcPort = int(ctx.UDPLayer.SrcPort)
		dstPort = int(ctx.UDPLayer.DstPort)
		proto = "UDP"
	}

	if srcIP == "" {
		return "Non-IP Packet"
	}

	return fmt.Sprintf("%s:%d -> %s:%d [%s]", srcIP, srcPort, dstIP, dstPort, proto)
}

// ParsePacket parses a raw packet and extracts basic layers
func ParsePacket(rawPacket []byte) (*PacketContext, error) {
	if len(rawPacket) == 0 {
		return nil, fmt.Errorf("empty packet")
	}

	ctx := &PacketContext{
		RawPacket: rawPacket,
		Fields:    make(map[string]interface{}),
	}

	// Try to determine if it's Ethernet or IP
	// IPv4 starts with 0x45-0x4f (version 4)
	// IPv6 starts with 0x6x (version 6)
	var packet gopacket.Packet
	version := rawPacket[0] >> 4
	if version == 4 {
		packet = gopacket.NewPacket(rawPacket, layers.LayerTypeIPv4, gopacket.Default)
	} else if version == 6 {
		packet = gopacket.NewPacket(rawPacket, layers.LayerTypeIPv6, gopacket.Default)
	} else {
		packet = gopacket.NewPacket(rawPacket, layers.LayerTypeEthernet, gopacket.Default)
	}

	ctx.Packet = packet

	// Extract common layers
	if etherLayer := packet.Layer(layers.LayerTypeEthernet); etherLayer != nil {
		ctx.EtherLayer = etherLayer.(*layers.Ethernet)
	}

	if ipLayer := packet.Layer(layers.LayerTypeIPv4); ipLayer != nil {
		ctx.IPv4Layer = ipLayer.(*layers.IPv4)
	}

	if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
		ctx.TCPLayer = tcpLayer.(*layers.TCP)
	}

	if udpLayer := packet.Layer(layers.LayerTypeUDP); udpLayer != nil {
		ctx.UDPLayer = udpLayer.(*layers.UDP)
	}

	return ctx, nil
}

// ExtractField extracts a field value from the packet based on field definition
func ExtractField(ctx *PacketContext, field models.Field) (interface{}, error) {
	// Handle built-in 5-tuple fields
	if field.Type == "builtin" {
		return extractBuiltinField(ctx, field.Name)
	}

	// Handle offset-based fields
	offset := field.Offset
	length := field.Length

	if offset < 0 || offset+length > len(ctx.RawPacket) {
		return nil, fmt.Errorf("invalid offset/length for field %s", field.Name)
	}

	data := ctx.RawPacket[offset : offset+length]

	switch field.Type {
	case "hex":
		return hex.EncodeToString(data), nil
	case "decimal":
		return bytesToDecimal(data), nil
	case "string":
		return strings.TrimRight(string(data), "\x00"), nil
	default:
		return hex.EncodeToString(data), nil
	}
}

func extractBuiltinField(ctx *PacketContext, fieldName string) (interface{}, error) {
	switch strings.ToLower(fieldName) {
	case "src_ip":
		if ctx.IPv4Layer != nil {
			return ctx.IPv4Layer.SrcIP.String(), nil
		}
	case "dst_ip":
		if ctx.IPv4Layer != nil {
			return ctx.IPv4Layer.DstIP.String(), nil
		}
	case "src_port":
		if ctx.TCPLayer != nil {
			return int(ctx.TCPLayer.SrcPort), nil
		}
		if ctx.UDPLayer != nil {
			return int(ctx.UDPLayer.SrcPort), nil
		}
	case "dst_port":
		if ctx.TCPLayer != nil {
			return int(ctx.TCPLayer.DstPort), nil
		}
		if ctx.UDPLayer != nil {
			return int(ctx.UDPLayer.DstPort), nil
		}
	case "protocol":
		if ctx.IPv4Layer != nil {
			return int(ctx.IPv4Layer.Protocol), nil
		}
	}
	return nil, fmt.Errorf("builtin field %s not available", fieldName)
}

func bytesToDecimal(data []byte) int64 {
	switch len(data) {
	case 1:
		return int64(data[0])
	case 2:
		return int64(binary.BigEndian.Uint16(data))
	case 4:
		return int64(binary.BigEndian.Uint32(data))
	case 8:
		return int64(binary.BigEndian.Uint64(data))
	default:
		// For other lengths, try to parse as big endian
		var result int64
		for _, b := range data {
			result = (result << 8) | int64(b)
		}
		return result
	}
}

// ExtractAllFields extracts all defined fields from packet
func ExtractAllFields(ctx *PacketContext, fields []models.Field) error {
	for _, field := range fields {
		value, err := ExtractField(ctx, field)
		if err != nil {
			// Don't fail on individual field extraction errors
			ctx.Fields[field.Name] = nil
			continue
		}
		ctx.Fields[field.Name] = value
	}
	return nil
}

// FormatFieldValue formats a field value for display
func FormatFieldValue(value interface{}, fieldType string) string {
	if value == nil {
		return "<not available>"
	}

	switch fieldType {
	case "hex":
		return fmt.Sprintf("%s", value)
	case "decimal":
		return fmt.Sprintf("%d", value)
	case "string":
		return fmt.Sprintf("%q", value)
	default:
		return fmt.Sprintf("%v", value)
	}
}

// CompareFieldValue compares a field value with expected value
func CompareFieldValue(actual interface{}, expected string, fieldType string) bool {
	if actual == nil {
		return false
	}

	switch fieldType {
	case "hex":
		actualStr, ok := actual.(string)
		if !ok {
			return false
		}
		// Remove any spaces and compare case-insensitively
		actualStr = strings.ReplaceAll(strings.ToLower(actualStr), " ", "")
		expected = strings.ReplaceAll(strings.ToLower(expected), " ", "")
		return actualStr == expected

	case "decimal":
		actualInt, ok := actual.(int64)
		if !ok {
			return false
		}
		expectedInt, err := strconv.ParseInt(expected, 10, 64)
		if err != nil {
			return false
		}
		return actualInt == expectedInt

	case "string":
		actualStr, ok := actual.(string)
		if !ok {
			return false
		}
		// Trim quotes if present in expected
		expected = strings.Trim(expected, "\"")
		return actualStr == expected

	default:
		return fmt.Sprintf("%v", actual) == expected
	}
}

// HexDump generates a custom hex and ASCII dump matching Wireshark style
func HexDump(data []byte) string {
	var sb strings.Builder
	for i := 0; i < len(data); i += 16 {
		// 1. Offset (4 hex digits)
		sb.WriteString(fmt.Sprintf("%04x  ", i))

		// 2. Hex values (16 bytes, with extra space after 8th byte)
		for j := 0; j < 16; j++ {
			if i+j < len(data) {
				sb.WriteString(fmt.Sprintf("%02x ", data[i+j]))
			} else {
				sb.WriteString("   ")
			}
			if j == 7 {
				sb.WriteString(" ")
			}
		}

		// 3. Large spacer before ASCII
		sb.WriteString("  ")

		// 4. ASCII representation
		for j := 0; j < 16; j++ {
			if i+j < len(data) {
				b := data[i+j]
				// Show printable characters, use dot otherwise
				if b >= 32 && b <= 126 {
					sb.WriteByte(b)
				} else {
					sb.WriteByte('.')
				}
			} else {
				sb.WriteByte(' ')
			}
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}
