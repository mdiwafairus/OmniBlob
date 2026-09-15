package main

import (
	"crypto/ed25519"
	"fmt"
)

func main() {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Private Key (hex): %x\n", priv)
	fmt.Printf("Public Key (hex): %x\n", pub)
}
