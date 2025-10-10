# InfraPulse

InfraPulse is a lightweight and efficient Go-based tool designed for comprehensive infrastructure health monitoring. It continuously checks the status of your servers and services, providing real-time insights and sending immediate email alerts upon detecting any failures.

## Table of Contents

- [Features](#features)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Usage](#usage)
- [Configuration](#configuration)
- [Building from Source](#building-from-source)
- [Development](#development)
- [Contributing](#contributing)
- [License](#license)

## Features

- **Concurrent Health Checks:** Uses goroutines to check multiple hosts and ports in parallel.
- **Multi-faceted Monitoring:**
  - ICMP Ping to check host reachability.
  - TCP Port checks to verify service availability.
- **Daemon Mode:** Run InfraPulse as a background service to continuously monitor your infrastructure.
- **YAML Configuration:** Easily define servers and SMTP settings in simple `.yaml` files.
- **Email Alerts:** Automatically sends an email via SMTP when a service is detected as down.
- **CLI Reporting:** Clean, color-coded status reports in the terminal.

## Prerequisites

- Go version 1.20 or later.

## Installation

An installation script is provided to build and install InfraPulse.

1. Clone the repository:
   ```sh
   git clone https://github.com/saransridatha/InfraPulse
   cd InfraPulse
   ```

2. Make the installation script executable:
   ```sh
   chmod +x install.sh
   ```

3. Run the installation script:
   ```sh
   ./install.sh
   ```

The script will build the `infrapulse` binary and install it to `$HOME/.local/bin`. It will also create a configuration directory at `$HOME/.config/infrapulse` with a default `servers.yaml` file.

## Usage

InfraPulse can be run in several modes:

- **One-time health check:**
  ```sh
  infrapulse
  ```

- **Run in monitoring loop mode:**
  ```sh
  infrapulse -d
  ```
  To run in the background, use `nohup` or a service manager (e.g., `systemd`):
  ```sh
  nohup infrapulse -d &
  ```
  To stop the background process, use OS-level commands (e.g., `pkill -f infrapulse`).

### Command-Line Flags

- `-config /path/to/servers.yaml`: Specify a custom path to the `servers.yaml` file.
- `-i <interval>`: Override the check interval in daemon mode (e.g., `30s`, `5m`, `1h`).
      Example: `infrapulse -d -i 30s` to run checks every 30 seconds.
- `-d`: Run in monitoring loop mode. This will keep running until manually stopped. Use `nohup` or a service manager to run in the background.
- `--stop`: This flag is deprecated. Use OS-level commands to stop background processes.

## Configuration

InfraPulse is configured using two YAML files located in `$HOME/.config/infrapulse/`.

### `servers.yaml`

This file contains the list of servers and services to monitor.

**Example:**
```yaml
servers:
  - name: "Web Server"
    host: "example.com"
    ports:
      - 80
      - 443
  - name: "Database Server"
    host: "db.example.com"
    ports:
      - 5432
```

### `config.yaml`

This file contains your SMTP server details for email alerts. You will need to create this file yourself.

**Example for Gmail:**

For Gmail, you will need to generate an **App Password**. See the Google Help documentation for instructions on how to do this.

```yaml
smtp:
  host: "smtp.gmail.com"
  port: 587
  username: "your_gmail_address@gmail.com"
  password: "the_16_character_app_password"
alert_recipient: "your_email@example.com, another_email@example.com"

Note: The `alert_recipient` field now supports multiple email addresses separated by commas. For example: `"admin@example.com, ops@example.com"`.
```

### Handling Sensitive Information with .env Files

For better security, especially for sensitive data like SMTP passwords, it's recommended to use environment variables and a `.env` file. You can then parse these values into your `config.yaml` or `servers.yaml` using a simple shell script or a tool like `envsubst`.

**Example using a shell script to populate `config.yaml`:**

1.  **Create a `.env` file:**
    ```
    SMTP_HOST="smtp.gmail.com"
    SMTP_PORT=587
    SMTP_USERNAME="your_gmail_address@gmail.com"
    SMTP_PASSWORD="the_16_character_app_password"
    ALERT_RECIPIENT="your_email@example.com, another_email@example.com"
    ```

2.  **Create a template `config.yaml.template`:**
    ```yaml
    smtp:
      host: "${SMTP_HOST}"
      port: ${SMTP_PORT}
      username: "${SMTP_USERNAME}"
      password: "${SMTP_PASSWORD}"
    alert_recipient: "${ALERT_RECIPIENT}"
    ```

3.  **Use a script to generate `config.yaml` from `.env`:**
    ```bash
    #!/bin/bash
    set -a # Automatically export all variables
    source .env
    set +a

    envsubst < config.yaml.template > $HOME/.config/infrapulse/config.yaml
    ```
    (You might need to install `gettext` for `envsubst`: `sudo apt-get install gettext`)

This approach keeps your sensitive credentials out of version control and makes your configuration more flexible.


## Building from Source

If you want to build the binary manually, you can use the following command:

```sh
go build -o infrapulse main.go
```

## Development

The daemonization process is implemented by re-executing the `infrapulse` binary with an internal `-internal-daemon` flag. This is handled automatically when you use the `-d` flag.

## Contributing

Contributions are welcome! Please see the [CONTRIBUTING.md](CONTRIBUTING.md) file for details.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
