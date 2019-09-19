// Copyright (c) 2013-2014 The btcsuite developers
// Copyright (c) 2017 BitGo
// Copyright (c) 2019 Tranquility Node Ltd
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package provautil

const (
	// AtomsPerGram is the number of Atoms in one gram (1 RMG).
	AtomsPerGram = 1e6

	// MaxAtoms is the maximum transaction amount allowed in Atoms.
	MaxAtoms = 21e8 * AtomsPerGram
)
