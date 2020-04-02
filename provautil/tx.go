// Copyright (c) 2013-2016 The btcsuite developers
// Copyright (c) 2017 BitGo
// Copyright (c) 2019 Tranquility Node Ltd
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package provautil

import (
	"bytes"
	"io"

	"github.com/pyx-partners/dmgd/chaincfg/chainhash"
	"github.com/pyx-partners/dmgd/wire"
)

// TxIndexUnknown is the value returned for a transaction index that is unknown.
// This is typically because the transaction has not been inserted into a block
// yet.
const TxIndexUnknown = -1

// Tx defines a bitcoin transaction that provides easier and more efficient
// manipulation of raw transactions.  It also memoizes the hash for the
// transaction on its first access so subsequent accesses don't have to repeat
// the relatively expensive hashing operations.
type Tx struct {
	msgTx         *wire.MsgTx     // Underlying MsgTx
	txHash        *chainhash.Hash // Cached transaction hash
	TxHashWithSig *chainhash.Hash // Cached tx-over-sig hash
	txIndex       int             // Position within a block or TxIndexUnknown
}

// IsCoinbase returns whether the transaction is a coinbase transaction.
func (t *Tx) IsCoinbase() bool {
	return t.txIndex == 0
}

// MsgTx returns the underlying wire.MsgTx for the transaction.
func (t *Tx) MsgTx() *wire.MsgTx {
	// Return the cached transaction.
	return t.msgTx
}

// Hash returns the hash of the transaction.  This is equivalent to
// calling TxHash on the underlying wire.MsgTx, however it caches the
// result so subsequent calls are more efficient.
func (t *Tx) HashWithSig() *chainhash.Hash {
	// Return the cached hash if it has already been generated.
	if t.TxHashWithSig != nil {
		return t.TxHashWithSig
	}

	// Cache the hash and return it.
	hash := t.msgTx.TxHashWithSig()
	t.TxHashWithSig = &hash
	return &hash
}

// Hash returns the hash of a transaction without scriptSigs.
func (t *Tx) Hash() *chainhash.Hash {
	// Return the cached hash if it has already been generated.
	if t.txHash != nil {
		return t.txHash
	}

	// Cache the hash and return it.
	hash := t.msgTx.TxHash()
	t.txHash = &hash
	return &hash
}

// Index returns the saved index of the transaction within a block.  This value
// will be TxIndexUnknown if it hasn't already explicitly been set.
func (t *Tx) Index() int {
	return t.txIndex
}

// SetIndex sets the index of the transaction in within a block.
func (t *Tx) SetIndex(index int) {
	t.txIndex = index
}

// NewTx returns a new instance of a bitcoin transaction given an underlying
// wire.MsgTx.  See Tx.
func NewTx(msgTx *wire.MsgTx) *Tx {
	return &Tx{
		msgTx:   msgTx,
		txIndex: TxIndexUnknown,
	}
}

// NewTxFromBytes returns a new instance of a bitcoin transaction given the
// serialized bytes.  See Tx.
func NewTxFromBytes(serializedTx []byte) (*Tx, error) {
	br := bytes.NewReader(serializedTx)
	return NewTxFromReader(br)
}

// NewTxFromReader returns a new instance of a bitcoin transaction given a
// Reader to deserialize the transaction.  See Tx.
func NewTxFromReader(r io.Reader) (*Tx, error) {
	// Deserialize the bytes into a MsgTx.
	var msgTx wire.MsgTx
	err := msgTx.Deserialize(r)
	if err != nil {
		return nil, err
	}

	t := Tx{
		msgTx:   &msgTx,
		txIndex: TxIndexUnknown,
	}
	return &t, nil
}
