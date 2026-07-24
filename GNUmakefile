default: build

.PHONY: build
build:
	go build -o bin/terraform-provider-diskbg .

.PHONY: install
install: build
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/diskbg/diskbg/0.1.0/$$(go env GOOS)_$$(go env GOARCH)
	cp bin/terraform-provider-diskbg ~/.terraform.d/plugins/registry.terraform.io/diskbg/diskbg/0.1.0/$$(go env GOOS)_$$(go env GOARCH)/

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
