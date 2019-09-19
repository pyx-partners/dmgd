// Copyright (c) 2016 The btcsuite developers
// Copyright (c) 2017 BitGo
// Copyright (c) 2019 Tranquility Node Ltd
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package chaincfg

import (
	"fmt"
	"testing"
)

// TestInvalidHashStr ensures the newShaHashFromStr function panics when used to
// with an invalid hash string.
func TestInvalidHashStr(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic for invalid hash, got nil")
		}
	}()
	newHashFromStr("banana")
}

// TestMustRegisterPanic ensures the mustRegister function panics when used to
// register an invalid network.
func TestMustRegisterPanic(t *testing.T) {
	t.Parallel()

	// Setup a defer to catch the expected panic to ensure it actually
	// paniced.
	defer func() {
		if err := recover(); err == nil {
			t.Error("mustRegister did not panic as expected")
		}
	}()

	// Intentionally try to register duplicate params to force a panic.
	mustRegister(&MainNetParams)
}

// TestMinValidateKeySetSize checks that the min validate key set size is
// consistent with the expected set size given the param rate limit settings.
func TestMinValidateKeySetSize(t *testing.T) {
	chainParams := Params{
		PowAveragingWindow:   31,
		ChainWindowMaxBlocks: 3,
	}
	minValidateKeySetSize := chainParams.MinValidateKeySetSize()
	expectedSetSize := 11
	if minValidateKeySetSize != expectedSetSize {
		str := fmt.Sprintf("MinValidateKeySetSize got %d min "+
			"validate keys, expected %d.", minValidateKeySetSize,
			expectedSetSize)
		t.Error(str)
	}

	smallShareLimitParams := Params{
		PowAveragingWindow:   17,
		ChainWindowMaxBlocks: 1,
	}
	minSmallShareLimitValidateKeySetSize := smallShareLimitParams.MinValidateKeySetSize()
	expectedSmallShareLimitSetSize := 17
	if minSmallShareLimitValidateKeySetSize != expectedSmallShareLimitSetSize {
		str := fmt.Sprintf("MinValidateKeySetSize got %d min "+
			"validate keys, expected %d.", minSmallShareLimitValidateKeySetSize,
			expectedSmallShareLimitSetSize)
		t.Error(str)
	}

	largeShareLimitParams := Params{
		PowAveragingWindow:   31,
		ChainWindowMaxBlocks: 13,
	}
	minLargeShareLimitValidateKeySetSize := largeShareLimitParams.MinValidateKeySetSize()
	expectedLargeShareLimitSetSize := 3
	if minLargeShareLimitValidateKeySetSize != expectedLargeShareLimitSetSize {
		str := fmt.Sprintf("MinValidateKeySetSize got %d min "+
			"validate keys, expected %d.", minLargeShareLimitValidateKeySetSize,
			expectedLargeShareLimitSetSize)
		t.Error(str)
	}
}
