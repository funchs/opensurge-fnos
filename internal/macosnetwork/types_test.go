package macosnetwork

import "testing"

func TestInterfaceOptionsAreSortedAndNamed(t *testing.T) {
	options := interfaceOptions(map[string]string{
		"USB 10/100/1000 LAN": "en7",
		"Wi-Fi":               "en0",
		"":                    "en9",
	})
	if len(options) != 2 {
		t.Fatalf("options = %#v", options)
	}
	if options[0].Interface != "en0" || options[0].NetworkService != "Wi-Fi" {
		t.Fatalf("first option = %#v", options[0])
	}
	if options[1].Interface != "en7" || options[1].NetworkService != "USB 10/100/1000 LAN" {
		t.Fatalf("second option = %#v", options[1])
	}
}

func TestValidateManual(t *testing.T) {
	valid := ManualConfig{NetworkService: "Wi-Fi", Interface: "en0", IPv4: "192.168.1.20", SubnetMask: "255.255.255.0", Router: "192.168.1.1", DNS: []string{"1.1.1.1"}}
	if err := ValidateManual(valid); err != nil {
		t.Fatal(err)
	}
	invalid := valid
	invalid.Router = "192.168.2.1"
	if err := ValidateManual(invalid); err == nil {
		t.Fatal("router outside subnet should fail")
	}
}

func TestVerifyManualRequiresManualModeAndExpectedIPv4(t *testing.T) {
	expected := ManualConfig{NetworkService: "Wi-Fi", Interface: "en0", IPv4: "192.168.1.20", SubnetMask: "255.255.255.0", Router: "192.168.1.1"}
	applied := Snapshot{NetworkService: "Wi-Fi", Interface: "en0", IPv4Mode: IPv4ModeManual, IPv4: expected.IPv4, SubnetMask: expected.SubnetMask, Router: expected.Router}
	if err := VerifyManual(applied, expected); err != nil {
		t.Fatal(err)
	}
	applied.IPv4Mode = IPv4ModeDHCP
	if err := VerifyManual(applied, expected); err == nil {
		t.Fatal("DHCP configuration should not verify as fixed IPv4")
	}
	applied.IPv4Mode = IPv4ModeManual
	applied.IPv4 = "192.168.1.99"
	if err := VerifyManual(applied, expected); err == nil {
		t.Fatal("unexpected manual IPv4 should not verify")
	}
}
