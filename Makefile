.PHONY: build test vet structure-gate structure-gate-baseline

build:
	go build ./...

test:
	go test ./...

vet:
	go vet ./...

structure-gate:
	cd ../servertools/structure-gate/go-structure-gate && go run . --root ../../../go-ygctl --mode changed

structure-gate-baseline:
	cd ../servertools/structure-gate/go-structure-gate && go run . --root ../../../go-ygctl --mode baseline
