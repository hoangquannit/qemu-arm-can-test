.PHONY: build test check fmt setup-vcan run-ecu run-test build-arm run-qemu poc-image poc clean

build:
	go build ./cmd/...

test:
	go test ./...

check: fmt test build build-arm

fmt:
	gofmt -w cmd internal

setup-vcan:
	./scripts/setup-vcan.sh vcan0

run-ecu:
	go run ./cmd/body-ecu --interface vcan0

run-test:
	go run ./cmd/testctl run examples/headlight-test.yaml

build-arm:
	mkdir -p build/guest
	GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o build/guest/body-ecu ./cmd/body-ecu

run-qemu:
	./qemu/run-sabrelite.sh

poc-image:
	docker build -f Dockerfile.poc -t qemu-arm-can-poc:local .

poc: poc-image
	mkdir -p runs
	docker run --rm -v "$(CURDIR)/runs:/work/runs" qemu-arm-can-poc:local

clean:
	rm -rf build runs .cache
