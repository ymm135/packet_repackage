package engine

import (
	"encoding/hex"
	"fmt"
	"packet-repackage/models"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// RepackagePacket rebuilds the packet based on output template
// Template format: "field1 + 0xHH + field2" where fields are replaced with their values
func RepackagePacket(template string, ctx *PacketContext, fields []models.Field) ([]byte, error) {
	if strings.TrimSpace(template) == "" {
		// No template, return original packet
		return ctx.RawPacket, nil
	}

	// Build field map
	fieldMap := make(map[string]models.Field)
	for _, f := range fields {
		fieldMap[f.Name] = f
	}

	// Parse template and build new packet
	parts := strings.Split(template, "+")
	var outputBytes []byte

	for _, part := range parts {
		part = strings.TrimSpace(part)

		// Check if it's a hex literal (0xHH format)
		if strings.HasPrefix(part, "0x") {
			hexStr := strings.TrimPrefix(part, "0x")
			bytes, err := hex.DecodeString(hexStr)
			if err != nil {
				return nil, fmt.Errorf("invalid hex literal %s: %w", part, err)
			}
			outputBytes = append(outputBytes, bytes...)
			continue
		}

		// Check if it's a field name
		if field, exists := fieldMap[part]; exists {
			value := ctx.Fields[part]
			bytes, err := valueToBytes(value, field)
			if err != nil {
				return nil, fmt.Errorf("failed to convert field %s to bytes: %w", part, err)
			}
			outputBytes = append(outputBytes, bytes...)
			continue
		}

		// Try as literal string
		outputBytes = append(outputBytes, []byte(part)...)
	}

	// Recalculate checksums if necessary
	outputBytes, err := recalculateChecksums(outputBytes, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to recalculate checksums: %w", err)
	}

	return outputBytes, nil
}

func valueToBytes(value interface{}, field models.Field) ([]byte, error) {
	if value == nil {
		return make([]byte, field.Length), nil
	}

	switch field.Type {
	case "hex":
		strVal, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("expected string for hex field")
		}
		bytes, err := hex.DecodeString(strVal)
		if err != nil {
			return nil, err
		}
		// Pad or truncate to field length
		return padOrTruncate(bytes, field.Length), nil

	case "decimal":
		var intVal int64
		switch v := value.(type) {
		case int64:
			intVal = v
		case int:
			intVal = int64(v)
		case string:
			parsed, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return nil, err
			}
			intVal = parsed
		default:
			return nil, fmt.Errorf("unsupported type for decimal: %T", value)
		}
		return intToBytes(intVal, field.Length), nil

	case "string":
		strVal, ok := value.(string)
		if !ok {
			strVal = fmt.Sprintf("%v", value)
		}
		bytes := []byte(strVal)
		return padOrTruncate(bytes, field.Length), nil

	default:
		return nil, fmt.Errorf("unknown field type: %s", field.Type)
	}
}

func padOrTruncate(data []byte, length int) []byte {
	if len(data) == length {
		return data
	}
	if len(data) > length {
		return data[:length]
	}
	// Pad with zeros
	padded := make([]byte, length)
	copy(padded, data)
	return padded
}

func intToBytes(value int64, length int) []byte {
	bytes := make([]byte, length)
	for i := length - 1; i >= 0; i-- {
		bytes[i] = byte(value & 0xFF)
		value >>= 8
	}
	return bytes
}

func recalculateChecksums(packetData []byte, ctx *PacketContext) ([]byte, error) {
	// Parse the modified packet
	packet := gopacket.NewPacket(packetData, layers.LayerTypeEthernet, gopacket.Default)

	// Create a buffer to serialize the packet with recalculated checksums
	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	// Get layers and serialize with checksum computation
	if etherLayer := packet.Layer(layers.LayerTypeEthernet); etherLayer != nil {
		eth := etherLayer.(*layers.Ethernet)
		
		if ipLayer := packet.Layer(layers.LayerTypeIPv4); ipLayer != nil {
			ip := ipLayer.(*layers.IPv4)
			
			if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
				tcp := tcpLayer.(*layers.TCP)
				tcp.SetNetworkLayerForChecksum(ip)
				
				payload := packet.ApplicationLayer()
				var payloadBytes []byte
				if payload != nil {
					payloadBytes = payload.Payload()
				}
				
				err := gopacket.SerializeLayers(buffer, opts, eth, ip, tcp, gopacket.Payload(payloadBytes))
				if err != nil {
					return nil, err
				}
				return buffer.Bytes(), nil
			}
			
			if udpLayer := packet.Layer(layers.LayerTypeUDP); udpLayer != nil {
				udp := udpLayer.(*layers.UDP)
				udp.SetNetworkLayerForChecksum(ip)
				
				payload := packet.ApplicationLayer()
				var payloadBytes []byte
				if payload != nil {
					payloadBytes = payload.Payload()
				}
				
				err := gopacket.SerializeLayers(buffer, opts, eth, ip, udp, gopacket.Payload(payloadBytes))
				if err != nil {
					return nil, err
				}
				return buffer.Bytes(), nil
			}
			
			// IP packet without TCP/UDP
			payload := packet.ApplicationLayer()
			var payloadBytes []byte
			if payload != nil {
				payloadBytes = payload.Payload()
			}
			
			err := gopacket.SerializeLayers(buffer, opts, eth, ip, gopacket.Payload(payloadBytes))
			if err != nil {
				return nil, err
			}
			return buffer.Bytes(), nil
		}
	}

	// If we can't parse as IP packet, return as-is
	return packetData, nil
}

// BuildOutputFromTemplate is a helper that parses template and generates output
// This handles more complex templates with field references
func BuildOutputFromTemplate(template string, ctx *PacketContext) ([]byte, error) {
	// Simple regex to find field references and hex literals
	fieldRegex := regexp.MustCompile(`\{(\w+)\}`)
	hexRegex := regexp.MustCompile(`0x([0-9a-fA-F]+)`)
	
	var output []byte
	remaining := template
	
	for len(remaining) > 0 {
		// Find next field reference
		fieldMatch := fieldRegex.FindStringIndex(remaining)
		hexMatch := hexRegex.FindStringIndex(remaining)
		
		// Determine which comes first
		if fieldMatch != nil && (hexMatch == nil || fieldMatch[0] < hexMatch[0]) {
			// Add literal text before field reference
			output = append(output, []byte(remaining[:fieldMatch[0]])...)
			
			// Extract field name and value
			fieldName := remaining[fieldMatch[0]+1:fieldMatch[1]-1]
			if value, exists := ctx.Fields[fieldName]; exists {
				// Convert value to bytes (simplified)
				output = append(output, []byte(fmt.Sprintf("%v", value))...)
			}
			
			remaining = remaining[fieldMatch[1]:]
		} else if hexMatch != nil {
			// Add literal text before hex
			output = append(output, []byte(remaining[:hexMatch[0]])...)
			
			// Decode hex
			hexStr := remaining[hexMatch[0]+2:hexMatch[1]]
			bytes, _ := hex.DecodeString(hexStr)
			output = append(output, bytes...)
			
			remaining = remaining[hexMatch[1]:]
		} else {
			// No more matches, add remaining text
			output = append(output, []byte(remaining)...)
			break
		}
	}
	
	return output, nil
}
