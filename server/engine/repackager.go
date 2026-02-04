package engine

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"packet-repackage/models"
	"sort"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// FieldSegment represents a segment of the packet (either user-defined or built-in)
type FieldSegment struct {
	Offset      int
	Length      int
	IsUserField bool
	FieldName   string // Empty for built-in fields
	Field       *models.Field
}

// RepackagePacket rebuilds the packet by preserving built-in fields and updating user-defined fields
func RepackagePacket(outputOptions string, ctx *PacketContext, fields []models.Field) ([]byte, error) {
	if len(fields) == 0 {
		// No fields defined, return original packet
		return ctx.RawPacket, nil
	}

	// Extract built-in fields (gaps between user-defined fields)
	segments := extractFieldSegments(ctx.RawPacket, fields)

	// Reassemble packet with modified user fields and preserved built-in fields
	reassembled := reassemblePacket(ctx.RawPacket, segments, ctx)

	// Apply output options (e.g., compute checksum)
	result, err := applyOutputOptions(reassembled, outputOptions, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to apply output options: %w", err)
	}

	return result, nil
}

// extractFieldSegments analyzes the packet and creates an ordered list of all field segments
func extractFieldSegments(rawPacket []byte, userFields []models.Field) []FieldSegment {
	var segments []FieldSegment

	// Sort user fields by offset
	sortedFields := make([]models.Field, len(userFields))
	copy(sortedFields, userFields)
	sort.Slice(sortedFields, func(i, j int) bool {
		return sortedFields[i].Offset < sortedFields[j].Offset
	})

	currentOffset := 0
	packetLen := len(rawPacket)

	for _, field := range sortedFields {
		// Add built-in field before this user field (if there's a gap)
		if currentOffset < field.Offset {
			segments = append(segments, FieldSegment{
				Offset:      currentOffset,
				Length:      field.Offset - currentOffset,
				IsUserField: false,
			})
		}

		// Add user-defined field
		fieldCopy := field
		segments = append(segments, FieldSegment{
			Offset:      field.Offset,
			Length:      field.Length,
			IsUserField: true,
			FieldName:   field.Name,
			Field:       &fieldCopy,
		})

		currentOffset = field.Offset + field.Length
	}

	// Add trailing built-in field (if any bytes remain)
	if currentOffset < packetLen {
		segments = append(segments, FieldSegment{
			Offset:      currentOffset,
			Length:      packetLen - currentOffset,
			IsUserField: false,
		})
	}

	return segments
}

// reassemblePacket reconstructs the packet from segments
func reassemblePacket(rawPacket []byte, segments []FieldSegment, ctx *PacketContext) []byte {
	var output []byte

	for _, segment := range segments {
		if segment.IsUserField {
			// Use modified value from context
			value := ctx.Fields[segment.FieldName]
			bytes, err := valueToBytes(value, *segment.Field)
			if err != nil {
				// Fallback to original bytes on error
				output = append(output, rawPacket[segment.Offset:segment.Offset+segment.Length]...)
			} else {
				output = append(output, bytes...)
			}
		} else {
			// Preserve original bytes for built-in fields
			endOffset := segment.Offset + segment.Length
			if endOffset <= len(rawPacket) {
				output = append(output, rawPacket[segment.Offset:endOffset]...)
			}
		}
	}

	return output
}

// applyOutputOptions processes output options like checksum computation
func applyOutputOptions(packetData []byte, optionsJSON string, ctx *PacketContext) ([]byte, error) {
	if optionsJSON == "" {
		return packetData, nil
	}

	var options []string
	err := json.Unmarshal([]byte(optionsJSON), &options)
	if err != nil {
		// If not valid JSON, treat as no options
		return packetData, nil
	}

	result := packetData
	for _, option := range options {
		switch option {
		case "compute_checksum":
			result, err = recalculateChecksums(result, ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to compute checksum: %w", err)
			}
		}
	}

	return result, nil
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
		case float64:
			intVal = int64(v)
		case string:
			// Try to parse as integer
			fmt.Sscanf(v, "%d", &intVal)
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
