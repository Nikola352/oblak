BIN_DIR := ./bin
SERVER  := $(BIN_DIR)/server
ROOTFS  := ./deployment/firecracker/rootfs.squashfs

.PHONY: build-server run-server rootfs run clean

build-server:
	go build -o $(SERVER) ./cmd/server

run-server: build-server
	sudo $(SERVER)

rootfs:
	./deployment/build-rootfs.sh

run: rootfs build-server
	sudo $(SERVER)

clean:
	rm -f $(SERVER) $(ROOTFS)
