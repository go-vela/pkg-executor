# Copyright (c) 2020 Target Brands, Inc. All rights reserved.
#
# Use of this source code is governed by the LICENSE file in this repository.

build: binary-build

run: build docker-run kubernetes-run

docker-run: build docker-run

kubernetes-run: build kubernetes-run

#################################
######      Go clean       ######
#################################

clean:

	@go mod tidy
	@go vet ./...
	@go fmt ./...
	@echo "I'm kind of the only name in clean energy right now"

#################################
######    Build Binary     ######
#################################

binary-build:

	GOOS=darwin CGO_ENABLED=0 \
		go build \
		-o release/vela-runtime \
		github.com/go-vela/pkg-executor/cmd/vela-executor

########################################
#####          Docker Run          #####
########################################

docker-run:

	release/vela-executor \
		--log.level trace \
		--runtime.driver docker

############################################
#####          Kubernetes Run          #####
############################################

kubernetes-run:

	release/vela-runtime \
		--log.level trace \
		--runtime.driver kubernetes \
		--runtime.config ~/.kube/config \
		--runtime.namespace docker
