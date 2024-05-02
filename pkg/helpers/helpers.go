package helpers

import (
	"fmt"
	"os/exec"
	"strings"

	types100 "github.com/containernetworking/cni/pkg/types/100"
)

const (
	TypeVeth      = "veth"
	TypeNetwork   = "network"
	NetNSLocation = "/run/netns"
)

// FindExecutable checks if an executable exists inside the container. If so, it returns that path.
// Otherwise, it also checks on /host.
func FindExecutable(name string) ([]string, error) {
	cmd := exec.Command("which", name)
	if out, err := cmd.Output(); err == nil {
		outStr := strings.TrimSuffix(string(out), "\n")
		return []string{outStr}, nil
	}

	cmd = exec.Command("chroot", "/host", "which", name)
	if out, err := cmd.Output(); err == nil {
		outStr := strings.TrimSuffix(string(out), "\n")
		return []string{"chroot", "/host", outStr}, nil
	}
	return nil, fmt.Errorf("could not find executable %q", name)
}

func GetNetNSLocation() string {
	return NetNSLocation
}

// RunCommand runs 'c parameters[0] parameters[1] ...'.'
func RunCommand(c string, parameters ...string) ([]byte, error) {
	bin, err := FindExecutable(c)
	if err != nil {
		return []byte{}, err
	}
	bin = append(bin, parameters...)
	cmd := exec.Command(bin[0], bin[1:]...)
	out, err := cmd.Output()
	return out, err
}

func ExtractInterfaceProperties(interfaces []*types100.Interface, interfaceName string) (string, string, error) {
	for _, intf := range interfaces {
		if intf.Name == interfaceName {
			if intf.Sandbox == "" {
				return "", "", fmt.Errorf("expected interfaces %q to be inside a namespace", intf.Name)
			}
			namespace := intf.Sandbox
			veth, err := ExtractVeth(interfaces, interfaceName, namespace)
			if err != nil {
				return "", "", err
			}
			return namespace, veth, nil
		}
	}
	return "", "", fmt.Errorf("could not find veth pair")
}

// TODO
func ExtractVeth(interfaces []*types100.Interface, interfaceName, namespace string) (string, error) {
	for _, intf := range interfaces {
		if intf.Sandbox == "" {
			return intf.Name, nil
		}
	}
	return "", fmt.Errorf("could not find veth pair")
}
