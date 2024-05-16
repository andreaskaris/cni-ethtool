module github.com/andreaskaris/cni-ethtool

go 1.22.0

toolchain go1.22.2

require (
	github.com/containernetworking/cni v1.2.0
	github.com/containernetworking/plugins v1.4.1
	github.com/vishvananda/netlink v1.2.1-beta.2
	k8s.io/apimachinery v0.30.1
	k8s.io/utils v0.0.0-20240502163921-fe8a2dddb1d0
)

require (
	github.com/vishvananda/netns v0.0.4 // indirect
	golang.org/x/sys v0.18.0 // indirect
)
