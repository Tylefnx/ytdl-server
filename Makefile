# --- SYSTEM DETECTION ---
OS := $(shell uname -s)
ARCH := $(shell uname -m)

# Load local config if it exists
ifneq ("$(wildcard .ytdl-config)","")
    include .ytdl-config
    export $(shell sed 's/=.*//' .ytdl-config)
endif

ifeq ($(OS), Linux)
    ifneq ("$(wildcard /etc/gentoo-release)","")
        DISTRO := gentoo
    else
        DISTRO := $(shell grep -w "ID" /etc/os-release 2>/dev/null | cut -d= -f2 | tr -d '"')
    endif
endif

# --- VARIABLES ---
BINARY_NAME=ytdl-server
CMD_PATH=./cmd/server/main.go
INSTALL_PATH=/usr/local/bin/$(BINARY_NAME)
CONF_DEST=/usr/local/etc/ytdl
INITD_PATH=/etc/init.d/$(BINARY_NAME)
BSD_RC_PATH=/usr/local/etc/rc.d/$(BINARY_NAME)

.PHONY: all setup install-deps build build-freebsd clean install check-root generate-env

all: setup build

check-root:
	@if [ $$(id -u) -ne 0 ]; then \
		echo "❌ ERROR: Root privileges required."; \
		exit 1; \
	fi

setup:
	@if [ ! -f .ytdl-config ]; then \
		echo "❌ ERROR: .ytdl-config not found!"; \
		echo "Please run: cp .ytdl-config.example .ytdl-config"; \
		echo "Then edit .ytdl-config with your settings."; \
		exit 1; \
	fi
	@echo ">>> 🔍 Analyzing System: $(OS) ($(DISTRO))"
	@$(MAKE) install-deps
	@$(MAKE) generate-env
	@go mod tidy
	@echo ">>> ✅ Setup complete."

install-deps: check-root
ifeq ($(OS), FreeBSD)
	pkg install -y ffmpeg go gmake
else ifeq ($(OS), Linux)
	@case "$(DISTRO)" in \
		ubuntu|debian|kali|raspbian) apt update && apt install -y ffmpeg golang ;; \
		arch|manjaro) pacman -S --noconfirm ffmpeg go ;; \
		gentoo) emerge --ask --noreplace media-video/ffmpeg dev-lang/go ;; \
		*) echo "⚠️  Manual dependency install might be needed for $(DISTRO)" ;; \
	esac
endif

generate-env:
	@echo ">>> Generating .env file..."
	@echo "PORT=$(PORT)" > .env
	@echo "MAX_CONCURRENT_JOBS=$(MAX_CONCURRENT_JOBS)" >> .env
	@echo "DOWNLOAD_DIR=$(DOWNLOAD_DIR)" >> .env
	@echo "TEMP_DIR=$(TEMP_DIR)" >> .env
	@echo "CLEAN_UP_AFTER_MINUTES=$(CLEAN_UP_MINUTES)" >> .env
	@echo "ALLOWED_ORIGINS=$(ALLOWED_ORIGINS)" >> .env

build:
	@echo ">>> 🔨 Building binary..."
	go build -o $(BINARY_NAME) $(CMD_PATH)

build-freebsd:
	@echo ">>> 🔨 Cross-compiling for FreeBSD..."
	GOOS=freebsd GOARCH=amd64 go build -o $(BINARY_NAME)-freebsd $(CMD_PATH)

clean:
	@echo ">>> 🧹 Cleaning up..."
	rm -f $(BINARY_NAME) $(BINARY_NAME)-freebsd .env
	rm -rf temp/* downloads/*

install: check-root build generate-env
	@echo ">>> ⚠️  Checking for user: $(SERVICE_USER)"
	@id $(SERVICE_USER) >/dev/null 2>&1 || (echo "❌ Error: User $(SERVICE_USER) does not exist!"; exit 1)
	
	@echo ">>> 🚀 Installing binary..."
	cp $(BINARY_NAME) $(INSTALL_PATH)
	chmod +x $(INSTALL_PATH)
	
	@echo ">>> 📂 Creating config and data directories..."
	# Create all directories including the base DATA_DIR
	mkdir -p $(CONF_DEST)
	mkdir -p $(DATA_DIR)
	mkdir -p $(DOWNLOAD_DIR)
	mkdir -p $(TEMP_DIR)
	cp .env $(CONF_DEST)/.env
	
	@echo ">>> 🔑 Setting permissions for $(SERVICE_USER)..."
	chown -R $(SERVICE_USER):$(SERVICE_USER) $(DATA_DIR)
	chown -R $(SERVICE_USER):$(SERVICE_USER) $(CONF_DEST)
	
	@echo ">>> ⚙️ Configuring Service..."
	@if [ "$(OS)" = "FreeBSD" ]; then \
		echo ">>> FreeBSD detected. Installing rc.d script..."; \
		sed -e "s|{{USER}}|$(SERVICE_USER)|g" \
			-e "s|{{DIR}}|$(DATA_DIR)|g" \
			-e "s|{{CONF}}|$(CONF_DEST)/.env|g" \
			ytdl-server.freebsd.template > $(BSD_RC_PATH); \
		chmod +x $(BSD_RC_PATH); \
		echo "✅ FreeBSD service installed. Enable with: sysrc $(BINARY_NAME)_enable=YES"; \
	elif [ -d "/run/systemd/system" ]; then \
		echo ">>> systemd detected. Installing unit file..."; \
		sed -e "s|{{USER}}|$(SERVICE_USER)|g" \
			-e "s|{{DIR}}|$(DATA_DIR)|g" \
			-e "s|{{CONF}}|$(CONF_DEST)/.env|g" \
			ytdl-server.service.template > /etc/systemd/system/$(BINARY_NAME).service; \
		systemctl daemon-reload; \
		echo "✅ systemctl service installed."; \
	elif [ -d "/etc/init.d" ]; then \
		echo ">>> OpenRC detected. Installing init.d script..."; \
		sed -e "s|{{USER}}|$(SERVICE_USER)|g" \
			-e "s|{{DIR}}|$(DATA_DIR)|g" \
			-e "s|{{CONF}}|$(CONF_DEST)/.env|g" \
			ytdl-server.initd.template > $(INITD_PATH); \
		chmod +x $(INITD_PATH); \
		echo "✅ OpenRC service installed."; \
	fi