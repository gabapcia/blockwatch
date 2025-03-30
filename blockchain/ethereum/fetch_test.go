package ethereum

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestNewRequestBody(t *testing.T) {
	t.Run("no params", func(t *testing.T) {
		method := "eth_blockNumber"
		req := newRequestBody(method)

		if req.JsonRPC != "2.0" {
			t.Errorf("expected JsonRPC \"2.0\", got %q", req.JsonRPC)
		}

		if req.ID != 1 {
			t.Errorf("expected ID 1, got %d", req.ID)
		}

		if req.Method != method {
			t.Errorf("expected method %q, got %q", method, req.Method)
		}

		if len(req.Params) != 0 {
			t.Errorf("expected no params, got %d", len(req.Params))
		}
	})

	t.Run("with params", func(t *testing.T) {
		var (
			method = "eth_getBlockByNumber"
			param1 = "0x1"
			param2 = true
		)

		req := newRequestBody(method, param1, param2)

		if req.JsonRPC != "2.0" {
			t.Errorf("expected JsonRPC \"2.0\", got %q", req.JsonRPC)
		}

		if req.ID != 1 {
			t.Errorf("expected ID 1, got %d", req.ID)
		}

		if req.Method != method {
			t.Errorf("expected method %q, got %q", method, req.Method)
		}

		expectedParams := []any{param1, param2}
		if !reflect.DeepEqual(req.Params, expectedParams) {
			t.Errorf("expected params %v, got %v", expectedParams, req.Params)
		}
	})
}

func TestErrorResponse_toError(t *testing.T) {
	var (
		errResp  = errorResponse{Code: 404, Message: "Not Found"}
		expected = "RPC error: 404 - Not Found"
		err      = errResp.toError()
	)

	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestFetch(t *testing.T) {
	t.Run("Successful Response", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"result": "0x1234", "error": null}`))
		}))
		defer ts.Close()

		s := &service{
			rpcUrl:     ts.URL,
			httpClient: http.DefaultClient,
		}

		result, err := s.fetch(t.Context(), "test_method")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		expected := `"0x1234"`
		if string(result) != expected {
			t.Errorf("expected result %q, got %q", expected, string(result))
		}
	})

	t.Run("Error Response", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"result": null, "error": {"code": 123, "message": "Test error"}}`))
		}))
		defer ts.Close()

		s := &service{
			rpcUrl:     ts.URL,
			httpClient: http.DefaultClient,
		}

		_, err := s.fetch(t.Context(), "test_method")
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		expectedError := "RPC error: 123 - Test error"
		if err.Error() != expectedError {
			t.Errorf("expected error %q, got %q", expectedError, err.Error())
		}
	})

	t.Run("Malformed JSON", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`invalid json`))
		}))
		defer ts.Close()

		s := &service{
			rpcUrl:     ts.URL,
			httpClient: http.DefaultClient,
		}

		_, err := s.fetch(t.Context(), "test_method")
		if err == nil {
			t.Fatal("expected error due to malformed JSON, got nil")
		}
	})
}
