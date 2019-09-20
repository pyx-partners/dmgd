package provautil_test

import (
	"fmt"
	"math"

	"github.com/opacey/dmgd/provautil"
)

func ExampleAmount() {

	a := provautil.Amount(0)
	fmt.Println("Zero Atoms:", a)

	a = provautil.Amount(1e6)
	fmt.Println("1,000,000 Atoms:", a)

	a = provautil.Amount(1e5)
	fmt.Println("100,000 Atoms:", a)
	// Output:
	// Zero Atoms: 0 DMG
	// 1,000,000 Atoms: 1 DMG
	// 100,000 Atoms: 0.1 DMG
}

func ExampleNewAmount() {
	amountOne, err := provautil.NewAmount(1)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(amountOne) //Output 1

	amountFraction, err := provautil.NewAmount(0.012345)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(amountFraction) //Output 2

	amountZero, err := provautil.NewAmount(0)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(amountZero) //Output 3

	amountNaN, err := provautil.NewAmount(math.NaN())
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(amountNaN) //Output 4

	// Output: 1 DMG
	// 0.012345 DMG
	// 0 DMG
	// invalid amount
}

func ExampleAmount_unitConversions() {
	amount := provautil.Amount(444333222111)

	fmt.Println("Atom to kDMG:", amount.Format(provautil.AmountKiloDMG))
	fmt.Println("Atom to DMG:", amount)
	fmt.Println("Atom to MilliDMG:", amount.Format(provautil.AmountMilliDMG))
	fmt.Println("Atom to Atom:", amount.Format(provautil.AmountAtoms))

	// Output:
	// Atom to kDMG: 444.333222111 kDMG
	// Atom to DMG: 444333.222111 DMG
	// Atom to MilliDMG: 444333222.111 mDMG
	// Atom to Atom: 444333222111 Atom
}
