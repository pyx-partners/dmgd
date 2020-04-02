// Copyright (c) 2013-2016 The btcsuite developers
// Copyright (c) 2017 BitGo
// Copyright (c) 2019 Tranquility Node Ltd
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package mempool

import (
	"fmt"
	"time"

	"github.com/pyx-partners/dmgd/blockchain"
	"github.com/pyx-partners/dmgd/provautil"
	"github.com/pyx-partners/dmgd/txscript"
	"github.com/pyx-partners/dmgd/wire"
)

const (
	// maxStandardP2SHSigOps is the maximum number of signature operations
	// that are considered standard in a pay-to-script-hash script.
	maxStandardP2SHSigOps = 15

	// MaxStandardTxSize is the maximum size allowed for transactions that
	// are considered standard and will therefore be relayed and considered
	// for mining.
	MaxStandardTxSize = 100000

	// maxStandardSigScriptSize is the maximum size allowed for a
	// transaction input signature script to be considered standard.  This
	// value allows for a 15-of-15 CHECKMULTISIG pay-to-script-hash with
	// compressed keys.
	//
	// The form of the overall script is: OP_0 <15 signatures> OP_PUSHDATA2
	// <2 bytes len> [OP_15 <15 pubkeys> OP_15 OP_CHECKMULTISIG]
	//
	// For the p2sh script portion, each of the 15 compressed pubkeys are
	// 33 bytes (plus one for the OP_DATA_33 opcode), and the thus it totals
	// to (15*34)+3 = 513 bytes.  Next, each of the 15 signatures is a max
	// of 73 bytes (plus one for the OP_DATA_73 opcode).  Also, there is one
	// extra byte for the initial extra OP_0 push and 3 bytes for the
	// OP_PUSHDATA2 needed to specify the 513 bytes for the script push.
	// That brings the total to 1+(15*74)+3+513 = 1627.  This value also
	// adds a few extra bytes to provide a little buffer.
	// (1 + 15*74 + 3) + (15*34 + 3) + 23 = 1650
	maxStandardSigScriptSize = 1650

	// DefaultMinRelayTxFee is the minimum fee in atoms that is required
	// for a transaction to be treated as free for relay and mining
	// purposes.  It is also used to help determine if a transaction is
	// considered dust and as a base for calculating minimum required fees
	// for larger transactions.  This value is in Atoms/1000 bytes.
	DefaultMinRelayTxFee = provautil.Amount(0)
)

// calcMinRequiredTxRelayFee returns the minimum transaction fee required for a
// transaction with the passed serialized size to be accepted into the memory
// pool and relayed.
func calcMinRequiredTxRelayFee(serializedSize int64, minRelayTxFee provautil.Amount) int64 {
	// Calculate the minimum fee for a transaction to be allowed into the
	// mempool and relayed by scaling the base fee (which is the minimum
	// free transaction relay fee).  minTxRelayFee is in Atoms/kB so
	// multiply by serializedSize (which is in bytes) and divide by 1000 to
	// get minimum Atoms.
	minFee := (serializedSize * int64(minRelayTxFee)) / 1000

	if minFee == 0 && minRelayTxFee > 0 {
		minFee = int64(minRelayTxFee)
	}

	// Set the minimum fee to the maximum possible value if the calculated
	// fee is not in the valid range for monetary amounts.
	if minFee < 0 || minFee > provautil.MaxAtoms {
		minFee = provautil.MaxAtoms
	}

	return minFee
}

// checkInputsStandard performs a series of checks on a transaction's inputs
// to ensure they are "standard".  A standard transaction input within the
// context of this function is one whose referenced public key script is of a
// standard form.  However, it should also be noted
// that standard inputs also are those which have a clean stack after execution
// and only contain pushed data in their signature scripts.  This function does
// not perform those checks because the script engine already does this more
// accurately and concisely via the txscript.ScriptVerifyCleanStack and
// txscript.ScriptVerifySigPushOnly flags.
func checkInputsStandard(tx *provautil.Tx, utxoView *blockchain.UtxoViewpoint) error {
	// NOTE: The reference implementation also does a coinbase check here,
	// but coinbases have already been rejected prior to calling this
	// function so no need to recheck.

	threadInt, _ := txscript.GetAdminDetails(tx)
	hasAdminOut := (threadInt >= 0)
	hasAdminIn := false
	thisPkScript := tx.MsgTx().TxOut[0].PkScript
	for txInIndex, txIn := range tx.MsgTx().TxIn {
		// It is safe to elide existence and index checks here since
		// they have already been checked prior to calling this
		// function.
		prevOut := txIn.PreviousOutPoint
		entry := utxoView.LookupEntry(&prevOut.Hash)
		originPkScript := entry.PkScriptByIndex(prevOut.Index)
		switch txscript.GetScriptClass(originPkScript) {
		case txscript.ProvaTy:
			fallthrough
		case txscript.GeneralProvaTy:
			break
		case txscript.ProvaAdminTy:
			sigPops, err := txscript.ParseScript(txIn.SignatureScript)
			if err != nil {
				str := fmt.Sprintf("transaction input #%d has "+
					"error %v", err)
				return txRuleError(wire.RejectNonstandard, str)
			}
			// we expect pairs of <pub><sig><pub><sig>
			if len(sigPops)%2 != 0 {
				str := fmt.Sprintf("transaction input #%d has "+
					"odd amount of sigPops %d", txInIndex, len(sigPops))
				return txRuleError(wire.RejectNonstandard, str)
			}
			// check input position
			if txInIndex != 0 {
				str := fmt.Sprintf("transaction %v tried to spend admin "+
					"thread transaction %v with input at position "+
					"%d. Only input #0 may spend an admin threads.",
					tx.Hash(), prevOut.Hash, txInIndex)
				return txRuleError(wire.RejectInvalidAdmin, str)
			}
			if !hasAdminOut {
				str := fmt.Sprintf("transaction %v spends admin output, "+
					"yet does not continue admin thread. Should have admin "+
					"output at position 0.", tx.Hash())
				return txRuleError(wire.RejectInvalidAdmin, str)
			}
			hasAdminIn = true
			// check admin thread input is spend to same thread
			if thisPkScript[0] != originPkScript[0] ||
				thisPkScript[1] != originPkScript[1] {
				str := fmt.Sprintf("admin transaction input #%d is "+
					"spending wrong thread.", txInIndex)
				return txRuleError(wire.RejectInvalidAdmin, str)
			}
		case txscript.NonStandardTy:
			str := fmt.Sprintf("transaction input #%d has a "+
				"non-standard script form", txInIndex)
			return txRuleError(wire.RejectNonstandard, str)
		}

		// If current transaction has admin output, but doesn't spend
		// an admin thread, it is not valid
		if hasAdminOut && !hasAdminIn {
			str := fmt.Sprintf("tried to issue admin operation "+
				"at transaction %s:%d without spending valid thread ",
				tx.Hash(), txInIndex)
			return txRuleError(wire.RejectInvalidAdmin, str)
		}

	}

	return nil
}

// checkPkScriptStandard performs a series of checks on a transaction output
// script (public key script) to ensure it is a "standard" public key script.
// A standard public key script is one that is a recognized form.
func checkPkScriptStandard(pkScript []byte, scriptClass txscript.ScriptClass) error {
	switch scriptClass {
	case txscript.ProvaTy:
		fallthrough
	case txscript.GeneralProvaTy:
		break
	case txscript.ProvaAdminTy:
		// TODO(prova): apply validation rules here
		break
	case txscript.NonStandardTy:
		return txRuleError(wire.RejectNonstandard,
			"non-standard script form")
	}

	return nil
}

// isDust returns whether or not the passed transaction output amount is
// considered dust or not based on the passed minimum transaction relay fee.
// Dust is defined in terms of the minimum transaction relay fee.  In
// particular, if the cost to the network to spend coins is more than 1/3 of the
// minimum transaction relay fee, it is considered dust.
func isDust(txOut *wire.TxOut, minRelayTxFee provautil.Amount) bool {
	// Unspendable outputs are considered dust.
	if txscript.IsUnspendable(txOut.PkScript) {
		return true
	}

	// The total serialized size consists of the output and the associated
	// input script to redeem it.  Since there is no input script
	// to redeem it yet, use the minimum size of a typical input script.
	//
	// Pay-to-pubkey-hash bytes breakdown:
	//
	//  Output to hash (34 bytes):
	//   8 value, 1 script len, 25 script [1 OP_DUP, 1 OP_HASH_160,
	//   1 OP_DATA_20, 20 hash, 1 OP_EQUALVERIFY, 1 OP_CHECKSIG]
	//
	//  Input with compressed pubkey (148 bytes):
	//   36 prev outpoint, 1 script len, 107 script [1 OP_DATA_72, 72 sig,
	//   1 OP_DATA_33, 33 compressed pubkey], 4 sequence
	//
	//  Input with uncompressed pubkey (180 bytes):
	//   36 prev outpoint, 1 script len, 139 script [1 OP_DATA_72, 72 sig,
	//   1 OP_DATA_65, 65 compressed pubkey], 4 sequence
	//
	// Pay-to-pubkey bytes breakdown:
	//
	//  Output to compressed pubkey (44 bytes):
	//   8 value, 1 script len, 35 script [1 OP_DATA_33,
	//   33 compressed pubkey, 1 OP_CHECKSIG]
	//
	//  Output to uncompressed pubkey (76 bytes):
	//   8 value, 1 script len, 67 script [1 OP_DATA_65, 65 pubkey,
	//   1 OP_CHECKSIG]
	//
	//  Input (114 bytes):
	//   36 prev outpoint, 1 script len, 73 script [1 OP_DATA_72,
	//   72 sig], 4 sequence
	//
	// Theoretically this could examine the script type of the output script
	// and use a different size for the typical input script size for
	// pay-to-pubkey vs pay-to-pubkey-hash inputs per the above breakdowns,
	// but the only combinination which is less than the value chosen is
	// a pay-to-pubkey script with a compressed pubkey, which is not very
	// common.
	//
	// The most common scripts are pay-to-pubkey-hash, and as per the above
	// breakdown, the minimum size of a p2pkh input script is 148 bytes.  So
	// that figure is used.
	totalSize := txOut.SerializeSize() + 148

	// The output is considered dust if the cost to the network to spend the
	// coins is more than 1/3 of the minimum free transaction relay fee.
	// minFreeTxRelayFee is in Atoms/KB, so multiply by 1000 to
	// convert to bytes.
	//
	// Using the typical values for a pay-to-pubkey-hash transaction from
	// the breakdown above and the default minimum free transaction relay
	// fee of 1000, this equates to values less than 546 atoms being
	// considered dust.
	//
	// The following is equivalent to (value/totalSize) * (1/3) * 1000
	// without needing to do floating point math.
	return txOut.Value*1000/(3*int64(totalSize)) < int64(minRelayTxFee)
}

// checkTransactionStandard performs a series of checks on a transaction to
// ensure it is a "standard" transaction.  A standard transaction is one that
// conforms to several additional limiting cases over what is considered a
// "sane" transaction such as having a version in the supported range, being
// finalized, conforming to more stringent size constraints, having scripts
// of recognized forms, and not containing "dust" outputs (those that are
// so small it costs more to process them than they are worth).
// TODO(prova): Notice that this code is a duplicate of transaction
// validation code in CheckTransactionSanity() of validate.go
// TODO(prova): extract functionality into admin tx validator.
func checkTransactionStandard(tx *provautil.Tx, height uint32,
	medianTimePast time.Time, minRelayTxFee provautil.Amount,
	maxTxVersion int32) error {
	// The transaction must be a currently supported version.
	msgTx := tx.MsgTx()
	if msgTx.Version > maxTxVersion || msgTx.Version < 1 {
		str := fmt.Sprintf("transaction version %d is not in the "+
			"valid range of %d-%d", msgTx.Version, 1,
			maxTxVersion)
		return txRuleError(wire.RejectNonstandard, str)
	}

	// The transaction must be finalized to be standard and therefore
	// considered for inclusion in a block.
	if !blockchain.IsFinalizedTransaction(tx, height, medianTimePast) {
		return txRuleError(wire.RejectNonstandard,
			"transaction is not finalized")
	}

	// Since extremely large transactions with a lot of inputs can cost
	// almost as much to process as the sender fees, limit the maximum
	// size of a transaction.  This also helps mitigate CPU exhaustion
	// attacks.
	serializedLen := msgTx.SerializeSize()
	if serializedLen > MaxStandardTxSize {
		str := fmt.Sprintf("transaction size of %v is larger than max "+
			"allowed size of %v", serializedLen, MaxStandardTxSize)
		return txRuleError(wire.RejectNonstandard, str)
	}

	for i, txIn := range msgTx.TxIn {
		// Each transaction input signature script must not exceed the
		// maximum size allowed for a standard transaction.  See
		// the comment on maxStandardSigScriptSize for more details.
		sigScriptLen := len(txIn.SignatureScript)
		if sigScriptLen > maxStandardSigScriptSize {
			str := fmt.Sprintf("transaction input %d: signature "+
				"script size of %d bytes is large than max "+
				"allowed size of %d bytes", i, sigScriptLen,
				maxStandardSigScriptSize)
			return txRuleError(wire.RejectNonstandard, str)
		}

		// Each transaction input signature script must only contain
		// opcodes which push data onto the stack.
		if !txscript.IsPushOnlyScript(txIn.SignatureScript) {
			str := fmt.Sprintf("transaction input %d: signature "+
				"script is not push only", i)
			return txRuleError(wire.RejectNonstandard, str)
		}
	}

	// None of the output public key scripts can be a non-standard script or
	// be "dust" (except when the script is a null data script).
	numNullDataOutputs := 0
	threadInt, adminOutputs := txscript.GetAdminDetails(tx)
	hasAdminOut := (threadInt >= 0)
	for txInIndex, txOut := range msgTx.TxOut {
		scriptClass := txscript.GetScriptClass(txOut.PkScript)
		err := checkPkScriptStandard(txOut.PkScript, scriptClass)
		if err != nil {
			// Attempt to extract a reject code from the error so
			// it can be retained.  When not possible, fall back to
			// a non standard error.
			rejectCode := wire.RejectNonstandard
			if rejCode, found := extractRejectCode(err); found {
				rejectCode = rejCode
			}
			str := fmt.Sprintf("transaction output %d: %v", txInIndex, err)
			return txRuleError(rejectCode, str)
		}

		// Only first output can be admin output
		if scriptClass == txscript.ProvaAdminTy {
			if txInIndex != 0 {
				str := fmt.Sprintf("transaction output %d: admin output "+
					"only allowed at position 0.", txInIndex)
				return txRuleError(wire.RejectInvalid, str)
			}
		}

		// All Admin tx output values must be 0 value
		if hasAdminOut {
			threadId := provautil.ThreadID(threadInt)
			if threadId != provautil.IssueThread && txOut.Value != 0 {
				str := fmt.Sprintf("admin transaction with non-zero value "+
					"output #%d.", txInIndex)
				return txRuleError(wire.RejectInvalid, str)
			}
		}

		// Accumulate the number of outputs which only carry data.  For
		// all other script types, ensure the output value is not
		// "dust".
		if scriptClass == txscript.NullDataTy {
			numNullDataOutputs++
		} else if !tx.IsCoinbase() && !hasAdminOut && isDust(txOut, minRelayTxFee) {
			str := fmt.Sprintf("transaction output %d: payment "+
				"of %d is dust", txInIndex, txOut.Value)
			return txRuleError(wire.RejectDust, str)
		}
	}

	// A standard transaction must not have more than one output script that
	// only carries data.
	if !hasAdminOut && numNullDataOutputs > 1 {
		str := "more than one transaction output in a nulldata script"
		return txRuleError(wire.RejectNonstandard, str)
	}

	// Check admin transaction on ROOT and PROVISION thread
	// TODO(prova): Notice that this code is a dupclicate of transaction
	// validation code in CheckTransactionSanity() of validate.go
	// TODO(prova): extract functionality into admin tx validator.
	if hasAdminOut {
		threadId := provautil.ThreadID(threadInt)
		if threadId == provautil.RootThread || threadId == provautil.ProvisionThread {
			// Admin tx may not have any other inputs
			if len(msgTx.TxIn) > 1 {
				str := fmt.Sprintf("admin transaction with more than 1 input.")
				return txRuleError(wire.RejectInvalid, str)
			}
			// Admin tx must have at least 2 outputs
			if len(msgTx.TxOut) < 2 {
				str := fmt.Sprintf("admin transaction with no admin operations.")
				return txRuleError(wire.RejectInvalid, str)
			}

			// op pkscript
			for _, adminOpOut := range adminOutputs {
				// check conditions for admin ops
				// - Admin tx additional outputs must be nulldata scripts
				// - Key in nulldata script must be valid
				// - Data in nulldata scripts must match proper form expected for
				//   the thread
				if !txscript.IsValidAdminOp(adminOpOut, threadId) {
					str := fmt.Sprintf("admin transaction with invalid admin " +
						"operation found.")
					return txRuleError(wire.RejectInvalid, str)
				}
			}
		}

		if threadId == provautil.IssueThread {
			// TODO(prova): take care of issue thread
			// If issuance/destruction tx, any non-nulldata outputs must be valid Prova scripts
		}
	}

	return nil
}
