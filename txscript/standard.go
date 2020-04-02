// Copyright (c) 2013-2017 The btcsuite developers
// Copyright (c) 2017 BitGo
// Copyright (c) 2019 Tranquility Node Ltd
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package txscript

import (
	"fmt"

	"github.com/pyx-partners/dmgd/btcec"
	"github.com/pyx-partners/dmgd/chaincfg"
	"github.com/pyx-partners/dmgd/provautil"
	"github.com/pyx-partners/dmgd/wire"
)

const (
	// MaxDataCarrierSize is the maximum number of bytes allowed in pushed
	// data to be considered a nulldata transaction
	MaxDataCarrierSize = 80

	// StandardVerifyFlags are the script flags which are used when
	// executing transaction scripts to enforce additional checks which
	// are required for the script to be considered standard.  These checks
	// help reduce issues related to transaction malleability as well as
	// allow pay-to-script hash transactions.  Note these flags are
	// different than what is required for the consensus rules in that they
	// are more strict.
	//
	// TODO: This definition does not belong here.  It belongs in a policy
	// package.
	StandardVerifyFlags = ScriptBip16 |
		ScriptVerifyDERSignatures |
		ScriptVerifyStrictEncoding |
		ScriptVerifyMinimalData |
		ScriptStrictMultiSig |
		ScriptDiscourageUpgradableNops |
		ScriptVerifyCleanStack |
		ScriptVerifyNullFail |
		ScriptVerifyCheckLockTimeVerify |
		ScriptVerifyCheckSequenceVerify |
		ScriptVerifyLowS
)

// ScriptClass is an enumeration for the list of standard types of script.
type ScriptClass byte

// Classes of script payment known about in the blockchain.
const (
	NonStandardTy  ScriptClass = iota // None of the recognized forms.
	PubKeyTy                          // Pay pubkey.
	PubKeyHashTy                      // Pay pubkey hash.
	ScriptHashTy                      // Pay to script hash.
	MultiSigTy                        // Multi signature.
	NullDataTy                        // Empty data-only (provably prunable).
	ProvaTy                           // Prova standard 2-of-3 type (subset of GeneralProvaTy)
	GeneralProvaTy                    // Prova (generalized m-of-n) script
	ProvaAdminTy                      // Prova Admin Operations
)

// scriptClassToName houses the human-readable strings which describe each
// script class.
var scriptClassToName = []string{
	// TODO(prova): clean up non-used types
	NonStandardTy:  "nonstandard",
	NullDataTy:     "nulldata",
	ProvaTy:        "safe_multisig",
	GeneralProvaTy: "safe_multisig",
	ProvaAdminTy:   "admin",
}

// String implements the Stringer interface by returning the name of
// the enum script class. If the enum is invalid then "Invalid" will be
// returned.
func (t ScriptClass) String() string {
	if int(t) > len(scriptClassToName) || int(t) < 0 {
		return "Invalid"
	}
	return scriptClassToName[t]
}

// isGeneralProva returns true if the passed script is an Prova script (generalized m-of-n)
func isGeneralProva(pops []parsedOpcode) bool {
	// The absolute minimum is 3 keys:
	// OP_2 <pkhash> <keyid> <keyid> OP_3 OP_CHECKMULTISIG
	sLen := len(pops)
	if sLen < 6 {
		return false
	}
	if !isSmallInt(pops[0].opcode) {
		return false
	}
	if !isSmallInt(pops[sLen-2].opcode) {
		return false
	}
	if pops[sLen-1].opcode.value != OP_CHECKSAFEMULTISIG {
		return false
	}

	// Get the number of sigs and keys for further validation
	nSigs := asSmallInt(pops[0].opcode)
	nKeys := asSmallInt(pops[sLen-2].opcode)

	// No effective single-sig allowed
	if nSigs < 2 {
		return false
	}

	// Verify the number of pubkeys specified matches the actual number
	// of pubkeys provided.
	if sLen-2-1 != nKeys {
		return false
	}

	// Run through the key ids and key hashes, counting each
	// Also, require that key ids come first, followed by key hashes.
	nKeyIDs := 0
	nKeyHashes := 0
	seenKeyIDs := make(map[int32]bool)
	for _, pop := range pops[1 : sLen-2] {
		dataLen := len(pop.data)
		// Valid pubkey hashes are 20 bytes
		if dataLen == 20 {
			// Key hashes MUST come before any key ids
			if nKeyIDs > 0 {
				return false
			}
			nKeyHashes++
		} else if isUint32(pop.opcode) {
			// Otherwise, it should look like a KeyID
			keyID, err := asInt32(pop)
			if err != nil {
				return false
			}
			_, seen := seenKeyIDs[keyID]
			if seen {
				// Duplicate key ids not allowed
				return false
			}
			seenKeyIDs[keyID] = true
			nKeyIDs++
		}
	}

	// Cannot allow raw key hashes to move funds without at least 1 KeyID
	if nKeyHashes >= nSigs {
		return false
	}

	// All key ids should be able to move funds in collaboration
	if nKeyIDs < nSigs {
		return false
	}

	return true
}

// isProva returns true if the passed script is a 2 of 3 Prova transaction.
func isProva(pops []parsedOpcode) bool {
	if len(pops) < 6 {
		return false
	}
	if !isGeneralProva(pops) {
		return false
	}

	// Handle standard common case
	if len(pops) == 6 &&
		pops[0].opcode.value == OP_2 &&
		pops[4].opcode.value == OP_3 {
		return true
	}

	// Handle less common case of n-1 of n (with n-1 keyids)
	// Errors can be ignored since already checked by isGeneralProva
	m, _ := asInt32(pops[0])
	n, _ := asInt32(pops[len(pops)-2])
	return m == n-1
}

// IsProvaTx determines if a transaction is a standard prova transaction
// consisting of only outputs to standard prova scripts and 0-value nulldata
// scripts.
func IsProvaTx(tx *provautil.Tx) bool {
	msgTx := tx.MsgTx()

	// A Prova transaction must have at least one output.
	if len(msgTx.TxOut) == 0 {
		return false
	}

	hasNullOutput := false
	for _, txOut := range msgTx.TxOut {
		atoms := txOut.Value
		pops, err := ParseScript(txOut.PkScript)
		if err != nil {
			return false
		}
		// NullData outputs are allowed, but only one, with zero value.
		if isNullData(pops) {
			if hasNullOutput {
				return false
			}
			hasNullOutput = true
			if atoms != 0 {
				return false
			}
		} else if !isGeneralProva(pops) {
			return false
		}
	}
	return true
}

// GetAdminDetails will read threadID and admin outputs from an admin transaction.
func GetAdminDetailsMsgTx(msgTx *wire.MsgTx) (int, [][]parsedOpcode) {
	// The first output of the admin transaction is the thread transaction.
	// Additional outputs modify the chain state. We expect at least one additional.
	if len(msgTx.TxOut) < 1 {
		return -1, nil
	}
	pops, err := ParseScript(msgTx.TxOut[0].PkScript)
	if err != nil {
		return -1, nil
	}
	if TypeOfScript(pops) != ProvaAdminTy {
		return -1, nil
	}
	threadID, err := ExtractThreadID(pops)
	if err != nil {
		return -1, nil
	}
	adminOutputs := make([][]parsedOpcode, len(msgTx.TxOut)-1)
	for i := 1; i < len(msgTx.TxOut); i++ {
		pops, err := ParseScript(msgTx.TxOut[i].PkScript)
		if err != nil {
			return -1, nil
		}
		adminOutputs[i-1] = pops
	}
	return int(threadID), adminOutputs
}

// GetAdminDetails will read threadID and admin outputs from an admin transaction.
func GetAdminDetails(tx *provautil.Tx) (int, [][]parsedOpcode) {
	return GetAdminDetailsMsgTx(tx.MsgTx())
}

// isProvaAdmin returns true if the passed script is admin thread script.
func isProvaAdmin(pops []parsedOpcode) bool {
	sLen := len(pops)
	if sLen != 2 {
		return false
	}
	if pops[sLen-1].opcode.value != OP_CHECKTHREAD {
		return false
	}
	threadID := provautil.ThreadID(asSmallInt(pops[0].opcode))
	if threadID < provautil.RootThread || threadID > provautil.IssueThread {
		return false
	}
	return true
}

// IsValidAdminOp returns true if the passed script is a valid admin
// operation at the given thread.
func IsValidAdminOp(pops []parsedOpcode, threadID provautil.ThreadID) bool {
	// always expect two ops
	// <OP_RETURN><OP_DATA>
	if len(pops) != 2 {
		return false
	}
	if pops[0].opcode.value != OP_RETURN {
		return false
	}
	if pops[1].opcode.value != OP_DATA_34 &&
		pops[1].opcode.value != OP_DATA_38 {
		return false
	}
	// read op type byte and check valid key
	op, _, err := ExtractAdminData(pops)
	if err != nil {
		return false
	}
	// check thread specific operations
	switch threadID {
	case provautil.RootThread:
		if op == AdminOpProvisionKeyAdd ||
			op == AdminOpProvisionKeyRevoke ||
			op == AdminOpIssueKeyAdd ||
			op == AdminOpIssueKeyRevoke {
			return true
		}
	case provautil.ProvisionThread:
		if op == AdminOpValidateKeyAdd ||
			op == AdminOpValidateKeyRevoke {
			return true
		}
		if op == AdminOpASPKeyAdd ||
			op == AdminOpASPKeyRevoke {
			// check length of data for ASP ops
			if len(pops[1].data) == 1+btcec.PubKeyBytesLenCompressed+btcec.KeyIDSize {
				return true
			}
		}
	case provautil.IssueThread:
		return false
	}
	return false
}

// isNullData returns true if the passed script is a null data transaction,
// false otherwise.
func isNullData(pops []parsedOpcode) bool {
	// A nulldata transaction is either a single OP_RETURN or an
	// OP_RETURN SMALLDATA (where SMALLDATA is a data push up to
	// MaxDataCarrierSize bytes).
	l := len(pops)
	if l == 1 && pops[0].opcode.value == OP_RETURN {
		return true
	}

	return l == 2 &&
		pops[0].opcode.value == OP_RETURN &&
		(isSmallInt(pops[1].opcode) || pops[1].opcode.value <=
			OP_PUSHDATA4) &&
		len(pops[1].data) <= MaxDataCarrierSize
}

// TypeOfScript returns the type of the script being inspected from the known
// standard types.
func TypeOfScript(pops []parsedOpcode) ScriptClass {
	return typeOfScript(pops)
}

// typeOfScript returns the type of the script being inspected from the known
// standard types.
func typeOfScript(pops []parsedOpcode) ScriptClass {
	if isNullData(pops) {
		return NullDataTy
	} else if isProva(pops) {
		return ProvaTy
	} else if isGeneralProva(pops) {
		return GeneralProvaTy
	} else if isProvaAdmin(pops) {
		return ProvaAdminTy
	}
	return NonStandardTy
}

// GetScriptClass returns the class of the script passed.
//
// NonStandardTy will be returned when the script does not parse.
func GetScriptClass(script []byte) ScriptClass {
	pops, err := ParseScript(script)
	if err != nil {
		return NonStandardTy
	}
	return typeOfScript(pops)
}

// ScriptInfo houses information about a script pair that is determined by
// CalcScriptInfo.
type ScriptInfo struct {
	// PkScriptClass is the class of the public key script and is equivalent
	// to calling GetScriptClass on it.
	PkScriptClass ScriptClass

	// NumInputs is the number of inputs provided by the public key script.
	NumInputs int

	// ExpectedInputs is the number of outputs required by the signature
	// script and any pay-to-script-hash scripts. The number will be -1 if
	// unknown.
	ExpectedInputs int

	// SigOps is the number of signature operations in the script pair.
	SigOps int
}

// CalcMultiSigStats returns the number of public keys and signatures from
// a multi-signature transaction script.  The passed script MUST already be
// known to be a multi-signature script.
func CalcMultiSigStats(script []byte) (int, int, error) {
	pops, err := ParseScript(script)
	if err != nil {
		return 0, 0, err
	}

	// A multi-signature script is of the pattern:
	//  NUM_SIGS PUBKEY PUBKEY PUBKEY... NUM_PUBKEYS OP_CHECKMULTISIG
	// Therefore the number of signatures is the oldest item on the stack
	// and the number of pubkeys is the 2nd to last.  Also, the absolute
	// minimum for a multi-signature script is 1 pubkey, so at least 4
	// items must be on the stack per:
	//  OP_1 PUBKEY OP_1 OP_CHECKMULTISIG
	if len(pops) < 4 {
		str := fmt.Sprintf("script %x is not a multisig script", script)
		return 0, 0, scriptError(ErrNotMultisigScript, str)
	}

	numSigs := asSmallInt(pops[0].opcode)
	numPubKeys := asSmallInt(pops[len(pops)-2].opcode)
	return numPubKeys, numSigs, nil
}

// payToProvaScript creates a new script to pay a transaction output to an
// Prova 2-of-3 address.
func payToProvaScript(pubKeyHash []byte, keyIDs []btcec.KeyID) ([]byte, error) {
	if len(keyIDs) != 2 {
		return nil, scriptError(ErrInvalidNumberOfKeyIds, "prova script must have 2 key ids")
	}
	return NewScriptBuilder().
		AddInt64(int64(len(keyIDs))).
		AddData(pubKeyHash).
		AddInt64(int64(keyIDs[0])).
		AddInt64(int64(keyIDs[1])).
		AddInt64(int64(len(keyIDs) + 1)).
		AddOp(OP_CHECKSAFEMULTISIG).
		Script()
}

// PayToAddrScript creates a new script to pay a transaction output to a the
// specified address.
func PayToAddrScript(addr provautil.Address) ([]byte, error) {
	switch addr := addr.(type) {
	case *provautil.AddressProva:
		if addr == nil {
			return nil, scriptError(ErrUnsupportedAddress, "address is nil")
		}
		return payToProvaScript(addr.ScriptAddress(), addr.ScriptKeyIDs())
	}

	return nil, scriptError(ErrUnsupportedAddress, "unsupported address type")
}

// ProvaThreadScript creates a new script to pay a transaction output to an
// Prova Admin Thread.
func ProvaThreadScript(threadID provautil.ThreadID) ([]byte, error) {
	return NewScriptBuilder().
		AddInt64(int64(threadID)).
		AddOp(OP_CHECKTHREAD).Script()
}

// NullDataScript creates a provably-prunable script containing OP_RETURN
// followed by the passed data.  An Error with the error code ErrTooMuchNullData
// will be returned if the length of the passed data exceeds MaxDataCarrierSize.
func NullDataScript(data []byte) ([]byte, error) {
	if len(data) > MaxDataCarrierSize {
		str := fmt.Sprintf("data size %d is larger than max "+
			"allowed size %d", len(data), MaxDataCarrierSize)
		return nil, scriptError(ErrTooMuchNullData, str)
	}

	return NewScriptBuilder().AddOp(OP_RETURN).AddData(data).Script()
}

// MultiSigScript returns a valid script for a multisignature redemption where
// nrequired of the keys in pubkeys are required to have signed the transaction
// for success.  An ErrBadNumRequired will be returned if nrequired is larger
// than the number of keys provided.
func MultiSigScript(pubkeys []*provautil.AddressPubKey, nrequired int) ([]byte, error) {
	if len(pubkeys) < nrequired {
		str := fmt.Sprintf("unable to generate multisig script with "+
			"%d required signatures when there are only %d public "+
			"keys available", nrequired, len(pubkeys))
		return nil, scriptError(ErrTooManyRequiredSigs, str)
	}

	builder := NewScriptBuilder().AddInt64(int64(nrequired))
	for _, key := range pubkeys {
		builder.AddData(key.ScriptAddress())
	}
	builder.AddInt64(int64(len(pubkeys)))
	builder.AddOp(OP_CHECKMULTISIG)

	return builder.Script()
}

// PushedData returns an array of byte slices containing any pushed data found
// in the passed script.  This includes OP_0, but not OP_1 - OP_16.
func PushedData(script []byte) ([][]byte, error) {
	pops, err := ParseScript(script)
	if err != nil {
		return nil, err
	}

	var data [][]byte
	for _, pop := range pops {
		if pop.data != nil {
			data = append(data, pop.data)
		} else if pop.opcode.value == OP_0 {
			data = append(data, nil)
		}
	}
	return data, nil
}

// ExtractPkScriptAddrs returns the type of script, addresses and required
// signatures associated with the passed PkScript.  Note that it only works for
// 'standard' transaction script types.  Any data such as public keys which are
// invalid are omitted from the results.
func ExtractPkScriptAddrs(pkScript []byte, chainParams *chaincfg.Params) (ScriptClass, []provautil.Address, int, error) {
	var addrs []provautil.Address
	var requiredSigs int

	// No valid addresses or required signatures if the script doesn't
	// parse.
	pops, err := ParseScript(pkScript)
	if err != nil {
		return NonStandardTy, nil, 0, err
	}

	scriptClass := typeOfScript(pops)
	switch scriptClass {

	case ProvaTy:
		numKeyIDs := len(pops) - 4
		requiredSigs = numKeyIDs
		keyIDError := false
		keyIDs := []btcec.KeyID{}
		for i := 0; i < numKeyIDs; i++ {
			keyID, err := asInt32(pops[2+i])
			if err != nil {
				keyIDError = true
			}
			keyIDs = append(keyIDs, btcec.KeyID(keyID))
		}
		addr, err := provautil.NewAddressProva(pops[1].data, keyIDs, chainParams)
		if err == nil && !keyIDError {
			addrs = append(addrs, addr)
		}

	case GeneralProvaTy:
		// TODO(prova): define what to do for generalized prova scripts

	case ProvaAdminTy:
		requiredSigs = 2

	case NullDataTy:
		// Null data transactions have no addresses or required
		// signatures.

	case NonStandardTy:
		// Don't attempt to extract addresses or required signatures for
		// nonstandard transactions.
	}

	return scriptClass, addrs, requiredSigs, nil
}
