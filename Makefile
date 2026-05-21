BIN_DIR := ./bin
SERVER  := $(BIN_DIR)/server
ROOTFS  := ./deployment/firecracker/rootfs.squashfs

.PHONY: build-server run-server rootfs run clean

build-server:
	go build -o $(SERVER) ./cmd/server

run-server: build-server
	sudo setcap cap_net_admin,cap_sys_admin+ep $(SERVER)
	$(SERVER)

rootfs:
	./deployment/build-rootfs.sh

run: rootfs build-server
	sudo setcap cap_net_admin,cap_sys_admin+ep $(SERVER)
	$(SERVER)

clean:
	rm -f $(SERVER) $(AGENT) $(ROOTFS)
