package main

import (
	"flag"
	"fmt"
	ps "github.com/mitchellh/go-ps"
	"github.com/patcharp/processtracker/notify"
	"gopkg.in/yaml.v3"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type Process struct {
	PID            int
	Name           string
	Polling        bool
	Stage          int
	LastNotifiedAt *time.Time
}

func (p *Process) Running() bool {
	return p.PID != -1
}

func (p *Process) Start(pid int) {
	p.PID = pid
}

func (p *Process) Stop() {
	p.PID = -1
}

type Config struct {
	Process ProcessCfg `yaml:"process"`
	Alert   *Alert     `yaml:"alert"`
}

type ProcessCfg struct {
	Name     string `yaml:"name"`
	Interval string `yaml:"interval"`
}

type Alert struct {
	Line    *LineAlert    `yaml:"line"`
	Discord *DiscordAlert `yaml:"discord"`
}

type LineAlert struct {
	To    string `yaml:"to"`
	Token string `yaml:"token"`
}

type DiscordAlert struct {
	WhUrl string `yaml:"webhook"`
}

const (
	ProcessStageUnknown   = -1
	ProcessStageNormal    = 0
	ProcessStageStop      = 1
	ProcessStagePIDChange = 2
)

var (
	DefaultInterval = time.Second * 60 // interval 1 min
	defaultConfig   = `process:
  name: <process name>
  interval: <duration eg. 10s>
alert:
  line:
    to: <User or Group ID>
    token: <Linebot Token>
  discord:
    webhook: <Discord Webhook>
`
)

func main() {
	helpCmd := flag.Bool("help", false, "Show help.")
	listCmd := flag.Bool("list", false, "List running process list.")
	allCmd := flag.Bool("all", false, "Set to list all process.")
	findCmd := flag.String("find", "", "Find process by name.")
	cfgFileCmd := flag.String("config-file", "config.yml", "Define configuration file.")
	genCfgCmd := flag.Bool("gen-config", false, "Generate example configuration file.")
	flag.Parse()
	// Help
	if *helpCmd {
		helpCmdHandler()
		os.Exit(0)
	}
	// List process
	if *listCmd {
		listPSCmdHandler("", *allCmd)
		os.Exit(0)
	}
	// Find specific process
	if *findCmd != "" {
		listPSCmdHandler(*findCmd, *allCmd)
		os.Exit(0)
	}
	// Gen config file
	if *genCfgCmd {
		genConfigFile(*cfgFileCmd)
		os.Exit(0)
	}
	// Default start command
	startCmdHandler(loadConfig(*cfgFileCmd))
}

func loadConfig(file string) *Config {
	f, err := os.ReadFile(file)
	if err != nil {
		fmt.Println("[ERR] Read config file error -:", err)
		os.Exit(1)
	}
	var cfg Config
	if err := yaml.Unmarshal(f, &cfg); err != nil {
		fmt.Println("[ERR] Config file error -:", err)
		os.Exit(2)
	}
	return &cfg
}

func sendNotify(msg string, lvl int, p *Process, cfg *Alert) {
	if cfg.Line != nil {
		go notify.SendLineNotify(msg, lvl, cfg.Line.Token, cfg.Line.To)
	}
	if cfg.Discord != nil {
		go notify.SendDiscordNotify(msg, lvl, p.PID, p.Name, cfg.Discord.WhUrl)
	}
}

func helpCmdHandler() {
	var desc []string
	desc = append(desc, "----------------------------------------")
	desc = append(desc, fmt.Sprintf("proctracker - Process tracker"))
	desc = append(desc, "----------------------------------------")
	desc = append(desc, "")
	desc = append(desc, fmt.Sprintf("The following options are available:"))
	desc = append(desc, fmt.Sprintf("  %-30s%s", "--help", "Display help instruction."))
	desc = append(desc, fmt.Sprintf("  %-30s%s", "--list", "List available process (only main process)."))
	desc = append(desc, fmt.Sprintf("  %-30s%s", "--find=<process name>", "Find specific process."))
	desc = append(desc, fmt.Sprintf("  %-30s%s", "--all", "Use with --list and --find to show all process found."))
	desc = append(desc, fmt.Sprintf("  %-30s%s", "--config-file=<file>", "Set specific configure file located. (default: config.yml)"))
	desc = append(desc, fmt.Sprintf("  %-30s%s", "--gen-config", "Generate example config file."))
	desc = append(desc, "")
	desc = append(desc, "Example:")
	desc = append(desc, "  # Start process tracker service")
	desc = append(desc, "    $ ./proctracker")
	desc = append(desc, "")
	desc = append(desc, "  # Finding process name nginx")
	desc = append(desc, "    $ ./proctracker --find=nginx")
	desc = append(desc, "")
	desc = append(desc, "  # List all available process")
	desc = append(desc, "    $ ./proctracker --list --all")
	desc = append(desc, "")
	desc = append(desc, "  # Generate starter configuration with specific file")
	desc = append(desc, "    $ ./proctracker --gen-config --config-file=custom.yml")
	desc = append(desc, "")
	fmt.Println(strings.Join(desc, "\n"))
}

func listPSCmdHandler(name string, all bool) {
	processList, err := ps.Processes()
	if err != nil {
		fmt.Println("[ERR] Gather process list error -:", err)
		os.Exit(3)
	}
	fmt.Println("[+] Total process count", len(processList))
	fmt.Println("----------------------------------------")
	fmt.Printf("%8s%8s%s%s\n", "PID", "PPID", "    ", "Process name")
	fmt.Println("----------------------------------------")
	foundPS := 0
	for _, v := range processList {
		if v.PPid() != 1 && !all {
			continue
		}
		if name == "" || (name != "" && strings.Contains(strings.ToLower(v.Executable()), strings.ToLower(name))) {
			fmt.Printf("%8d%8d%s%s\n", v.Pid(), v.PPid(), "    ", v.Executable())
			foundPS += 1
		}
	}
	if foundPS == 0 {
		fmt.Println("[x] No process found.")
	}
	fmt.Println("----------------------------------------")
	fmt.Println("[*] PID = Process ID, PPID = Parent PID")
}

func genConfigFile(file string) {
	fmt.Println("[*] Generating starter configuration.")
	if err := os.WriteFile(file, []byte(defaultConfig), 0644); err != nil {
		fmt.Println("[ERR] Write starter config file error -:", err)
		return
	}
	fmt.Println("[+] Wrote config file success at", file)
}

func startCmdHandler(cfg *Config) {
	interval, err := time.ParseDuration(cfg.Process.Interval)
	if err != nil {
		fmt.Println("[ERR] Invalid interval duration -:", err)
		interval = DefaultInterval
		cfg.Process.Interval = interval.String()
	}
	if interval < time.Second*10 || interval > time.Hour {
		fmt.Println("[ERR] Interval was not in range 10s to 1hr")
		os.Exit(4)
	}
	// Trim space
	cfg.Process.Name = strings.TrimSpace(cfg.Process.Name)
	if cfg.Process.Name == "" {
		fmt.Println("[ERR] Process name should not empty")
		os.Exit(5)
	}
	process := Process{
		PID:            -1,
		Name:           cfg.Process.Name,
		Polling:        false,
		Stage:          ProcessStageUnknown,
		LastNotifiedAt: nil,
	}
	fmt.Printf("[+] Start tracking process: `%s`, interval every: `%s`\n", cfg.Process.Name, cfg.Process.Interval)
	PollingJob(cfg, &process)
	ticker := time.NewTicker(interval)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	for {
		select {
		case <-sigs:
			fmt.Println("[+] Process tracker was stopped.")
			return
		case <-ticker.C:
			PollingJob(cfg, &process)
		}
	}
}

func PollingJob(cfg *Config, process *Process) {
	if process.Polling {
		fmt.Println("[*] Previous job in progress.")
		go sendNotify(fmt.Sprintf("[x] Previuos tracker job in progress."), notify.AlertSeverityInfo, process, cfg.Alert)
		return
	}
	process.Polling = true
	defer func() {
		process.Polling = false
	}()
	psList, err := ps.Processes()
	if err != nil {
		sendNotify(fmt.Sprintf("[x] Cannot gather system process and service was terminated."), notify.AlertSeverityCritical, process, cfg.Alert)
		fmt.Println("[x] Gather system process error -:", err)
		return
	}
	// check ps
	count := 0
	for _, p := range psList {
		// Match process name
		if strings.ToLower(process.Name) == strings.ToLower(p.Executable()) {
			// found process
			if !process.Running() {
				// First found process was running
				process.Start(p.Pid())
				process.Stage = ProcessStageNormal
				fmt.Printf("[+] Process `%s` was running at PID: `%d`\n", process.Name, process.PID)
				if process.LastNotifiedAt != nil {
					// Process become ready after stop
					go sendNotify(fmt.Sprintf("[+] Process `%s` was back to normal at PID: `%d`\n", process.Name, process.PID), notify.AlertSeverityInfo, process, cfg.Alert)
					// reset notify
					process.LastNotifiedAt = nil
					fmt.Println("[*] Send notify at", time.Now().Format(time.DateTime))
				}
			} else if process.PID != p.Pid() {
				// PID Changed
				process.Start(p.Pid())
				process.Stage = ProcessStagePIDChange
				fmt.Printf("[+] Process `%s` PID was changed to `%d`\n", process.Name, p.Pid())
				now := time.Now()
				process.LastNotifiedAt = &now
				go sendNotify(fmt.Sprintf("[+] Process `%s` PID was changed to `%d`\n", process.Name, p.Pid()), notify.AlertSeverityWarn, process, cfg.Alert)
				fmt.Println("[*] Send notify at", now.Format(time.DateTime))
			}
			// stop loop if found
			break
		}
		count += 1
	}

	// Process not found
	if count >= len(psList) && process.Stage != ProcessStageStop {
		fmt.Printf("[x] Process `%s` was stopped.\n", process.Name)
		process.Stop()
		process.Stage = ProcessStageStop
		now := time.Now()
		process.LastNotifiedAt = &now
		go sendNotify(fmt.Sprintf("[x] Process `%s` was stopped.", process.Name), notify.AlertSeverityCritical, process, cfg.Alert)
		fmt.Println("[*] Send notify at", now.Format(time.DateTime))
	}
}
