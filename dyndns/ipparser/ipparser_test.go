package ipparser

import (
	"testing"
)

func TestValidIP4ToReturnTrueOnValidAddress(t *testing.T) {
	result := ValidIP4("1.2.3.4")

	if result != true {
		t.Fatalf("Expected ValidIP(1.2.3.4) to be true but got false")
	}
}

func TestValidIP4ToReturnFalseOnInvalidAddress(t *testing.T) {
	result := ValidIP4("abcd")

	if result == true {
		t.Fatalf("Expected ValidIP(abcd) to be false but got true")
	}
}

func TestValidIP4ToReturnFalseOnEmptyAddress(t *testing.T) {
	result := ValidIP4("")

	if result == true {
		t.Fatalf("Expected ValidIP() to be false but got true")
	}
}

func TestMergeIP6NetworkHostAddressTypical(t *testing.T) {
	net := "1:2:3:4:5:6:7:8"
	host := "a1:a2:a3:a4:a5:a6:a7:a8"
	size := 64
	expect := "1:2:3:4:a5:a6:a7:a8"
	result := MergeIP6NetworkHostAddress(net, host, size)

	if result.String() != expect {
		t.Fatalf("Got unexpected merged IPv6 address %s but expected %s",
			result, expect)
	}
}

func TestMergeIP6NetworkHostAddressOdd(t *testing.T) {
	net := "1:2:3:4:5:6:7abc:8"
	host := "a1:a2:a3:a4:a5:a6:8def:a8"
	size := 20
	expect := "1:2:3:4:5:6:7abf:a8"
	result := MergeIP6NetworkHostAddress(net, host, size)

	if result.String() != expect {
		t.Fatalf("Got unexpected merged IPv6 address %s but expected %s",
			result, expect)
	}
}
