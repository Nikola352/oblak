BIN_DIR  := ./bin
SERVER   := $(BIN_DIR)/server
VM       := $(BIN_DIR)/vm
ANALYZER := $(BIN_DIR)/analyzer
ROOTFS   := ./deployment/firecracker/rootfs.squashfs

AGENT_SOURCES := deployment/build-rootfs.sh $(shell find cmd/agent internal/agent internal/agentproto -name '*.go')

.PHONY: build-server build-vm build-analyzer build \
        run-server run-vm run-analyzer \
        run install clean

# --- build targets ---

build-server:
	go build -o $(SERVER) ./cmd/server

build-vm:
	go build -o $(VM) ./cmd/vm
	sudo setcap cap_net_admin,cap_sys_admin,cap_dac_override,cap_kill+eip $(VM)

build-analyzer:
	go build -o $(ANALYZER) ./cmd/analyzer

build: build-server build-vm build-analyzer

# --- rootfs (rebuilt only when agent sources change) ---

$(ROOTFS): $(AGENT_SOURCES)
	./deployment/build-rootfs.sh

# --- install images to /srv/firecracker (run once after host-setup.sh) ---

install: $(ROOTFS)
	sudo install -m 644 ./deployment/firecracker/vmlinux-6.1.155 /srv/firecracker/
	sudo install -m 644 $(ROOTFS) /srv/firecracker/

# --- run targets (separate terminals) ---

run-server: build-server
	$(SERVER)

run-vm: build-vm $(ROOTFS) install
	$(VM)

run-analyzer: build-analyzer
	$(ANALYZER)

# --- combined launcher (prefixed output, single terminal) ---

run: build $(ROOTFS)
	@( $(SERVER) 2>&1 | sed 's/^/[server]   /' ) & \
	 ( $(VM) 2>&1 | sed 's/^/[vm]       /' ) & \
	 ( $(ANALYZER) 2>&1 | sed 's/^/[analyzer] /' ) & \
	 wait

# --- misc ---

clean:
	rm -f $(SERVER) $(VM) $(ANALYZER) $(ROOTFS)
