package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/andreaskaris/cni-ethtool/pkg/ethtool"
	"github.com/andreaskaris/cni-ethtool/pkg/helpers"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	types100 "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/cni/pkg/version"
	"github.com/containernetworking/plugins/pkg/ns"
	bv "github.com/containernetworking/plugins/pkg/utils/buildversion"
)

const (
	pluginName = "cni-ethtool"
)

// PluginConf is whatever you expect your configuration json to be. This is whatever
// is passed in on stdin. Your plugin may wish to expose its functionality via
// runtime args, see CONVENTIONS.md in the CNI spec.
type PluginConf struct {
	// This embeds the standard NetConf structure which allows your plugin
	// to more easily parse standard fields like Name, Type, CNIVersion,
	// and PrevResult.c
	types.NetConf

	Debug   bool                   `json:"debug"`
	LogFile string                 `json:"logfile"`
	Ethtool ethtool.EthtoolConfigs `json:"ethtool"`
}

type customLogger struct {
	slog.Logger
	PluginName string
}

func newCustomLogger(conf *PluginConf) (*customLogger, error) {
	var err error
	f := os.Stderr
	if conf.LogFile != "" {
		if f, err = os.OpenFile(conf.LogFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600); err != nil {
			return nil, fmt.Errorf("could not write to file %q, err: %q", conf.LogFile, err)
		}
	}
	programLevel := slog.LevelInfo
	if conf.Debug {
		programLevel = slog.LevelDebug
	}
	return &customLogger{*slog.New(slog.NewJSONHandler(f, &slog.HandlerOptions{Level: programLevel})), pluginName}, nil
}

func (c *customLogger) Debug(msg string, args ...any) {
	a := append([]any{"cni-plugin", c.PluginName}, args...)
	c.Logger.Debug(msg, a...)
}

func (c *customLogger) Info(msg string, args ...any) {
	a := append([]any{"cni-plugin", c.PluginName}, args...)
	c.Logger.Info(msg, a...)
}

func parseConfig(stdin []byte) (*PluginConf, error) {
	conf := PluginConf{}

	if err := json.Unmarshal(stdin, &conf); err != nil {
		return nil, fmt.Errorf("failed to parse network configuration: %v", err)
	}

	if err := version.ParsePrevResult(&conf.NetConf); err != nil {
		return nil, fmt.Errorf("could not parse prevResult: %v", err)
	}

	if !conf.Ethtool.IsValid() {
		return nil, fmt.Errorf("provided ethtool configuration %+v is not valid", conf.Ethtool)
	}

	return &conf, nil
}

// cmdAdd is called for ADD requests
func cmdAdd(args *skel.CmdArgs) error {
	conf, err := parseConfig(args.StdinData)
	if err != nil {
		return err
	}
	logger, err := newCustomLogger(conf)
	if err != nil {
		return err
	}
	logger.Debug("cmdAdd", "conf", conf, "conf.Logfile", conf.LogFile, "conf.Debug", conf.Debug)

	// This plugin must be called as a chained plugin.
	if conf.PrevResult == nil {
		return fmt.Errorf("must be called as chained plugin")
	}

	// Convert the PrevResult to a concrete Result type that can be modified.
	prevResult, err := types100.GetResult(conf.PrevResult)
	if err != nil {
		return fmt.Errorf("failed to convert prevResult: %v", err)
	}
	logger.Debug("cmdAdd", "prevResult", prevResult)

	// Iterate over each interface of the Ethtool config, e.g. "eth0", "eth1", ...
	for interfaceName, ethtoolConfig := range conf.Ethtool {
		// Get the namespace name and the netns.
		namespace, err := helpers.ExtractInterfaceNamespace(prevResult.Interfaces, interfaceName)
		if err != nil {
			return err
		}
		netns, err := ns.GetNS(namespace)
		if err != nil {
			return err
		}

		// Get the interface index of the interface inside the namespace (e.g. "eth0" has index "2").
		var interfaceIndex int
		err = netns.Do(func(_ ns.NetNS) error {
			var err error
			interfaceIndex, err = helpers.GetInterfaceIndex(interfaceName)
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return err
		}

		logger.Debug("cmdAdd", "step", "found interface namespace and index", "interfaceName", interfaceName,
			"namespace", namespace, "interfaceIndex", interfaceIndex)

		// Set ethtool parameters inside the pod. The "self" index.
		// Set ethtool parameters inside the pod, one by one.
		for parameter, setting := range ethtoolConfig.GetSelf() {
			logger.Debug("cmdAdd", "step", "ethtool set parameter inside namespace", "namespace", namespace,
				"interfaceName", interfaceName, "parameter", parameter, "setting", setting)
			err = netns.Do(func(_ ns.NetNS) error {
				_, err := ethtool.Set(interfaceName, parameter, setting)
				return err
			})
			if err != nil {
				return err
			}
		}
		// Set ethtool parameters for veth peer in global namespace, if one exists. The "peer" index.
		if peerSettings := ethtoolConfig.GetPeer(); peerSettings != nil {
			netnsID, err := helpers.FindNetNSID(namespace)
			if err != nil {
				return fmt.Errorf("could not find namespace id for netns %s, err: %q", namespace, err)
			}
			peerInterfaceName, err := helpers.ExtractVeth(prevResult.Interfaces, netnsID, interfaceIndex)
			if err != nil {
				return fmt.Errorf("could not find veth peer for interface %s in netns %s, err: %q",
					interfaceName, namespace, err)
			}
			logger.Debug("cmdAdd", "step", "found netnsID and peerInterfaceName", "netnsID", netnsID,
				"peerInterfaceName", peerInterfaceName)
			// Set ethtool parameters in the global namespace, one by one.
			for parameter, setting := range peerSettings {
				logger.Debug("cmdAdd", "step", "ethtool set parameter inside global namespace",
					"peerInterfaceName", peerInterfaceName, "parameter", parameter, "setting", setting)
				if _, err := ethtool.Set(peerInterfaceName, parameter, setting); err != nil {
					return err
				}
			}
		}
	}
	logger.Debug("cmdAdd", "done", true)
	// Pass through the result for the next plugin
	return types.PrintResult(prevResult, conf.CNIVersion)
}

func main() {
	skel.PluginMainFuncs(skel.CNIFuncs{Add: cmdAdd}, version.All, bv.BuildString("cni-ethtool"))
}
