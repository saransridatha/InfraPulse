package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"os"
	"os/signal"
	"path/filepath"
	
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/prometheus-community/pro-bing"
	"gopkg.in/yaml.v3"
)

// --- Structs for Configuration ---

type Server struct {
	Name  string   `yaml:"name"`
	Host  string   `yaml:"host"`
	Ports []int    `yaml:"ports"`
}

type SMTPConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type Config struct {
	Servers        []Server   `yaml:"servers"`
	SMTP           SMTPConfig `yaml:"smtp"`
	AlertRecipient string     `yaml:"alert_recipient"`
	CheckInterval  string     `yaml:"check_interval"`
}

// --- Structs for Service and Status ---

type Service struct {
	Name string
	Host string
	Port int // 0 for ping
}

type CheckResult struct {
	Service Service
	Status  string // "UP" or "DOWN"
	Error   error
}

// --- Main Application Logic ---

func main() {
	// --- Command-Line Flags ---
defaultServerFile := ""
home, err := os.UserHomeDir()
	if err == nil {
		defaultServerFile = filepath.Join(home, ".config", "infrapulse", "servers.yaml")
	}

	serverFile := flag.String("config", defaultServerFile, "Path to the servers.yaml configuration file.")
daemon := flag.Bool("d", false, "Run in monitoring loop mode. Use 'nohup' or a service manager to run in background.")
	
	interval := flag.String("i", "", "Check interval in monitoring loop mode (e.g., '60s', '5m'). Overrides config file.")
	flag.Parse()

	// --- Load Configuration ---
	if *serverFile == "" {
		slog.Error("Could not find default config path. Please use the -config flag.")
		os.Exit(1)
	}

	configFile := filepath.Join(filepath.Dir(*serverFile), "config.yaml")
	cfg, err := loadConfig(*serverFile, configFile)
	if err != nil {
		slog.Error("Error loading configuration", "error", err)
		os.Exit(1)
	}

	// --- Create Services ---
	services := createServices(cfg.Servers)

	// --- Monitoring Loop Mode ---
	if *daemon {
		runMonitoringLoop(cfg, services, *interval)
		return
	}

	// --- One-Time Run ---
	runOnce(cfg, services)
}

func runMonitoringLoop(cfg *Config, services []Service, intervalFlag string) {
	// --- Signal Handling ---
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// --- State Management ---
	statusMap := make(map[string]string)

	// --- Interval ---
	checkInterval := cfg.CheckInterval
	if intervalFlag != "" {
		checkInterval = intervalFlag
	}
	if checkInterval == "" {
		checkInterval = "60s" // Default to 60 seconds if not specified
	}
	duration, err := time.ParseDuration(checkInterval)
	if err != nil {
		slog.Error("Invalid check interval", "error", err)
		os.Exit(1)
	}

	color.Cyan("InfraPulse: Starting monitoring loop...")
	color.Cyan("Check interval: %s", duration)

	// --- Main Loop ---
	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			var wg sync.WaitGroup
			results := make(chan CheckResult)

			for _, service := range services {
				wg.Add(1)
				go checkService(service, &wg, results)
			}

			go func() {
				wg.Wait()
				close(results)
			}()

			var alerts []string
			for result := range results {
				printResult(result)
				serviceID := fmt.Sprintf("%s:%d", result.Service.Host, result.Service.Port)
				previousStatus := statusMap[serviceID]
				if result.Status == "DOWN" && previousStatus != "DOWN" {
					alerts = append(alerts, formatAlert(result))
				}
				statusMap[serviceID] = result.Status
			}

			if len(alerts) > 0 {
				if cfg.SMTP.Host != "" {
					color.Yellow("Sending failure alerts via email...")
					sendAlertEmail(cfg, alerts)
				} else {
					color.Yellow("SMTP configuration not found, skipping email alerts.")
				}
			}
		case <-sigChan:
			color.Cyan("\nShutting down monitoring loop...")
			return
		}
	}
}


func createServices(servers []Server) []Service {
	var services []Service
	for _, server := range servers {
		if len(server.Ports) == 0 {
			services = append(services, Service{Name: server.Name, Host: server.Host, Port: 0})
		} else {
			for _, port := range server.Ports {
				services = append(services, Service{Name: server.Name, Host: server.Host, Port: port})
			}
		}
	}
	return services
}

func runOnce(cfg *Config, services []Service) {
	var wg sync.WaitGroup
	results := make(chan CheckResult)

	color.Cyan("InfraPulse: Starting health checks...")

	for _, service := range services {
		wg.Add(1)
		go checkService(service, &wg, results)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var alerts []string
	for result := range results {
		printResult(result)
		if result.Status == "DOWN" {
			alerts = append(alerts, formatAlert(result))
		}
	}

	if len(alerts) > 0 {
		if cfg.SMTP.Host != "" {
			color.Yellow("Sending failure alerts via email...")
			sendAlertEmail(cfg, alerts)
		} else {
			color.Yellow("SMTP configuration not found, skipping email alerts.")
		}
	}

	color.Cyan("All checks complete.")
}



func checkService(service Service, wg *sync.WaitGroup, results chan<- CheckResult) {
	defer wg.Done()

	if service.Port == 0 { // Ping
		pinger, err := probing.NewPinger(service.Host)
		if err != nil {
			results <- CheckResult{Service: service, Status: "DOWN", Error: err}
			return
		}
		pinger.Count = 3
		pinger.Timeout = 2 * time.Second
		err = pinger.Run()
		if err != nil || pinger.Statistics().PacketsRecv == 0 {
			results <- CheckResult{Service: service, Status: "DOWN", Error: err}
		} else {
			results <- CheckResult{Service: service, Status: "UP"}
		}
	} else { // TCP Port Check
		address := fmt.Sprintf("%s:%d", service.Host, service.Port)
		conn, err := net.DialTimeout("tcp", address, 2*time.Second)
		if err != nil {
			results <- CheckResult{Service: service, Status: "DOWN", Error: err}
		} else {
			conn.Close()
			results <- CheckResult{Service: service, Status: "UP"}
		}
	}
}

func printResult(result CheckResult) {
	if result.Service.Port == 0 { // Ping
		if result.Status == "UP" {
			color.Green("  [UP] %s (%s): Host is up", result.Service.Name, result.Service.Host)
		} else {
			color.Red("  [DOWN] %s (%s): Host is down", result.Service.Name, result.Service.Host)
		}
	} else { // Port
		if result.Status == "UP" {
			color.Green("    - Port %d: [UP]", result.Service.Port)
		} else {
			color.Red("    - Port %d: [DOWN]", result.Service.Port)
		}
	}
}

func formatAlert(result CheckResult) string {
	timestamp := time.Now().Format(time.RFC1123)
	var errorMsg string
	if result.Error != nil {
		errorMsg = result.Error.Error()
	} else {
		errorMsg = "No specific error message."
	}

	if result.Service.Port == 0 {
		return fmt.Sprintf("Host Down Alert\n\nHost: %s (%s)\nTime: %s\nDetails: Ping failed.\nError: %s\n", result.Service.Name, result.Service.Host, timestamp, errorMsg)
	}
	return fmt.Sprintf("Service Down Alert\n\nService: %s\nHost: %s\nPort: %d\nTime: %s\nError: %s\n", result.Service.Name, result.Service.Host, result.Service.Port, timestamp, errorMsg)
}

// loadConfig reads and merges server and SMTP configurations.
func loadConfig(serverFile, configFile string) (*Config, error) {
	// Load server list
	serverData, err := os.ReadFile(serverFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", serverFile, err)
	}
	var serverConfig struct {
		Servers       []Server `yaml:"servers"`
		CheckInterval string   `yaml:"check_interval"`
	}
	if err := yaml.Unmarshal(serverData, &serverConfig); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", serverFile, err)
	}

	// Load private config (SMTP, etc.)
	configData, err := os.ReadFile(configFile)
	if err != nil {
		// If the config file is not found, we just return the server config
		// and assume no email alerts are needed.
		if os.IsNotExist(err) {
			return &Config{
				Servers:       serverConfig.Servers,
				CheckInterval: serverConfig.CheckInterval,
			}, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", configFile, err)
	}
	var privateConfig struct {
		SMTP           SMTPConfig `yaml:"smtp"`
		AlertRecipient string     `yaml:"alert_recipient"`
	}
	if err := yaml.Unmarshal(configData, &privateConfig); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", configFile, err)
	}

	// Combine into a single config struct
	fullConfig := &Config{
		Servers:        serverConfig.Servers,
		SMTP:           privateConfig.SMTP,
		AlertRecipient: privateConfig.AlertRecipient,
		CheckInterval:  serverConfig.CheckInterval,
	}

	return fullConfig, nil
}

// sendAlertEmail sends a consolidated email with all failure alerts.
func sendAlertEmail(cfg *Config, alerts []string) {
	if cfg.AlertRecipient == "" {
		slog.Warn("Email alert failed: AlertRecipient is not set in config.yaml")
		return
	}

	from := cfg.SMTP.Username
	password := cfg.SMTP.Password
	to := strings.Split(cfg.AlertRecipient, ",")
	for i, email := range to {
		to[i] = strings.TrimSpace(email)
	}
	smtpHost := cfg.SMTP.Host
	smtpPort := cfg.SMTP.Port

	subject := "Subject: InfraPulse Alert: Service Degradation Detected\n"
	body := "One or more services are down:\n\n"
	body += strings.Join(alerts, "\n---------------------------------\n\n")
	
	message := []byte(subject + body)


	auth := smtp.PlainAuth("", from, password, smtpHost)
	addr := fmt.Sprintf("%s:%d", smtpHost, smtpPort)

	err := smtp.SendMail(addr, auth, from, to, message)
	if err != nil {
		slog.Error("Email alert failed to send", "error", err)
		return
	}

	slog.Info("Email alert sent successfully.")
}