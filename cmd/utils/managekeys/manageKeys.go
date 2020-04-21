package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
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

const (
	maxProtocolVersion = 70002
)

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\nDMG : Key management tool")
	fmt.Println("-------------------------\n")

	fmt.Println("Do you want to add a new key or revoke an existing key:")
	fmt.Println("[1] Add new key")
	fmt.Println("[2] Revoke existing key")

	option := getLine(reader)
	var addOrRevoke string
	switch option {
	case "1":
		addOrRevoke = "add"
	case "2":
		addOrRevoke = "revoke"
	}

	fmt.Println("What type of key do you want to %s:", addOrRevoke)
	fmt.Println("[1] Issue key")
	fmt.Println("[2] Validate key")
	fmt.Println("[3] Provision key")
	fmt.Println("[4] ASP key")
	option2 := getLine(reader)

	var operation byte
	switch option + option2 {
	case "11":
		operation = txscript.AdminOpIssueKeyAdd
	case "12":
		operation = txscript.AdminOpValidateKeyAdd
	case "13":
		operation = txscript.AdminOpProvisionKeyAdd
	case "14":
		operation = txscript.AdminOpASPKeyAdd
	case "21":
		operation = txscript.AdminOpIssueKeyRevoke
	case "22":
		operation = txscript.AdminOpValidateKeyRevoke
	case "23":
		operation = txscript.AdminOpProvisionKeyRevoke
	case "24":
		operation = txscript.AdminOpASPKeyRevoke
	default:
		return
	}

	var threadAsString string = "root"
	threadID := (provautil.ThreadID)(0) // Root thread
	if operation == txscript.AdminOpValidateKeyAdd || operation == txscript.AdminOpASPKeyAdd || operation == txscript.AdminOpValidateKeyRevoke || operation == txscript.AdminOpASPKeyRevoke {
		threadID = (provautil.ThreadID)(1) // these actions require the provision thread
		threadAsString = "provision"
	}

	// Get the current issue thread tip
	fmt.Printf("Enter the %s thread tip hash:\n", threadAsString)
	issueThreadTip := getLine(reader)

	fmt.Printf("Enter %s thread tip index:\n", threadAsString)
	issueThreadTipIndex := getLine(reader)
	tipIndex64, _ := strconv.ParseUint(issueThreadTipIndex, 10, 32)
	tipIndex := uint32(tipIndex64)

	fmt.Printf("Enter public key to %s:\n", addOrRevoke)
	pubKey1String := getLine(reader)
	pubKey1Bytes, _ := hex.DecodeString(pubKey1String)
	pubKey1, _ := btcec.ParsePubKey(pubKey1Bytes, btcec.S256())

	// Need a keyID if we are adding or revoking an ASP key
	var keyID uint32
	if operation == txscript.AdminOpASPKeyAdd || operation == txscript.AdminOpASPKeyRevoke {
		fmt.Println("Enter the key ID:")
		keyIDString := getLine(reader)
		keyID64, _ := strconv.ParseInt(keyIDString, 10, 32)
		keyID = (uint32)(keyID64)
	}

	// Grab the keys
	fmt.Printf("Enter private key 1 (%s):\n", threadAsString)
	privKey1String := getLine(reader)
	privKey1Bytes, _ := hex.DecodeString(privKey1String)
	privKey1, _ := btcec.PrivKeyFromBytes(btcec.S256(), privKey1Bytes)

	fmt.Printf("Enter private key 2 (%s):\n", threadAsString)
	privKey2String := getLine(reader)
	privKey2Bytes, _ := hex.DecodeString(privKey2String)
	privKey2, _ := btcec.PrivKeyFromBytes(btcec.S256(), privKey2Bytes)

	lookupKey := func(a provautil.Address) ([]txscript.PrivateKey, error) {
		return []txscript.PrivateKey{
			txscript.PrivateKey{privKey1, true},
			txscript.PrivateKey{privKey2, true},
		}, nil
	}

	prevOutHash, _ := chainhash.NewHashFromStr(issueThreadTip)
	prevOut := wire.NewOutPoint(prevOutHash, tipIndex)

	keyAdminTransaction := createAdminTx(*prevOut, threadID, operation, pubKey1, keyID, lookupKey)

	// Convert to hex and output
	mtxHex := messageToHex(keyAdminTransaction)
	fmt.Println("---------------------")
	fmt.Printf(mtxHex)
}

// createAdminTx creates an admin transaction
func createAdminTx(outPoint wire.OutPoint, threadID provautil.ThreadID, op byte, pubKey *btcec.PublicKey, keyID uint32, lookupKey func(a provautil.Address) ([]txscript.PrivateKey, error)) *wire.MsgTx {
	spendTx := wire.NewMsgTx(1)
	spendTx.AddTxIn(&wire.TxIn{
		PreviousOutPoint: outPoint,
		Sequence:         wire.MaxTxInSequenceNum,
		SignatureScript:  nil,
	})
	txValue := int64(0) // how much the tx is spending. 0 for admin tx.
	spendTx.AddTxOut(wire.NewTxOut(txValue, provaThreadScript(threadID)))
	if op == txscript.AdminOpASPKeyAdd || op == txscript.AdminOpASPKeyRevoke {
		spendTx.AddTxOut(wire.NewTxOut(txValue, provaAdminScriptForASP(op, pubKey, keyID)))
	} else {
		spendTx.AddTxOut(wire.NewTxOut(txValue, provaAdminScript(op, pubKey)))
	}

	// Select the appropriate thread PK script
	var threadPkScript []byte
	if threadID == 1 {
		threadPkScript = []byte{
			0x00, 0xbb, // Root Thread Id, OP_CHECKTHREAD
		}
	} else {
		threadPkScript = []byte{
			0x51, 0xbb, // Provision Thread, OP_CHECKTHREAD
		}
	}

	// Sign the admin outpoint
	sigScript, _ := txscript.SignTxOutput(&chaincfg.TestNetParams, spendTx,
		0, 0, threadPkScript, txscript.SigHashAll, txscript.KeyClosure(lookupKey), nil)

	// Attach the signature to the transaction
	spendTx.TxIn[0].SignatureScript = sigScript

	return spendTx
}

// provaAdminScript creates a new script that executes an admin op.
func provaAdminScript(opcode byte, pubKey *btcec.PublicKey) []byte {
	// size as: <operation (1 byte)> <compressed public key (33 bytes)>>
	data := make([]byte, 1+btcec.PubKeyBytesLenCompressed)
	data[0] = opcode
	copy(data[1:], pubKey.SerializeCompressed())

	builder := txscript.NewScriptBuilder()
	script, err := builder.
		AddOp(txscript.OP_RETURN).
		AddData(data).Script()

	if err != nil {
		panic(err)
	}
	return script
}

// provaAdminScript creates a new script that executes an admin op.
func provaAdminScriptForASP(opcode byte, pubKey *btcec.PublicKey, keyID uint32) []byte {
	// size as: <operation (1 byte)> <compressed public key (33 bytes)> <keyID (4 bytes)>>
	data := make([]byte, 5+btcec.PubKeyBytesLenCompressed)
	data[0] = opcode
	copy(data[1:], pubKey.SerializeCompressed())

	bs := make([]byte, 4)
	binary.LittleEndian.PutUint32(bs, keyID)
	copy(data[34:], bs)

	builder := txscript.NewScriptBuilder()
	script, err := builder.
		AddOp(txscript.OP_RETURN).
		AddData(data).Script()

	if err != nil {
		panic(err)
	}
	return script
}

func getLine(reader *bufio.Reader) string {
	line, _ := reader.ReadString('\n')
	line = strings.TrimSuffix(line, "\n")
	return line
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
