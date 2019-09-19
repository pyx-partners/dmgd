// Copyright (c) 2013, 2014 The btcsuite developers
// Copyright (c) 2017 BitGo
// Copyright (c) 2019 Tranquility Node Ltd
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package provautil_test

import (
	"math"
	"testing"

	. "github.com/opacey/dmgd/provautil"
)

func TestAmountCreation(t *testing.T) {
	tests := []struct {
		name     string
		amount   float64
		valid    bool
		expected Amount
	}{
		// Positive tests.
		{
			name:     "zero",
			amount:   0,
			valid:    true,
			expected: 0,
		},
		{
			name:     "max producible",
			amount:   21e8,
			valid:    true,
			expected: MaxAtoms,
		},
		{
			name:     "min producible",
			amount:   -21e8,
			valid:    true,
			expected: -MaxAtoms,
		},
		{
			name:     "exceeds max producible",
			amount:   21e8 + 1e-6,
			valid:    true,
			expected: MaxAtoms + 1,
		},
		{
			name:     "exceeds min producible",
			amount:   -21e8 - 1e-6,
			valid:    true,
			expected: -MaxAtoms - 1,
		},
		{
			name:     "one hundred",
			amount:   100,
			valid:    true,
			expected: 100 * AtomsPerGram,
		},
		{
			name:     "fraction",
			amount:   0.123456,
			valid:    true,
			expected: 123456,
		},
		{
			name:     "rounding up",
			amount:   54.999999999999943157,
			valid:    true,
			expected: 55 * AtomsPerGram,
		},
		{
			name:     "rounding down",
			amount:   55.000000000000056843,
			valid:    true,
			expected: 55 * AtomsPerGram,
		},

		// Negative tests.
		{
			name:   "not-a-number",
			amount: math.NaN(),
			valid:  false,
		},
		{
			name:   "-infinity",
			amount: math.Inf(-1),
			valid:  false,
		},
		{
			name:   "+infinity",
			amount: math.Inf(1),
			valid:  false,
		},
	}

	for _, test := range tests {
		a, err := NewAmount(test.amount)
		switch {
		case test.valid && err != nil:
			t.Errorf("%v: Positive test Amount creation failed with: %v", test.name, err)
			continue
		case !test.valid && err == nil:
			t.Errorf("%v: Negative test Amount creation succeeded (value %v) when should fail", test.name, a)
			continue
		}

		if a != test.expected {
			t.Errorf("%v: Created amount %v does not match expected %v", test.name, a, test.expected)
			continue
		}
	}
}

func TestAmountUnitConversions(t *testing.T) {
	tests := []struct {
		name      string
		amount    Amount
		unit      AmountUnit
		converted float64
		s         string
	}{
		{
			name:      "MRMG",
			amount:    MaxAtoms,
			unit:      AmountMegaRMG,
			converted: 2100,
			s:         "2100 MRMG",
		},
		{
			name:      "kRMG",
			amount:    44433322211100,
			unit:      AmountKiloRMG,
			converted: 44433.322211100,
			s:         "44433.3222111 kRMG",
		},
		{
			name:      "RMG",
			amount:    44433322211100,
			unit:      AmountRMG,
			converted: 44433322.211100,
			s:         "44433322.2111 RMG",
		},
		{
			name:      "mRMG",
			amount:    44433322211100,
			unit:      AmountMilliRMG,
			converted: 44433322211.100,
			s:         "44433322211.1 mRMG",
		},
		{

			name:      "Atom",
			amount:    444333222111,
			unit:      AmountAtoms,
			converted: 444333222111,
			s:         "444333222111 Atom",
		},
		{

			name:      "non-standard unit",
			amount:    44433322211100,
			unit:      AmountUnit(-1),
			converted: 444333222.11100,
			s:         "444333222.111 1e-1 RMG",
		},
	}

	for _, test := range tests {
		f := test.amount.ToUnit(test.unit)
		if f != test.converted {
			t.Errorf("%v: converted value %v does not match expected %v", test.name, f, test.converted)
			continue
		}

		s := test.amount.Format(test.unit)
		if s != test.s {
			t.Errorf("%v: format '%v' does not match expected '%v'", test.name, s, test.s)
			continue
		}

		// Verify that Amount.ToRMG works as advertised.
		f1 := test.amount.ToUnit(AmountRMG)
		f2 := test.amount.ToRMG()
		if f1 != f2 {
			t.Errorf("%v: ToRMG does not match ToUnit(AmountRMG): %v != %v", test.name, f1, f2)
		}

		// Verify that Amount.String works as advertised.
		s1 := test.amount.Format(AmountRMG)
		s2 := test.amount.String()
		if s1 != s2 {
			t.Errorf("%v: String does not match Format(AmountGrams): %v != %v", test.name, s1, s2)
		}
	}
}

func TestAmountMulF64(t *testing.T) {
	tests := []struct {
		name string
		amt  Amount
		mul  float64
		res  Amount
	}{
		{
			name: "Multiply 0.1 RMG by 2",
			amt:  100e5, // 0.1 RMG
			mul:  2,
			res:  200e5, // 0.2 RMG
		},
		{
			name: "Multiply 0.2 RMG by 0.02",
			amt:  200e5, // 0.2 RMG
			mul:  1.02,
			res:  204e5, // 0.204 RMG
		},
		{
			name: "Multiply 0.1 RMG by -2",
			amt:  100e5, // 0.1 RMG
			mul:  -2,
			res:  -200e5, // -0.2 RMG
		},
		{
			name: "Multiply 0.2 RMG by -0.02",
			amt:  200e5, // 0.2 RMG
			mul:  -1.02,
			res:  -204e5, // -0.204 RMG
		},
		{
			name: "Multiply -0.1 RMG by 2",
			amt:  -100e5, // -0.1 RMG
			mul:  2,
			res:  -200e5, // -0.2 RMG
		},
		{
			name: "Multiply -0.2 RMG by 0.02",
			amt:  -200e5, // -0.2 RMG
			mul:  1.02,
			res:  -204e5, // -0.204 RMG
		},
		{
			name: "Multiply -0.1 RMG by -2",
			amt:  -100e5, // -0.1 RMG
			mul:  -2,
			res:  200e5, // 0.2 RMG
		},
		{
			name: "Multiply -0.2 RMG by -0.02",
			amt:  -200e5, // -0.2 RMG
			mul:  -1.02,
			res:  204e5, // 0.204 RMG
		},
		{
			name: "Round down",
			amt:  49, // 49 Atoms
			mul:  0.01,
			res:  0,
		},
		{
			name: "Round up",
			amt:  50, // 50 Atoms
			mul:  0.01,
			res:  1, // 1 Atom
		},
		{
			name: "Multiply by 0.",
			amt:  1e8, // 100 RMG
			mul:  0,
			res:  0, // 0 RMG
		},
		{
			name: "Multiply 1 by 0.5.",
			amt:  1, // 1 Atom
			mul:  0.5,
			res:  1, // 1 Atom
		},
		{
			name: "Multiply 100 by 66%.",
			amt:  100, // 100 Atoms
			mul:  0.66,
			res:  66, // 66 Atoms
		},
		{
			name: "Multiply 100 by 66.6%.",
			amt:  100, // 100 Atoms
			mul:  0.666,
			res:  67, // 67 Atoms
		},
		{
			name: "Multiply 100 by 2/3.",
			amt:  100, // 100 Atoms
			mul:  2.0 / 3,
			res:  67, // 67 Atoms
		},
	}

	for _, test := range tests {
		a := test.amt.MulF64(test.mul)
		if a != test.res {
			t.Errorf("%v: expected %v got %v", test.name, test.res, a)
		}
	}
}
