package prompt

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	mgr := NewManager()
	require.NotNil(t, mgr)
	assert.NotEmpty(t, mgr.templates)
}

func TestManager_Add(t *testing.T) {
	mgr := NewManager()

	tmpl := &Template{
		ID:          "test-template",
		Name:        "Test Template",
		Description: "A test template",
		Provider:    "test",
		Template:    "Hello {{.Name}}",
	}

	err := mgr.Add(tmpl)
	require.NoError(t, err)

	retrieved, err := mgr.Get("test-template")
	require.NoError(t, err)
	assert.Equal(t, "Test Template", retrieved.Name)
}

func TestManager_Add_Duplicate(t *testing.T) {
	mgr := NewManager()

	tmpl := &Template{
		ID:       "llama3-chat",
		Name:     "Duplicate",
		Provider: "test",
	}

	err := mgr.Add(tmpl)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestManager_Add_NoID(t *testing.T) {
	mgr := NewManager()

	tmpl := &Template{
		Name:     "No ID",
		Provider: "test",
	}

	err := mgr.Add(tmpl)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ID is required")
}

func TestManager_Update(t *testing.T) {
	mgr := NewManager()

	tmpl := &Template{
		ID:          "llama3-chat",
		Name:        "Updated Name",
		Description: "Updated description",
		Provider:    "meta",
	}

	err := mgr.Update(tmpl)
	require.NoError(t, err)

	retrieved, err := mgr.Get("llama3-chat")
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", retrieved.Name)
}

func TestManager_Update_NotFound(t *testing.T) {
	mgr := NewManager()

	tmpl := &Template{
		ID:   "nonexistent",
		Name: "Test",
	}

	err := mgr.Update(tmpl)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestManager_Delete(t *testing.T) {
	mgr := NewManager()

	tmpl := &Template{
		ID:       "to-delete",
		Name:     "To Delete",
		Provider: "test",
	}
	mgr.Add(tmpl)

	err := mgr.Delete("to-delete")
	require.NoError(t, err)

	_, err = mgr.Get("to-delete")
	assert.Error(t, err)
}

func TestManager_Delete_NotFound(t *testing.T) {
	mgr := NewManager()

	err := mgr.Delete("nonexistent")
	assert.Error(t, err)
}

func TestManager_GetByProvider(t *testing.T) {
	mgr := NewManager()

	templates := mgr.GetByProvider("meta")
	assert.NotEmpty(t, templates)
	for _, tmpl := range templates {
		assert.Equal(t, "meta", tmpl.Provider)
	}
}

func TestManager_List(t *testing.T) {
	mgr := NewManager()

	list := mgr.List()
	assert.NotEmpty(t, list)
}

func TestTemplate_Render(t *testing.T) {
	tmpl := &Template{
		ID:       "test",
		Template: "Hello, {{.Name}}!",
	}

	result, err := tmpl.Render(map[string]string{"Name": "World"})
	require.NoError(t, err)
	assert.Equal(t, "Hello, World!", result)
}

func TestTemplate_Render_InvalidTemplate(t *testing.T) {
	tmpl := &Template{
		ID:       "test",
		Template: "Hello, {{.Name",
	}

	_, err := tmpl.Render(map[string]string{"Name": "World"})
	assert.Error(t, err)
}

func TestRenderMessages(t *testing.T) {
	messages := []Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
	}

	result, err := RenderMessages(messages, "chatml")
	require.NoError(t, err)
	assert.Contains(t, result, "user")
	assert.Contains(t, result, "Hello")
	assert.Contains(t, result, "assistant")
	assert.Contains(t, result, "Hi there!")
}

func TestRenderMessages_TemplateNotFound(t *testing.T) {
	messages := []Message{
		{Role: "user", Content: "Hello"},
	}

	_, err := RenderMessages(messages, "nonexistent")
	assert.Error(t, err)
}

func TestRenderSimpleChat(t *testing.T) {
	result, err := RenderSimpleChat("You are helpful.", "What is AI?", "chatml")
	require.NoError(t, err)
	assert.Contains(t, result, "system")
	assert.Contains(t, result, "You are helpful")
	assert.Contains(t, result, "user")
	assert.Contains(t, result, "What is AI?")
}

func TestConvertMessagesToPrompt(t *testing.T) {
	messages := []Message{
		{Role: "system", Content: "Be helpful"},
		{Role: "user", Content: "Hi"},
		{Role: "assistant", Content: "Hello!"},
	}

	tests := []struct {
		provider string
		contains string
		notEmpty bool
	}{
		{"llama", "<|start_header_id|>", true},
		{"chatglm", "问：", true},
		{"qwen", "<|im_start|>", true},
		{"mistral", "[INST]", true},
		{"deepseek", "<｜", true},
		{"yi", "<|im_start|>", true},
		{"unknown", "System:", true},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			result := ConvertMessagesToPrompt(messages, tt.provider)
			assert.True(t, len(result) > 0, "result should not be empty")
			if tt.contains != "" {
				assert.Contains(t, result, tt.contains)
			}
		})
	}
}
