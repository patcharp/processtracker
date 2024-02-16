package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	MaxDiscordEmbed = 10
	ColorRed        = 0x992D22
	ColorOrange     = 0xF0B816
	ColorGreen      = 0x2ECC71
	ColorGrey       = 0x95A5A6
	ColorBlue       = 0x58b9ff
)

type alertManAlert struct {
	Annotations struct {
		Description string `json:"description"`
		Summary     string `json:"summary"`
	} `json:"annotations"`
	EndsAt       string            `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	Labels       map[string]string `json:"labels"`
	StartsAt     string            `json:"startsAt"`
	Status       string            `json:"status"`
}

type alertManOut struct {
	Alerts            []alertManAlert `json:"alerts"`
	CommonAnnotations struct {
		Summary string `json:"summary"`
	} `json:"commonAnnotations"`
	CommonLabels struct {
		Alertname string `json:"alertname"`
	} `json:"commonLabels"`
	ExternalURL string `json:"externalURL"`
	GroupKey    string `json:"groupKey"`
	GroupLabels struct {
		Alertname string `json:"alertname"`
	} `json:"groupLabels"`
	Receiver string `json:"receiver"`
	Status   string `json:"status"`
	Version  string `json:"version"`
}

type discordOut struct {
	Content string         `json:"content"`
	Embeds  []discordEmbed `json:"embeds"`
}

type discordEmbed struct {
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Color       int                 `json:"color"`
	Fields      []discordEmbedField `json:"fields"`
}

type discordEmbedField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func SendDiscordNotify(msg string, lvl int, pId int, pName string, whUrl string) {
	re := regexp.MustCompile(`https://discord(?:app)?.com/api/webhooks/[0-9]{18,19}/[a-zA-Z0-9_-]+`)
	if !re.MatchString(whUrl) {
		// invalid discord webhook
		fmt.Println("[x] Invalid discord webhook url.")
		return
	}

	hostname, _ := os.Hostname()
	severity := "normal"
	color := ColorGrey
	status := "changed"
	switch lvl {
	case AlertSeverityInfo:
		color = ColorGreen
		break
	case AlertSeverityWarn:
		color = ColorOrange
		severity = "warning"
		break
	case AlertSeverityError, AlertSeverityCritical:
		color = ColorRed
		severity = "critical"
		break
	}
	if pId == -1 {
		status = "stopped"
	} else if color == ColorGreen {
		status = "running"
	}

	title := fmt.Sprintf("[%s] %s - %s process was %s", strings.ToUpper(severity), hostname, pName, status)
	var labels []string
	var desc []string
	labels = append(labels, fmt.Sprintf(": - **_%s:_** %s", "Hostname", hostname))
	labels = append(labels, fmt.Sprintf(": - **_%s:_** %s", "Process", pName))
	if pId != -1 {
		labels = append(labels, fmt.Sprintf(": - **_%s:_** %d", "PID", pId))
	}
	now := time.Now()
	desc = append(desc, fmt.Sprintf("**⏰ Event Time:** %s", now.Format(time.DateTime)))
	desc = append(desc, fmt.Sprintf("**🏷️ Alert labels:**\n%s", strings.Join(labels, "\n")))
	desc = append(desc, "------")
	desc = append(desc, fmt.Sprintf("**📖 Description:**\n%s", msg))
	DO := discordOut{
		Content: fmt.Sprintf("=== Alert: %s - %s ===", "Process tracker", pName),
		Embeds: []discordEmbed{
			{
				Title:       title,
				Color:       color,
				Description: strings.Join(desc, "\n"),
				Fields:      []discordEmbedField{},
			},
		},
	}
	fireDiscordMessageOut(&DO, whUrl)
}

func fireDiscordMessageOut(msg *discordOut, whUrl string) {
	DOD, _ := json.Marshal(msg)
	r, err := http.Post(whUrl, "application/json", bytes.NewReader(DOD))
	if err != nil {
		fmt.Println("[x] Send discord error -:", err)
		return
	}
	if r.StatusCode >= http.StatusBadRequest {
		b, _ := ioutil.ReadAll(r.Body)
		fmt.Println("[x] Discord server return error -:", r.StatusCode, string(b))
	}
}
