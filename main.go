package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/akaris/cni-ethtool/pkg/ethtool"
	"github.com/akaris/cni-ethtool/pkg/helpers"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	types100 "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/cni/pkg/version"
	bv "github.com/containernetworking/plugins/pkg/utils/buildversion"
)

// PluginConf is whatever you expect your configuration json to be. This is whatever
// is passed in on stdin. Your plugin may wish to expose its functionality via
// runtime args, see CONVENTIONS.md in the CNI spec.
type PluginConf struct {
	// This embeds the standard NetConf structure which allows your plugin
	// to more easily parse standard fields like Name, Type, CNIVersion,
	// and PrevResult.c
	types.NetConf

	Debug   bool                                  `json:"debug"`
	LogFile string                                `json:"logfile"`
	Ethtool map[string]map[string]map[string]bool `json:"ethtool"`
}

type customLogger struct {
	slog.Logger
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
	return &customLogger{*slog.New(slog.NewJSONHandler(f, &slog.HandlerOptions{Level: programLevel}))}, nil
}

func parseConfig(stdin []byte) (*PluginConf, error) {
	conf := PluginConf{}

	if err := json.Unmarshal(stdin, &conf); err != nil {
		return nil, fmt.Errorf("failed to parse network configuration: %v", err)
	}

	if err := version.ParsePrevResult(&conf.NetConf); err != nil {
		return nil, fmt.Errorf("could not parse prevResult: %v", err)
	}

	// Do any validation here
	/*if conf.AnotherAwesomeArg == "" {
		return nil, fmt.Errorf("anotherAwesomeArg must be specified")
	}*/

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

	for interfaceName, ethtoolConfig := range conf.Ethtool {
		namespace, peerInterfaceName, err := helpers.ExtractInterfaceProperties(prevResult.Interfaces, interfaceName)
		if err != nil {
			return err
		}
		logger.Debug("cmdAdd", "eth", interfaceName, "namespace", namespace, "veth", peerInterfaceName)
		if selfSettings, ok := ethtoolConfig["self"]; ok {
			for parameter, setting := range selfSettings {
				logger.Debug("cmdAdd ethtool.Set", "namespace", namespace, "interfaceName", interfaceName, "parameter", parameter, "setting", setting)
				ethtool.Set(namespace, interfaceName, parameter, setting)
			}
		}
		if peerSettings, ok := ethtoolConfig["peer"]; ok {
			for parameter, setting := range peerSettings {
				logger.Debug("cmdAdd ethtool.Set", "namespace", "", "peerInterfaceName", peerInterfaceName, "parameter", parameter, "setting", setting)
				ethtool.Set("", peerInterfaceName, parameter, setting)
			}
		}
	}

	// Pass through the result for the next plugin
	return types.PrintResult(prevResult, conf.CNIVersion)
}

// cmdDel is called for DELETE requests
func cmdDel(args *skel.CmdArgs) error {
	conf, err := parseConfig(args.StdinData)
	if err != nil {
		return err
	}
	logger, err := newCustomLogger(conf)
	if err != nil {
		return err
	}
	logger.Debug("cmdDel", "conf", conf)

	// Do your delete here

	return nil
}

func main() {
	skel.PluginMainFuncs(skel.CNIFuncs{Add: cmdAdd, Check: cmdCheck, Del: cmdDel}, version.All, bv.BuildString("cni-ethtool"))
}

func cmdCheck(_ *skel.CmdArgs) error {
	return fmt.Errorf("not implemented")
}
