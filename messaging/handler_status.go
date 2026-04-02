package messaging

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"time"
)

func (h *Handler) buildStatus() string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var b strings.Builder

	// System info
	b.WriteString(fmt.Sprintf("**ringclaw %s** (%s/%s)\n", h.version, runtime.GOOS, runtime.GOARCH))
	b.WriteString(fmt.Sprintf("* uptime: %s\n", formatDuration(time.Since(h.startTime))))
	b.WriteString(fmt.Sprintf("* go: %s\n", runtime.Version()))
	b.WriteString("\n")

	// Default agent
	b.WriteString("**Default Agent**\n")
	if h.defaultName == "" {
		b.WriteString("* name: none (echo mode)\n")
	} else if ag, ok := h.agents[h.defaultName]; !ok {
		b.WriteString(fmt.Sprintf("* name: %s (not started)\n", h.defaultName))
	} else {
		info := ag.Info()
		b.WriteString(fmt.Sprintf("* name: %s\n", h.defaultName))
		b.WriteString(fmt.Sprintf("* type: %s\n", info.Type))
		if info.Model != "" {
			b.WriteString(fmt.Sprintf("* model: %s\n", info.Model))
		}
		if info.PID > 0 {
			b.WriteString(fmt.Sprintf("* pid: %d\n", info.PID))
		}
	}

	// Active sessions
	activeSessions := 0
	for range h.agents {
		activeSessions++
	}
	b.WriteString(fmt.Sprintf("* active sessions: %d\n", activeSessions))

	// All available agents
	if len(h.agentMetas) > 0 {
		b.WriteString("\n**Available Agents**\n")
		for _, m := range h.agentMetas {
			marker := ""
			if m.Name == h.defaultName {
				marker = " *"
			}
			model := m.Model
			if model == "" {
				model = "-"
			}
			b.WriteString(fmt.Sprintf("* %s%s  type=%s  model=%s\n", m.Name, marker, m.Type, model))
		}
	}

	return strings.TrimRight(b.String(), "\n")
}

func (h *Handler) buildStatusCard() json.RawMessage {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// System facts
	systemFacts := []map[string]string{
		{"title": "Version", "value": h.version},
		{"title": "Platform", "value": fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)},
		{"title": "Uptime", "value": formatDuration(time.Since(h.startTime))},
		{"title": "Go", "value": runtime.Version()},
	}

	// Default agent facts
	var agentFacts []map[string]string
	if h.defaultName == "" {
		agentFacts = []map[string]string{
			{"title": "Name", "value": "none (echo mode)"},
		}
	} else if ag, ok := h.agents[h.defaultName]; !ok {
		agentFacts = []map[string]string{
			{"title": "Name", "value": fmt.Sprintf("%s (not started)", h.defaultName)},
		}
	} else {
		info := ag.Info()
		agentFacts = []map[string]string{
			{"title": "Name", "value": h.defaultName},
			{"title": "Type", "value": info.Type},
		}
		if info.Model != "" {
			agentFacts = append(agentFacts, map[string]string{"title": "Model", "value": info.Model})
		}
		if info.PID > 0 {
			agentFacts = append(agentFacts, map[string]string{"title": "PID", "value": fmt.Sprintf("%d", info.PID)})
		}
	}

	activeSessions := len(h.agents)
	agentFacts = append(agentFacts, map[string]string{"title": "Active Sessions", "value": fmt.Sprintf("%d", activeSessions)})

	// Build card body
	body := []any{
		map[string]any{
			"type":   "TextBlock",
			"text":   "ringclaw Status",
			"weight": "bolder",
			"size":   "medium",
		},
		map[string]any{
			"type":      "TextBlock",
			"text":      "System",
			"weight":    "bolder",
			"spacing":   "medium",
			"separator": true,
		},
		map[string]any{
			"type":  "FactSet",
			"facts": systemFacts,
		},
		map[string]any{
			"type":      "TextBlock",
			"text":      "Default Agent",
			"weight":    "bolder",
			"spacing":   "medium",
			"separator": true,
		},
		map[string]any{
			"type":  "FactSet",
			"facts": agentFacts,
		},
	}

	// Available agents table
	if len(h.agentMetas) > 0 {
		body = append(body, map[string]any{
			"type":      "TextBlock",
			"text":      "Available Agents",
			"weight":    "bolder",
			"spacing":   "medium",
			"separator": true,
		})

		// Header row
		columns := []map[string]any{
			{"type": "Column", "width": "stretch", "items": []map[string]any{
				{"type": "TextBlock", "text": "Name", "weight": "bolder"},
			}},
			{"type": "Column", "width": "auto", "items": []map[string]any{
				{"type": "TextBlock", "text": "Type", "weight": "bolder"},
			}},
			{"type": "Column", "width": "stretch", "items": []map[string]any{
				{"type": "TextBlock", "text": "Model", "weight": "bolder"},
			}},
		}
		body = append(body, map[string]any{
			"type":    "ColumnSet",
			"columns": columns,
		})

		for _, m := range h.agentMetas {
			name := m.Name
			if m.Name == h.defaultName {
				name = m.Name + " *"
			}
			model := m.Model
			if model == "" {
				model = "-"
			}
			row := []map[string]any{
				{"type": "Column", "width": "stretch", "items": []map[string]any{
					{"type": "TextBlock", "text": name},
				}},
				{"type": "Column", "width": "auto", "items": []map[string]any{
					{"type": "TextBlock", "text": m.Type},
				}},
				{"type": "Column", "width": "stretch", "items": []map[string]any{
					{"type": "TextBlock", "text": model},
				}},
			}
			body = append(body, map[string]any{
				"type":    "ColumnSet",
				"columns": row,
			})
		}
	}

	card := map[string]any{
		"type":    "AdaptiveCard",
		"version": "1.3",
		"body":    body,
	}

	data, _ := json.Marshal(card)
	return json.RawMessage(data)
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	secs := int(d.Seconds()) % 60
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, mins, secs)
	}
	if mins > 0 {
		return fmt.Sprintf("%dm %ds", mins, secs)
	}
	return fmt.Sprintf("%ds", secs)
}
