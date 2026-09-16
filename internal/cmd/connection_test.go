package cmd_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

const connectionResponse = `{"connection_id":"c-1","merchant_connection_id":"adyen-us-001",
	"provider_id":"ADYEN","status":"ACTIVE","flow_type":"PAYIN",
	"payment_methods":["CARD","GOOGLE_PAY"],"params":[{"param_id":"API_KEY","value":"secret"}]}`

func TestConnectionCatalog_HitsTheCatalogPath(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	response := `{"payment_method_type":["CARD"],"params":[{"param_id":"API_KEY",
		"field_type":"string","description":"Stripe Secret API Key","optional":false,"secret":true}]}`

	got := startAPI(t, http.StatusOK, response)

	out, err := runCLI(t, "", "connection", "catalog", "STRIPE")
	if err != nil {
		t.Fatalf("connection catalog failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/connections/catalog/STRIPE" {
		t.Errorf("expected GET /v1/connections/catalog/STRIPE, got %s %s", got.Method, got.Path)
	}

	for _, want := range []string{"PARAM_ID", "API_KEY", "Stripe Secret API Key"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestConnectionGet_HitsTheConnectionPath(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, connectionResponse)

	out, err := runCLI(t, "", "connection", "get", "c-1")
	if err != nil {
		t.Fatalf("connection get failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/connections/c-1" {
		t.Errorf("expected GET /v1/connections/c-1, got %s %s", got.Method, got.Path)
	}

	for _, want := range []string{"ADYEN", "ACTIVE", "CARD,GOOGLE_PAY"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestConnectionGet_AcceptsNonStringParamValues(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	response := `{"connection_id":"c-1","provider_id":"ADYEN","status":"ACTIVE",
		"params":[{"param_id":"API_KEY","value":"secret"},{"param_id":"SANDBOX","value":true},
		{"param_id":"RETRIES","value":3}]}`

	startAPI(t, http.StatusOK, response)

	out, err := runCLI(t, "", "connection", "get", "c-1", "--json")
	if err != nil {
		t.Fatalf("connection get failed: %v (%s)", err, out)
	}

	var got struct {
		Params []struct {
			ParamID string `json:"param_id"`
			Value   any    `json:"value"`
		} `json:"params"`
	}

	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("decode output: %v (%s)", err, out)
	}

	want := map[string]any{"API_KEY": "secr****", "SANDBOX": true, "RETRIES": 3.0}

	if len(got.Params) != len(want) {
		t.Fatalf("expected %d params, got %d (%s)", len(want), len(got.Params), out)
	}

	for _, param := range got.Params {
		if param.Value != want[param.ParamID] {
			t.Errorf("param %s: expected %v, got %v", param.ParamID, want[param.ParamID], param.Value)
		}
	}
}

func TestConnectionCreate_SendsTheBody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusCreated, connectionResponse)

	out, err := runCLI(t, "", "connection", "create", "--yes",
		"--account-id", "acc-1",
		"--merchant-connection-id", "adyen-us-001",
		"--provider-id", "ADYEN",
		"--flow-type", "PAYIN",
		"--payment-method", "CARD",
		"--payment-method", "GOOGLE_PAY",
		"--params", `[{"param_id":"API_KEY","value":"secret"}]`)
	if err != nil {
		t.Fatalf("connection create failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/connections" {
		t.Errorf("expected POST /v1/connections, got %s %s", got.Method, got.Path)
	}

	var body map[string]any
	if err := json.Unmarshal([]byte(got.Body), &body); err != nil {
		t.Fatalf("request body is not json: %v (%s)", err, got.Body)
	}

	methods, ok := body["payment_methods"].([]any)
	if !ok || len(methods) != 2 || methods[1] != "GOOGLE_PAY" {
		t.Errorf("expected both payment methods in the body, got %s", got.Body)
	}

	if params, ok := body["params"].([]any); !ok || len(params) != 1 {
		t.Errorf("expected the params array in the body, got %s", got.Body)
	}
}

func TestConnectionCreate_ConfirmsBeforeSending(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	startAPI(t, http.StatusCreated, connectionResponse)

	_, err := runCLI(t, "n\n", "connection", "create", "--provider-id", "ADYEN")
	if err == nil {
		t.Fatal("expected the declined confirmation to abort the request")
	}
}

func TestConnection_ForbiddenMentionsTheScope(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	startAPI(t, http.StatusForbidden,
		`{"type":"forbidden","code":"INSUFFICIENT_SCOPE","message":"missing scope"}`)

	_, err := runCLI(t, "", "connection", "get", "c-1")
	if err == nil {
		t.Fatal("expected a 403 error")
	}

	if !strings.Contains(err.Error(), "connections:read") {
		t.Errorf("expected the error to mention the connections scope, got: %v", err)
	}
}
