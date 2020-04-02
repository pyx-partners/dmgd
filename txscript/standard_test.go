// Copyright (c) 2013-2017 The btcsuite developers
// Copyright (c) 2017 BitGo
// Copyright (c) 2019 Tranquility Node Ltd
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package txscript

import (
	"bytes"
	"encoding/hex"
	"github.com/pyx-partners/dmgd/btcec"
	"github.com/pyx-partners/dmgd/chaincfg"
	"github.com/pyx-partners/dmgd/provautil"
	"github.com/pyx-partners/dmgd/wire"
	"reflect"
	"testing"
)

// decodeHex decodes the passed hex string and returns the resulting bytes.  It
// panics if an error occurs.  This is only used in the tests as a helper since
// the only way it can fail is if there is an error in the test source code.
func decodeHex(hexStr string) []byte {
	b, err := hex.DecodeString(hexStr)
	if err != nil {
		panic("invalid hex string in test source: err " + err.Error() +
			", hex: " + hexStr)
	}

	return b
}

// mustParseShortForm parses the passed short form script and returns the
// resulting bytes.  It panics if an error occurs.  This is only used in the
// tests as a helper since the only way it can fail is if there is an error in
// the test source code.
func mustParseShortForm(script string) []byte {
	s, err := parseShortForm(script)
	if err != nil {
		panic("invalid short form script in test source: err " +
			err.Error() + ", script: " + script)
	}

	return s
}

func newAddressProva(pkHash []byte, keyIDs []btcec.KeyID) provautil.Address {
	addr, err := provautil.NewAddressProva(pkHash, keyIDs, &chaincfg.MainNetParams)
	if err != nil {
		panic("invalid prova address in test source")
	}

	return addr
}

// TestExtractPkScriptAddrs ensures that extracting the type, addresses, and
// number of required signatures from PkScripts works as intended.
func TestExtractPkScriptAddrs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		script  []byte
		addrs   []provautil.Address
		reqSigs int
		class   ScriptClass
	}{
		{
			name: "standard prova",
			script: decodeHex("521435dbbf04bca061e49dace08f858d87" +
				"75c0a57c8e030000015153ba"),
			addrs: []provautil.Address{
				newAddressProva(decodeHex("35dbbf04bca061e49dace08f858d8775c0a57c8e"),
					[]btcec.KeyID{0x10000, 1}),
			},
			reqSigs: 2,
			class:   ProvaTy,
		},
		{
			name:    "empty script",
			script:  []byte{},
			addrs:   nil,
			reqSigs: 0,
			class:   NonStandardTy,
		},
		{
			name:    "script that does not parse",
			script:  []byte{OP_DATA_45},
			addrs:   nil,
			reqSigs: 0,
			class:   NonStandardTy,
		},
	}

	t.Logf("Running %d tests.", len(tests))
	for i, test := range tests {
		class, addrs, reqSigs, err := ExtractPkScriptAddrs(
			test.script, &chaincfg.MainNetParams)
		if err != nil {
		}

		if !reflect.DeepEqual(addrs, test.addrs) {
			t.Errorf("ExtractPkScriptAddrs #%d (%s) unexpected "+
				"addresses\ngot  %v\nwant %v", i, test.name,
				addrs, test.addrs)
			continue
		}

		if reqSigs != test.reqSigs {
			t.Errorf("ExtractPkScriptAddrs #%d (%s) unexpected "+
				"number of required signatures - got %d, "+
				"want %d", i, test.name, reqSigs, test.reqSigs)
			continue
		}

		if class != test.class {
			t.Errorf("ExtractPkScriptAddrs #%d (%s) unexpected "+
				"script type - got %s, want %s", i, test.name,
				class, test.class)
			continue
		}
	}
}

// TestIsValidAdminOp tests the IsValidAdminOp function.
func TestIsValidAdminOp(t *testing.T) {
	// Create some dummy admin op output.
	_, pubKey := btcec.PrivKeyFromBytes(btcec.S256(), []byte{
		0x2b, 0x8c, 0x52, 0xb7, 0x7b, 0x32, 0x7c, 0x75,
		0x5b, 0x9b, 0x37, 0x55, 0x00, 0xd3, 0xf4, 0xb2,
		0xda, 0x9b, 0x0a, 0x1f, 0xf6, 0x5f, 0x68, 0x91,
		0xd3, 0x11, 0xfe, 0x94, 0x29, 0x5b, 0xc2, 0x6a,
	})
	// provision key add
	data := make([]byte, 1+btcec.PubKeyBytesLenCompressed)
	data[0] = AdminOpProvisionKeyAdd
	copy(data[1:], pubKey.SerializeCompressed())
	adminOpPkScript, _ := NewScriptBuilder().AddOp(OP_RETURN).AddData(data).Script()
	adminOpTxOut := wire.TxOut{
		Value:    0,
		PkScript: adminOpPkScript,
	}
	// asp add
	aspData := make([]byte, 1+btcec.PubKeyBytesLenCompressed+btcec.KeyIDSize)
	aspData[0] = AdminOpASPKeyAdd
	copy(aspData[1:], pubKey.SerializeCompressed())
	btcec.KeyID(1).ToAddressFormat(aspData[1+btcec.PubKeyBytesLenCompressed:])
	provOpPkScript, _ := NewScriptBuilder().AddOp(OP_RETURN).AddData(aspData).Script()
	provOpTxOut := wire.TxOut{
		Value:    0,
		PkScript: provOpPkScript,
	}
	// create root tx out
	rootPkScript, _ := ProvaThreadScript(provautil.RootThread)
	rootTxOut := wire.TxOut{
		Value:    0,
		PkScript: rootPkScript,
	}
	// create provision tx out
	provisionPkScript, _ := ProvaThreadScript(provautil.ProvisionThread)
	provisionTxOut := wire.TxOut{
		Value:    0, // 0 DMG
		PkScript: provisionPkScript,
	}

	tests := []struct {
		name    string
		tx      wire.MsgTx
		isValid bool
	}{
		{
			name: "Typical admin transaction",
			tx: wire.MsgTx{
				TxOut: []*wire.TxOut{&rootTxOut, &adminOpTxOut},
			},
			isValid: true,
		}, {
			name: "Admin transaction adding asp",
			tx: wire.MsgTx{
				TxOut: []*wire.TxOut{&provisionTxOut, &provOpTxOut},
			},
			isValid: true,
		}, {
			name: "Admin transaction with operation on wrong thread",
			tx: wire.MsgTx{
				TxOut: []*wire.TxOut{&provisionTxOut, &adminOpTxOut},
			},
			isValid: false,
		},
	}

	for _, test := range tests {
		mtx := provautil.NewTx(&test.tx)
		// Ensure standardness is as expected.
		threadInt, adminOutputs := GetAdminDetails(mtx)
		if threadInt < 0 {
			t.Errorf("IsValidAdminOp (%s): non-admin thread "+
				" when it should be", test.name)
			continue
		}
		isValid := IsValidAdminOp(adminOutputs[0], provautil.ThreadID(threadInt))
		if isValid == test.isValid {
			// Test passes since function returned valid for an
			// op which is intended to be valid.
			continue
		}
		if isValid {
			t.Errorf("IsValidAdminOp (%s): valid when "+
				"it should not be", test.name)
			continue
		}
		if !isValid {
			t.Errorf("IsValidAdminOp (%s): invalid when "+
				"it should be valid", test.name)
			continue
		}
	}
}

// bogusAddress implements the provautil.Address interface so the tests can ensure
// unsupported address types are handled properly.
type bogusAddress struct{}

// EncodeAddress simply returns an empty string.  It exists to satsify the
// provautil.Address interface.
func (b *bogusAddress) EncodeAddress() string {
	return ""
}

// ScriptAddress simply returns an empty byte slice.  It exists to satsify the
// provautil.Address interface.
func (b *bogusAddress) ScriptAddress() []byte {
	return nil
}

// IsForNet lies blatantly to satisfy the provautil.Address interface.
func (b *bogusAddress) IsForNet(chainParams *chaincfg.Params) bool {
	return true // why not?
}

// String simply returns an empty string.  It exists to satsify the
// provautil.Address interface.
func (b *bogusAddress) String() string {
	return ""
}

func (b *bogusAddress) ScriptKeyIDs() []btcec.KeyID {
	return make([]btcec.KeyID, 0)
}

// TestPayToAddrScript ensures the PayToAddrScript function generates the
// correct scripts for the various types of addresses.
func TestPayToAddrScript(t *testing.T) {
	t.Parallel()

	// TCq7ZvyjTugZ3xDY8m1Mdgm95v4QmMpMfm3Fg8GCeE1uf
	provaTest, err := provautil.NewAddressProva(
		decodeHex("35dbbf04bca061e49dace08f858d8775c0a57c8e"),
		[]btcec.KeyID{0x10000, 1}, &chaincfg.TestNetParams)
	if err != nil {
		t.Fatalf("Unable to create prova address: %v", err)
	}

	errUnsupportedAddress := scriptError(ErrUnsupportedAddress, "")

	tests := []struct {
		in       provautil.Address
		expected string
		err      error
	}{
		// pay-to-pubkey-hash address on mainnet
		{
			provaTest,
			"521435dbbf04bca061e49dace08f858d8775c0a57c8e030000015153ba",
			nil,
		},

		// Supported address types with nil pointers.
		{(*provautil.AddressProva)(nil), "", errUnsupportedAddress},

		// Unsupported address type.
		{&bogusAddress{}, "", errUnsupportedAddress},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		pkScript, err := PayToAddrScript(test.in)
		if e := tstCheckScriptError(err, test.err); e != nil {
			t.Fatalf("PayToAddrScript #%d unexpected error - "+
				"got %v, want %v", i, err, test.err)
		}

		expected := decodeHex(test.expected)
		if !bytes.Equal(pkScript, expected) {
			t.Fatalf("PayToAddrScript #%d got: %x\nwant: %x",
				i, pkScript, expected)
		}
	}
}

// TestMultiSigScript ensures the MultiSigScript function returns the expected
// scripts and errors.
func TestMultiSigScript(t *testing.T) {
	t.Parallel()

	//  mainnet p2pk 13CG6SJ3yHUXo4Cr2RY4THLLJrNFuG3gUg
	p2pkCompressedMain, err := provautil.NewAddressPubKey(decodeHex("02192d7"+
		"4d0cb94344c9569c2e77901573d8d7903c3ebec3a957724895dca52c6b4"),
		&chaincfg.MainNetParams)
	if err != nil {
		t.Fatalf("Unable to create pubkey address (compressed): %v",
			err)
	}
	p2pkCompressed2Main, err := provautil.NewAddressPubKey(decodeHex("03b0bd"+
		"634234abbb1ba1e986e884185c61cf43e001f9137f23c2c409273eb16e65"),
		&chaincfg.MainNetParams)
	if err != nil {
		t.Fatalf("Unable to create pubkey address (compressed 2): %v",
			err)
	}

	p2pkUncompressedMain, err := provautil.NewAddressPubKey(decodeHex("0411d"+
		"b93e1dcdb8a016b49840f8c53bc1eb68a382e97b1482ecad7b148a6909a5c"+
		"b2e0eaddfb84ccf9744464f82e160bfa9b8b64f9d4c03f999b8643f656b41"+
		"2a3"), &chaincfg.MainNetParams)
	if err != nil {
		t.Fatalf("Unable to create pubkey address (uncompressed): %v",
			err)
	}

	tests := []struct {
		keys      []*provautil.AddressPubKey
		nrequired int
		expected  string
		err       error
	}{
		{
			[]*provautil.AddressPubKey{
				p2pkCompressedMain,
				p2pkCompressed2Main,
			},
			1,
			"1 DATA_33 0x02192d74d0cb94344c9569c2e77901573d8d7903c" +
				"3ebec3a957724895dca52c6b4 DATA_33 0x03b0bd634" +
				"234abbb1ba1e986e884185c61cf43e001f9137f23c2c4" +
				"09273eb16e65 2 CHECKMULTISIG",
			nil,
		},
		{
			[]*provautil.AddressPubKey{
				p2pkCompressedMain,
				p2pkCompressed2Main,
			},
			2,
			"2 DATA_33 0x02192d74d0cb94344c9569c2e77901573d8d7903c" +
				"3ebec3a957724895dca52c6b4 DATA_33 0x03b0bd634" +
				"234abbb1ba1e986e884185c61cf43e001f9137f23c2c4" +
				"09273eb16e65 2 CHECKMULTISIG",
			nil,
		},
		{
			[]*provautil.AddressPubKey{
				p2pkCompressedMain,
				p2pkCompressed2Main,
			},
			3,
			"",
			scriptError(ErrTooManyRequiredSigs, ""),
		},
		{
			[]*provautil.AddressPubKey{
				p2pkUncompressedMain,
			},
			1,
			"1 DATA_65 0x0411db93e1dcdb8a016b49840f8c53bc1eb68a382" +
				"e97b1482ecad7b148a6909a5cb2e0eaddfb84ccf97444" +
				"64f82e160bfa9b8b64f9d4c03f999b8643f656b412a3 " +
				"1 CHECKMULTISIG",
			nil,
		},
		{
			[]*provautil.AddressPubKey{
				p2pkUncompressedMain,
			},
			2,
			"",
			scriptError(ErrTooManyRequiredSigs, ""),
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		script, err := MultiSigScript(test.keys, test.nrequired)
		if e := tstCheckScriptError(err, test.err); e != nil {
			t.Errorf("MultiSigScript #%d: %v", i, e)
			continue
		}

		expected := mustParseShortForm(test.expected)
		if !bytes.Equal(script, expected) {
			t.Errorf("MultiSigScript #%d got: %x\nwant: %x",
				i, script, expected)
			continue
		}
	}
}

// TestCalcMultiSigStats ensures the CalcMutliSigStats function returns the
// expected errors.
func TestCalcMultiSigStats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		script string
		err    error
	}{
		{
			name: "short script",
			script: "0x046708afdb0fe5548271967f1a67130b7105cd6a828" +
				"e03909a67962e0ea1f61d",
			err: scriptError(ErrMalformedPush, ""),
		},
		{
			name: "stack underflow",
			script: "RETURN DATA_41 0x046708afdb0fe5548271967f1a" +
				"67130b7105cd6a828e03909a67962e0ea1f61deb649f6" +
				"bc3f4cef308",
			err: scriptError(ErrNotMultisigScript, ""),
		},
		{
			name: "multisig script",
			script: "0 DATA_72 0x30450220106a3e4ef0b51b764a2887226" +
				"2ffef55846514dacbdcbbdd652c849d395b4384022100" +
				"e03ae554c3cbb40600d31dd46fc33f25e47bf8525b1fe" +
				"07282e3b6ecb5f3bb2801 CODESEPARATOR 1 DATA_33 " +
				"0x0232abdc893e7f0631364d7fd01cb33d24da45329a0" +
				"0357b3a7886211ab414d55a 1 CHECKMULTISIG",
			err: nil,
		},
	}

	for i, test := range tests {
		script := mustParseShortForm(test.script)
		_, _, err := CalcMultiSigStats(script)
		if e := tstCheckScriptError(err, test.err); e != nil {
			t.Errorf("CalcMultiSigStats #%d (%s): %v", i, test.name,
				e)
			continue
		}
	}
}

// scriptClassTests houses several test scripts used to ensure various class
// determination is working as expected.  It's defined as a test global versus
// inside a function scope since this spans both the standard tests and the
// consensus tests (pay-to-script-hash is part of consensus).
var scriptClassTests = []struct {
	name   string
	script string
	class  ScriptClass
}{
	{
		name: "Pay Pubkey",
		script: "DATA_65 0x0411db93e1dcdb8a016b49840f8c53bc1eb68a382e" +
			"97b1482ecad7b148a6909a5cb2e0eaddfb84ccf9744464f82e16" +
			"0bfa9b8b64f9d4c03f999b8643f656b412a3 CHECKSIG",
		class: NonStandardTy,
	},
	// tx 599e47a8114fe098103663029548811d2651991b62397e057f0c863c2bc9f9ea
	{
		name: "Pay PubkeyHash",
		script: "DUP HASH160 DATA_20 0x660d4ef3a743e3e696ad990364e555" +
			"c271ad504b EQUALVERIFY CHECKSIG",
		class: NonStandardTy,
	},
	// part of tx 6d36bc17e947ce00bb6f12f8e7a56a1585c5a36188ffa2b05e10b4743273a74b
	// codeseparator parts have been elided. (bitcoin core's checks for
	// multisig type doesn't have codesep either).
	{
		name: "multisig",
		script: "1 DATA_33 0x0232abdc893e7f0631364d7fd01cb33d24da4" +
			"5329a00357b3a7886211ab414d55a 1 CHECKMULTISIG",
		class: NonStandardTy,
	},
	// tx e5779b9e78f9650debc2893fd9636d827b26b4ddfa6a8172fe8708c924f5c39d
	{
		name: "P2SH",
		script: "HASH160 DATA_20 0x433ec2ac1ffa1b7b7d027f564529c57197f" +
			"9ae88 EQUAL",
		class: NonStandardTy,
	},
	{
		// Nulldata with no data at all.
		name:   "nulldata with no data",
		script: "RETURN",
		class:  NullDataTy,
	},
	{
		// Nulldata with single zero push.
		name:   "nulldata zero",
		script: "RETURN 0",
		class:  NullDataTy,
	},
	{
		// Nulldata with small integer push.
		name:   "nulldata small int",
		script: "RETURN 1",
		class:  NullDataTy,
	},
	{
		// Nulldata with max small integer push.
		name:   "nulldata max small int",
		script: "RETURN 16",
		class:  NullDataTy,
	},
	{
		// Nulldata with small data push.
		name:   "nulldata small data",
		script: "RETURN DATA_8 0x046708afdb0fe554",
		class:  NullDataTy,
	},
	{
		// Canonical nulldata with 60-byte data push.
		name: "canonical nulldata 60-byte push",
		script: "RETURN 0x3c 0x046708afdb0fe5548271967f1a67130b7105cd" +
			"6a828e03909a67962e0ea1f61deb649f6bc3f4cef3046708afdb" +
			"0fe5548271967f1a67130b7105cd6a",
		class: NullDataTy,
	},
	{
		// Non-canonical nulldata with 60-byte data push.
		name: "non-canonical nulldata 60-byte push",
		script: "RETURN PUSHDATA1 0x3c 0x046708afdb0fe5548271967f1a67" +
			"130b7105cd6a828e03909a67962e0ea1f61deb649f6bc3f4cef3" +
			"046708afdb0fe5548271967f1a67130b7105cd6a",
		class: NullDataTy,
	},
	{
		// Nulldata with max allowed data to be considered standard.
		name: "nulldata max standard push",
		script: "RETURN PUSHDATA1 0x50 0x046708afdb0fe5548271967f1a67" +
			"130b7105cd6a828e03909a67962e0ea1f61deb649f6bc3f4cef3" +
			"046708afdb0fe5548271967f1a67130b7105cd6a828e03909a67" +
			"962e0ea1f61deb649f6bc3f4cef3",
		class: NullDataTy,
	},
	{
		// Nulldata with more than max allowed data to be considered
		// standard (so therefore nonstandard)
		name: "nulldata exceed max standard push",
		script: "RETURN PUSHDATA1 0x51 0x046708afdb0fe5548271967f1a67" +
			"130b7105cd6a828e03909a67962e0ea1f61deb649f6bc3f4cef3" +
			"046708afdb0fe5548271967f1a67130b7105cd6a828e03909a67" +
			"962e0ea1f61deb649f6bc3f4cef308",
		class: NonStandardTy,
	},
	{
		// Almost nulldata, but add an additional opcode after the data
		// to make it nonstandard.
		name:   "almost nulldata",
		script: "RETURN 4 TRUE",
		class:  NonStandardTy,
	},

	// The next few are almost multisig (it is the more complex script type)
	// but with various changes to make it fail.
	{
		// Multisig but invalid nsigs.
		name: "strange 1",
		script: "DUP DATA_33 0x0232abdc893e7f0631364d7fd01cb33d24da45" +
			"329a00357b3a7886211ab414d55a 1 CHECKSAFEMULTISIG",
		class: NonStandardTy,
	},
	{
		// Multisig but invalid pubkey.
		name:   "strange 2",
		script: "1 1 1 CHECKSAFEMULTISIG",
		class:  NonStandardTy,
	},
	{
		// Multisig but no matching npubkeys opcode.
		name: "strange 3",
		script: "1 DATA_33 0x0232abdc893e7f0631364d7fd01cb33d24da4532" +
			"9a00357b3a7886211ab414d55a DATA_33 0x0232abdc893e7f0" +
			"631364d7fd01cb33d24da45329a00357b3a7886211ab414d55a " +
			"CHECKSAFEMULTISIG",
		class: NonStandardTy,
	},
	{
		// Multisig but with multisigverify.
		name: "strange 4",
		script: "1 DATA_33 0x0232abdc893e7f0631364d7fd01cb33d24da4532" +
			"9a00357b3a7886211ab414d55a 1 CHECKSAFEMULTISIG",
		class: NonStandardTy,
	},
	{
		// Multisig but wrong length.
		name:   "strange 5",
		script: "1 CHECKSAFEMULTISIG",
		class:  NonStandardTy,
	},
	{
		name:   "doesn't parse",
		script: "DATA_5 0x01020304",
		class:  NonStandardTy,
	},
	{
		name: "multisig script with wrong number of pubkeys",
		script: "2 " +
			"DATA_33 " +
			"0x027adf5df7c965a2d46203c781bd4dd8" +
			"21f11844136f6673af7cc5a4a05cd29380 " +
			"DATA_33 " +
			"0x02c08f3de8ee2de9be7bd770f4c10eb0" +
			"d6ff1dd81ee96eedd3a9d4aeaf86695e80 " +
			"3 CHECKSAFEMULTISIG",
		class: NonStandardTy,
	},
	{
		name: "standard prova script",
		script: "2 DATA_20 0x433ec2ac1ffa1b7b7d027f564529c57197f" +
			"9ae88 1 2 3 CHECKSAFEMULTISIG",
		class: ProvaTy,
	},
	{
		name: "prova script with additional key ids",
		script: "2 DATA_20 0x433ec2ac1ffa1b7b7d027f564529c57197f" +
			"9ae88 1 2 3 4 5 CHECKSAFEMULTISIG",
		class: GeneralProvaTy,
	},
	{
		name:   "prova admin script",
		script: "0 CHECKTHREAD",
		class:  ProvaAdminTy,
	},
}

// TestScriptClass ensures all the scripts in scriptClassTests have the expected
// class.
func TestScriptClass(t *testing.T) {
	t.Parallel()

	for _, test := range scriptClassTests {
		script := mustParseShortForm(test.script)
		class := GetScriptClass(script)
		if class != test.class {
			t.Errorf("%s: expected %s got %s (script %x)", test.name,
				test.class, class, script)
			continue
		}
	}
}

// TestStringifyClass ensures the script class string returns the expected
// string for each script class.
func TestStringifyClass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		class    ScriptClass
		stringed string
	}{
		{
			name:     "nonstandardty",
			class:    NonStandardTy,
			stringed: "nonstandard",
		},
		{
			name:     "nulldataty",
			class:    NullDataTy,
			stringed: "nulldata",
		},
		{
			name:     "broken",
			class:    ScriptClass(255),
			stringed: "Invalid",
		},
	}

	for _, test := range tests {
		typeString := test.class.String()
		if typeString != test.stringed {
			t.Errorf("%s: got %#q, want %#q", test.name,
				typeString, test.stringed)
		}
	}
}

// TestNullDataScript tests whether NullDataScript returns a valid script.
func TestNullDataScript(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected []byte
		err      error
		class    ScriptClass
	}{
		{
			name:     "small int",
			data:     hexToBytes("01"),
			expected: mustParseShortForm("RETURN 1"),
			err:      nil,
			class:    NullDataTy,
		},
		{
			name:     "max small int",
			data:     hexToBytes("10"),
			expected: mustParseShortForm("RETURN 16"),
			err:      nil,
			class:    NullDataTy,
		},
		{
			name: "data of size before OP_PUSHDATA1 is needed",
			data: hexToBytes("0102030405060708090a0b0c0d0e0f10111" +
				"2131415161718"),
			expected: mustParseShortForm("RETURN 0x18 0x01020304" +
				"05060708090a0b0c0d0e0f101112131415161718"),
			err:   nil,
			class: NullDataTy,
		},
		{
			name: "just right",
			data: hexToBytes("000102030405060708090a0b0c0d0e0f101" +
				"112131415161718191a1b1c1d1e1f202122232425262" +
				"728292a2b2c2d2e2f303132333435363738393a3b3c3" +
				"d3e3f404142434445464748494a4b4c4d4e4f"),
			expected: mustParseShortForm("RETURN PUSHDATA1 0x50 " +
				"0x000102030405060708090a0b0c0d0e0f101112131" +
				"415161718191a1b1c1d1e1f20212223242526272829" +
				"2a2b2c2d2e2f303132333435363738393a3b3c3d3e3" +
				"f404142434445464748494a4b4c4d4e4f"),
			err:   nil,
			class: NullDataTy,
		},
		{
			name: "too big",
			data: hexToBytes("000102030405060708090a0b0c0d0e0f101" +
				"112131415161718191a1b1c1d1e1f202122232425262" +
				"728292a2b2c2d2e2f303132333435363738393a3b3c3" +
				"d3e3f404142434445464748494a4b4c4d4e4f50"),
			expected: nil,
			err:      scriptError(ErrTooMuchNullData, ""),
			class:    NonStandardTy,
		},
	}

	for i, test := range tests {
		script, err := NullDataScript(test.data)
		if e := tstCheckScriptError(err, test.err); e != nil {
			t.Errorf("NullDataScript: #%d (%s): %v", i, test.name,
				e)
			continue

		}

		// Check that the expected result was returned.
		if !bytes.Equal(script, test.expected) {
			t.Errorf("NullDataScript: #%d (%s) wrong result\n"+
				"got: %x\nwant: %x", i, test.name, script,
				test.expected)
			continue
		}

		// Check that the script has the correct type.
		scriptType := GetScriptClass(script)
		if scriptType != test.class {
			t.Errorf("GetScriptClass: #%d (%s) wrong result -- "+
				"got: %v, want: %v", i, test.name, scriptType,
				test.class)
			continue
		}
	}
}
