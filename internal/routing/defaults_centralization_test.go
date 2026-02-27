package routing

import (
	"testing"

	"ai-gateway/internal/constants"
)

func TestDefaultModelScoresSourcedFromConstants(t *testing.T) {
	defaults := DefaultModelScores()
	if len(defaults) != len(constants.RoutingDefaultModelScores) {
		t.Fatalf("default model score count mismatch: got %d want %d", len(defaults), len(constants.RoutingDefaultModelScores))
	}
}

func TestDefaultCascadeRulesSourcedFromConstants(t *testing.T) {
	rules := DefaultCascadeRules()
	if len(rules) != len(constants.RoutingDefaultCascadeRules) {
		t.Fatalf("default cascade rule count mismatch: got %d want %d", len(rules), len(constants.RoutingDefaultCascadeRules))
	}
}
