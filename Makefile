.PHONY: indigo android ios indigo-cross swarm evm all test clean
.PHONY: indigo-linux indigo-linux-386 indigo-linux-amd64 indigo-linux-mips64 indigo-linux-mips64le
.PHONY: indigo-linux-arm indigo-linux-arm-5 indigo-linux-arm-6 indigo-linux-arm-7 indigo-linux-arm64
.PHONY: indigo-darwin indigo-darwin-386 indigo-darwin-amd64
.PHONY: indigo-windows indigo-windows-386 indigo-windows-amd64
.PHONY: dep docker release

GOBIN = $(shell pwd)/build/bin
GO ?= latest

dep:
	dep ensure --vendor-only

indigo:
	cd cmd/indigo; go build -o ../../bin/indigo
	@echo "Done building."
	@echo "Run \"bin/indigo\" to launch indigo."

bootnode:
	cd cmd/bootnode; go build -o ../../bin/indigo-bootnode
	@echo "Done building."
	@echo "Run \"bin/indigo-bootnode\" to launch indigo."

docker:
	docker build -t indigo/indigo .

all: bootnode indigo

release:
	./release.sh

install: all
	cp bin/indigo-bootnode $(GOPATH)/bin/indigo-bootnode
	cp bin/indigo $(GOPATH)/bin/indigo

android:
	build/env.sh go run build/ci.go aar --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/indigo.aar\" to use the library."

ios:
	build/env.sh go run build/ci.go xcode --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/indigo.framework\" to use the library."

test:
	go test ./...

clean:
	rm -fr build/_workspace/pkg/ $(GOBIN)/*

# The devtools target installs tools required for 'go generate'.
# You need to put $GOBIN (or $GOPATH/bin) in your PATH to use 'go generate'.

devtools:
	env GOBIN= go get -u golang.org/x/tools/cmd/stringer
	env GOBIN= go get -u github.com/kevinburke/go-bindata/go-bindata
	env GOBIN= go get -u github.com/fjl/gencodec
	env GOBIN= go get -u github.com/golang/protobuf/protoc-gen-go
	env GOBIN= go install ./cmd/abigen
	@type "npm" 2> /dev/null || echo 'Please install node.js and npm'
	@type "solc" 2> /dev/null || echo 'Please install solc'
	@type "protoc" 2> /dev/null || echo 'Please install protoc'

# Cross Compilation Targets (xgo)

indigo-cross: indigo-linux indigo-darwin indigo-windows indigo-android indigo-ios
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/indigo-*

indigo-linux: indigo-linux-386 indigo-linux-amd64 indigo-linux-arm indigo-linux-mips64 indigo-linux-mips64le
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/indigo-linux-*

indigo-linux-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/386 -v ./cmd/indigo
	@echo "Linux 386 cross compilation done:"
	@ls -ld $(GOBIN)/indigo-linux-* | grep 386

indigo-linux-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/amd64 -v ./cmd/indigo
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/indigo-linux-* | grep amd64

indigo-linux-arm: indigo-linux-arm-5 indigo-linux-arm-6 indigo-linux-arm-7 indigo-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -ld $(GOBIN)/indigo-linux-* | grep arm

indigo-linux-arm-5:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-5 -v ./cmd/indigo
	@echo "Linux ARMv5 cross compilation done:"
	@ls -ld $(GOBIN)/indigo-linux-* | grep arm-5

indigo-linux-arm-6:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-6 -v ./cmd/indigo
	@echo "Linux ARMv6 cross compilation done:"
	@ls -ld $(GOBIN)/indigo-linux-* | grep arm-6

indigo-linux-arm-7:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-7 -v ./cmd/indigo
	@echo "Linux ARMv7 cross compilation done:"
	@ls -ld $(GOBIN)/indigo-linux-* | grep arm-7

indigo-linux-arm64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm64 -v ./cmd/indigo
	@echo "Linux ARM64 cross compilation done:"
	@ls -ld $(GOBIN)/indigo-linux-* | grep arm64

indigo-linux-mips:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips --ldflags '-extldflags "-static"' -v ./cmd/indigo
	@echo "Linux MIPS cross compilation done:"
	@ls -ld $(GOBIN)/indigo-linux-* | grep mips

indigo-linux-mipsle:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mipsle --ldflags '-extldflags "-static"' -v ./cmd/indigo
	@echo "Linux MIPSle cross compilation done:"
	@ls -ld $(GOBIN)/indigo-linux-* | grep mipsle

indigo-linux-mips64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64 --ldflags '-extldflags "-static"' -v ./cmd/indigo
	@echo "Linux MIPS64 cross compilation done:"
	@ls -ld $(GOBIN)/indigo-linux-* | grep mips64

indigo-linux-mips64le:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64le --ldflags '-extldflags "-static"' -v ./cmd/indigo
	@echo "Linux MIPS64le cross compilation done:"
	@ls -ld $(GOBIN)/indigo-linux-* | grep mips64le

indigo-darwin: indigo-darwin-386 indigo-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/indigo-darwin-*

indigo-darwin-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./cmd/indigo
	@echo "Darwin 386 cross compilation done:"
	@ls -ld $(GOBIN)/indigo-darwin-* | grep 386

indigo-darwin-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 -v ./cmd/indigo
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/indigo-darwin-* | grep amd64

indigo-windows: indigo-windows-386 indigo-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/indigo-windows-*

indigo-windows-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/386 -v ./cmd/indigo
	@echo "Windows 386 cross compilation done:"
	@ls -ld $(GOBIN)/indigo-windows-* | grep 386

indigo-windows-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/amd64 -v ./cmd/indigo
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/indigo-windows-* | grep amd64
