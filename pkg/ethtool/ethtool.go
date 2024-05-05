package ethtool

import (
	"encoding/json"

	"github.com/andreaskaris/veth-ethtool/pkg/helpers"
)

const (
	SelfClassifier = "self"
	PeerClassifier = "peer"
)

type EthtoolConfig map[string]map[string]bool

func (e EthtoolConfig) GetSelf() map[string]bool {
	self, ok := e[SelfClassifier]
	if ok {
		return self
	}
	return nil
}

func (e EthtoolConfig) GetPeer() map[string]bool {
	self, ok := e[PeerClassifier]
	if ok {
		return self
	}
	return nil
}

func (e EthtoolConfig) IsValid() bool {
	if len(e) == 2 {
		_, ok1 := e[SelfClassifier]
		_, ok2 := e[PeerClassifier]
		return ok1 && ok2
	}
	if len(e) == 1 {
		_, ok := e[SelfClassifier]
		return ok
	}
	return false
}

func (e EthtoolConfig) String() string {
	b, err := json.Marshal(e)
	if err != nil {
		return ""
	}
	return string(b)
}

type EthtoolConfigs map[string]EthtoolConfig

func (es EthtoolConfigs) IsValid() bool {
	for _, ethtoolConfig := range es {
		if !ethtoolConfig.IsValid() {
			return false
		}

	}
	return true
}

func (es EthtoolConfigs) String() string {
	b, err := json.Marshal(es)
	if err != nil {
		return ""
	}
	return string(b)
}

// Set sets the offloading attribute of an interface.
func Set(iface, field string, enable bool) ([]byte, error) {
	set := "off"
	if enable {
		set = "on"
	}
	return ethtool("-K", iface, field, set)
}

var ethtool = func(parameters ...string) ([]byte, error) {
	return helpers.RunCommand("ethtool", parameters...)
}
