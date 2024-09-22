package networking

import (
	"fmt"
	"net"
)

func FindSubnetForIP(ip net.IP) (*net.IPNet, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			// Check if the IP belongs to this subnet
			if ipNet.Contains(ip) {
				return ipNet, nil
			}
		}
	}

	return nil, fmt.Errorf("subnet not found for IP %s", ip.String())
}

func GetIPRange(ipNet *net.IPNet) []net.IP {
	network := ipNet.IP.Mask(ipNet.Mask)

	broadcast := make(net.IP, len(network))
	for i := 0; i < len(network); i++ {
		broadcast[i] = network[i] | ^ipNet.Mask[i]
	}

	firstUsable := make(net.IP, len(network))
	copy(firstUsable, network)
	firstUsable[len(firstUsable)-1]++

	lastUsable := make(net.IP, len(broadcast))
	copy(lastUsable, broadcast)
	lastUsable[len(lastUsable)-1]--

	// Calculate all IPs in the range
	var ips []net.IP
	for ip := firstUsable; !ip.Equal(lastUsable); ip = nextIP(ip) {
		ips = append(ips, ip)
	}
	ips = append(ips, lastUsable) // Include the last usable IP

	return ips
}

func nextIP(ip net.IP) net.IP {
	nextIP := make(net.IP, len(ip))
	copy(nextIP, ip)

	for i := len(ip) - 1; i >= 0; i-- {
		nextIP[i]++
		if nextIP[i] > 0 {
			break
		}
	}

	return nextIP
}
