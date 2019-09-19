// Copyright (c) 2013-2016 The btcsuite developers
// Copyright (c) 2017 BitGo
// Copyright (c) 2019 Tranquility Node Ltd
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain_test

import (
	"testing"

	"github.com/opacey/dmgd/blockchain"
	"github.com/opacey/dmgd/provautil"
)

// BenchmarkIsCoinBase performs a simple benchmark against the IsCoinBase
// function.
func BenchmarkIsCoinBase(b *testing.B) {
	tx, _ := provautil.NewBlock(&SomeBlock).Tx(1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		blockchain.IsCoinBase(tx)
	}
}

// BenchmarkIsCoinBaseTx performs a simple benchmark against the IsCoinBaseTx
// function.
func BenchmarkIsCoinBaseTx(b *testing.B) {
	tx := SomeBlock.Transactions[1]
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		blockchain.IsCoinBaseTx(tx)
	}
}
