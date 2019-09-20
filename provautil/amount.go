// Copyright (c) 2013, 2014 The btcsuite developers
// Copyright (c) 2017 BitGo
// Copyright (c) 2019 Tranquility Node Ltd
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package provautil

import (
	"errors"
	"math"
	"strconv"
)

// AmountUnit describes a method of converting an Amount to something
// other than the base unit of a gram.  The value of the AmountUnit
// is the exponent component of the decadic multiple to convert from
// an amount in grams to an amount counted in units.
type AmountUnit int

// These constants define various units used when describing a gram monetary
// amount.
const (
	AmountMegaDMG  AmountUnit = 6
	AmountKiloDMG  AmountUnit = 3
	AmountDMG      AmountUnit = 0
	AmountMilliDMG AmountUnit = -3
	AmountAtoms    AmountUnit = -6
)

// String returns the unit as a string.  For recognized units, the SI
// prefix is used.  For all unrecognized units, "1eN DMG" is returned, where
// N is the AmountUnit.
func (u AmountUnit) String() string {
	switch u {
	case AmountMegaDMG:
		return "MDMG"
	case AmountKiloDMG:
		return "kDMG"
	case AmountDMG:
		return "DMG"
	case AmountMilliDMG:
		return "mDMG"
	case AmountAtoms:
		return "Atom"
	default:
		return "1e" + strconv.FormatInt(int64(u), 10) + " DMG"
	}
}

// Amount represents the base monetary unit (colloquially referred to as an
// 'Atom').  A single Amount is equal to 1e-6 of a gram.
type Amount int64

// round converts a floating point number, which may or may not be representable
// as an integer, to the Amount integer type by rounding to the nearest integer.
// This is performed by adding or subtracting 0.5 depending on the sign, and
// relying on integer truncation to round the value to the nearest Amount.
func round(f float64) Amount {
	if f < 0 {
		return Amount(f - 0.5)
	}
	return Amount(f + 0.5)
}

// NewAmount creates an Amount from a floating point value representing
// some value in grams.  NewAmount errors if f is NaN or +-Infinity, but
// does not check that the amount is within the total amount of grams
// producible as f may not refer to an amount at a single moment in time.
//
// NewAmount is for specifically for converting DMG to Atoms.
// For creating a new Amount with an int64 value which denotes a quantity of
// Atoms, do a simple type conversion from type int64 to Amount.
// See GoDoc for example: http://godoc.org/github.com/opacey/dmgd/provautil#example-Amount
func NewAmount(f float64) (Amount, error) {
	// The amount is only considered invalid if it cannot be represented
	// as an integer type.  This may happen if f is NaN or +-Infinity.
	switch {
	case math.IsNaN(f):
		fallthrough
	case math.IsInf(f, 1):
		fallthrough
	case math.IsInf(f, -1):
		return 0, errors.New("invalid amount")
	}

	return round(f * AtomsPerGram), nil
}

// ToUnit converts a monetary amount counted in gram base units to a
// floating point value representing an amount.
func (a Amount) ToUnit(u AmountUnit) float64 {
	return float64(a) / math.Pow10(int(u+6))
}

// ToDMG is the equivalent of calling ToUnit with AmountDMG.
func (a Amount) ToDMG() float64 {
	return a.ToUnit(AmountDMG)
}

// Format formats a monetary amount counted in gram base units as a
// string for a given unit.  The conversion will succeed for any unit,
// however, known units will be formated with an appended label describing
// the units with SI notation, or "Atoms" for the base unit.
func (a Amount) Format(u AmountUnit) string {
	units := " " + u.String()
	return strconv.FormatFloat(a.ToUnit(u), 'f', -int(u+6), 64) + units
}

// String is the equivalent of calling Format with AmountDMG.
func (a Amount) String() string {
	return a.Format(AmountDMG)
}

// MulF64 multiplies an Amount by a floating point value.  While this is not
// an operation that must typically be done by a full node or wallet, it is
// useful for services that build on top of bitcoin (for example, calculating
// a fee by multiplying by a percentage).
func (a Amount) MulF64(f float64) Amount {
	return round(float64(a) * f)
}
