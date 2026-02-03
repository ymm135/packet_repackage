package network

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
)

// InterfaceInfo represents network interface information
type InterfaceInfo struct {
	Name         string   `json:"name"`
	HardwareAddr string   `json:"hardware_addr"`
	IPAddresses  []string `json:"ip_addresses"`
	IsUp         bool     `json:"is_up"`
}

// getInterfaceRealStatus checks actual interface status from system
func getInterfaceRealStatus(ifaceName string) bool {
	cmd := exec.Command("ip", "link", "show", ifaceName)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	
	// Parse output looking for "state UP" or "state DOWN"
	outputStr := string(output)
	if strings.Contains(outputStr, "state UP") {
		return true
	}
	return false
}

// ListInterfaces returns all available network interfaces
func ListInterfaces() ([]InterfaceInfo, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to list interfaces: %w", err)
	}

	var result []InterfaceInfo
	for _, iface := range interfaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		var ipAddresses []string
		for _, addr := range addrs {
			ipAddresses = append(ipAddresses, addr.String())
		}

		info := InterfaceInfo{
			Name:         iface.Name,
			HardwareAddr: iface.HardwareAddr.String(),
			IPAddresses:  ipAddresses,
			IsUp:         getInterfaceRealStatus(iface.Name),
		}

		// Filter out virtual interfaces we're managing
		if !strings.HasPrefix(iface.Name, "vlan_") && iface.Name != "Bridge" {
			result = append(result, info)
		}
	}

	return result, nil
}

// GetInterfaceByName returns information about a specific interface
func GetInterfaceByName(name string) (*InterfaceInfo, error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return nil, fmt.Errorf("interface not found: %w", err)
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return nil, err
	}

	var ipAddresses []string
	for _, addr := range addrs {
		ipAddresses = append(ipAddresses, addr.String())
	}

	return &InterfaceInfo{
		Name:         iface.Name,
		HardwareAddr: iface.HardwareAddr.String(),
		IPAddresses:  ipAddresses,
		IsUp:         getInterfaceRealStatus(iface.Name),
	}, nil
}
