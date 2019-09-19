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
	// Zero Atoms: 0 RMG
	// 1,000,000 Atoms: 1 RMG
	// 100,000 Atoms: 0.1 RMG
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

	// Output: 1 RMG
	// 0.012345 RMG
	// 0 RMG
	// invalid amount
}

func ExampleAmount_unitConversions() {
	amount := provautil.Amount(444333222111)

	fmt.Println("Atom to kRMG:", amount.Format(provautil.AmountKiloRMG))
	fmt.Println("Atom to RMG:", amount)
	fmt.Println("Atom to MilliRMG:", amount.Format(provautil.AmountMilliRMG))
	fmt.Println("Atom to Atom:", amount.Format(provautil.AmountAtoms))

	// Output:
	// Atom to kRMG: 444.333222111 kRMG
	// Atom to RMG: 444333.222111 RMG
	// Atom to MilliRMG: 444333222.111 mRMG
	// Atom to Atom: 444333222111 Atom
}
