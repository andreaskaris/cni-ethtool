FROM golang:1.22 as build
WORKDIR /build
COPY . .
RUN make build

FROM registry.fedoraproject.org/fedora-minimal:latest
COPY --from=build /build/_output/cni-ethtool /usr/local/bin/cni-ethtool
