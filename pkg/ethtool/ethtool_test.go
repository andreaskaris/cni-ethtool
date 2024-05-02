package ethtool

import (
	"fmt"
	"strings"
	"testing"

	"k8s.io/utils/pointer"
)

const (
	notFoundOutput = `[  ]`
	notFoundError  = `netlink error: no device matches name (offset 24)
netlink error: No such device`

	dummy0Output = `[ {
        "ifname": "dummy0",
        "rx-checksumming": {
            "active": false,
            "fixed": true,
            "requested": false
        },
        "tx-checksumming": {
            "active": true,
            "fixed": null,
            "requested": null
        },
        "tx-checksum-ipv4": {
            "active": false,
            "fixed": true,
            "requested": false
        },
        "tx-checksum-ip-generic": {
            "active": true,
            "fixed": false,
            "requested": true
        },
        "tx-checksum-ipv6": {
            "active": false,
            "fixed": true,
            "requested": false
        },
        "tx-checksum-fcoe-crc": {
            "active": false,
            "fixed": true,
            "requested": false
        },
        "tx-checksum-sctp": {
            "active": false,
            "fixed": true,
            "requested": false
        },
        "scatter-gather": {
            "active": true,
            "fixed": null,
            "requested": null
        },
        "tx-scatter-gather": {
            "active": true,
            "fixed": false,
            "requested": true
        },
        "tx-scatter-gather-fraglist": {
            "active": true,
            "fixed": false,
            "requested": true
        },
        "tcp-segmentation-offload": {
            "active": true,
            "fixed": null,
            "requested": null
        },
        "tx-tcp-segmentation": {
            "active": true,
            "fixed": false,
            "requested": true
        },
        "tx-tcp-ecn-segmentation": {
            "active": true,
            "fixed": false,
            "requested": true
        },
        "tx-tcp-mangleid-segmentation": {
            "active": true,
            "fixed": false,
            "requested": true
        },
        "tx-tcp6-segmentation": {
            "active": true,
            "fixed": false,
            "requested": true
        },
        "generic-segmentation-offload": {
            "active": true,
            "fixed": false,
            "requested": true
        },
        "generic-receive-offload": {
            "active": true,
            "fixed": false,
            "requested": true
        },
        "large-receive-offload": {
            "active": false,
            "fixed": true,
            "requested": false
        },
        "rx-vlan-offload": {
            "active": false,
            "fixed": true,
            "requested": false
        },
        "tx-vlan-offload": {
            "active": false,
            "fixed": true,
            "requested": false
        },
        "ntuple-filters": {
            "active": false,
            "fixed": true,
            "requested": false
        },
        "receive-hashing": {
            "active": false,
            "fixed": true,
            "requested": false
        },
        "highdma": {
            "active": true,
            "fixed": false,
            "requested": true
        },
        "rx-vlan-filter": {
            "active": false,
            "fixed": true,
            "requested": false
        },
        "vlan-challenged": {
            "active": false,
            "fixed": true,
            "requested": false
        },
        "tx-lockless": {
            "active": true,
            "fixed": true,
            "requested": true
        },
        "netns-local": {
            "active": false,
            "fixed": true,
            "requested": false
        },
        "tx-gso-robust": {
            "active": false,
            "fixed": true,
            "requested": false
        },
        "tx-fcoe-segmentation": {
            "active": false,
            "fixed": true,
            "requested": false
        },
        "tx-gre-segmentation": {
            "active": true,
            "fixed": false,
            "requested": true
        },
        "tx-gre-csum-segmentation": {
            "active": true,
            "fixed": false,
            "requested": true
        },
        "tx-ipxip4-segmentation": {
            "active": true,
            "fixed": false,
            "requested": true
        },
        "tx-ipxip6-segmentation": {
            "active": true,
            "fixed": false,
            "requested": true
        },
        "tx-udp_tnl-segmentation": {
            "active": true,
            "fixed": false,
            "requested": true
        },
        "tx-udp_tnl-csum-segmentation": {
            "active": true,
            "fixed": false,
            "requested": true
        },
        "tx-gso-partial": {
            "active": false,
            "fixed": true,
            "requested": false
        },
        "tx-tunnel-remcsum-segmentation": {
            "active": false,
            "fixed": true,
            "requested": false
        },
        "tx-sctp-segmentation": {
            "active": true,
            "fixed": false,
            "requested": true
        },
        "tx-esp-segmentation": {
            "active": false,
            "fixed": true,
            "requested": false
        },
        "tx-udp-segmentation": {
            "active": true,
            "fixed": false,
            "requested": true
        },
        "tx-gso-list": {
            "active": true,
            "fixed": false,
            "requested": true
        },
        "fcoe-mtu": {
            "active": false,
            "fixed": true,
            "requested": false
        },
        "tx-nocache-copy": {
            "active": false,
            "fixed": false,
            "requested": false
        },
        "loopback": {
            "active": false,
            "fixed": true,
            "requested": false
        },
        "rx-fcs": {
            "active": false,
            "fixed": true,
            "requested": false
        },
        "rx-all": {
            "active": false,
            "fixed": true,
            "requested": false
        },
        "tx-vlan-stag-hw-insert": {
            "active": false,
            "fixed": true,
            "requested": false
        },
        "rx-vlan-stag-hw-parse": {
            "active": false,
            "fixed": true,
            "requested": false
        },
        "rx-vlan-stag-filter": {
            "active": false,
            "fixed": true,
            "requested": false
        },
        "l2-fwd-offload": {
            "active": false,
            "fixed": true,
            "requested": false
        },
        "hw-tc-offload": {
            "active": false,
            "fixed": true,
            "requested": false
        },
        "esp-hw-offload": {
            "active": false,
            "fixed": true,
            "requested": false
        },
        "esp-tx-csum-hw-offload": {
            "active": false,
            "fixed": true,
            "requested": false
        },
        "rx-udp_tunnel-port-offload": {
            "active": false,
            "fixed": true,
            "requested": false
        },
        "tls-hw-tx-offload": {
            "active": false,
            "fixed": true,
            "requested": false
        },
        "tls-hw-rx-offload": {
            "active": false,
            "fixed": true,
            "requested": false
        },
        "rx-gro-hw": {
            "active": false,
            "fixed": true,
            "requested": false
        },
        "tls-hw-record": {
            "active": false,
            "fixed": true,
            "requested": false
        },
        "rx-gro-list": {
            "active": false,
            "fixed": false,
            "requested": false
        },
        "macsec-hw-offload": {
            "active": false,
            "fixed": true,
            "requested": false
        },
        "rx-udp-gro-forwarding": {
            "active": false,
            "fixed": false,
            "requested": false
        },
        "hsr-tag-ins-offload": {
            "active": false,
            "fixed": true,
            "requested": false
        },
        "hsr-tag-rm-offload": {
            "active": false,
            "fixed": true,
            "requested": false
        },
        "hsr-fwd-offload": {
            "active": false,
            "fixed": true,
            "requested": false
        },
        "hsr-dup-offload": {
            "active": false,
            "fixed": true,
            "requested": false
        }
    } ]
`
)

var (
	dummy0OutputParsed = OffloadList{
		"tx-sctp-segmentation":           {pointer.Bool(true), pointer.Bool(false), pointer.Bool(true)},
		"rx-udp-gro-forwarding":          {pointer.Bool(false), pointer.Bool(false), pointer.Bool(false)},
		"tls-hw-record":                  {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"fcoe-mtu":                       {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"tx-nocache-copy":                {pointer.Bool(false), pointer.Bool(false), pointer.Bool(false)},
		"tx-udp_tnl-segmentation":        {pointer.Bool(true), pointer.Bool(false), pointer.Bool(true)},
		"tcp-segmentation-offload":       {pointer.Bool(true), nil, nil},
		"rx-all":                         {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"tx-tcp-ecn-segmentation":        {pointer.Bool(true), pointer.Bool(false), pointer.Bool(true)},
		"tx-gso-list":                    {pointer.Bool(true), pointer.Bool(false), pointer.Bool(true)},
		"rx-udp_tunnel-port-offload":     {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"receive-hashing":                {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"tx-udp_tnl-csum-segmentation":   {pointer.Bool(true), pointer.Bool(false), pointer.Bool(true)},
		"rx-vlan-stag-hw-parse":          {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"rx-vlan-offload":                {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"tx-ipxip6-segmentation":         {pointer.Bool(true), pointer.Bool(false), pointer.Bool(true)},
		"tx-checksum-ipv6":               {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"hsr-tag-ins-offload":            {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"hsr-fwd-offload":                {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"tx-checksum-sctp":               {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"tx-tcp-segmentation":            {pointer.Bool(true), pointer.Bool(false), pointer.Bool(true)},
		"rx-vlan-filter":                 {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"rx-vlan-stag-filter":            {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"tls-hw-rx-offload":              {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"tx-vlan-offload":                {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"tx-checksum-fcoe-crc":           {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"vlan-challenged":                {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"generic-segmentation-offload":   {pointer.Bool(true), pointer.Bool(false), pointer.Bool(true)},
		"tx-scatter-gather-fraglist":     {pointer.Bool(true), pointer.Bool(false), pointer.Bool(true)},
		"tx-gso-partial":                 {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"tx-gre-segmentation":            {pointer.Bool(true), pointer.Bool(false), pointer.Bool(true)},
		"tx-udp-segmentation":            {pointer.Bool(true), pointer.Bool(false), pointer.Bool(true)},
		"loopback":                       {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"rx-gro-hw":                      {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"ntuple-filters":                 {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"macsec-hw-offload":              {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"rx-checksumming":                {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"tx-checksum-ipv4":               {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"tx-tcp6-segmentation":           {pointer.Bool(true), pointer.Bool(false), pointer.Bool(true)},
		"large-receive-offload":          {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"hsr-tag-rm-offload":             {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"tx-lockless":                    {pointer.Bool(true), pointer.Bool(true), pointer.Bool(true)},
		"tx-gre-csum-segmentation":       {pointer.Bool(true), pointer.Bool(false), pointer.Bool(true)},
		"hsr-dup-offload":                {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"tx-checksumming":                {pointer.Bool(true), nil, nil},
		"tx-gso-robust":                  {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"esp-tx-csum-hw-offload":         {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"l2-fwd-offload":                 {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"tx-checksum-ip-generic":         {pointer.Bool(true), pointer.Bool(false), pointer.Bool(true)},
		"tx-tcp-mangleid-segmentation":   {pointer.Bool(true), pointer.Bool(false), pointer.Bool(true)},
		"hw-tc-offload":                  {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"esp-hw-offload":                 {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"tx-scatter-gather":              {pointer.Bool(true), pointer.Bool(false), pointer.Bool(true)},
		"rx-gro-list":                    {pointer.Bool(false), pointer.Bool(false), pointer.Bool(false)},
		"tls-hw-tx-offload":              {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"tx-ipxip4-segmentation":         {pointer.Bool(true), pointer.Bool(false), pointer.Bool(true)},
		"netns-local":                    {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"rx-fcs":                         {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"tx-vlan-stag-hw-insert":         {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"tx-fcoe-segmentation":           {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"tx-tunnel-remcsum-segmentation": {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"scatter-gather":                 {pointer.Bool(true), nil, nil},
		"highdma":                        {pointer.Bool(true), pointer.Bool(false), pointer.Bool(true)},
		"tx-esp-segmentation":            {pointer.Bool(false), pointer.Bool(true), pointer.Bool(false)},
		"generic-receive-offload":        {pointer.Bool(true), pointer.Bool(false), pointer.Bool(true)},
	}

	fakeEthtool = func(parameters ...string) ([]byte, error) {
		if len(parameters) == 3 && parameters[0] == "--json" && parameters[1] == "-k" {
			iface := parameters[2]
			if iface == "dummy0" {
				return []byte(dummy0Output), nil
			}
			return []byte(notFoundOutput), fmt.Errorf(notFoundError)
		}
		if len(parameters) == 4 && parameters[0] == "-K" {
			iface := parameters[1]
			field := parameters[2]
			set := parameters[3]
			if iface == "dummy0" && field == "tx-checksumming" && (set == "on" || set == "off") {
				return []byte("Actual changes: <etc>"), nil
			}
			return []byte(notFoundOutput), fmt.Errorf(notFoundError)
		}
		return []byte{}, fmt.Errorf("unsupported input for fakeEthtool")
	}
)

func TestList(t *testing.T) {
	ethtool = fakeEthtool
	tcs := []struct {
		iface  string
		errStr string
	}{
		{"dummy0", ""},
		{"", "No such device"},
	}
	for _, tc := range tcs {
		iface, err := List(tc.iface)
		if tc.errStr != "" {
			if err == nil || !strings.Contains(err.Error(), tc.errStr) {
				t.Fatalf("Get(%s): expected to see error %q but got %q", tc.iface, tc.errStr, err)
			}
			continue
		}
		if err != nil {
			t.Fatalf("Get(%s): expected to see no error but got %q", tc.iface, err)
		}
		if !iface.Equals(dummy0OutputParsed) {
			t.Fatalf("Get(%s): Parsed attributes and expected attributes do not match, expected: %v, got: %v",
				tc.iface, dummy0OutputParsed, iface)
		}
	}
}

func TestSet(t *testing.T) {
	ethtool = fakeEthtool
	tcs := []struct {
		iface  string
		field  string
		enable bool
		errStr string
	}{
		{"dummy0", "tx-checksumming", true, ""},
		{"dummy10", "tx-checksumming", true, "No such device"},
	}
	for _, tc := range tcs {
		_, err := Set(tc.iface, tc.field, tc.enable)
		if tc.errStr != "" {
			if err == nil || !strings.Contains(err.Error(), tc.errStr) {
				t.Fatalf("Get(%s): expected to see error %q but got %q", tc.iface, tc.errStr, err)
			}
			continue
		}
		if err != nil {
			t.Fatalf("Get(%s): expected to see no error but got %q", tc.iface, err)
		}
	}
}
