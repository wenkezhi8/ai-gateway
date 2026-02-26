package routing

import "testing"

func TestClampClassifierConfig_PreserveControlFlagsWhenDisabled(t *testing.T) {
	cfg := DefaultClassifierConfig()
	cfg.Control.Enable = false
	cfg.Control.ShadowOnly = false
	cfg.Control.ToolGateEnable = true
	cfg.Control.ModelFitEnable = true

	clamped := clampClassifierConfig(cfg)

	if clamped.Control.ToolGateEnable != cfg.Control.ToolGateEnable {
		t.Fatalf("expected ToolGateEnable preserved, got %v", clamped.Control.ToolGateEnable)
	}
	if clamped.Control.ModelFitEnable != cfg.Control.ModelFitEnable {
		t.Fatalf("expected ModelFitEnable preserved, got %v", clamped.Control.ModelFitEnable)
	}
	if clamped.Control.ShadowOnly != cfg.Control.ShadowOnly {
		t.Fatalf("expected ShadowOnly preserved, got %v", clamped.Control.ShadowOnly)
	}
}

func TestSelectModelByControlFit(t *testing.T) {
	r := NewSmartRouter()

	var modelA, modelB string
	for model, score := range r.config.ModelScores {
		if !score.Enabled {
			continue
		}
		if modelA == "" {
			modelA = model
			continue
		}
		if modelB == "" {
			modelB = model
			break
		}
	}

	if modelA == "" || modelB == "" {
		t.Fatal("expected at least 2 enabled models in default config")
	}

	r.config.Classifier.Control.Enable = true
	r.config.Classifier.Control.ShadowOnly = false
	r.config.Classifier.Control.ModelFitEnable = true

	assessment := &AssessmentResult{
		TaskType: TaskTypeChat,
		ControlSignals: &ControlSignals{
			ModelFit: map[string]float64{
				modelA: 0.31,
				modelB: 0.89,
			},
		},
	}

	selected := r.selectModelByControlFit(assessment, []string{modelA, modelB})
	if selected != modelB {
		t.Fatalf("expected %s, got %s", modelB, selected)
	}

	r.config.Classifier.Control.ModelFitEnable = false
	selected = r.selectModelByControlFit(assessment, []string{modelA, modelB})
	if selected != "" {
		t.Fatalf("expected empty selection when feature disabled, got %s", selected)
	}

	r.config.Classifier.Control.ModelFitEnable = true
	r.config.Classifier.Control.ShadowOnly = true
	selected = r.selectModelByControlFit(assessment, []string{modelA, modelB})
	if selected != "" {
		t.Fatalf("expected shadow mode to skip apply, got %s", selected)
	}
}
