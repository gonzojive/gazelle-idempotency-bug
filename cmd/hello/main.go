package main

import (
	"fmt"

	example "github.com/gonzojive/gazelle-idempotency-bug/proto/example"
)

func main() {
	fmt.Println("Hello, world!")
	_ = example.DoSomethingRequest{} // Use the imported type to avoid compiler error.
}
