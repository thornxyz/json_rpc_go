package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// RPCRequest represents an incoming JSON-RPC request
type RPCRequest struct {
	Jsonrpc string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      interface{}     `json:"id"`
}

// RPCResponse represents the outgoing response
type RPCResponse struct {
	Jsonrpc string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

// Handler for RPC calls
func rpcHandler(w http.ResponseWriter, r *http.Request) {
	var req RPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	var result interface{}
	var errMsg interface{}

	switch req.Method {
	case "add":
		var params []float64
		json.Unmarshal(req.Params, &params)
		result = params[0] + params[1]

	case "getUser":
		var params map[string]interface{}
		json.Unmarshal(req.Params, &params)
		id := params["userId"].(float64)
		result = map[string]interface{}{
			"id":   id,
			"name": "Alice",
			"role": "Admin",
		}

	case "greet":
		var params map[string]interface{}
		json.Unmarshal(req.Params, &params)
		name := params["name"].(string)
		result = fmt.Sprintf("Hello, %s! Welcome to the RPC System.", name)

	default:
		errMsg = fmt.Sprintf("Unknown method: %s", req.Method)
	}

	resp := RPCResponse{
		Jsonrpc: "2.0",
		Result:  result,
		Error:   errMsg,
		ID:      req.ID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	http.HandleFunc("/rpc", rpcHandler)
	fmt.Println("🚀 RPC Server running on http://localhost:8080/rpc")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
