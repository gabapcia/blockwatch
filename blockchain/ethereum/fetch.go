package ethereum

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type (
	// requestBody represents the JSON-RPC request payload to be sent to an Ethereum node.
	// It conforms to the standard JSON-RPC 2.0 specification.
	requestBody struct {
		// JsonRPC is the JSON-RPC version.
		JsonRPC string `json:"jsonrpc"`
		// ID is the identifier for the request.
		ID int `json:"id"`
		// Method specifies the JSON-RPC method to be invoked.
		Method string `json:"method"`
		// Params holds the parameters for the JSON-RPC method.
		Params []any `json:"params"`
	}

	// errorResponse encapsulates the error information returned by an Ethereum JSON-RPC endpoint.
	errorResponse struct {
		// Code is the error code.
		Code int `json:"code"`
		// Message provides details about the error.
		Message string `json:"message"`
	}

	// responseBody represents the JSON-RPC response returned from an Ethereum node.
	// It contains either the result or an error.
	responseBody struct {
		// Result contains the raw JSON result returned by the request.
		Result json.RawMessage `json:"result"`
		// Error contains the error information if the request failed.
		Error *errorResponse `json:"error"`
	}
)

// newRequestBody creates and returns a new requestBody for the given JSON-RPC method
// and parameters. It initializes the JSON-RPC version to "2.0" and sets a fixed request ID.
func newRequestBody(method string, params ...any) requestBody {
	return requestBody{
		JsonRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	}
}

// toError converts an errorResponse to a standard Go error.
// It formats the error code and message into a readable error string.
func (e errorResponse) toError() error {
	return fmt.Errorf("RPC error: %d - %s", e.Code, e.Message)
}

// fetch sends a JSON-RPC request to the Ethereum node using the specified method and parameters.
// It constructs the request payload, sends an HTTP POST request, and decodes the JSON response.
// On success, it returns the raw JSON result. If the response contains an error or if the request
// fails, it returns an appropriate error.
func (s *service) fetch(ctx context.Context, method string, params ...any) (json.RawMessage, error) {
	reqBody, err := json.Marshal(newRequestBody(method, params...))
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.rpcUrl, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var data responseBody
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		return nil, err
	}

	if data.Error != nil {
		return nil, data.Error.toError()
	}

	return data.Result, nil
}
