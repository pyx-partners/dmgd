// Copyright (c) 2013-2015 The btcsuite developers
// Copyright (c) 2017 BitGo
// Copyright (c) 2019 Tranquility Node Ltd
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package txscript

import (
	"errors"
	"fmt"
	"github.com/pyx-partners/dmgd/btcec"
	"github.com/pyx-partners/dmgd/chaincfg"
	"github.com/pyx-partners/dmgd/provautil"
	"github.com/pyx-partners/dmgd/wire"
	"sort"
)

// RawTxInSignature returns the serialized ECDSA signature for the input idx of
// the given transaction, with hashType appended to it.
func RawTxInSignature(tx *wire.MsgTx, idx int, subScript []byte,
	hashType SigHashType, key *btcec.PrivateKey) ([]byte, error) {

	parsedScript, err := ParseScript(subScript)
	if err != nil {
		return nil, fmt.Errorf("cannot parse output script: %v", err)
	}
	hash := calcSignatureHash(parsedScript, hashType, tx, idx)
	signature, err := key.Sign(hash)
	if err != nil {
		return nil, fmt.Errorf("cannot sign tx input: %s", err)
	}

	return append(signature.Serialize(), byte(hashType)), nil
}

// RawTxInSignatureNew returns the serialized ECDSA signature for the input idx of
// the given transaction, with hashType appended to it.
// TODO(prova): need to cleanup the old/new versions
func RawTxInSignatureNew(tx *wire.MsgTx, idx int, txSigHashes *TxSigHashes, amt int64, subScript []byte,
	hashType SigHashType, key *btcec.PrivateKey) ([]byte, error) {

	parsedScript, err := ParseScript(subScript)
	if err != nil {
		return nil, fmt.Errorf("cannot parse output script: %v", err)
	}

	hash := calcSignatureHashNew(parsedScript, txSigHashes, hashType, tx, idx, amt)
	signature, err := key.Sign(hash)
	if err != nil {
		return nil, fmt.Errorf("cannot sign tx input: %s", err)
	}

	return append(signature.Serialize(), byte(hashType)), nil
}

// SignatureScript creates an input signature script for tx to spend DMG sent
// from a previous output to the owner of privKey. tx must include all
// transaction inputs and outputs, however txin scripts are allowed to be filled
// or empty. The returned script is calculated to be used as the idx'th txin
// sigscript for tx. subscript is the PkScript of the previous output being used
// as the idx'th input. privKey is serialized in either a compressed or
// uncompressed format based on compress. This format must match the same format
// used to generate the payment address, or the script validation will fail.
func SignatureScript(tx *wire.MsgTx, idx int, subscript []byte, hashType SigHashType, privKey *btcec.PrivateKey, compress bool) ([]byte, error) {
	sig, err := RawTxInSignature(tx, idx, subscript, hashType, privKey)
	if err != nil {
		return nil, err
	}

	pk := (*btcec.PublicKey)(&privKey.PublicKey)
	var pkData []byte
	if compress {
		pkData = pk.SerializeCompressed()
	} else {
		pkData = pk.SerializeUncompressed()
	}

	return NewScriptBuilder().AddData(sig).AddData(pkData).Script()
}

// signSafeMultiSig signs as many of the outputs in the provided multisig script as
// possible. It returns the generated script and a boolean if the script fulfils
// the contract (i.e. nrequired signatures are provided).  Since it is arguably
// legal to not be able to sign any of the outputs, no error is returned.
func signSafeMultiSig(tx *wire.MsgTx, idx int, txSigHashes *TxSigHashes, amt int64, subScript []byte, hashType SigHashType,
	keys []PrivateKey, nRequired int, kdb KeyDB) ([]byte, bool) {
	builder := NewScriptBuilder()
	signed := 0

	for _, key := range keys {

		// add pubKey
		pk := (*btcec.PublicKey)(&key.Key.PublicKey)
		builder.AddData(pk.SerializeCompressed())

		// add signature
		sig, err := RawTxInSignatureNew(tx, idx, txSigHashes, amt, subScript, hashType, key.Key)
		if err != nil {
			// we silently ignore errors, because not all keys need to sign for a valid tx.
			continue
		}
		builder.AddData(sig)
		signed++
		if signed == nRequired {
			break
		}

	}

	script, _ := builder.Script()
	return script, signed == nRequired
}

func sign(chainParams *chaincfg.Params, tx *wire.MsgTx, idx int, inputAmt int64,
	subScript []byte, hashType SigHashType, kdb KeyDB) (
	[]byte, ScriptClass, []provautil.Address, int, error) {

	class, addresses, nrequired, err := ExtractPkScriptAddrs(subScript,
		chainParams)
	if err != nil {
		return nil, NonStandardTy, nil, 0, err
	}

	// Create a new HashCache adding the intermediate sigHashes of this tx to it.
	// The size of the HashCache is chosen big enough for any transaction.
	// TODO(prova) find a better way to set size of HashCache
	hashCache := NewHashCache(90)
	hashCache.AddSigHashes(tx)
	txHash := tx.TxHash()
	txSigHashes, found := hashCache.GetSigHashes(&txHash)
	if !found {
		return nil, class, nil, 0, errors.New("unable to find sighashes")
	}

	switch class {
	case ProvaTy:
		// We use the keysDb lookup to get a list of privKeys
		// that are needed for signing.
		keys, err := kdb.GetKey(addresses[0])
		if err != nil {
			return nil, class, nil, 0, err
		}
		// do the signing
		script, _ := signSafeMultiSig(tx, idx, txSigHashes, inputAmt, subScript, hashType,
			keys, nrequired, kdb)
		return script, class, addresses, nrequired, nil
	case ProvaAdminTy:
		// We use the keysDb lookup to get a list of privKeys that are needed
		// for signing. Passing nil will give us all keys.
		keys, err := kdb.GetKey(nil)
		if err != nil {
			return nil, class, nil, 0, err
		}
		// do the signing
		script, _ := signSafeMultiSig(tx, idx, txSigHashes, inputAmt, subScript, hashType,
			keys, nrequired, kdb)
		return script, class, addresses, nrequired, nil
	case NullDataTy:
		return nil, class, nil, 0,
			errors.New("can't sign NULLDATA transactions")
	default:
		return nil, class, nil, 0,
			errors.New("can't sign unknown transactions")
	}
}

// mergeScripts merges sigScript and prevScript assuming they are both
// partial solutions for pkScript spending output idx of tx. class, addresses
// and nrequired are the result of extracting the addresses from pkscript.
// The return value is the best effort merging of the two scripts. Calling this
// function with addresses, class and nrequired that do not match pkScript is
// an error and results in undefined behaviour.
func mergeScripts(chainParams *chaincfg.Params, tx *wire.MsgTx, idx int,
	pkScript []byte, class ScriptClass, addresses []provautil.Address,
	nRequired int, sigScript, prevScript []byte) []byte {

	switch class {
	case ProvaTy:
		return mergeProvaSig(tx, idx, addresses, nRequired, pkScript,
			sigScript, prevScript)
	case ProvaAdminTy:
		return mergeProvaAdminSig(tx, idx, addresses, nRequired, pkScript,
			sigScript, prevScript)

	// It doesn't actually make sense to merge anything other than multiig
	// and scripthash (because it could contain multisig). Everything else
	// has either zero signature, can't be spent, or has a single signature
	// which is either present or not. The other two cases are handled
	// above. In the conflict case here we just assume the longest is
	// correct (this matches behaviour of the reference implementation).
	default:
		if len(sigScript) > len(prevScript) {
			return sigScript
		}
		return prevScript
	}
}

// mergeProvaSig combines the two signature scripts sigScript and prevScript
// that both provide signatures for pkScript in output idx of tx.
func mergeProvaSig(tx *wire.MsgTx, idx int, addresses []provautil.Address,
	nRequired int, pkScript, sigScript, prevScript []byte) []byte {
	return append(prevScript[:], sigScript[:]...)
}

// mergeProvaAdminSig combines the two signature scripts sigScript and prevScript
// that both provide signatures for pkScript in output idx of tx.
func mergeProvaAdminSig(tx *wire.MsgTx, idx int, addresses []provautil.Address,
	nRequired int, pkScript, sigScript, prevScript []byte) []byte {

	sigPops, err := ParseScript(sigScript)
	if err != nil || len(sigPops) == 0 {
		return prevScript
	}

	prevPops, err := ParseScript(prevScript)
	if err != nil || len(prevPops) == 0 {
		return sigScript
	}

	// create a map of pub to sig
	pubToOps := make(map[string][2]parsedOpcode)
	for i := 0; i < len(sigPops); i = i + 2 {
		pubKey, _ := btcec.ParsePubKey(sigPops[i].data, btcec.S256())
		pubKeyStr := fmt.Sprintf("%x", pubKey.SerializeCompressed())
		pubToOps[pubKeyStr] = [2]parsedOpcode{sigPops[i], sigPops[i+1]}
	}
	for i := 0; i < len(prevPops); i = i + 2 {
		pubKey, _ := btcec.ParsePubKey(prevPops[i].data, btcec.S256())
		pubKeyStr := fmt.Sprintf("%x", pubKey.SerializeCompressed())
		pubToOps[pubKeyStr] = [2]parsedOpcode{prevPops[i], prevPops[i+1]}
	}
	// sort pubs alphanumerically
	pubs := make([]string, 0, len(pubToOps))
	for pub := range pubToOps {
		pubs = append(pubs, pub)
	}
	sort.Strings(pubs)

	// create new script with right ordering
	builder := NewScriptBuilder()
	doneSigs := 0
	for _, pub := range pubs {
		builder.AddData(pubToOps[pub][0].data)
		builder.AddData(pubToOps[pub][1].data)
		doneSigs++
		if doneSigs == nRequired {
			break
		}
	}
	script, _ := builder.Script()
	return script
}

type PrivateKey struct {
	Key        *btcec.PrivateKey
	Compressed bool
}

// KeyDB is an interface type provided to SignTxOutput, it encapsulates
// any user state required to get the private keys for an address.
type KeyDB interface {
	GetKey(provautil.Address) ([]PrivateKey, error)
}

// KeyClosure implements KeyDB with a closure.
type KeyClosure func(provautil.Address) ([]PrivateKey, error)

// GetKey implements KeyDB by returning the result of calling the closure.
func (kc KeyClosure) GetKey(address provautil.Address) ([]PrivateKey, error) {
	return kc(address)
}

// SignTxOutput signs output idx of the given tx to resolve the script given in
// pkScript with a signature type of hashType. Any keys required will be
// looked up by calling getKey() with the string of the given address.
// Any pay-to-script-hash signatures will be similarly looked up by calling
// getScript. If previousScript is provided then the results in previousScript
// will be merged in a type-dependent manner with the newly generated.
// signature script.
func SignTxOutput(chainParams *chaincfg.Params, tx *wire.MsgTx, idx int, inputAmt int64,
	pkScript []byte, hashType SigHashType, kdb KeyDB,
	previousScript []byte) ([]byte, error) {

	sigScript, class, addresses, nrequired, err := sign(chainParams, tx,
		idx, inputAmt, pkScript, hashType, kdb)
	if err != nil {
		return nil, err
	}

	// Merge scripts. with any previous data, if any.
	mergedScript := mergeScripts(chainParams, tx, idx, pkScript, class,
		addresses, nrequired, sigScript, previousScript)
	return mergedScript, nil
}
