package cmd_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// paymentResponse is the payment payload the fake API returns.
const paymentResponse = `{"id":"pay-1","account_id":"acc-1","merchant_order_id":"order-42",
	"description":"Test payment","country":"CO","status":"SUCCEEDED","sub_status":"APPROVED",
	"amount":{"currency":"USD","value":150},
	"payment_method":{"type":"CARD","detail":{"card":{"brand":"VISA","lfd":"1234"}}},
	"transactions":[{"id":"t-1","type":"PURCHASE","status":"SUCCEEDED","category":"CARD",
	"provider_id":"STRIPE","amount":{"currency":"USD","value":150},"provider_data":{"code":"00"}}],
	"created_at":"2026-09-16T10:00:00Z"}`

// decodeBody parses the JSON body the fake API received.
func decodeBody(t *testing.T, raw string) map[string]any {
	t.Helper()

	var body map[string]any
	if err := json.Unmarshal([]byte(raw), &body); err != nil {
		t.Fatalf("request body is not json: %v (%s)", err, raw)
	}

	return body
}

func TestPaymentCreate_PostsTheBody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusCreated, paymentResponse)

	out, err := runCLI(t, "", "payment", "create", "--yes",
		"--account-id", "acc-1", "--merchant-order-id", "order-42",
		"--description", "Test payment", "--country", "CO",
		"--currency", "USD", "--amount", "150",
		"--checkout-session", "sess-1", "--workflow", "DIRECT",
		"--payment-method-type", "CARD", "--payment-method-token", "tok-1",
		"--customer-id", "cus-1")
	if err != nil {
		t.Fatalf("payment create failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/payments" {
		t.Errorf("expected POST /v1/payments, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)

	amount, ok := body["amount"].(map[string]any)
	if !ok || amount["currency"] != "USD" || amount["value"] != float64(150) {
		t.Errorf("unexpected amount in body: %s", got.Body)
	}

	checkout, ok := body["checkout"].(map[string]any)
	if !ok || checkout["session"] != "sess-1" {
		t.Errorf("unexpected checkout in body: %s", got.Body)
	}

	method, ok := body["payment_method"].(map[string]any)
	if !ok || method["type"] != "CARD" || method["token"] != "tok-1" {
		t.Errorf("unexpected payment method in body: %s", got.Body)
	}

	if payer, ok := body["customer_payer"].(map[string]any); !ok || payer["id"] != "cus-1" {
		t.Errorf("unexpected customer payer in body: %s", got.Body)
	}

	for _, want := range []string{"MERCHANT_ORDER_ID", "SUCCEEDED", "CARD VISA 1234"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected table output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestPaymentCreate_RequiresABody(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusCreated, paymentResponse)

	_, err := runCLI(t, "", "payment", "create", "--yes")
	if err == nil {
		t.Fatal("expected an error when no body is given")
	}

	if !strings.Contains(err.Error(), "request body is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPaymentGet_HitsThePaymentPath(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, paymentResponse)

	out, err := runCLI(t, "", "payment", "get", "pay-1",
		"--raw-response", "--transactions-history")
	if err != nil {
		t.Fatalf("payment get failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/payments/pay-1" {
		t.Errorf("expected GET /v1/payments/pay-1, got %s %s", got.Method, got.Path)
	}

	if got.Query != "raw_response=true&transactions_history=true" {
		t.Errorf("unexpected query: %s", got.Query)
	}
}

func TestPaymentGet_JSONKeepsUntypedFields(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, paymentResponse)

	out, err := runCLI(t, "", "payment", "get", "pay-1", "--json")
	if err != nil {
		t.Fatalf("payment get failed: %v (%s)", err, out)
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("output is not json: %v (%s)", err, out)
	}

	transactions, ok := decoded["transactions"].([]any)
	if !ok || len(transactions) != 1 {
		t.Fatalf("expected one transaction in the json output, got: %s", out)
	}

	first, ok := transactions[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected transaction shape: %s", out)
	}

	if _, ok := first["provider_data"]; !ok {
		t.Errorf("expected provider_data to survive the round trip, got: %s", out)
	}
}

func TestPaymentGetByOrderID_QueriesTheMerchantOrder(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, "["+paymentResponse+"]")

	out, err := runCLI(t, "", "payment", "get-by-order-id", "order-42")
	if err != nil {
		t.Fatalf("payment get-by-order-id failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/payments" {
		t.Errorf("expected GET /v1/payments, got %s %s", got.Method, got.Path)
	}

	if got.Query != "merchant_order_id=order-42" {
		t.Errorf("unexpected query: %s", got.Query)
	}

	if !strings.Contains(out, "pay-1") {
		t.Errorf("expected the payment id in the output, got:\n%s", out)
	}
}

func TestPaymentList_QueriesTheMerchantOrder(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, "["+paymentResponse+"]")

	if out, err := runCLI(t, "", "payment", "list", "--merchant-order-id", "order-42"); err != nil {
		t.Fatalf("payment list failed: %v (%s)", err, out)
	}

	if got.Query != "merchant_order_id=order-42" {
		t.Errorf("unexpected query: %s", got.Query)
	}
}

func TestPaymentList_RequiresTheMerchantOrderID(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, `[]`)

	if _, err := runCLI(t, "", "payment", "list"); err == nil {
		t.Fatal("expected an error when no merchant order id is given")
	}
}

func TestPaymentIssuers_QueriesTheCatalog(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, `{"issuers":[{"id":"1001","name":"BANCO DE BOGOTA"}]}`)

	out, err := runCLI(t, "", "payment", "issuers",
		"--country-code", "CO", "--payment-method", "PSE", "--checkout-session", "sess-1")
	if err != nil {
		t.Fatalf("payment issuers failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodGet || got.Path != "/v1/issuers" {
		t.Errorf("expected GET /v1/issuers, got %s %s", got.Method, got.Path)
	}

	if got.Query != "checkout_session=sess-1&country_code=CO&payment_method=PSE" {
		t.Errorf("unexpected query: %s", got.Query)
	}

	if !strings.Contains(out, "BANCO DE BOGOTA") {
		t.Errorf("expected the issuer name in the output, got:\n%s", out)
	}
}

func TestPaymentRefund_PostsToTheTransaction(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, paymentResponse)

	out, err := runCLI(t, "", "payment", "refund", "pay-1", "t-1", "--yes",
		"--merchant-reference", "ref-1", "--reason", "REQUESTED_BY_CUSTOMER",
		"--currency", "USD", "--amount", "50")
	if err != nil {
		t.Fatalf("payment refund failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/payments/pay-1/transactions/t-1/refund" {
		t.Errorf("expected POST the refund path, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)
	if body["merchant_reference"] != "ref-1" || body["reason"] != "REQUESTED_BY_CUSTOMER" {
		t.Errorf("unexpected refund body: %s", got.Body)
	}

	if amount, ok := body["amount"].(map[string]any); !ok || amount["value"] != float64(50) {
		t.Errorf("unexpected refund amount: %s", got.Body)
	}
}

func TestPaymentCancel_PostsToTheTransaction(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, paymentResponse)

	out, err := runCLI(t, "", "payment", "cancel", "pay-1", "t-1", "--yes",
		"--merchant-reference", "ref-1")
	if err != nil {
		t.Fatalf("payment cancel failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/payments/pay-1/transactions/t-1/cancel" {
		t.Errorf("expected POST the cancel path, got %s %s", got.Method, got.Path)
	}

	if body := decodeBody(t, got.Body); body["merchant_reference"] != "ref-1" {
		t.Errorf("unexpected cancel body: %s", got.Body)
	}
}

func TestPaymentCapture_PostsToTheTransaction(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, paymentResponse)

	out, err := runCLI(t, "", "payment", "capture", "pay-1", "t-1", "--yes",
		"--merchant-reference", "ref-1", "--reason", "CAPTURE",
		"--currency", "USD", "--amount", "150")
	if err != nil {
		t.Fatalf("payment capture failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/payments/pay-1/transactions/t-1/capture" {
		t.Errorf("expected POST the capture path, got %s %s", got.Method, got.Path)
	}

	if body := decodeBody(t, got.Body); body["reason"] != "CAPTURE" {
		t.Errorf("unexpected capture body: %s", got.Body)
	}
}

func TestPaymentCancelOrRefund_PaymentLevel(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, paymentResponse)

	out, err := runCLI(t, "", "payment", "cancel-or-refund", "pay-1", "--yes",
		"--reason", "REQUESTED_BY_CUSTOMER")
	if err != nil {
		t.Fatalf("payment cancel-or-refund failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/payments/pay-1/cancel-or-refund" {
		t.Errorf("expected POST the payment level path, got %s %s", got.Method, got.Path)
	}

	if body := decodeBody(t, got.Body); body["reason"] != "REQUESTED_BY_CUSTOMER" {
		t.Errorf("unexpected body: %s", got.Body)
	}
}

func TestPaymentCancelOrRefund_TransactionLevel(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, paymentResponse)

	out, err := runCLI(t, "", "payment", "cancel-or-refund", "pay-1", "t-1", "--yes",
		"--reason", "REQUESTED_BY_CUSTOMER")
	if err != nil {
		t.Fatalf("payment cancel-or-refund failed: %v (%s)", err, out)
	}

	if got.Path != "/v1/payments/pay-1/transactions/t-1/cancel-or-refund" {
		t.Errorf("expected the transaction level path, got %s", got.Path)
	}
}

func TestPaymentDisputeCreate_PostsTheEvidence(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, `{"status":"RECEIVED"}`)

	out, err := runCLI(t, "", "payment", "dispute", "create", "pay-1", "t-1", "--yes",
		"--account-id", "acc-1", "--evidence", `{"files":["ev-1"]}`)
	if err != nil {
		t.Fatalf("payment dispute create failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/payments/pay-1/transactions/t-1/dispute" {
		t.Errorf("expected POST the dispute path, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)
	if body["account_id"] != "acc-1" {
		t.Errorf("unexpected dispute body: %s", got.Body)
	}

	if _, ok := body["evidence"].(map[string]any); !ok {
		t.Errorf("expected the evidence object in the body: %s", got.Body)
	}

	if !strings.Contains(out, "RECEIVED") {
		t.Errorf("expected the response in the output, got:\n%s", out)
	}
}

func TestPaymentDisputeUpdate_PatchesTheEvidence(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, `{"status":"UPDATED"}`)

	out, err := runCLI(t, "", "payment", "dispute", "update", "pay-1", "t-1", "--yes",
		"--account-id", "acc-1", "--evidence", `{"files":["ev-2"]}`)
	if err != nil {
		t.Fatalf("payment dispute update failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPatch || got.Path != "/v1/payments/pay-1/transactions/t-1/dispute" {
		t.Errorf("expected PATCH the dispute path, got %s %s", got.Method, got.Path)
	}
}

func TestPaymentDisputeCreate_RejectsMalformedEvidence(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, `{}`)

	_, err := runCLI(t, "", "payment", "dispute", "create", "pay-1", "t-1", "--yes",
		"--account-id", "acc-1", "--evidence", "{not json")
	if err == nil {
		t.Fatal("expected an error for malformed evidence json")
	}

	if !strings.Contains(err.Error(), "--evidence is not valid json") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPaymentFulfillments_PostsTheStatus(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)

	got := startAPI(t, http.StatusOK, `{"status":"FULFILLED"}`)

	out, err := runCLI(t, "", "payment", "fulfillments", "pay-1", "--yes",
		"--status", "FULFILLED", "--fulfillments", `[{"id":"f-1"}]`)
	if err != nil {
		t.Fatalf("payment fulfillments failed: %v (%s)", err, out)
	}

	if got.Method != http.MethodPost || got.Path != "/v1/payments/pay-1/fulfillments" {
		t.Errorf("expected POST the fulfillments path, got %s %s", got.Method, got.Path)
	}

	body := decodeBody(t, got.Body)
	if body["status"] != "FULFILLED" {
		t.Errorf("unexpected fulfillments body: %s", got.Body)
	}

	if _, ok := body["fulfillments"].([]any); !ok {
		t.Errorf("expected the fulfillments array in the body: %s", got.Body)
	}
}

func TestPaymentGet_ForbiddenMentionsTheScope(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusForbidden, `{"code":"INSUFFICIENT_SCOPE","message":"forbidden"}`)

	_, err := runCLI(t, "", "payment", "get", "pay-1")
	if err == nil {
		t.Fatal("expected an error on 403")
	}

	if !strings.Contains(err.Error(), "payments:read") {
		t.Errorf("expected the scope hint in the error, got: %v", err)
	}
}

func TestPaymentRefund_RequiresConfirmation(t *testing.T) {
	isolateConfig(t)
	seedCredentials(t)
	startAPI(t, http.StatusOK, paymentResponse)

	_, err := runCLI(t, "n\n", "payment", "refund", "pay-1", "t-1",
		"--merchant-reference", "ref-1")
	if err == nil {
		t.Fatal("expected the refund to be declined")
	}
}
