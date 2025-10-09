#!/bin/bash

# Exit on error
set -e

# --- Helper Functions ---
print_info() {
    echo -e "\e[34m[INFO]\e[0m $1"
}

print_success() {
    echo -e "\e[32m[SUCCESS]\e[0m $1"
}

print_warning() {
    echo -e "\e[33m[WARNING]\e[0m $1"
}

print_error() {
    echo -e "\e[31m[ERROR]\e[0m $1"
    exit 1
}

# --- Installation Steps ---

# 0. Check for existing installation
check_for_existing_installation() {
    INSTALL_DIR="$HOME/.local/bin"
    BINARY_PATH="$INSTALL_DIR/infrapulse"

    if [ -f "$BINARY_PATH" ]; then
        print_warning "Existing 'infrapulse' binary found at $BINARY_PATH."
        read -p "Do you want to remove the existing installation and proceed with a fresh install? (y/N): " -n 1 -r
        echo # (optional) move to a new line
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            print_info "Removing existing 'infrapulse' binary..."
            rm "$BINARY_PATH"
            print_success "Existing 'infrapulse' removed."
        else
            print_info "Skipping installation. Existing 'infrapulse' binary will not be replaced."
            exit 0
        fi
    fi
}

check_for_existing_installation

# 1. Check for Go
print_info "Checking for Go installation..."
if ! command -v go &> /dev/null; then
    print_error "Go is not installed. Please install Go (version 1.20 or later) and try again."
    echo "Installation instructions can be found at: https://golang.org/doc/install"
    exit 1
fi
print_success "Go is installed."

# 2. Tidy Go modules
print_info "Tidying Go modules..."
go mod tidy
print_success "Go modules tidied."

# 3. Build the binary
print_info "Building the 'infrapulse' binary..."
go build -o infrapulse main.go
print_success "Binary built successfully."

# 4. Create configuration directory
CONFIG_DIR="$HOME/.config/infrapulse"
print_info "Creating configuration directory at $CONFIG_DIR..."
mkdir -p "$CONFIG_DIR"
print_success "Configuration directory created."

# 5. Copy default configuration
if [ ! -f "$CONFIG_DIR/servers.yaml" ]; then
    print_info "Copying default 'servers.yaml' to $CONFIG_DIR..."
    cat > "$CONFIG_DIR/servers.yaml" << EOL
servers:
  - name: "Localhost"
    host: "127.0.0.1"
    ports:
      - 80
      - 443
  - name: "Google DNS"
    host: "8.8.8.8"
    ports:
      - 53
EOL
    print_success "Default 'servers.yaml' created."
else
    print_info "Configuration file already exists at $CONFIG_DIR/servers.yaml. Skipping creation of default."
fi

# 6. Install the binary
INSTALL_DIR="$HOME/.local/bin"
print_info "Installing 'infrapulse' to $INSTALL_DIR..."
mkdir -p "$INSTALL_DIR"
mv infrapulse "$INSTALL_DIR/"
print_success "'infrapulse' installed successfully."

# 7. Check if INSTALL_DIR is in PATH
if [[ ":$PATH:" != ":$INSTALL_DIR:"* ]]; then
    print_warning "$INSTALL_DIR is not in your PATH. You might need to add it to your shell's configuration file (e.g., ~/.bashrc, ~/.zshrc)."
    echo "You can add it by running:"
    echo "echo 'export PATH=\"$HOME/.local/bin:$PATH\"' >> ~/.bashrc && source ~/.bashrc"
fi

# 8. Final instructions
print_success "Installation complete!"
echo "You can now run 'infrapulse' from anywhere in your terminal."
echo "To customize the servers to monitor, edit the configuration file at: $CONFIG_DIR/servers.yaml"
echo "To enable email alerts, create a 'config.yaml' file in the same directory with your SMTP server details."
echo "To run InfraPulse in the background, use 'nohup infrapulse -d &' or a service manager like systemd."
