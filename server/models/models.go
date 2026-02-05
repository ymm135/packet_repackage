package models

import (
	"time"

	"gorm.io/gorm"
)

// Field represents a field definition for packet parsing
type Field struct {
	gorm.Model
	Name   string `gorm:"uniqueIndex;not null" json:"name"`
	Offset int    `gorm:"not null" json:"offset"`             // Starting offset in bytes (can be hex like 0x58)
	Length int    `gorm:"not null" json:"length"`             // Field length in bytes
	Type   string `gorm:"not null;default:'hex'" json:"type"` // hex, decimal, string, or builtin (for 5-tuple)
}

// Rule represents a packet modification rule
type Rule struct {
	gorm.Model
	Name           string `gorm:"uniqueIndex;not null" json:"name"`
	Enabled        bool   `gorm:"default:true" json:"enabled"`
	MatchCondition string `gorm:"type:text" json:"match_condition"` // Expression like: tagName == "BHB10A01YP01_pmt" && option == "opset"
	Actions        string `gorm:"type:text" json:"actions"`         // JSON array of actions like: [{"field": "tagName", "op": "set", "value": "BHB10A01YP01"}]
	OutputOptions  string `gorm:"type:text" json:"output_options"`  // JSON array of processing options like: ["compute_checksum"]
	Priority       int    `gorm:"default:0" json:"priority"`        // Higher priority rules evaluated first
}

// InterfaceConfig represents network interface VLAN configuration
type InterfaceConfig struct {
	gorm.Model
	OutInterface string `gorm:"uniqueIndex;not null" json:"out_interface"` // Interface name
	LinkType     string `gorm:"not null" json:"link_type"`                 // access or trunk
	VlanId       string `json:"vlan_id"`                                   // For access mode
	TrunkVlanId  string `json:"trunk_vlan_id"`                             // For trunk mode, comma-separated or ranges like "2,3,5-10"
	DefaultId    string `json:"default_id"`                                // Default VLAN ID for trunk
}

// VlanConfig represents VLAN interface configuration
type VlanConfig struct {
	gorm.Model
	OutInterface      string         `gorm:"uniqueIndex;not null" json:"out_interface"` // vlan_X interface name
	VlanId            int            `gorm:"not null" json:"vlan_id"`
	NickName          string         `json:"nick_name"`
	Type              string         `gorm:"default:'2'" json:"type"`     // 1=route, 2=transparent
	PhysicalInterface string         `json:"physical_interface"`          // Comma-separated physical interfaces
	Status            int            `gorm:"default:1" json:"status"`     // 1=up, 0=down
	IsManager         int            `gorm:"default:0" json:"is_manager"` // 0=normal, 1=management
	VlanConfigArray   []VlanConfigIP `gorm:"foreignKey:OutInterface;references:OutInterface" json:"vlan_config_array"`
}

// VlanConfigIP represents IP addresses assigned to VLAN interfaces
type VlanConfigIP struct {
	gorm.Model
	OutInterface string `gorm:"index" json:"out_interface"`
	IpAddress    string `gorm:"not null" json:"ip_address"`
	SubnetMask   string `gorm:"not null" json:"subnet_mask"` // Can be CIDR notation like "24" or full mask "255.255.255.0"
}

// ProcessLog represents a packet processing log entry
type ProcessLog struct {
	gorm.Model
	RuleID         uint      `gorm:"index" json:"rule_id"`
	RuleName       string    `json:"rule_name"`
	OriginalPacket string    `gorm:"type:text" json:"original_packet"` // Hex string
	ModifiedPacket string    `gorm:"type:text" json:"modified_packet"` // Hex string
	FieldValues    string    `gorm:"type:text" json:"field_values"`    // JSON object with before/after values
	Result         string    `json:"result"`                           // success, error, dropped
	ErrorMessage   string    `gorm:"type:text" json:"error_message"`
	ProcessedAt    time.Time `gorm:"index" json:"processed_at"`

	// 5-Tuple info
	SrcIP    string `json:"src_ip"`
	DstIP    string `json:"dst_ip"`
	SrcPort  int    `json:"src_port"`
	DstPort  int    `json:"dst_port"`
	Protocol string `json:"protocol"`
}

// NFTRule represents an nftables firewall rule
type NFTRule struct {
	gorm.Model
	Name     string `gorm:"uniqueIndex;not null" json:"name"`
	Enabled  bool   `gorm:"default:true" json:"enabled"`
	Priority int    `gorm:"default:100" json:"priority"` // Lower = higher priority

	// 5-Tuple filtering (empty = any)
	SrcIP    string `json:"src_ip"`   // Source IP/CIDR, e.g., "192.168.1.0/24"
	DstIP    string `json:"dst_ip"`   // Destination IP/CIDR
	SrcPort  string `json:"src_port"` // Source port or range, e.g., "80" or "1024-65535"
	DstPort  string `json:"dst_port"` // Destination port or range
	Protocol string `json:"protocol"` // tcp/udp/icmp/any (empty = any)

	// Logging
	LogEnabled bool   `gorm:"default:false" json:"log_enabled"`
	LogPrefix  string `json:"log_prefix"` // Optional log prefix

	// Action
	Action   string `gorm:"not null" json:"action"` // accept/drop/queue
	QueueNum string `json:"queue_num"`              // Queue number or range for queue action (e.g., "0" or "0-3")
}

// StatusMap maps status integer to string commands
var StatusMap = map[int]string{
	0: "down",
	1: "up",
}
