package main

import (
	"fmt"
	"test/internal/server"
)

func main() {
	server.TxInit()

	fmt.Println("[ Tx service Loaded ]")
}
