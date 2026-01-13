package ipparser

import (
	"net"
)

// ValidIP4 tells you if a given string is a valid IPv4 address.
func ValidIP4(ipAddress string) bool {
	testInput := net.ParseIP(ipAddress)
	if testInput == nil {
		return false
	}

	return testInput.To4() != nil
}

// ValidIP6 tells you if a given string is a valid IPv6 address.
func ValidIP6(ip6Address string) bool {
	testInputIP6 := net.ParseIP(ip6Address)
	if testInputIP6 == nil {
		return false
	}

	return testInputIP6.To16() != nil
}

func MergeIP6NetworkHostAddress(networkAddress string, hostAddress string, hostAddressSize int) net.IP {
	netAddr := net.ParseIP(networkAddress)
	hostAddr := net.ParseIP(hostAddress)
	mask := net.CIDRMask(128-hostAddressSize, 128)

	// result = netAddr.Mask(mask)

	netByte := []byte(netAddr.Mask(mask))
	maskByte := []byte(mask)
	hostByte := []byte(hostAddr)
	resultByte := make([]byte, len(netByte))
	for i := range resultByte {
		hostMask := maskByte[i] ^ 0xff
		resultByte[i] = netAddr[i]&maskByte[i] | hostByte[i]&hostMask
	}

	return net.IP(resultByte)
}
