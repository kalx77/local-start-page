package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/kalx77/local-start-page/config"
	"github.com/kalx77/local-start-page/server"
)

//go:embed web
var webFS embed.FS

const pidFile = "/tmp/local-start-page.pid"
const logFile = "/tmp/local-start-page.log"

func main() {
	configPath := flag.String("config", "./config.toml", "path to config file")
	port := flag.Int("port", 0, "port to listen on (overrides config, default 1221)")
	daemon := flag.Bool("daemon", false, "run server in background")
	stop := flag.Bool("stop", false, "stop background server")
	status := flag.Bool("status", false, "show background server status")
	install := flag.Bool("install", false, "install as system service (launchd/systemd/Task Scheduler)")
	uninstall := flag.Bool("uninstall", false, "remove system service")
	flag.Parse()

	if *stop {
		stopDaemon()
		return
	}

	if *status {
		showStatus()
		return
	}

	if *uninstall {
		uninstallService()
		return
	}

	if err := config.EnsureExists(*configPath); err != nil {
		log.Fatalf("failed to create default config: %v", err)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Port priority: --port flag > config file > default 1221
	effectivePort := cfg.Port
	if effectivePort == 0 {
		effectivePort = 1221
	}
	if *port != 0 {
		effectivePort = *port
		cfg.Port = effectivePort
		if saveErr := config.Save(*configPath, cfg); saveErr != nil {
			log.Printf("warning: could not save port to config: %v", saveErr)
		}
	}

	if *install {
		installService(effectivePort, *configPath)
		return
	}

	if *daemon {
		startDaemon(*configPath, effectivePort)
		return
	}

	addr := fmt.Sprintf(":%d", effectivePort)
	log.Printf("starting local-start-page on http://localhost%s", addr)
	if err := server.Start(addr, *configPath, webFS); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func startDaemon(configPath string, port int) {
	exe, err := os.Executable()
	if err != nil {
		log.Fatalf("cannot find executable: %v", err)
	}

	// Build args without --daemon
	args := []string{"--config", configPath, "--port", strconv.Itoa(port)}
	for _, a := range os.Args[1:] {
		if a == "--daemon" || a == "-daemon" {
			continue
		}
		// Skip already-included flags
		if strings.HasPrefix(a, "--config") || strings.HasPrefix(a, "--port") ||
			strings.HasPrefix(a, "-config") || strings.HasPrefix(a, "-port") {
			continue
		}
		args = append(args, a)
	}

	lf, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("cannot open log file: %v", err)
	}

	cmd := exec.Command(exe, args...)
	cmd.Stdout = lf
	cmd.Stderr = lf
	cmd.Stdin = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		log.Fatalf("failed to start daemon: %v", err)
	}

	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0644); err != nil {
		log.Printf("warning: could not write pid file: %v", err)
	}

	fmt.Printf("local-start-page started (PID %d)\n", cmd.Process.Pid)
	fmt.Printf("  URL:  http://localhost:%d\n", port)
	fmt.Printf("  Log:  %s\n", logFile)
	fmt.Printf("  Stop: %s --stop\n", exe)
}

func stopDaemon() {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		fmt.Println("no running daemon found (pid file missing)")
		return
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		fmt.Printf("invalid pid file: %v\n", err)
		return
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		fmt.Printf("process not found: %v\n", err)
		return
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		fmt.Printf("failed to stop daemon (PID %d): %v\n", pid, err)
		return
	}
	os.Remove(pidFile)
	fmt.Printf("stopped local-start-page (PID %d)\n", pid)
}

func showStatus() {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		fmt.Println("status: not running")
		return
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		fmt.Println("status: not running (bad pid file)")
		return
	}
	proc, err := os.FindProcess(pid)
	if err != nil || proc.Signal(syscall.Signal(0)) != nil {
		fmt.Printf("status: not running (stale PID %d)\n", pid)
		os.Remove(pidFile)
		return
	}
	fmt.Printf("status: running (PID %d)\n", pid)
	fmt.Printf("  log: %s\n", logFile)
}
