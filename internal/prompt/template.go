package prompt

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"text/template"
)

type Template struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Provider    string             `json:"provider"`
	Model       string             `json:"model,omitempty"`
	Template    string             `json:"template"`
	Variables   []TemplateVariable `json:"variables,omitempty"`
	CreatedAt   int64              `json:"created_at"`
	UpdatedAt   int64              `json:"updated_at"`
}

type TemplateVariable struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Default     string `json:"default,omitempty"`
}

type RenderOptions struct {
	Variables map[string]interface{}
}

type Manager struct {
	templates map[string]*Template
	mu        sync.RWMutex
}

func NewManager() *Manager {
	m := &Manager{
		templates: make(map[string]*Template),
	}
	m.loadDefaults()
	return m
}

func (m *Manager) loadDefaults() {
	defaults := []Template{
		{
			ID:          "llama3-chat",
			Name:        "Llama3 Chat Template",
			Description: "Llama3 default chat template",
			Provider:    "meta",
			Template:    `<|begin_of_text|>{{range .Messages}}<|start_header_id|>{{.Role}}<|end_header_id|>\n\n{{.Content}}<|eot_id|>{{end}}<|start_header_id|>assistant<|end_header_id|>\n\n`,
			Variables: []TemplateVariable{
				{Name: "Messages", Type: "array", Description: "List of chat messages", Required: true},
			},
		},
		{
			ID:          "chatml",
			Name:        "ChatML Template",
			Description: "ChatML format used by Qwen and other models",
			Provider:    "qwen",
			Template:    `{{range .Messages}}<|im_start|>{{.Role}}\n{{.Content}}<|im_end|>\n{{end}}<|im_start|>assistant\n`,
			Variables: []TemplateVariable{
				{Name: "Messages", Type: "array", Description: "List of chat messages", Required: true},
			},
		},
		{
			ID:          "chatglm3",
			Name:        "ChatGLM3 Template",
			Description: "ChatGLM3 chat template",
			Provider:    "chatglm",
			Template:    `{{range $i, $m := .Messages}}{{if eq $i 0}}{{if eq $m.Role "system"}}{{.Content}}{{else}}[Round 0]\n问：{{.Content}}\n答：{{end}}{{else}}[Round {{$i}}]\n问：{{.Content}}\n答：{{end}}{{end}}`,
			Variables: []TemplateVariable{
				{Name: "Messages", Type: "array", Description: "List of chat messages", Required: true},
			},
		},
		{
			ID:          "alpaca",
			Name:        "Alpaca Template",
			Description: "Alpaca instruction template",
			Provider:    "alpaca",
			Template:    `{{if .System}}{{.System}}\n\n{{end}}### Instruction:\n{{.Instruction}}\n\n### Response:`,
			Variables: []TemplateVariable{
				{Name: "System", Type: "string", Description: "System prompt"},
				{Name: "Instruction", Type: "string", Description: "User instruction", Required: true},
			},
		},
		{
			ID:          "mistral",
			Name:        "Mistral Template",
			Description: "Mistral chat template",
			Provider:    "mistral",
			Template:    `{{range .Messages}}{{if eq .Role "system"}}[INST] {{.Content}} [/INST]{{else if eq .Role "user"}}[INST] {{.Content}} [/INST]{{else}}{{.Content}}{{end}}{{end}}`,
			Variables: []TemplateVariable{
				{Name: "Messages", Type: "array", Description: "List of chat messages", Required: true},
			},
		},
		{
			ID:          "deepseek-chat",
			Name:        "DeepSeek Chat Template",
			Description: "DeepSeek chat template",
			Provider:    "deepseek",
			Template:    `{{range .Messages}}<｜{{.Role}}｜>{{.Content}}{{end}}<｜assistant｜>`,
			Variables: []TemplateVariable{
				{Name: "Messages", Type: "array", Description: "List of chat messages", Required: true},
			},
		},
		{
			ID:          "yi",
			Name:        "Yi Chat Template",
			Description: "Yi model chat template",
			Provider:    "yi",
			Template:    `{{range .Messages}}<|im_start|>{{.Role}}\n{{.Content}}<|im_end|>\n{{end}}<|im_start|>assistant\n`,
			Variables: []TemplateVariable{
				{Name: "Messages", Type: "array", Description: "List of chat messages", Required: true},
			},
		},
	}

	for _, t := range defaults {
		m.templates[t.ID] = &t
	}
}

func (m *Manager) Add(t *Template) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if t.ID == "" {
		return fmt.Errorf("template ID is required")
	}

	if _, exists := m.templates[t.ID]; exists {
		return fmt.Errorf("template %s already exists", t.ID)
	}

	m.templates[t.ID] = t
	return nil
}

func (m *Manager) Update(t *Template) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.templates[t.ID]; !exists {
		return fmt.Errorf("template %s not found", t.ID)
	}

	m.templates[t.ID] = t
	return nil
}

func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.templates[id]; !exists {
		return fmt.Errorf("template %s not found", id)
	}

	delete(m.templates, id)
	return nil
}

func (m *Manager) Get(id string) (*Template, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	t, exists := m.templates[id]
	if !exists {
		return nil, fmt.Errorf("template %s not found", id)
	}

	return t, nil
}

func (m *Manager) GetByProvider(provider string) []*Template {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*Template
	for _, t := range m.templates {
		if t.Provider == provider {
			result = append(result, t)
		}
	}
	return result
}

func (m *Manager) List() []*Template {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Template, 0, len(m.templates))
	for _, t := range m.templates {
		result = append(result, t)
	}
	return result
}

func (t *Template) Render(data interface{}) (string, error) {
	tmpl, err := template.New(t.ID).Parse(t.Template)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}

	return buf.String(), nil
}

type Message struct {
	Role    string
	Content string
}

func RenderMessages(messages []Message, templateID string) (string, error) {
	mgr := NewManager()
	tmpl, err := mgr.Get(templateID)
	if err != nil {
		return "", err
	}

	data := struct {
		Messages []Message
	}{
		Messages: messages,
	}

	return tmpl.Render(data)
}

func RenderSimpleChat(system, user string, templateID string) (string, error) {
	messages := []Message{}
	if system != "" {
		messages = append(messages, Message{Role: "system", Content: system})
	}
	messages = append(messages, Message{Role: "user", Content: user})
	return RenderMessages(messages, templateID)
}

func ConvertMessagesToPrompt(messages []Message, provider string) string {
	switch strings.ToLower(provider) {
	case "llama", "llama2", "llama3", "meta":
		return convertLlama(messages)
	case "chatglm", "chatglm2", "chatglm3":
		return convertChatGLM(messages)
	case "qwen", "alibaba":
		return convertChatML(messages)
	case "mistral":
		return convertMistral(messages)
	case "deepseek":
		return convertDeepSeek(messages)
	case "yi":
		return convertYi(messages)
	default:
		return convertGeneric(messages)
	}
}

func convertLlama(messages []Message) string {
	var sb strings.Builder
	for _, m := range messages {
		switch m.Role {
		case "system":
			sb.WriteString(fmt.Sprintf("<|begin_of_text|><|start_header_id|>system<|end_header_id|>\n\n%s<|eot_id|>", m.Content))
		case "user":
			sb.WriteString(fmt.Sprintf("<|start_header_id|>user<|end_header_id|>\n\n%s<|eot_id|>", m.Content))
		case "assistant":
			sb.WriteString(fmt.Sprintf("<|start_header_id|>assistant<|end_header_id|>\n\n%s<|eot_id|>", m.Content))
		}
	}
	sb.WriteString("<|start_header_id|>assistant<|end_header_id|>\n\n")
	return sb.String()
}

func convertChatGLM(messages []Message) string {
	var sb strings.Builder
	for i, m := range messages {
		if m.Role == "system" {
			sb.WriteString(m.Content)
			continue
		}
		if m.Role == "user" {
			sb.WriteString(fmt.Sprintf("[Round %d]\n问：%s\n答：", i, m.Content))
		} else if m.Role == "assistant" {
			sb.WriteString(m.Content)
		}
	}
	return sb.String()
}

func convertChatML(messages []Message) string {
	var sb strings.Builder
	for _, m := range messages {
		sb.WriteString(fmt.Sprintf("<|im_start|>%s\n%s<|im_end|>\n", m.Role, m.Content))
	}
	sb.WriteString("<|im_start|>assistant\n")
	return sb.String()
}

func convertMistral(messages []Message) string {
	var sb strings.Builder
	for _, m := range messages {
		if m.Role == "user" {
			sb.WriteString(fmt.Sprintf("[INST] %s [/INST]", m.Content))
		} else if m.Role == "assistant" {
			sb.WriteString(m.Content)
		} else if m.Role == "system" {
			sb.WriteString(fmt.Sprintf("[INST] %s [/INST]", m.Content))
		}
	}
	return sb.String()
}

func convertDeepSeek(messages []Message) string {
	var sb strings.Builder
	for _, m := range messages {
		sb.WriteString(fmt.Sprintf("<｜%s｜>%s", m.Role, m.Content))
	}
	sb.WriteString("<｜assistant｜>")
	return sb.String()
}

func convertYi(messages []Message) string {
	return convertChatML(messages)
}

func convertGeneric(messages []Message) string {
	var sb strings.Builder
	for _, m := range messages {
		sb.WriteString(fmt.Sprintf("%s: %s\n", strings.Title(m.Role), m.Content))
	}
	sb.WriteString("Assistant: ")
	return sb.String()
}
