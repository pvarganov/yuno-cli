package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPCIProxyDestination_KeepsTheRawBody(t *testing.T) {
	t.Parallel()

	const body = `{"id":"d-1","hostname":"api.processor.com","unknown_field":"kept"}`

	var destination PCIProxyDestination
	if err := json.Unmarshal([]byte(body), &destination); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	replayed, err := json.Marshal(destination)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if !strings.Contains(string(replayed), "unknown_field") {
		t.Errorf("expected the raw body to survive, got %s", replayed)
	}
}

func TestPCIProxyDestination_MarshalsWithoutARawBody(t *testing.T) {
	t.Parallel()

	data, err := json.Marshal(PCIProxyDestination{ID: "d-1", Hostname: "api.processor.com"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if !strings.Contains(string(data), `"hostname"`) {
		t.Errorf("expected the typed fields, got %s", data)
	}
}

func TestPCIProxyDestinationViews_FlattenTheList(t *testing.T) {
	t.Parallel()

	destinations := []PCIProxyDestination{
		{ID: "d-1", Hostname: "api.processor.com", Status: "ENABLED", Purpose: "charges"},
		{ID: "d-2", Hostname: "api.other.com", Status: "DISABLED"},
	}

	views := PCIProxyDestinationViews(destinations)
	if len(views) != 2 || views[0].Purpose != "charges" || views[1].Status != "DISABLED" {
		t.Errorf("unexpected views: %+v", views)
	}
}
