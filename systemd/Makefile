# Systemd Services Deployment Makefile
# Deploys systemd service files to BeagleBone Black

# Deployment configuration
DEPLOY_HOST ?= bbb1.wegman
SYSTEMD_DIR = /etc/systemd/system

# Service files to deploy
SERVICES = \
	enable-remote-detection-pru.service \
	enable-audio-pru.service \
	led-sound-light-control.service \
	pwm-setup.service \
	gpio-ir-setup.service

.PHONY: all deploy install enable disable start stop restart status clean help

# Default target
all: help

# Deploy service files to target device
deploy:
	@echo "Deploying systemd services to $(DEPLOY_HOST)..."
	@for service in $(SERVICES); do \
		echo "Copying $$service..."; \
		scp $$service $(DEPLOY_HOST):/tmp/$$service; \
		ssh $(DEPLOY_HOST) "sudo mv /tmp/$$service $(SYSTEMD_DIR)/$$service && sudo chmod 644 $(SYSTEMD_DIR)/$$service"; \
	done
	@echo "Reloading systemd daemon..."
	@ssh $(DEPLOY_HOST) "sudo systemctl daemon-reload"
	@echo "Service files deployed successfully"

# Install and enable services (run after deploy)
install: deploy
	@echo "Enabling services on $(DEPLOY_HOST)..."
	@for service in $(SERVICES); do \
		echo "Enabling $$service..."; \
		ssh $(DEPLOY_HOST) "sudo systemctl enable $$service" || true; \
	done
	@echo "Services enabled"

# Enable services without deploying
enable:
	@echo "Enabling services on $(DEPLOY_HOST)..."
	@for service in $(SERVICES); do \
		echo "Enabling $$service..."; \
		ssh $(DEPLOY_HOST) "sudo systemctl enable $$service" || true; \
	done
	@echo "Services enabled"

# Disable services
disable:
	@echo "Disabling services on $(DEPLOY_HOST)..."
	@for service in $(SERVICES); do \
		echo "Disabling $$service..."; \
		ssh $(DEPLOY_HOST) "sudo systemctl disable $$service" || true; \
	done
	@echo "Services disabled"

# Start services
start:
	@echo "Starting services on $(DEPLOY_HOST)..."
	@ssh $(DEPLOY_HOST) "sudo systemctl start enable-remote-detection-pru.service"
	@ssh $(DEPLOY_HOST) "sudo systemctl start enable-audio-pru.service"
	@ssh $(DEPLOY_HOST) "sudo systemctl start pwm-setup.service"
	@ssh $(DEPLOY_HOST) "sudo systemctl start gpio-ir-setup.service"
	@ssh $(DEPLOY_HOST) "sudo systemctl start led-sound-light-control.service"
	@echo "Services started"

# Stop services
stop:
	@echo "Stopping services on $(DEPLOY_HOST)..."
	@ssh $(DEPLOY_HOST) "sudo systemctl stop led-sound-light-control.service" || true
	@ssh $(DEPLOY_HOST) "sudo systemctl stop enable-remote-detection-pru.service" || true
	@ssh $(DEPLOY_HOST) "sudo systemctl stop enable-audio-pru.service" || true
	@echo "Services stopped"

# Restart services
restart: stop start

# Check status of services
status:
	@echo "Checking service status on $(DEPLOY_HOST)..."
	@echo ""
	@echo "=== PRU0 Remote Detection Service ==="
	@ssh $(DEPLOY_HOST) "sudo systemctl status enable-remote-detection-pru.service --no-pager" || true
	@echo ""
	@echo "=== PRU1 Audio Service ==="
	@ssh $(DEPLOY_HOST) "sudo systemctl status enable-audio-pru.service --no-pager" || true
	@echo ""
	@echo "=== Main Application Service ==="
	@ssh $(DEPLOY_HOST) "sudo systemctl status led-sound-light-control.service --no-pager" || true
	@echo ""
	@echo "=== PRU States ==="
	@echo "PRU0 (remoteproc1): $$(ssh $(DEPLOY_HOST) 'cat /sys/class/remoteproc/remoteproc1/state' 2>/dev/null || echo 'unavailable')"
	@echo "PRU1 (remoteproc2): $$(ssh $(DEPLOY_HOST) 'cat /sys/class/remoteproc/remoteproc2/state' 2>/dev/null || echo 'unavailable')"

# Clean - remove service files from target
clean:
	@echo "WARNING: This will remove service files from $(DEPLOY_HOST)"
	@echo "Press Ctrl+C to cancel, or Enter to continue..."
	@read dummy
	@echo "Stopping services..."
	@$(MAKE) stop
	@echo "Disabling services..."
	@$(MAKE) disable
	@echo "Removing service files..."
	@for service in $(SERVICES); do \
		echo "Removing $$service..."; \
		ssh $(DEPLOY_HOST) "sudo rm -f $(SYSTEMD_DIR)/$$service" || true; \
	done
	@ssh $(DEPLOY_HOST) "sudo systemctl daemon-reload"
	@echo "Service files removed"

# Display help
help:
	@echo "Systemd Services Deployment"
	@echo ""
	@echo "Usage:"
	@echo "  make deploy          - Copy service files to $(DEPLOY_HOST)"
	@echo "  make install         - Deploy and enable services"
	@echo "  make enable          - Enable services (autostart on boot)"
	@echo "  make disable         - Disable services (no autostart)"
	@echo "  make start           - Start all services"
	@echo "  make stop            - Stop all services"
	@echo "  make restart         - Restart all services"
	@echo "  make status          - Check service status"
	@echo "  make clean           - Remove service files from target (interactive)"
	@echo "  make help            - Show this help message"
	@echo ""
	@echo "Services managed:"
	@for service in $(SERVICES); do \
		echo "  - $$service"; \
	done
	@echo ""
	@echo "Override deployment host:"
	@echo "  make deploy DEPLOY_HOST=bbb2.wegman"
	@echo ""
	@echo "Typical workflow:"
	@echo "  1. make install      - Deploy and enable services"
	@echo "  2. make start        - Start services"
	@echo "  3. make status       - Verify services are running"
