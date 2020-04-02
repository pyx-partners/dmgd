// Copyright (c) 2013-2016 The btcsuite developers
// Copyright (c) 2017 BitGo
// Copyright (c) 2019 Tranquility Node Ltd
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain_test

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/pyx-partners/dmgd/blockchain"
	"github.com/pyx-partners/dmgd/txscript"
)

// TestCheckBlockScripts ensures that validating the all of the scripts in a
// known-good block doesn't return an error.
//func TestCheckBlockScripts(t *testing.T) {
//TODO(prova) fix test
func CheckBlockScripts(t *testing.T) {
	runtime.GOMAXPROCS(runtime.NumCPU())

	testBlockNum := 277647
	blockDataFile := fmt.Sprintf("%d.dat.bz2", testBlockNum)
	blocks, err := loadBlocks(blockDataFile)
	if err != nil {
		t.Errorf("Error loading file: %v\n", err)
		return
	}
	if len(blocks) > 1 {
		t.Errorf("The test block file must only have one block in it")
		return
	}
	if len(blocks) == 0 {
		t.Errorf("The test block file may not be empty")
		return
	}

	storeDataFile := fmt.Sprintf("%d.utxostore.bz2", testBlockNum)
	utxoView, err := loadUtxoView(storeDataFile)
	if err != nil {
		t.Errorf("Error loading txstore: %v\n", err)
		return
	}

	scriptFlags := txscript.ScriptBip16
	err = blockchain.TstCheckBlockScripts(blocks[0], utxoView, nil, scriptFlags,
		nil, nil)
	if err != nil {
		t.Errorf("Transaction script validation failed: %v\n", err)
		return
	}
}
