package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"text/template"
)

// ── Public entry points ──────────────────────────────────────────────────────

func installService(port int, configPath string) {
	sc, err := buildServiceConfig(port, configPath)
	if err != nil {
		die("cannot prepare service config: %v", err)
	}

	for _, dir := range []string{filepath.Dir(sc.BinPath), filepath.Dir(sc.ConfigPath), filepath.Dir(sc.LogPath)} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			die("cannot create directory %s: %v", dir, err)
		}
	}

	if err := copyExe(sc.BinPath); err != nil {
		die("cannot install binary: %v", err)
	}

	switch runtime.GOOS {
	case "darwin":
		installMacOS(sc)
	case "linux":
		installLinux(sc)
	case "windows":
		installWindows(sc)
	default:
		die("unsupported OS: %s", runtime.GOOS)
	}
}

func uninstallService() {
	switch runtime.GOOS {
	case "darwin":
		uninstallMacOS()
	case "linux":
		uninstallLinux()
	case "windows":
		uninstallWindows()
	default:
		die("unsupported OS: %s", runtime.GOOS)
	}
}

// ── Config builder ───────────────────────────────────────────────────────────

type svcConfig struct {
	BinPath    string
	ConfigPath string
	LogPath    string
	Port       int
	PortStr    string // for templates that need a string
}

func buildServiceConfig(port int, configPath string) (svcConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return svcConfig{}, err
	}

	var binPath, cfgDir, logDir string

	switch runtime.GOOS {
	case "darwin":
		binPath = filepath.Join(home, ".local", "bin", "local-start-page")
		cfgDir = filepath.Join(home, "Library", "Application Support", "local-start-page")
		logDir = filepath.Join(home, "Library", "Logs", "local-start-page")
	case "windows":
		appData := os.Getenv("LOCALAPPDATA")
		binPath = filepath.Join(appData, "local-start-page", "local-start-page.exe")
		cfgDir = filepath.Join(os.Getenv("APPDATA"), "local-start-page")
		logDir = cfgDir
	default: // linux
		binPath = filepath.Join(home, ".local", "bin", "local-start-page")
		cfgDir = filepath.Join(home, ".config", "local-start-page")
		logDir = cfgDir
	}

	// If caller passed a non-default config path, use it; otherwise use install dir.
	if configPath == "./config.toml" {
		configPath = filepath.Join(cfgDir, "config.toml")
	}

	return svcConfig{
		BinPath:    binPath,
		ConfigPath: configPath,
		LogPath:    filepath.Join(logDir, "server.log"),
		Port:       port,
		PortStr:    strconv.Itoa(port),
	}, nil
}

// ── Binary copy ──────────────────────────────────────────────────────────────

func copyExe(dst string) error {
	src, err := os.Executable()
	if err != nil {
		return err
	}
	// Resolve symlinks so we copy the real binary.
	src, err = filepath.EvalSymlinks(src)
	if err != nil {
		return err
	}
	if src == dst {
		return nil // already installed in place
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// ── macOS — launchd LaunchAgent ──────────────────────────────────────────────

const plistLabel = "com.local-start-page"

const plistTmpl = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>com.local-start-page</string>
  <key>ProgramArguments</key>
  <array>
    <string>{{.BinPath}}</string>
    <string>--config</string>
    <string>{{.ConfigPath}}</string>
    <string>--port</string>
    <string>{{.PortStr}}</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>StandardOutPath</key>
  <string>{{.LogPath}}</string>
  <key>StandardErrorPath</key>
  <string>{{.LogPath}}</string>
</dict>
</plist>
`

func plistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", plistLabel+".plist")
}

func installMacOS(sc svcConfig) {
	p := plistPath()
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		die("cannot create LaunchAgents dir: %v", err)
	}
	if err := writeTemplate(p, plistTmpl, sc); err != nil {
		die("cannot write plist: %v", err)
	}
	// Unload first in case it was already loaded.
	exec.Command("launchctl", "unload", "-w", p).Run() //nolint:errcheck
	if out, err := exec.Command("launchctl", "load", "-w", p).CombinedOutput(); err != nil {
		die("launchctl load failed: %v\n%s", err, out)
	}
	printInstallSummary("launchd LaunchAgent", p, sc)
	fmt.Println("  Autostart : on login")
	fmt.Printf("  Logs      : %s\n", sc.LogPath)
	fmt.Println("  Uninstall : local-start-page --uninstall")
}

func uninstallMacOS() {
	p := plistPath()
	exec.Command("launchctl", "unload", "-w", p).Run() //nolint:errcheck
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		fmt.Printf("warning: could not remove plist: %v\n", err)
	}
	fmt.Println("Service removed (launchd).")
}

// ── Linux — systemd user service ─────────────────────────────────────────────

const unitName = "local-start-page.service"

const systemdTmpl = `[Unit]
Description=Local Start Page
After=network.target

[Service]
ExecStart={{.BinPath}} --config {{.ConfigPath}} --port {{.PortStr}}
Restart=on-failure
RestartSec=5
StandardOutput=append:{{.LogPath}}
StandardError=append:{{.LogPath}}

[Install]
WantedBy=default.target
`

func systemdUnitPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "systemd", "user", unitName)
}

func installLinux(sc svcConfig) {
	p := systemdUnitPath()
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		die("cannot create systemd user dir: %v", err)
	}
	if err := writeTemplate(p, systemdTmpl, sc); err != nil {
		die("cannot write unit file: %v", err)
	}
	runCmd("systemctl", "--user", "daemon-reload")
	runCmd("systemctl", "--user", "enable", "--now", unitName)
	printInstallSummary("systemd user service", p, sc)
	fmt.Println("  Autostart : on login (loginctl enable-linger for headless)")
	fmt.Printf("  Logs      : journalctl --user -u %s -f\n", unitName)
	fmt.Println("  Uninstall : local-start-page --uninstall")
}

func uninstallLinux() {
	exec.Command("systemctl", "--user", "disable", "--now", unitName).Run() //nolint:errcheck
	if err := os.Remove(systemdUnitPath()); err != nil && !os.IsNotExist(err) {
		fmt.Printf("warning: could not remove unit file: %v\n", err)
	}
	exec.Command("systemctl", "--user", "daemon-reload").Run() //nolint:errcheck
	fmt.Println("Service removed (systemd user).")
}

// ── Windows — Task Scheduler ─────────────────────────────────────────────────

const taskName = "local-start-page"

func installWindows(sc svcConfig) {
	// Quote paths that may contain spaces.
	tr := fmt.Sprintf(`"%s" --config "%s" --port %s`, sc.BinPath, sc.ConfigPath, sc.PortStr)
	out, err := exec.Command(
		"schtasks", "/create",
		"/tn", taskName,
		"/tr", tr,
		"/sc", "ONLOGON",
		"/rl", "HIGHEST",
		"/f",
	).CombinedOutput()
	if err != nil {
		die("schtasks /create failed: %v\n%s", err, out)
	}
	// Start immediately without waiting for next login.
	exec.Command("schtasks", "/run", "/tn", taskName).Run() //nolint:errcheck
	printInstallSummary("Task Scheduler", taskName, sc)
	fmt.Println("  Autostart : on login")
	fmt.Printf("  Logs      : %s\n", sc.LogPath)
	fmt.Println("  Uninstall : local-start-page.exe --uninstall")
}

func uninstallWindows() {
	exec.Command("schtasks", "/end", "/tn", taskName).Run()             //nolint:errcheck
	exec.Command("schtasks", "/delete", "/tn", taskName, "/f").Run()   //nolint:errcheck
	fmt.Println("Service removed (Task Scheduler).")
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func writeTemplate(path, tmpl string, data any) error {
	t, err := template.New("").Parse(tmpl)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0644)
}

func runCmd(name string, args ...string) {
	if out, err := exec.Command(name, args...).CombinedOutput(); err != nil {
		fmt.Printf("warning: %s %v: %v\n%s\n", name, args, err, out)
	}
}

func printInstallSummary(method, serviceFile string, sc svcConfig) {
	fmt.Printf("Installed via %s\n", method)
	fmt.Printf("  Binary    : %s\n", sc.BinPath)
	fmt.Printf("  Config    : %s\n", sc.ConfigPath)
	fmt.Printf("  Service   : %s\n", serviceFile)
	fmt.Printf("  URL       : http://localhost:%d\n", sc.Port)
}

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
