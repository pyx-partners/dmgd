package main

import (
	"encoding/hex"
	"fmt"
	"math"
	"os"

	"github.com/pyx-partners/dmgd/btcec"
)

func main() {

	privKeyStr := os.Args[1]
	privKeyBytes, _ := hex.DecodeString(privKeyStr)
	privKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), privKeyBytes)

	fmt.Printf("Compressed: %x\n", privKey.PubKey().SerializeCompressed())
	fmt.Printf("Uncompressed: %x\n", privKey.PubKey().SerializeUncompressed())

	fmt.Print("Private key (code):\n")
	for i, b := range privKeyBytes {
		if math.Mod(float64(i), 8) == 0 {
			fmt.Printf("\n")
		}
		fmt.Printf("0x%02x, ", b)
	}
}
