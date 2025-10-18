package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type RPCRequest struct {
	Method string      `json:"method"`
	Params interface{} `json:"params"`
	ID     int         `json:"id"`
}

type RPCResponse struct {
	Result interface{} `json:"result,omitempty"`
	Error  *RPCError   `json:"error,omitempty"`
	ID     int         `json:"id"`
}

type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func main() {
	http.HandleFunc("/rpc", handleRPC)
	fmt.Println("🚀 Custom RPC Server running on http://localhost:8080/rpc")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleRPC(w http.ResponseWriter, r *http.Request) {
	var req RPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, req.ID, "invalid request", err.Error())
		return
	}

	switch req.Method {
	case "add":
		// params expected as array [a, b]
		arr, ok := req.Params.([]interface{})
		if !ok || len(arr) < 2 {
			writeError(w, 400, req.ID, "invalid params for add", nil)
			return
		}
		a, _ := arr[0].(float64)
		b, _ := arr[1].(float64)
		writeResult(w, req.ID, a+b)

	case "getUser":
		// params expected as object {"userId": ...}
		m, ok := req.Params.(map[string]interface{})
		if !ok {
			writeError(w, 400, req.ID, "invalid params for getUser", nil)
			return
		}
		uidf, _ := m["userId"].(float64)
		uid := int(uidf)
		user := map[string]interface{}{
			"ID":   uid,
			"Name": "Alice",
			"Role": "Admin",
		}
		writeResult(w, req.ID, user)

	case "greet":
		m, ok := req.Params.(map[string]interface{})
		if !ok {
			writeError(w, 400, req.ID, "invalid params for greet", nil)
			return
		}
		name, _ := m["name"].(string)
		writeResult(w, req.ID, fmt.Sprintf("Hello, %s! 👋", name))

	default:
		writeError(w, 404, req.ID, "unknown method", nil)
	}
}

func writeResult(w http.ResponseWriter, id int, result interface{}) {
	w.Header().Set("Content-Type", "application/json")
	resp := RPCResponse{Result: result, ID: id}
	_ = json.NewEncoder(w).Encode(resp)
}

func writeError(w http.ResponseWriter, code, id int, msg string, data interface{}) {
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
