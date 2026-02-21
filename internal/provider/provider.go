package provider

// This file is kept for backward compatibility.
// Types and interfaces are now defined in types.go
// Provider implementations are in their respective packages:
// - volcengine/ : 火山方舟 (Volcengine Ark)
// - openai/ : OpenAI GPT
// - claude/ : Anthropic Claude
//
// Usage:
//
//	// Create registry and load providers from config
//	registry := provider.NewRegistry()
//	err := registry.LoadFromConfig(cfg.Providers)
//
//	// Get provider by name
//	p, ok := registry.Get("openai")
//
//	// Get provider by model
//	p, ok := registry.GetByModel("gpt-4")
//
//	// Make chat request
//	resp, err := p.Chat(ctx, &provider.ChatRequest{
//	    Model: "gpt-4",
//	    Messages: []provider.ChatMessage{
//	        {Role: "user", Content: "Hello!"},
//	    },
//	})
