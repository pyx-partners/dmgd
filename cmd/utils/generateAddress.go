package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pyx-partners/dmgd/btcec"
	"github.com/pyx-partners/dmgd/chaincfg"
	"github.com/pyx-partners/dmgd/provautil"
)

func main() {
	reader := bufio.NewReader(os.Stdin)

	// Get the primary and backup ASP key IDs
	fmt.Println("Enter the key ID of the primary ASP:")
	primaryKeyIDAsString := getLine(reader)
	keyID1_64, _ := strconv.ParseUint(primaryKeyIDAsString, 10, 32)
	keyID1 := btcec.KeyID((uint32)(keyID1_64))

	fmt.Println("Enter the key ID of the backup ASP:")
	backupKeyIDAsString := getLine(reader)
	keyID2_64, _ := strconv.ParseUint(backupKeyIDAsString, 10, 32)
	keyID2 := btcec.KeyID((uint32)(keyID2_64))

	// Create a random new private key
	privateKey, _ := btcec.NewPrivateKey(btcec.S256())

	publicKey := (*btcec.PublicKey)(&privateKey.PublicKey)
	pkHash := provautil.Hash160(publicKey.SerializeCompressed())

	// Generate an address from the new key
	payAddr, _ := provautil.NewAddressProva(pkHash, []btcec.KeyID{keyID1, keyID2}, &chaincfg.TestNetParams)

	fmt.Println("Address:")
	fmt.Println(payAddr)

	fmt.Println("Private key:")
	fmt.Printf("%x\n", privateKey.Serialize())
	fmt.Println("Public key:")
	fmt.Printf("%x\n", publicKey.SerializeCompressed())
}

func getLine(reader *bufio.Reader) string {
	line, _ := reader.ReadString('\n')
	line = strings.TrimSuffix(line, "\n")
	return line
}
