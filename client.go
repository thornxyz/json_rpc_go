package main

import (
	"context"
	"fmt"
	"log"

	"my_rpc/jsonrpc"
)

func main() {
	client := jsonrpc.NewClient("http://localhost:8080/rpc")
	ctx := context.Background()

	// Call "add"
	addResp, err := client.Call(ctx, "add", 5, 3)
	if err != nil {
		log.Fatal("Error calling add:", err)
	}
	addResult, _ := addResp.GetFloat()
	fmt.Println("✅ add(5,3) =", addResult)

	// Call "getUser"
	var user struct {
		ID   int
		Name string
		Role string
	}
	err = client.CallFor(ctx, &user, "getUser", map[string]interface{}{"userId": 101})
	if err != nil {
		log.Fatal("Error calling getUser:", err)
	}
	fmt.Printf("✅ getUser: %+v\n", user)

	// Call "greet"
	var greet string
	err = client.CallFor(ctx, &greet, "greet", map[string]interface{}{"name": "Subhrajyoti"})
	if err != nil {
		log.Fatal("Error calling greet:", err)
	}
	fmt.Println("✅ greet:", greet)

	// Try calling an unknown method
	_, err = client.Call(ctx, "unknownMethod")
	if err != nil {
		fmt.Println("❌ Error:", err)
	}
}
