package ethtool

import (
	"encoding/json"
	"fmt"

	"github.com/andreaskaris/veth-ethtool/pkg/helpers"
)

type OffloadList map[string]OffloadAttributes

func (o OffloadList) Equals(b OffloadList) bool {
	if len(o) != len(b) {
		return false
	}
	for k := range o {
		if !o[k].Equals(b[k]) {
			return false
		}
	}
	return true
}

type OffloadAttributes struct {
	Active    *bool `json:"active"`
	Fixed     *bool `json:"fixed"`
	Requested *bool `json:"requested"`
}

func (o OffloadAttributes) Equals(b OffloadAttributes) bool {
	if !boolPointerEquals(o.Active, b.Active) {
		return false
	}
	if !boolPointerEquals(o.Fixed, b.Fixed) {
		return false
	}
	if !boolPointerEquals(o.Requested, b.Requested) {
		return false
	}
	return true
}

func boolPointerEquals(a *bool, b *bool) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil {
		return false
	}
	if b == nil {
		return false
	}
	return *a == *b
}

// List lists the offloading attributes of an interfaces.
func List(namespace, iface string) (OffloadList, error) {
	if namespace != "" {
		// TODO
		return OffloadList{}, nil
	}
	// First, read the output of ethtool. Ethtool returns mixed JSON output, therefore read the nested elements
	// as json.RawMessage. In the next step, we will then filter out the "ifname" field which is the only one that does
	// not match our desired format.
	var ol []map[string]json.RawMessage
	out, err := ethtool("--json", "-k", iface)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(out, &ol)
	if err != nil {
		return nil, err
	}
	if len(ol) != 1 {
		return nil, fmt.Errorf("unexpected length for offload list: %v", ol)
	}

	// Now, filter out "ifname". Then, unmarshal all values individually and write everything into one homogenous
	// OffloadList.
	offloadList := OffloadList{}
	for k, v := range ol[0] {
		var ola OffloadAttributes
		if k == "ifname" {
			continue
		}
		err := json.Unmarshal(v, &ola)
		if err != nil {
			return nil, fmt.Errorf("cannot unmarshal attribute %q with value %q", k, v)
		}
		offloadList[k] = ola
	}
	return offloadList, nil
}

// Set sets the offloading attribute of an interface.
func Set(namespace, iface, field string, enable bool) ([]byte, error) {
	if namespace != "" {
		// TODO
		return []byte{}, nil
	}
	set := "off"
	if enable {
		set = "on"
	}
	return ethtool("-K", iface, field, set)
}

var ethtool = func(parameters ...string) ([]byte, error) {
	return helpers.RunCommand("ethtool", parameters...)
}
