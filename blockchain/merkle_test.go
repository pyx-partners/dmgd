// Copyright (c) 2013-2014 The btcsuite developers
// Copyright (c) 2017 BitGo
// Copyright (c) 2019 Tranquility Node Ltd
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain_test

import (
	"testing"

	"github.com/pyx-partners/dmgd/blockchain"
	"github.com/pyx-partners/dmgd/chaincfg/chainhash"
	"github.com/pyx-partners/dmgd/provautil"
)

// TestMerkle tests the BuildMerkleTreeStore API.
func TestMerkle(t *testing.T) {
	block := provautil.NewBlock(&SomeBlock)
	merkles := blockchain.BuildMerkleTreeStore(block.Transactions())
	calculatedMerkleRoot := merkles[len(merkles)-1]
	// TODO(prova) clean this up and generate a new block with correct merkle root
	merkleStr := "fe174a6ec736a412d8cc3e02328d1b1d250e24127dd74484dfa257bf983135f8"
	wantMerkle, _ := chainhash.NewHashFromStr(merkleStr)
	if !wantMerkle.IsEqual(calculatedMerkleRoot) {
		t.Errorf("BuildMerkleTreeStore: merkle root mismatch - "+
			"got %v, want %v", calculatedMerkleRoot, wantMerkle)
	}
}
