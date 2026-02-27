package constants

type RoutingModelScorePreset struct {
	Model        string
	Provider     string
	QualityScore int
	SpeedScore   int
	CostScore    int
	Enabled      bool
}

type RoutingTaskRulePreset struct {
	TaskType        string
	Keywords        []string
	PreferredModels []string
}

type RoutingCascadeRulePreset struct {
	TaskType               string
	Difficulty             string
	StartLevel             string
	MaxLevel               string
	FallbackEnabled        bool
	MaxRetries             int
	TimeoutPerLevelSeconds int
}

var RoutingDefaultModelScores = map[string]RoutingModelScorePreset{
	"deepseek-chat":              {Model: "deepseek-chat", Provider: "deepseek", QualityScore: 85, SpeedScore: 90, CostScore: 95, Enabled: true},
	"deepseek-reasoner":          {Model: "deepseek-reasoner", Provider: "deepseek", QualityScore: 95, SpeedScore: 60, CostScore: 90, Enabled: true},
	"deepseek-coder":             {Model: "deepseek-coder", Provider: "deepseek", QualityScore: 90, SpeedScore: 85, CostScore: 95, Enabled: true},
	"gpt-4o":                     {Model: "gpt-4o", Provider: "openai", QualityScore: 95, SpeedScore: 75, CostScore: 60, Enabled: true},
	"gpt-4o-mini":                {Model: "gpt-4o-mini", Provider: "openai", QualityScore: 80, SpeedScore: 95, CostScore: 85, Enabled: true},
	"gpt-4-turbo":                {Model: "gpt-4-turbo", Provider: "openai", QualityScore: 92, SpeedScore: 70, CostScore: 55, Enabled: true},
	"o1":                         {Model: "o1", Provider: "openai", QualityScore: 98, SpeedScore: 40, CostScore: 30, Enabled: true},
	"o1-mini":                    {Model: "o1-mini", Provider: "openai", QualityScore: 90, SpeedScore: 60, CostScore: 50, Enabled: true},
	"claude-3-5-sonnet-20241022": {Model: "claude-3-5-sonnet-20241022", Provider: "anthropic", QualityScore: 96, SpeedScore: 70, CostScore: 55, Enabled: true},
	"claude-3-5-haiku-20241022":  {Model: "claude-3-5-haiku-20241022", Provider: "anthropic", QualityScore: 82, SpeedScore: 95, CostScore: 80, Enabled: true},
	"claude-3-opus-20240229":     {Model: "claude-3-opus-20240229", Provider: "anthropic", QualityScore: 97, SpeedScore: 50, CostScore: 40, Enabled: true},
	"qwen-max":                   {Model: "qwen-max", Provider: "qwen", QualityScore: 90, SpeedScore: 80, CostScore: 75, Enabled: true},
	"qwen-plus":                  {Model: "qwen-plus", Provider: "qwen", QualityScore: 85, SpeedScore: 85, CostScore: 85, Enabled: true},
	"qwen-turbo":                 {Model: "qwen-turbo", Provider: "qwen", QualityScore: 75, SpeedScore: 95, CostScore: 95, Enabled: true},
	"qwen-long":                  {Model: "qwen-long", Provider: "qwen", QualityScore: 80, SpeedScore: 70, CostScore: 70, Enabled: true},
	"glm-4-plus":                 {Model: "glm-4-plus", Provider: "zhipu", QualityScore: 88, SpeedScore: 80, CostScore: 80, Enabled: true},
	"glm-4":                      {Model: "glm-4", Provider: "zhipu", QualityScore: 85, SpeedScore: 85, CostScore: 85, Enabled: true},
	"glm-4-flash":                {Model: "glm-4-flash", Provider: "zhipu", QualityScore: 70, SpeedScore: 98, CostScore: 98, Enabled: true},
	"glm-4-long":                 {Model: "glm-4-long", Provider: "zhipu", QualityScore: 80, SpeedScore: 70, CostScore: 75, Enabled: true},
	"moonshot-v1-8k":             {Model: "moonshot-v1-8k", Provider: "moonshot", QualityScore: 85, SpeedScore: 85, CostScore: 85, Enabled: true},
	"moonshot-v1-32k":            {Model: "moonshot-v1-32k", Provider: "moonshot", QualityScore: 85, SpeedScore: 80, CostScore: 80, Enabled: true},
	"moonshot-v1-128k":           {Model: "moonshot-v1-128k", Provider: "moonshot", QualityScore: 85, SpeedScore: 75, CostScore: 75, Enabled: true},
	"abab6.5s-chat":              {Model: "abab6.5s-chat", Provider: "minimax", QualityScore: 85, SpeedScore: 90, CostScore: 85, Enabled: true},
	"abab6.5g-chat":              {Model: "abab6.5g-chat", Provider: "minimax", QualityScore: 88, SpeedScore: 80, CostScore: 80, Enabled: true},
	"abab5.5-chat":               {Model: "abab5.5-chat", Provider: "minimax", QualityScore: 80, SpeedScore: 90, CostScore: 90, Enabled: true},
	"Baichuan4":                  {Model: "Baichuan4", Provider: "baichuan", QualityScore: 88, SpeedScore: 80, CostScore: 80, Enabled: true},
	"Baichuan3-Turbo":            {Model: "Baichuan3-Turbo", Provider: "baichuan", QualityScore: 82, SpeedScore: 90, CostScore: 90, Enabled: true},
	"doubao-pro-128k":            {Model: "doubao-pro-128k", Provider: "volcengine", QualityScore: 85, SpeedScore: 85, CostScore: 85, Enabled: true},
	"doubao-lite-32k":            {Model: "doubao-lite-32k", Provider: "volcengine", QualityScore: 75, SpeedScore: 95, CostScore: 95, Enabled: true},
	"gemini-2.0-flash":           {Model: "gemini-2.0-flash", Provider: "google", QualityScore: 88, SpeedScore: 90, CostScore: 80, Enabled: true},
	"gemini-1.5-pro":             {Model: "gemini-1.5-pro", Provider: "google", QualityScore: 92, SpeedScore: 75, CostScore: 70, Enabled: true},
}

var RoutingDefaultTaskRules = []RoutingTaskRulePreset{
	{TaskType: "code", Keywords: []string{"代码", "code", "编程", "bug", "debug", "function", "class", "实现"}, PreferredModels: []string{"deepseek-coder", "claude-3-5-sonnet-20241022", "gpt-4o"}},
	{TaskType: "reasoning", Keywords: []string{"推理", "reasoning", "分析", "逻辑", "证明", "数学", "math"}, PreferredModels: []string{"deepseek-reasoner", "o1", "o1-mini", "claude-3-opus-20240229"}},
	{TaskType: "long_context", Keywords: []string{"长文本", "总结", "摘要", "文档", "分析报告"}, PreferredModels: []string{"qwen-long", "glm-4-long", "moonshot-v1-128k", "claude-3-5-sonnet-20241022"}},
	{TaskType: "creative", Keywords: []string{"写作", "创意", "故事", "文案", "创作"}, PreferredModels: []string{"claude-3-5-sonnet-20241022", "gpt-4o", "qwen-max"}},
	{TaskType: "chat", Keywords: []string{}, PreferredModels: []string{"deepseek-chat", "gpt-4o-mini", "glm-4-flash", "qwen-turbo"}},
}

var RoutingDefaultProviderDefaults = map[string]string{
	"deepseek":   "deepseek-chat",
	"openai":     "gpt-4o",
	"anthropic":  "claude-3-5-sonnet-20241022",
	"qwen":       "qwen-max",
	"zhipu":      "glm-4-plus",
	"moonshot":   "moonshot-v1-8k",
	"minimax":    "abab6.5s-chat",
	"baichuan":   "Baichuan4",
	"volcengine": "doubao-pro-128k",
	"google":     "gemini-2.0-flash",
}

var RoutingDefaultCascadeRules = map[string]RoutingCascadeRulePreset{
	"chat:low":         {TaskType: "chat", Difficulty: "low", StartLevel: "small", MaxLevel: "medium", FallbackEnabled: true, MaxRetries: 2, TimeoutPerLevelSeconds: 10},
	"chat:medium":      {TaskType: "chat", Difficulty: "medium", StartLevel: "medium", MaxLevel: "large", FallbackEnabled: true, MaxRetries: 2, TimeoutPerLevelSeconds: 15},
	"chat:high":        {TaskType: "chat", Difficulty: "high", StartLevel: "large", MaxLevel: "large", FallbackEnabled: false, MaxRetries: 1, TimeoutPerLevelSeconds: 30},
	"code:low":         {TaskType: "code", Difficulty: "low", StartLevel: "small", MaxLevel: "medium", FallbackEnabled: true, MaxRetries: 2, TimeoutPerLevelSeconds: 15},
	"code:medium":      {TaskType: "code", Difficulty: "medium", StartLevel: "medium", MaxLevel: "large", FallbackEnabled: true, MaxRetries: 2, TimeoutPerLevelSeconds: 20},
	"code:high":        {TaskType: "code", Difficulty: "high", StartLevel: "large", MaxLevel: "large", FallbackEnabled: false, MaxRetries: 1, TimeoutPerLevelSeconds: 60},
	"reasoning:low":    {TaskType: "reasoning", Difficulty: "low", StartLevel: "medium", MaxLevel: "large", FallbackEnabled: true, MaxRetries: 2, TimeoutPerLevelSeconds: 20},
	"reasoning:medium": {TaskType: "reasoning", Difficulty: "medium", StartLevel: "medium", MaxLevel: "large", FallbackEnabled: true, MaxRetries: 2, TimeoutPerLevelSeconds: 30},
	"reasoning:high":   {TaskType: "reasoning", Difficulty: "high", StartLevel: "large", MaxLevel: "large", FallbackEnabled: false, MaxRetries: 1, TimeoutPerLevelSeconds: 120},
}

var RoutingDefaultModelLevels = map[string][]string{
	"small":  {"gpt-4o-mini", "glm-4-flash", "qwen-turbo", "doubao-lite-32k", "deepseek-chat", "abab5.5-chat", "Baichuan3-Turbo"},
	"medium": {"deepseek-coder", "gpt-4o", "claude-3-5-haiku-20241022", "qwen-plus", "glm-4", "moonshot-v1-8k", "abab6.5s-chat", "doubao-pro-128k", "gemini-2.0-flash"},
	"large":  {"deepseek-reasoner", "o1", "o1-mini", "claude-3-5-sonnet-20241022", "claude-3-opus-20240229", "qwen-max", "glm-4-plus", "Baichuan4", "gemini-1.5-pro", "gpt-4-turbo"},
}

var RoutingTaskTypeLevelModelPrefs = map[string]map[string][]string{
	"code": {
		"small":  {"deepseek-chat", "gpt-4o-mini"},
		"medium": {"deepseek-coder", "claude-3-5-haiku-20241022"},
		"large":  {"claude-3-5-sonnet-20241022", "gpt-4o", "o1-mini"},
	},
	"reasoning": {
		"small":  {"gpt-4o-mini", "glm-4-flash"},
		"medium": {"gpt-4o", "qwen-plus"},
		"large":  {"deepseek-reasoner", "o1", "claude-3-opus-20240229"},
	},
	"math": {
		"small":  {"gpt-4o-mini", "deepseek-chat"},
		"medium": {"gpt-4o", "qwen-plus"},
		"large":  {"deepseek-reasoner", "o1"},
	},
	"creative": {
		"small":  {"gpt-4o-mini", "qwen-turbo"},
		"medium": {"gpt-4o", "claude-3-5-haiku-20241022"},
		"large":  {"claude-3-5-sonnet-20241022", "claude-3-opus-20240229"},
	},
}
