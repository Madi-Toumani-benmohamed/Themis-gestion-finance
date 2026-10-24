package api

import (
	"context"
	"testing"
)

func TestAssistantDoesNotInventCompleteCoverage(t *testing.T) {
	srv := &Server{}
	srv.ai.client = scriptedLLM{reply: `{"answer":"Informations insuffisantes pour établir la couverture.","status":"ok"}`}
	result, err := srv.orchestrate(context.Background(), nil, 1, nil, "Quel est mon solde ?")
	if err != nil {
		t.Fatal(err)
	}
	if result.DataQuality["status"] != "unknown" {
		t.Fatalf("unverified coverage: %+v", result.DataQuality)
	}
}
