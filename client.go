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

	// add
	addResp, err := client.Call(ctx, "add", 5, 3)
	if err != nil {
		log.Fatal("Error calling add:", err)
	}
	addResult, _ := addResp.GetFloat()
	fmt.Println("✅ add(5,3) =", addResult)

	// getUser
	var user struct {
		ID   int
		Name string
		Role string
	}
	if err := client.CallFor(ctx, &user, "getUser", map[string]interface{}{"userId": 101}); err != nil {
		log.Fatal("Error calling getUser:", err)
	}
	fmt.Printf("✅ getUser: %+v\n", user)

	// greet
	var greet string
	if err := client.CallFor(ctx, &greet, "greet", map[string]interface{}{"name": "Subhrajyoti"}); err != nil {
		log.Fatal("Error calling greet:", err)
	}
	fmt.Println("✅ greet:", greet)

	// unknown method -> Call now returns RPC error as Go error
	_, err = client.Call(ctx, "unknownMethod")
	if err != nil {
		fmt.Println("❌ Error:", err)
	}
}
