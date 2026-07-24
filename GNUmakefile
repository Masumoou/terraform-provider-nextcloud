default: build

.PHONY: build
build:
	go build -o bin/terraform-provider-nextcloud .

.PHONY: install
install: build
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/Masumoou/nextcloud/0.1.0/$$(go env GOOS)_$$(go env GOARCH)
	cp bin/terraform-provider-nextcloud ~/.terraform.d/plugins/registry.terraform.io/Masumoou/nextcloud/0.1.0/$$(go env GOOS)_$$(go env GOARCH)/

.PHONY: fmt
fmt:
	gofmt -w .

.PHONY: vet
vet:
	go vet ./...

.PHONY: test
test:
	go test ./... -v

.PHONY: tidy
tidy:
	go mod tidy
