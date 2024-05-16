package helpers

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	types100 "github.com/containernetworking/cni/pkg/types/100"
	"github.com/vishvananda/netlink"
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

// ExtractInterfaceNamespace takes an interface name inside the namespace, e.g. "eth0", and searches the list of
// provided interfaces for that interface name. On success, it returns the name of the namespace of that interface
// (the content of the Sandbox field).
func ExtractInterfaceNamespace(interfaces []*types100.Interface, interfaceName string) (string, error) {
	for _, intf := range interfaces {
		if intf.Name == interfaceName {
			if intf.Sandbox == "" {
				return "", fmt.Errorf("expected interfaces %q to be inside a namespace", intf.Name)
			}
			namespace := intf.Sandbox
			return namespace, nil
		}
	}
	return "", fmt.Errorf("could not find namespaced interface %s", interfaceName)
}

// GetInterfaceIndex will return the interface index for the provided interface name.
func GetInterfaceIndex(interfaceName string) (int, error) {
	link, err := netlink.LinkByName(interfaceName)
	if err != nil {
		return -1, err
	}
	return link.Attrs().Index, nil
}

// ExtractVeth iterates over the list of provided interfaces. For each interface, it checks:
// * That the interface is in the global namespace.
// * That the ParentIndex (peer index) of the interface equals the peerInterfaceIndex that we are looking for.
// * That the NetNsID equals the provided netnsID.
// If these conditions are met, it will return the name of the matching interface.
func ExtractVeth(interfaces []*types100.Interface, netnsID, peerInterfaceIndex int) (string, error) {
	for _, intf := range interfaces {
		// We are only interested in interfaces in the global namespace.
		if intf.Sandbox != "" {
			continue
		}
		// Get the netlink interface.
		link, err := netlink.LinkByName(intf.Name)
		if err != nil {
			continue
		}
		// Next, make sure that the interface index of the peer matches the parent index.
		if link.Attrs().ParentIndex != peerInterfaceIndex {
			continue
		}
		// Next, make sure that the interface's peer netns matches what we are looking for.
		// link.Attrs().NetNsID holds the netns ID of the peer. We then compare to the netns IDs of files /run/netns.
		if link.Attrs().NetNsID != netnsID {
			continue
		}
		return intf.Name, nil
	}
	return "", fmt.Errorf("could not find veth peer for netnsID %d, peerInterfaceIndex %d", netnsID, peerInterfaceIndex)
}

// FindNetNSID expects a path to a netns and will return the ID of the corresponding netns.
func FindNetNSID(netnsPath string) (int, error) {
	f, err := os.Open(netnsPath)
	if err != nil {
		return -1, fmt.Errorf("could not open file %q for reading, err: %q", netnsPath, err)
	}

	id, err := netlink.GetNetNsIdByFd(int(f.Fd()))
	if err != nil {
		return -1, fmt.Errorf("issue running netlink.GetNetNsIdByFd, file: %q, err: %q", netnsPath, err)
	}

	return id, nil
}
