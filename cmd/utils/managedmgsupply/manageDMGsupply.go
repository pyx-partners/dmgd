package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pyx-partners/dmgd/btcec"
	"github.com/pyx-partners/dmgd/chaincfg"
	"github.com/pyx-partners/dmgd/chaincfg/chainhash"
	"github.com/pyx-partners/dmgd/provautil"
	"github.com/pyx-partners/dmgd/txscript"
	"github.com/pyx-partners/dmgd/wire"
)

var (
	keyId1 = btcec.KeyID(1)
	keyId2 = btcec.KeyID(2)
)

const (
	maxProtocolVersion = 70002
)

func main() {

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\nDMG : Issue tool")
	fmt.Println("----------------\n")

	fmt.Println("[1] Issue DMG")
	fmt.Println("[2] Destroy DMG")

	option := getLine(reader)

	switch option {
	case "1":
		issueDMG()
	case "2":
		destroyDMG()
	default:
		fmt.Println("Unknown option. Please enter either 1 or 2.")
		return
	}
}

func issueDMG() {
	reader := bufio.NewReader(os.Stdin)

	// Get the target address to issue to
	fmt.Println("Enter the target DMG address into which DMG will be issued:")
	targetAddressString, _ := reader.ReadString('\n')
	targetAddressString = strings.TrimSuffix(targetAddressString, "\n")

	payAddr, err := provautil.DecodeAddress(targetAddressString, &chaincfg.TestNetParams)

	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	// Get the amount
	fmt.Println("Enter the number of DMG that should be issued:")
	amountAsString := getLine(reader)
	amount, _ := strconv.ParseInt(amountAsString, 10, 64)
	amountInAtoms := amount * 1000000

	// Get the current issue thread tip
	fmt.Println("Enter the issue thread tip hash:")
	issueThreadTip := getLine(reader)

	fmt.Println("Enter issue thread tip index:")
	issueThreadTipIndex := getLine(reader)
	tipIndex64, _ := strconv.ParseUint(issueThreadTipIndex, 10, 32)
	tipIndex := uint32(tipIndex64)

	// Grab the keys
	fmt.Println("Enter private key 1:")
	privKey1String := getLine(reader)
	privKey1Bytes, _ := hex.DecodeString(privKey1String)
	privKey1, _ := btcec.PrivKeyFromBytes(btcec.S256(), privKey1Bytes)

	fmt.Println("Enter private key 2:")
	privKey2String := getLine(reader)
	privKey2Bytes, _ := hex.DecodeString(privKey2String)
	privKey2, _ := btcec.PrivKeyFromBytes(btcec.S256(), privKey2Bytes)

	lookupKey := func(a provautil.Address) ([]txscript.PrivateKey, error) {
		return []txscript.PrivateKey{
			txscript.PrivateKey{privKey1, true},
			txscript.PrivateKey{privKey2, true},
		}, nil
	}

	// Create the issue transaction
	fmt.Println("Starting the issue process\n")
	fmt.Printf("Issuing to this address: %s \n", payAddr)

	prevOutHash, _ := chainhash.NewHashFromStr(issueThreadTip)
	prevOut := wire.NewOutPoint(prevOutHash, tipIndex)

	issueTx := createIssueTx(amountInAtoms, 0, *prevOut, *prevOut, payAddr, lookupKey, nil)

	fmt.Println("---------------------")

	mtxHex := messageToHex(issueTx)
	fmt.Printf(mtxHex)
}

func destroyDMG() {
	reader := bufio.NewReader(os.Stdin)

	// Get the target address to issue to
	fmt.Println("Enter the target DMG address which holds the DMG to be destroyed:")
	targetAddressString, _ := reader.ReadString('\n')
	targetAddressString = strings.TrimSuffix(targetAddressString, "\n")

	payAddr, err := provautil.DecodeAddress(targetAddressString, &chaincfg.TestNetParams)

	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	// Get the amount
	fmt.Println("Enter the number of DMG that should be destroyed:")
	amountAsString := getLine(reader)
	amount, _ := strconv.ParseInt(amountAsString, 10, 64)
	amountInAtoms := amount * 1000000

	// Get the current issue thread tip
	fmt.Println("Enter the issue thread tip hash:")
	issueThreadTip := getLine(reader)

	fmt.Println("Enter issue thread tip index:")
	issueThreadTipIndex := getLine(reader)
	tipIndex64, _ := strconv.ParseUint(issueThreadTipIndex, 10, 32)
	tipIndex := uint32(tipIndex64)

	// Get the coins to destroy
	fmt.Println("Enter the hash for the transaction that the coins currently exist in:")
	txHash := getLine(reader)

	fmt.Println("Enter transaction index:")
	txIndexString := getLine(reader)
	txIndex64, _ := strconv.ParseUint(txIndexString, 10, 32)
	txIndex := uint32(txIndex64)

	// Grab the keys
	fmt.Println("Enter private key 1 (ISSUE KEY):")
	privKey1String := getLine(reader)
	privKey1Bytes, _ := hex.DecodeString(privKey1String)
	privKey1, _ := btcec.PrivKeyFromBytes(btcec.S256(), privKey1Bytes)

	fmt.Println("Enter private key 2 (ISSUE KEY):")
	privKey2String := getLine(reader)
	privKey2Bytes, _ := hex.DecodeString(privKey2String)
	privKey2, _ := btcec.PrivKeyFromBytes(btcec.S256(), privKey2Bytes)

	fmt.Println("Enter private key 3 (ASP KEY):")
	privKey3String := getLine(reader)
	privKey3Bytes, _ := hex.DecodeString(privKey3String)
	privKey3, _ := btcec.PrivKeyFromBytes(btcec.S256(), privKey3Bytes)

	fmt.Println("Enter private key 4 (Account key):")
	privKey4String := getLine(reader)
	privKey4Bytes, _ := hex.DecodeString(privKey4String)
	privKey4, _ := btcec.PrivKeyFromBytes(btcec.S256(), privKey4Bytes)

	lookupKey := func(a provautil.Address) ([]txscript.PrivateKey, error) {
		return []txscript.PrivateKey{
			txscript.PrivateKey{privKey1, true},
			txscript.PrivateKey{privKey2, true},
		}, nil
	}

	lookupKey2 := func(a provautil.Address) ([]txscript.PrivateKey, error) {
		return []txscript.PrivateKey{
			txscript.PrivateKey{privKey3, true},
			txscript.PrivateKey{privKey4, true},
		}, nil
	}

	// Create the destroy transaction
	fmt.Println("Starting the destroy process\n")

	prevOutHash, _ := chainhash.NewHashFromStr(issueThreadTip)
	prevOut := wire.NewOutPoint(prevOutHash, tipIndex)

	coinsToRevokeHash, _ := chainhash.NewHashFromStr(txHash)
	coinsToRevoke := wire.NewOutPoint(coinsToRevokeHash, txIndex)

	issueTx := createIssueTx(0, amountInAtoms, *prevOut, *coinsToRevoke, payAddr, lookupKey, lookupKey2)

	fmt.Println("---------------------")

	mtxHex := messageToHex(issueTx)
	fmt.Printf(mtxHex)

}

// createIssueTx creates an issue thread admin tx.
// If a spend output is passed, a revoke transaction is build.
// if spend is nil, new tokens of amount in value are issued.
func createIssueTx(value int64, revokeValue int64, previousOutpoint wire.OutPoint, coinsToRevoke wire.OutPoint, payToAddr provautil.Address, lookupKey func(a provautil.Address) ([]txscript.PrivateKey, error), lookupKey2 func(a provautil.Address) ([]txscript.PrivateKey, error)) *wire.MsgTx {
	spendTx := wire.NewMsgTx(1)
	// thread input
	spendTx.AddTxIn(&wire.TxIn{
		PreviousOutPoint: previousOutpoint,
		Sequence:         wire.MaxTxInSequenceNum,
		SignatureScript:  nil,
	})
	// thread output
	spendTx.AddTxOut(wire.NewTxOut(int64(0), provaThreadScript(provautil.IssueThread)))
	if revokeValue == 0 {
		scriptPkScript, _ := txscript.PayToAddrScript(payToAddr)
		spendTx.AddTxOut(wire.NewTxOut(value, scriptPkScript))
	} else {
		// destroy some tokens:
		// - spend output of amount x
		// - bind amount x in opReturn output
		//coinsToRevoke := wire.NewOutPoint(&previousOutpoint.Hash, 1)
		spendTx.AddTxIn(&wire.TxIn{
			PreviousOutPoint: coinsToRevoke,
			Sequence:         wire.MaxTxInSequenceNum,
			SignatureScript:  nil,
		})
		spendTx.AddTxOut(wire.NewTxOut(
			revokeValue,
			opReturnScript(),
		))
	}

	issueThreadpkScript := []byte{
		0x52, 0xbb, // Issue Thread, OP_CHECKTHREAD
	}

	// Sign the first input (this is on both issue and revoke transactions)
	sigScript, _ := txscript.SignTxOutput(&chaincfg.TestNetParams, spendTx,
		0, 0, issueThreadpkScript, txscript.SigHashAll, txscript.KeyClosure(lookupKey), nil)

	spendTx.TxIn[0].SignatureScript = sigScript

	if revokeValue != 0 {
		// tx output script
		scriptPkScript2, _ := txscript.PayToAddrScript(payToAddr)

		// sign the second input (only on revoke transactions)
		sigScript2, _ := txscript.SignTxOutput(&chaincfg.TestNetParams, spendTx,
			1, revokeValue, scriptPkScript2, txscript.SigHashAll, txscript.KeyClosure(lookupKey2), nil)

		spendTx.TxIn[1].SignatureScript = sigScript2
	}
	return spendTx
}

// spendableOut represents a transaction output that is spendable along with
// additional metadata such as the block its in and how much it pays.
type spendableOut struct {
	prevOut  wire.OutPoint
	pkScript []byte
	amount   provautil.Amount
}

// provaThreadScript creates a new script to pay a transaction output to an
// Prova Admin Thread.
func provaThreadScript(threadID provautil.ThreadID) []byte {
	builder := txscript.NewScriptBuilder()
	script, err := builder.
		AddInt64(int64(threadID)).
		AddOp(txscript.OP_CHECKTHREAD).Script()
	if err != nil {
		panic(err)
	}
	return script
}

// opReturnScript creates an op_return pkScript.
func opReturnScript() []byte {
	return []byte{txscript.OP_RETURN}
}

// messageToHex serializes a message to the wire protocol encoding using the
// latest protocol version and returns a hex-encoded string of the result.
func messageToHex(msg wire.Message) string {
	var buf bytes.Buffer
	if err := msg.BtcEncode(&buf, maxProtocolVersion); err != nil {
		return ""
	}

	return hex.EncodeToString(buf.Bytes())
}

func getLine(reader *bufio.Reader) string {
	line, _ := reader.ReadString('\n')
	line = strings.TrimSuffix(line, "\n")
	return line
}
