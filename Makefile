KIND_CLUSTER_NAME ?= cni-ethtool
MAKEFILE_DIR = $(shell cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd)
OUTPUT_DIR = $(MAKEFILE_DIR)/_output
KIND_CLUSTER_CONFIG ?= $(OUTPUT_DIR)/kubeconfig
CONTAINER_IMAGE ?= quay.io/akaris/cni-ethtool:latest
ETHTOOL_CONTAINER_IMAGE ?= quay.io/akaris/fedora:ethtool
SAVED_IMAGE ?= $(OUTPUT_DIR)/image.tar
ETHTOOL_SAVED_IMAGE ?= $(OUTPUT_DIR)/ethool-image.tar
CONTAINER_ENGINE ?= KIND_EXPERIMENTAL_PROVIDER=podman
# CONTAINER_ENGINE = 

.PHONY: build
build:
	CGO_ENABLED=0 go build -o _output/cni-ethtool

.PHONY: test
test:
	go test -v -count 1 ./...

.PHONY: test-coverage
test-coverage:
	go test -v -coverprofile=$(OUTPUT_DIR)/cover.out -count 1 ./...
	go tool cover -html=$(OUTPUT_DIR)/cover.out

.PHONY: build-container
build-container:
	podman build -t $(CONTAINER_IMAGE) .

.PHONY: build-ethtool-container
build-ethtool-container:
	podman build -t $(ETHTOOL_CONTAINER_IMAGE) -f Dockerfile.ethtool .

.PHONY: create-kind
create-kind:
	$(CONTAINER_ENGINE) kind create cluster --name $(KIND_CLUSTER_NAME) --kubeconfig $(KIND_CLUSTER_CONFIG)

.PHONY: destroy-kind
destroy-kind:
	$(CONTAINER_ENGINE) kind delete cluster --name $(KIND_CLUSTER_NAME)

.PHONY: load-container-image-kind
load-container-image-kind:
	podman save $(CONTAINER_IMAGE) > $(SAVED_IMAGE)
	$(CONTAINER_ENGINE) kind load image-archive $(SAVED_IMAGE) --name $(KIND_CLUSTER_NAME)
	rm -f $(SAVED_IMAGE)

.PHONY: load-ethtool-container-image-kind
load-ethtool-container-image-kind:
	podman save $(ETHTOOL_CONTAINER_IMAGE) > $(ETHTOOL_SAVED_IMAGE)
	$(CONTAINER_ENGINE) kind load image-archive $(ETHTOOL_SAVED_IMAGE) --name $(KIND_CLUSTER_NAME)
	rm -f $(ETHTOOL_SAVED_IMAGE)

.PHONY: deploy
deploy:
	kubectl kustomize $(MAKEFILE_DIR)/config/kubernetes | kubectl apply -f -

.PHONY: undeploy
undeploy:
	kubectl kustomize $(MAKEFILE_DIR)/config/kubernetes | kubectl delete -f -

.PHONY: build-and-deploy
build-and-deploy: build-container load-container-image-kind build-ethtool-container load-ethtool-container-image-kind deploy

.PHONY: e2e-test
e2e-test:
	export KUBECONFIG=$(KIND_CLUSTER_CONFIG) && \
	cd $(MAKEFILE_DIR)/e2e && \
	go test -v -count 1 ./... $(RUN)

.PHONY: build-and-e2e-test
build-and-e2e-test: build-container load-container-image-kind build-ethtool-container load-ethtool-container-image-kind e2e-test
