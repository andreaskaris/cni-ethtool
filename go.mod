module github.com/andreaskaris/cni-ethtool

go 1.22

toolchain go1.22.2

require (
	github.com/andreaskaris/veth-ethtool v0.0.0-20240425221904-71b87c0081a5
	github.com/containernetworking/cni v1.2.0
	github.com/containernetworking/plugins v1.4.1
	k8s.io/utils v0.0.0-20240502163921-fe8a2dddb1d0
)

require (
	github.com/vishvananda/netns v0.0.4 // indirect
	golang.org/x/sys v0.18.0 // indirect
	k8s.io/klog v1.0.0 // indirect
)
