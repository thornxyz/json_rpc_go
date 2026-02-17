package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type RPCRequest struct {
	Method string      `json:"method"`
	Params interface{} `json:"params"`
	ID     interface{} `json:"id"` // Can be null for notifications
}

type RPCResponse struct {
	Result interface{} `json:"result,omitempty"`
	Error  *RPCError   `json:"error,omitempty"`
	ID     interface{} `json:"id"`
}

type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type MethodHandler func(params interface{}) (interface{}, *methodError)

var methods = make(map[string]MethodHandler)

func RegisterMethod(name string, handler MethodHandler) {
	methods[name] = handler
}

func init() {
	RegisterMethod("add", handleAdd)
	RegisterMethod("getUser", handleGetUser)
	RegisterMethod("greet", handleGreet)
}

func handleAdd(params interface{}) (interface{}, *methodError) {
	arr, ok := params.([]interface{})
	if !ok || len(arr) < 2 {
		return nil, &methodError{Code: -32602, Message: "invalid params", Data: "expected array with at least 2 elements"}
	}
	a, ok := toFloat64(arr[0])
	if !ok {
		return nil, &methodError{Code: -32602, Message: "invalid params", Data: "first parameter must be a number"}
	}
	b, ok := toFloat64(arr[1])
	if !ok {
		return nil, &methodError{Code: -32602, Message: "invalid params", Data: "second parameter must be a number"}
	}
	return a + b, nil
}

func handleGetUser(params interface{}) (interface{}, *methodError) {
	m, ok := params.(map[string]interface{})
	if !ok {
		return nil, &methodError{Code: -32602, Message: "invalid params", Data: "expected object with userId field"}
	}
	uidf, ok := m["userId"].(float64)
	if !ok {
		return nil, &methodError{Code: -32602, Message: "invalid params", Data: "userId must be a number"}
	}
	uid := int(uidf)
	return map[string]interface{}{
		"ID":   uid,
		"Name": "Alice",
		"Role": "Admin",
	}, nil
}

func handleGreet(params interface{}) (interface{}, *methodError) {
	m, ok := params.(map[string]interface{})
	if !ok {
		return nil, &methodError{Code: -32602, Message: "invalid params", Data: "expected object with name field"}
	}
	name, ok := m["name"].(string)
	if !ok {
		return nil, &methodError{Code: -32602, Message: "invalid params", Data: "name must be a string"}
	}
	return fmt.Sprintf("Hello, %s!", name), nil
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/rpc", handleRPC)

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		fmt.Println("Server running on http://localhost:8080/rpc")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("\nShutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Server stopped")
}

func handleRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodPost {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		writeError(w, -32600, 0, "invalid request", "only POST method is supported")
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")

	var rawReq json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&rawReq); err != nil {
		writeError(w, -32700, 0, "parse error", err.Error())
		return
	}

	if len(rawReq) == 0 {
		writeError(w, -32600, 0, "invalid request", "empty request body")
		return
	}

	if rawReq[0] == '[' {
		var batchReqs []json.RawMessage
		if err := json.Unmarshal(rawReq, &batchReqs); err != nil {
			writeError(w, -32700, 0, "parse error", err.Error())
			return
		}
		if len(batchReqs) == 0 {
			writeError(w, -32600, 0, "invalid request", "empty batch")
			return
		}
		handleBatchRPC(w, batchReqs)
		return
	}

	var req RPCRequest
	if err := json.Unmarshal(rawReq, &req); err != nil {
		writeError(w, -32700, 0, "parse error", err.Error())
		return
	}

	if req.Method == "" {
		writeError(w, -32600, req.ID, "invalid request", "method is required")
		return
	}

	handleSingleRPC(w, req)
}

func handleSingleRPC(w http.ResponseWriter, req RPCRequest) {
	result, err := handleMethod(req)
	if err != nil {
		writeError(w, err.Code, req.ID, err.Message, err.Data)
		return
	}
	writeResult(w, req.ID, result)
}

func writeResult(w http.ResponseWriter, id interface{}, result interface{}) {
	if id == nil {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	resp := RPCResponse{Result: result, ID: id}
	_ = json.NewEncoder(w).Encode(resp)
}

func writeError(w http.ResponseWriter, code int, id interface{}, msg string, data interface{}) {
	if id == nil {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	resp := RPCResponse{
		Error: &RPCError{
			Code:    code,
			Message: msg,
			Data:    data,
		},
		ID: id,
	}
	_ = json.NewEncoder(w).Encode(resp)
}

type methodError struct {
	Code    int
	Message string
	Data    interface{}
}

func handleMethod(req RPCRequest) (interface{}, *methodError) {
	handler, exists := methods[req.Method]
	if !exists {
		return nil, &methodError{Code: -32601, Message: "method not found", Data: nil}
	}
	return handler(req.Params)
}

func handleBatchRPC(w http.ResponseWriter, batchReqs []json.RawMessage) {
	w.Header().Set("Content-Type", "application/json")

	responses := make([]RPCResponse, 0, len(batchReqs))
	for _, rawReq := range batchReqs {
		var req RPCRequest
		if err := json.Unmarshal(rawReq, &req); err != nil {
			responses = append(responses, RPCResponse{
				Error: &RPCError{Code: -32700, Message: "parse error", Data: err.Error()},
				ID:    0,
			})
			continue
		}

		if req.Method == "" {
			responses = append(responses, RPCResponse{
				Error: &RPCError{Code: -32600, Message: "invalid request", Data: "method is required"},
				ID:    req.ID,
			})
			continue
		}

		result, err := handleMethod(req)
		if err != nil {
			responses = append(responses, RPCResponse{
				Error: &RPCError{Code: err.Code, Message: err.Message, Data: err.Data},
				ID:    req.ID,
			})
			continue
		}

		responses = append(responses, RPCResponse{Result: result, ID: req.ID})
	}

	_ = json.NewEncoder(w).Encode(responses)
}

func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case json.Number:
		f, err := val.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}
