BIN      := smb-provisioner
REGISTRY ?= quay.io
IMAGE    ?= nixpanic/csi-provisioner-smb
VERSION  ?= latest

$(BIN): $(shell git ls-files './cmd/*.go' './internal/*.go')
	go build -o $(BIN) ./cmd/smbplugin

check:
	go test -a -v ./...

image:
	buildah bud -t $(REGISTRY)/$(IMAGE):$(VERSION) -f deploy/Containerfile .

push:
	podman push $(REGISTRY)/$(IMAGE):$(VERSION)

clean:
	$(RM) $(BIN)
