// Copyright (c) 2014-2016 The btcsuite developers
// Copyright (c) 2016 The Zcash developers
// Copyright (c) 2017 BitGo
// Copyright (c) 2019 Tranquility Node Ltd
// Copyright (c) 2020 Pyx Partners
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package chaincfg

import (
	"encoding/hex"
	"errors"
	"math"
	"math/big"
	"time"

	"github.com/pyx-partners/dmgd/btcec"
	"github.com/pyx-partners/dmgd/chaincfg/chainhash"
	"github.com/pyx-partners/dmgd/wire"
)

// These variables are the chain proof-of-work limit parameters for each default
// network.
var (
	// bigOne is 1 represented as a big.Int.  It is defined here to avoid
	// the overhead of creating it multiple times.
	bigOne = big.NewInt(1)

	// mainPowLimit is the highest proof of work value a block can
	// have for the main network.  It is the value 2^243 - 1.
	mainPowLimit = new(big.Int).Sub(new(big.Int).Lsh(bigOne, 243), bigOne)

	// regressionPowLimit is the highest proof of work value a block
	// can have for the regression test network.
	regressionPowLimit = powLimitFromStr("0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f")

	// testNetPowLimit is the highest proof of work value a block can have
	// for the test network (version 3).
	testNetPowLimit = powLimitFromStr("07ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")

	// simNetPowLimit is the highest proof of work value a Bitcoin block
	// can have for the simulation test network.  It is the value 2^255 - 1.
	simNetPowLimit = new(big.Int).Sub(new(big.Int).Lsh(bigOne, 255), bigOne)
)

// Checkpoint identifies a known good point in the block chain.  Using
// checkpoints allows a few optimizations for old blocks during initial download
// and also prevents forks from old blocks.
//
// Each checkpoint is selected based upon several factors.  See the
// documentation for blockchain.IsCheckpointCandidate for details on the
// selection criteria.
type Checkpoint struct {
	Height uint32
	Hash   *chainhash.Hash
}

// DNSSeed identifies a DNS seed.
type DNSSeed struct {
	// Host defines the hostname of the seed.
	Host string

	// HasFiltering defines whether the seed supports filtering
	// by service flags (wire.ServiceFlag).
	HasFiltering bool
}

// Params defines a Bitcoin network by its parameters.  These parameters may be
// used by Bitcoin applications to differentiate networks as well as addresses
// and keys for one network from those intended for use on another network.
type Params struct {
	// Name defines a human-readable identifier for the network.
	Name string

	// Net defines the magic bytes used to identify the network.
	Net wire.BitcoinNet

	// DefaultPort defines the default peer-to-peer port for the network.
	DefaultPort string

	// DNSSeeds defines a list of DNS seeds for the network that are used
	// as one method to discover peers.
	DNSSeeds []DNSSeed

	// GenesisBlock defines the first block of the chain.
	GenesisBlock *wire.MsgBlock

	// GenesisHash is the starting block hash.
	GenesisHash *chainhash.Hash

	// AdminKeySets is the set of keys governing the chain state.
	AdminKeySets map[btcec.KeySetType]btcec.PublicKeySet

	// ASPKeyIdMap are the provisioned keyIDs and respective pubKeys
	ASPKeyIdMap btcec.KeyIdMap

	// PowLimit defines the highest allowed proof of work value for a block
	// as a uint256.
	PowLimit *big.Int

	// PowLimitBits defines the highest allowed proof of work value for a
	// block in compact form.
	PowLimitBits uint32

	// CoinbaseMaturity is the number of blocks required before newly mined
	// coins (coinbase transactions) can be spent.
	CoinbaseMaturity uint16

	// SubsidyReductionInterval is the interval of blocks before the subsidy
	// is reduced.
	SubsidyReductionInterval uint32

	// TargetTimePerBlock is the desired amount of time to generate each
	// block.
	TargetTimePerBlock time.Duration

	// GenerateSupported specifies whether or not CPU mining is allowed.
	GenerateSupported bool

	// Checkpoints ordered from oldest to newest.
	Checkpoints []Checkpoint

	// Enforce current block version once network has
	// upgraded.  This is part of BIP0034.
	BlockEnforceNumRequired uint64

	// Reject previous block versions once network has
	// upgraded.  This is part of BIP0034.
	BlockRejectNumRequired uint64

	// The number of nodes to check.  This is part of BIP0034.
	BlockUpgradeNumToCheck uint64

	// Mempool parameters
	RelayNonStdTxs bool

	// Address encoding magics
	ProvaAddrID  byte // First byte of an Prova address
	PrivateKeyID byte // First byte of a WIF private key

	// BIP32 hierarchical deterministic extended key magics
	HDPrivateKeyID [4]byte
	HDPublicKeyID  [4]byte

	// BIP44 coin type used in the hierarchical deterministic path for
	// address generation.
	HDCoinType uint32

	// Number of blocks for the moving window of difficulty adjustment.
	PowAveragingWindow int

	// Maximum downward adjustment in pow difficulty, as a percentage.
	PowMaxAdjustDown int64

	// Maximum upward adjustment in pow difficulty, as a percentage.
	PowMaxAdjustUp int64

	// Maximum blocks signed by a single validate key in averaging window.
	ChainWindowMaxBlocks int

	// Maximum fee allowed in a single transaction, in atoms.
	MaximumFeeAmount int64
}

// MaxActualTimespan returns a timespan with the down-dampening factor applied.
func (p Params) MaxActualTimespan() time.Duration {
	dampenPercentage := time.Duration(100 + p.PowMaxAdjustDown)
	return (p.AveragingWindowTimespan() * dampenPercentage) / 100
}

// MinActualTimespan returns a timespan with the up dampening factor applied.
func (p Params) MinActualTimespan() time.Duration {
	dampenPercentage := time.Duration(100 - p.PowMaxAdjustUp)
	return (p.AveragingWindowTimespan() * dampenPercentage) / 100
}

// MinValidateKeySetSize returns the minimum number of validate keys required
// to progress the chain, given the ChainWindowShareLimit.
func (p Params) MinValidateKeySetSize() int {
	powAveragingWindow := float64(p.PowAveragingWindow)
	chainWindowMaxBlocks := float64(p.ChainWindowMaxBlocks)
	return int(math.Ceil(powAveragingWindow / chainWindowMaxBlocks))
}

// AveragingWindowTimespan returns the difficulty timespan to be averaged over.
func (p Params) AveragingWindowTimespan() time.Duration {
	return time.Duration(p.PowAveragingWindow) * p.TargetTimePerBlock
}

// hexToBytes converts the passed hex string into bytes and will panic if there
// is an error.  This is only provided for the hard-coded constants so errors in
// the source code can be detected. It will only (and must only) be called with
// hard-coded values.
func hexToBytes(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic("invalid hex in source file: " + s)
	}
	return b
}

// MainNetParams defines the network parameters for the main Bitcoin network.
var MainNetParams = Params{
	Name:        "mainnet",
	Net:         wire.MainNet,
	DefaultPort: "6464",
	DNSSeeds: []DNSSeed{
		{"mainnet.digitalmint.gold", false},
	},

	// Chain parameters
	GenesisBlock: &genesisBlock,
	GenesisHash:  &genesisHash,
	AdminKeySets: func() map[btcec.KeySetType]btcec.PublicKeySet {
		keySets := make(map[btcec.KeySetType]btcec.PublicKeySet)

		// Root keys
		keySets[btcec.RootKeySet], _ = btcec.ParsePubKeySet(btcec.S256(),
			// TRM Keys
			"024d31e55e3f6c93b11787bff2a5a2c671eb1b3deca35e4c53260a65ea4be6bf9d",
			"03ac5c0c5e34bcdc4b97a68d8126b15ae311278877c55ebcec34839b1b761ff9db",
		)

		// Provision Keys
		keySets[btcec.ProvisionKeySet], _ = btcec.ParsePubKeySet(btcec.S256(),
			// BitGo Key
			"024b175f51c0b65b159e52f068f1e8b24e487ec9723967268a41ea952f3e2ed30c",
			// TRM Key
			"03e27301983513e043792772f9ba6dceb443ac1f07e46753e95786fb48176bf41b",
		)

		// Issue Keys
		keySets[btcec.IssueKeySet], _ = btcec.ParsePubKeySet(btcec.S256(),
			// BitGo Key
			"024b175f51c0b65b159e52f068f1e8b24e487ec9723967268a41ea952f3e2ed30c",
			// TRM Key
			"0248707c5d4267a6b340e108d14aaec1d1ec0a800fc9dd75a8318b3d7eb198590d",
		)

		// Validate keys
		keySets[btcec.ValidateKeySet], _ = btcec.ParsePubKeySet(btcec.S256(),
			// AlphaPoint Validate Keys
			"0203f330c65512659eb4c37e879a230b085a35c5869d010337c43be95ef7e67cd9",
			"0280cfec2d2d07505a11c4aa8883924218af97e63a590b05c9abdac32bf1ab4092",
			"02f9eeafe3137ad8b6db4af4cd4d7166b0251698a117776ad7eed6a4428b3f0430",
			"0367ede4090dc7f79c016679ca507d5ca56da516bed8072c4367a7ad3869afca8c",
			"03b924a0ce7cfa109975019a6d736cfc1509b028d840745cea6231fa01a0e0fa49",
			"03d9c5b4e01f01a46bb565df463eece63d9085b6f42b09719398cb5344d99d0b32",
			// BitGo Validate Keys
			"03ca50d5cf8f56e4cc7e9b3618b4e603e6abc21b881dd74072d2bb9dc3fb9ffa17",
			"02f6d5048dd06f90add76fdf94c9a89c85adf3eee9c0ab6223ce729dcc8d8548ec",
			"02e1059aea17a90d14e41f92cbb89d54821ecc127ee34df5e2c2c4e936bb1fb512",
			"036986dd78e88c13f8a4e26709c027a3402b49eddd902dee759c2054607d4a9e37",
			"02099825c04dd5a81c04d391ca326f3cf7ac6259381169d06082a5c59cf95823c0",
			"0389102684dcc0a90b4cb7ba305b828b92e8eed416fffe70bad74bc102e342bb59",
			"036d8d4cb21aaab9d08193d6f4e9f8e60cf311cfd2ac2107f155a35b639efa49ac",
			"031f34e4c36ad48ade3daa6bfc461d9a40fc14f8bd9e6b91b52f8b59df8603b25f",
			"024c9ad978198253b0e64532a5ad5d57a883465b616cc5c42b3a1fb2168008181f",
			"0282f6f2d93bf088afd2edfdf9d3de1e28be45846d308db9c8e72e2b98c80a482a",
			"03615ed6b30e9487230b4a8db337187ce66e798d102672185e2d96daf7e3a91932",
			"0303e26ffe58433e20df2510532d1f3fde27f0081b6c36402b02300e8f6eecf15c",
			// CME Validate Keys
			"02e0fe48709edee99facb8307839947066d75c566e5fc18e71a63237aba16bd959",
			"022e3548372407b202509092a269da610c9e75db913d00a6d00a100b1c729e55ee",
			"0312995470c43908102aafd388fb657f2add89472e59d42ed688da639a54c2211f",
			"02a524f154a8399f98813608c817358dadea1da81469a5d8923c4b8a36ebd9900e",
			"0240025bfab065229e0e609c24212aa8777c159169c76729817e107e1c000347ee",
			"02c9f05dff6434eacf1095a76c09785b7611ca9610d306d0b710852b820aa76825",
			// TRM Validate Keys
			"039a7557e07517edb5d4098f33df34a644b8e79d022fd4149e7d85b6ed3e0f4959",
			"02236dd183e02ffaa59a1394c27fa83d6fb57336ba2c3da18ef0acd763628b54c2",
			"03fc6d723b5e8d9e549bed31d9378e48bbe18b130d55057797ed9babc33bf7085a",
			"02a54aaf214e489c8847c42730052978ba27eb9054b3a3869ad267ad10e11183b7",
			"03c2210cd4d440bb7f6261ce4ad90ea3fc6eec6fe50a05f17ee3f81154df43cc2a",
			"0299c6d6255eddb1eff2a971625e23dddb00c07f6fdf8a362f909fecb767e87cd4",
		)

		return keySets
	}(),
	ASPKeyIdMap: func() btcec.KeyIdMap {
		// BitGo ASP Key
		pubKey1, _ := btcec.ParsePubKey(hexToBytes("033fa570adba7413fbe0fb90f358e823b003371c47dd4e3769028e122f40ea7496"), btcec.S256())
		// TRM RRS Key
		pubKey2, _ := btcec.ParsePubKey(hexToBytes("0202a0aa7a9b3467fa2b934b751ec35cb8cfe031e2f4f304add323eadd5db282c9"), btcec.S256())
		return map[btcec.KeyID]*btcec.PublicKey{btcec.KeyID(1): pubKey1, btcec.KeyID(2): pubKey2}
	}(),
	PowLimit:                 mainPowLimit,
	PowLimitBits:             0x1f07ffff,
	CoinbaseMaturity:         100,
	SubsidyReductionInterval: 210000,
	TargetTimePerBlock:       time.Second * 150, // 2.5 minutes
	GenerateSupported:        true,

	// Checkpoints ordered from oldest to newest.
	Checkpoints: []Checkpoint{},

	// Enforce current block version once majority of the network has
	// upgraded.
	// 75% (750 / 1000)
	// Reject previous block versions once a majority of the network has
	// upgraded.
	// 95% (950 / 1000)
	BlockEnforceNumRequired: 750,
	BlockRejectNumRequired:  950,
	BlockUpgradeNumToCheck:  1000,

	// Mempool parameters
	RelayNonStdTxs: false,

	// Address encoding magics
	PrivateKeyID: 0x80, // starts with 5 (uncompressed) or K (compressed)
	ProvaAddrID:  0x33, // starts with G

	// BIP32 hierarchical deterministic extended key magics
	HDPrivateKeyID: [4]byte{0x04, 0x88, 0xad, 0xe4}, // starts with xprv
	HDPublicKeyID:  [4]byte{0x04, 0x88, 0xb2, 0x1e}, // starts with xpub

	// BIP44 coin type used in the hierarchical deterministic path for
	// address generation.
	HDCoinType: 0,

	// Number of blocks for the moving window of difficulty adjustment.
	PowAveragingWindow: 31,

	// Maximum downward adjustment in pow difficulty, as a percentage.
	PowMaxAdjustDown: 32,

	// Maximum upward adjustment in pow difficulty, as a percentage.
	PowMaxAdjustUp: 16,

	// Maximum blocks signed by a single validate key in averaging window.
	ChainWindowMaxBlocks: 3,

	// Maximum fee allowed in a single transaction, in atoms.
	MaximumFeeAmount: 5000000,
}

// RegressionNetParams defines the network parameters for the regression test
// Bitcoin network.  Not to be confused with the test Bitcoin network.
var RegressionNetParams = Params{
	Name:        "regtest",
	Net:         wire.RegNet,
	DefaultPort: "18989",
	DNSSeeds:    []DNSSeed{},

	// Chain parameters
	GenesisBlock: &regTestGenesisBlock,
	GenesisHash:  &regTestGenesisHash,
	AdminKeySets: func() map[btcec.KeySetType]btcec.PublicKeySet {
		keySets := make(map[btcec.KeySetType]btcec.PublicKeySet)

		// Root keys
		keySets[btcec.RootKeySet], _ = btcec.ParsePubKeySet(btcec.S256(),
			"025ceeba2ab4a635df2c0301a3d773da06ac5a18a7c3e0d09a795d7e57d233edf1", // priv eaf02ca348c524e6392655ba4d29603cd1a7347d9d65cfe93ce1ebffdca22694
			"038ef4a121bcaf1b1f175557a12896f8bc93b095e84817f90e9a901cd2113a8202", // priv 2b8c52b77b327c755b9b375500d3f4b2da9b0a1ff65f6891d311fe94295bc26a
		)

		// Provision Keys
		keySets[btcec.ProvisionKeySet], _ = btcec.ParsePubKeySet(btcec.S256(),
			"0248b3b4e579444e6b7cc414510109316c4c9ba7a2a46f50f8dcbb273efb1337ab", // priv f954b388f5db3a1d2915cda434206d791b47cf3d4e78cc32fbeb77ea25d20d7d
			"02ef86c70ae6afd2dd2f0efb07ea59789c27a1bf43f687b35dac435b539e1337ab", // priv 627f6f1d5d8f38bd60b6aaea2f74c72917deffcc2a5a64f67d3e0a28a2d711c1
		)

		// Issue Keys
		keySets[btcec.IssueKeySet], _ = btcec.ParsePubKeySet(btcec.S256(),
			"02ef7739dc67d38f2804a9c0aa1add89a992ced5f37e580ea8ccbb5742391337ab", // priv 3f9222ab4d30b1795941d9815e5833a4da70cb04bff59a5fd2ddc4641e58607e
			"021126d3d6158cf4f47eb2e08d12e9fa46d8da7b7e401220260bdc46446f1337ab", // priv 0a40defde0e49e1f78edb9cea5c499f704fabc140d6fd1a4df8405365e2e4f0f
		)

		// Validate keys
		keySets[btcec.ValidateKeySet], _ = btcec.ParsePubKeySet(btcec.S256(),
			"035f5103852bd7d9c9c28e44caf1f7188941e16295062ca4c89928a8ccff993cd3", // TODO(prova) add priv
			"0265de49399e78020026219492e2a6e1a41e93591b87220ae8a2f3ebf3473dbeef", // TODO(prova) add priv
			"039cb94c99c4700918250c40fa35b7fa0a75a967c9366aa19b8fc354373368beef", // TODO(prova) add priv
			"031337ab09070254638075c7b59643dce2d60c5260bf5841d2f8cc6f75f6790d4e", // TODO(prova) add priv
			"03133752072c8bc132679655c671b7953c2edabc575f42d60fa2e4caac09770061", // d36c82406d3c77ebc342aaa16f24a985fbfe63c75e6fd2afeffa1ba69632d252
			"02ab82d1531552c5b67e528047542ff9a2550ee4df7a88d67037754069db0541f9", // 05fa7a36092cc7accc8008365fd8d07229c794be2a4e9361c662b5cae9492fa3
			"02ab8264fdb09480d07b4db2d25db28065d51ea2950da89b7366d13b3c7be25e92", // a3262a6f506e4bfd4bc5b0708b2162e755410c8670e38c53928eb093ece2d37e
			"02ab829a899bb14365e45099062b9a9543e8c51d4ace1ddfb7ef52320bb6dabbdd", // 041bf76c17185bcddbbb5d40122d04528fbe6c68f488c16a4e85711410134b5e
			"02ab8201f1834926a5388ca4157f56008238b84fdcd0bd208647151ef617ac49ed", // 224688827325203eb53d0ec0f044b72312c8e11fc4fdada7b91416e7b54939d5
			"02ab821a995287881d383718476bf2305dee70174a4727688070701490c526061b", // c37e338bebe77d1ca77438ad7a382dc97c28703d793c732d88348eb5f26f9732
			"02ab82faed27170fc9c2dc3ab57ebba5c0d2649b6044011c77ca4b27aef05c6e07", // 6d4a926fec187ee0a0b0395cadb39360687b8416809c21ab32490e944784d6a3
		)

		return keySets
	}(),
	ASPKeyIdMap: func() btcec.KeyIdMap {
		pubKey1, _ := btcec.ParsePubKey(hexToBytes("025ceeba2ab4a635df2c0301a3d773da06ac5a18a7c3e0d09a795d7e57d233edf1"), btcec.S256())
		pubKey2, _ := btcec.ParsePubKey(hexToBytes("038ef4a121bcaf1b1f175557a12896f8bc93b095e84817f90e9a901cd2113a8202"), btcec.S256())
		return map[btcec.KeyID]*btcec.PublicKey{btcec.KeyID(1): pubKey1, btcec.KeyID(2): pubKey2}
	}(),
	PowLimit:                 regressionPowLimit,
	PowLimitBits:             0x200f0f0f,
	CoinbaseMaturity:         100,
	SubsidyReductionInterval: 150,
	TargetTimePerBlock:       time.Minute, // 1 minute
	GenerateSupported:        true,

	// Enforce current block version once majority of the network has
	// upgraded.
	// 75% (750 / 1000)
	// Reject previous block versions once a majority of the network has
	// upgraded.
	// 95% (950 / 1000)
	BlockEnforceNumRequired: 750,
	BlockRejectNumRequired:  950,
	BlockUpgradeNumToCheck:  1000,

	// Mempool parameters
	RelayNonStdTxs: false,

	// Address encoding magics
	ProvaAddrID:  0x58, // starts with T
	PrivateKeyID: 0xef, // starts with 9 (uncompressed) or c (compressed)

	// BIP32 hierarchical deterministic extended key magics
	HDPrivateKeyID: [4]byte{0x04, 0x35, 0x83, 0x94}, // starts with tprv
	HDPublicKeyID:  [4]byte{0x04, 0x35, 0x87, 0xcf}, // starts with tpub

	// BIP44 coin type used in the hierarchical deterministic path for
	// address generation.
	HDCoinType: 1,

	// Number of blocks for the moving window of difficulty adjustment
	PowAveragingWindow: 31,

	// Maximum downward adjustment in pow difficulty, as a percentage
	PowMaxAdjustDown: 32,

	// Maximum upward adjustment in pow difficulty, as a percentage
	PowMaxAdjustUp: 16,

	// Maximum fee allowed in a single transaction, in atoms.
	MaximumFeeAmount: 5000000,
}

// TestNetParams defines the network parameters for the test network.
var TestNetParams = Params{
	Name:        "testnet",
	Net:         wire.TestNet,
	DefaultPort: "16464",
	DNSSeeds: []DNSSeed{
		{"testnet.digitalmint.gold", false},
	},

	// Chain parameters
	GenesisBlock: &testNetGenesisBlock,
	GenesisHash:  &testNetGenesisHash,
	AdminKeySets: func() map[btcec.KeySetType]btcec.PublicKeySet {
		keySets := make(map[btcec.KeySetType]btcec.PublicKeySet)

		// Issue keys
		keySets[btcec.IssueKeySet], _ = btcec.ParsePubKeySet(btcec.S256(),
			// BitGo Keys
			"029cd0486fd4a5b260f956e1b16db17cd0e2f8914054b30eddda17950af7033855",
			"034d3083c7ad8537d5397fbb65aa6794e5db24d8c4ed3a1f4e12285a447c83ffc7",

			// TRM Keys
			"0358feab3764c99a46cbe403406a473485adafb0f195aac23ab2669eff3409284f",

			// Pyx Keys
			"038ab498ba67635c5025d06fb94a912c022184a1d9aed951813f056f5ba0411130",
			"0289841a7a5c335614296fcf8ae233be08dcf71f90f0311e1ebb40ca4bd01b1073",
			"03e11633439ec9f0698c34dd14913e76812a771b690adfa45e3ad27964954a8220",
			"03fceb2fe3ad2703c1859d01c3fb72523c5c93a19f732e136a375303e5fa6ef22b",
		)

		// Provision keys
		keySets[btcec.ProvisionKeySet], _ = btcec.ParsePubKeySet(btcec.S256(),
			// BitGo Keys
			"029cd0486fd4a5b260f956e1b16db17cd0e2f8914054b30eddda17950af7033855",
			"0307051d31f87efb84839115c09e6160cf7b9ac210860202792e1cea7bd32a7051",

			// TRM Keys
			"0321fa2bf73dcab470f7e0da5caf64b5ddd670a3833bdac5f7ed1de11a1215808f",

			// Pyx keys
			"031a14a8283cfb7d1e0f4ab6588f543f45529c80c7571f2b934527bd6c2c8efb3f",
			"03dded09c792230c87b4a16e4edcb27156627068bf678208b26ea82bd81879ef5f",
		)

		// Root keys
		keySets[btcec.RootKeySet], _ = btcec.ParsePubKeySet(btcec.S256(),
			// Pyx Keys
			"025d22ab7b6a80032171b486ba8a72dcaef9d653080f7a625fdc9e63aba60d9f85",
			"0392ff35e8302a926d4a85a9e30ae3865e5ebb772bf21a005fb3a6d33c022b64af",
			"02de6e479321206ff21f5a9b7b5fc78d10f7de1ac148da7c7ccfaa0b2c34a135f9",
			"02415747ae40bcc4e16237c98a31da75adc3f6440ebcd7a65b522bfea3466122fe",
		)

		// Validate keys
		keySets[btcec.ValidateKeySet], _ = btcec.ParsePubKeySet(btcec.S256(),
			"031337b4828f61541f003634ebe6c55a9cca4d13c7fec34f2698939272a76b15b0",
			"031337e5bbe1b43b283ab56d5ceb64c159b0b5abd5ab58edd7d0ccad5360a23130",
			"031337b35bb33a8bfa9c190f617e139efef3d307bb1a981f17fd0ea6a42010f5ae",
			"031337eeb34fd8eb0af9e171c448a66ea632311cfe8316f7aed49b954ac6c054f7",
			"0372d56e2288fdd17a0975baa9607d5143f4b330b801142db16405ec961dab1337",
			"02615e88cd09e731816926cd7f16caad87b6b7ae54ef440d3ffb7d935eedab1337",
			"03d9e4d1bccd8efae3f74ae57258dd41c3c250e8b3806e7891a0a4d07fc3ab1337",
			"031337b1b33037ad401a189389f65bda1f9abd1497c956834c99213f0bd2d931ed",
			"031337ef04ae0d9b7f46f11889bbb0362d8477661a8ec5fabbaf85e9beb24eef20",
			"031337a189ad45659f2eebd521c43775e3d6c69e1c25b9b3498909e95c5d2da3b6",
			"031337e3ab1c86cbb6d67208b273098df863d1f3080947602d894cae4bc2d19d5c",
			// Pyx Keys
			"02f54ba86dc1ccb5bed0224d23f01ed87e4a443c47fc690d7797a13d41d2340e1a",
			"02bd690a99ceec5eb928c42097d10f16498598fc018e21d375f37ba724cdb04c9b",
			"02e3920c25d8f3ed64f18e9f91294c0c616597feeb4df8dacdfe487901b2216ec6",
			"02dd325fcdc0b2aa6c31688fa5b4169d6b16d8f16184d1553d7a235ed562797468",
			"02383e235cdeadd4ab9bc1f8081e6bd9d8b1fea4e379f1f617762ccfe22b00bcd8",
			"02d900626e566dce40a08e9a6b0d9698b024f0d8a48d4c19f38deb22dcc839e104",
			"03aeee9577cd5827d492346491dcf6adcb7a8d80924f6cbf62215b0e72054db8fd",
			"03d2f5d590e0dbae2e716518410c00752dad41aad68e2e29905df99dc04d084303",
			"030549bf1cf8e2fb73489bdb2ec0939aa226ba01948244e12186103e082d9e7813",
			"03e504cb6728a22f4650b2d6382b96ed59d027cc944a2429d4a8f467f6dd3cecbd",
			"0386e2d898b78ab554a71b9e86847c9757d145b42b1f4dbc3ef7bd0e365957023f",
		)

		return keySets
	}(),
	ASPKeyIdMap: func() btcec.KeyIdMap {
		// BitGo Keys
		pubKey1, _ := btcec.ParsePubKey(hexToBytes("036bcc8bc2af28edd3b1b8d25baefa0f06dd4fc243da0587268b5899d5538fc8a6"), btcec.S256())
		pubKey3, _ := btcec.ParsePubKey(hexToBytes("021497b39f2f32eeaa1083c52ee265d0fad85338fb82bf8c0ae4a1dbe746e4a45b"), btcec.S256())

		// TRM Keys
		pubKey2, _ := btcec.ParsePubKey(hexToBytes("02cea696aa3388a06a42ede7aab3c50b7229cc98659413c65178b52a86f7499635"), btcec.S256())

		return map[btcec.KeyID]*btcec.PublicKey{btcec.KeyID(1): pubKey1, btcec.KeyID(2): pubKey2, btcec.KeyID(3): pubKey3}
	}(),
	PowLimit:                 testNetPowLimit,
	PowLimitBits:             0x2007ffff,
	CoinbaseMaturity:         100,
	SubsidyReductionInterval: 210000,
	TargetTimePerBlock:       time.Second * 150, // 2.5 minutes
	GenerateSupported:        true,

	// Checkpoints ordered from oldest to newest.
	Checkpoints: []Checkpoint{},

	// Enforce current block version once majority of the network has
	// upgraded.
	// 51% (51 / 100)
	// Reject previous block versions once a majority of the network has
	// upgraded.
	// 75% (75 / 100)
	BlockEnforceNumRequired: 51,
	BlockRejectNumRequired:  75,
	BlockUpgradeNumToCheck:  100,

	// Mempool parameters
	RelayNonStdTxs: false,

	// Address encoding magics
	PrivateKeyID: 0xef, // starts with 9 (uncompressed) or c (compressed)
	ProvaAddrID:  0x58, // starts with T

	// BIP32 hierarchical deterministic extended key magics
	HDPrivateKeyID: [4]byte{0x04, 0x35, 0x83, 0x94}, // starts with tprv
	HDPublicKeyID:  [4]byte{0x04, 0x35, 0x87, 0xcf}, // starts with tpub

	// BIP44 coin type used in the hierarchical deterministic path for
	// address generation.
	HDCoinType: 1,

	// Number of blocks for the moving window of difficulty adjustment.
	PowAveragingWindow: 31,

	// Maximum downward adjustment in pow difficulty, as a percentage.
	PowMaxAdjustDown: 64,

	// Maximum upward adjustment in pow difficulty, as a percentage.
	PowMaxAdjustUp: 64,

	// Maximum blocks signed by a single validate key in averaging window.
	ChainWindowMaxBlocks: 3,

	// Maximum fee allowed in a single transaction, in atoms.
	MaximumFeeAmount: 5000000,
}

// SimNetParams defines the network parameters for the simulation test Bitcoin
// network.  This network is similar to the normal test network except it is
// intended for private use within a group of individuals doing simulation
// testing.  The functionality is intended to differ in that the only nodes
// which are specifically specified are used to create the network rather than
// following normal discovery rules.  This is important as otherwise it would
// just turn into another public testnet.
var SimNetParams = Params{
	Name:        "simnet",
	Net:         wire.SimNet,
	DefaultPort: "10079",
	DNSSeeds:    []DNSSeed{}, // NOTE: There must NOT be any seeds.

	// Chain parameters
	GenesisBlock:             &simNetGenesisBlock,
	GenesisHash:              &simNetGenesisHash,
	PowLimit:                 simNetPowLimit,
	PowLimitBits:             0x207fffff,
	CoinbaseMaturity:         100,
	SubsidyReductionInterval: 210000,
	TargetTimePerBlock:       time.Second * 150, // 2.5 minutes
	GenerateSupported:        true,

	// Checkpoints ordered from oldest to newest.
	Checkpoints: nil,

	// Enforce current block version once majority of the network has
	// upgraded.
	// 51% (51 / 100)
	// Reject previous block versions once a majority of the network has
	// upgraded.
	// 75% (75 / 100)
	BlockEnforceNumRequired: 51,
	BlockRejectNumRequired:  75,
	BlockUpgradeNumToCheck:  100,

	// Mempool parameters
	RelayNonStdTxs: false,

	// Address encoding magics
	PrivateKeyID: 0x64, // starts with 4 (uncompressed) or F (compressed)

	// BIP32 hierarchical deterministic extended key magics
	HDPrivateKeyID: [4]byte{0x04, 0x20, 0xb9, 0x00}, // starts with sprv
	HDPublicKeyID:  [4]byte{0x04, 0x20, 0xbd, 0x3a}, // starts with spub

	// BIP44 coin type used in the hierarchical deterministic path for
	// address generation.
	HDCoinType: 115, // ASCII for s

	// Number of blocks for the moving window of difficulty adjustment
	PowAveragingWindow: 31,

	// Maximum downward adjustment in pow difficulty, as a percentage
	PowMaxAdjustDown: 32,

	// Maximum upward adjustment in pow difficulty, as a percentage
	PowMaxAdjustUp: 16,

	// Maximum blocks signed by a single validate key in averaging window.
	ChainWindowMaxBlocks: 3,

	// Maximum fee allowed in a single transaction, in atoms.
	MaximumFeeAmount: 5000000,
}

var (
	// ErrDuplicateNet describes an error where the parameters for a Bitcoin
	// network could not be set due to the network already being a standard
	// network or previously-registered into this package.
	ErrDuplicateNet = errors.New("duplicate Bitcoin network")

	// ErrUnknownHDKeyID describes an error where the provided id which
	// is intended to identify the network for a hierarchical deterministic
	// private extended key is not registered.
	ErrUnknownHDKeyID = errors.New("unknown hd private extended key bytes")
)

var (
	registeredNets    = make(map[wire.BitcoinNet]struct{})
	pubKeyHashAddrIDs = make(map[byte]struct{})
	scriptHashAddrIDs = make(map[byte]struct{})
	provaAddrIDs      = make(map[byte]struct{})
	hdPrivToPubKeyIDs = make(map[[4]byte][]byte)
)

// String returns the hostname of the DNS seed in human-readable form.
func (d DNSSeed) String() string {
	return d.Host
}

// Register registers the network parameters for a Bitcoin network.  This may
// error with ErrDuplicateNet if the network is already registered (either
// due to a previous Register call, or the network being one of the default
// networks).
//
// Network parameters should be registered into this package by a main package
// as early as possible.  Then, library packages may lookup networks or network
// parameters based on inputs and work regardless of the network being standard
// or not.
func Register(params *Params) error {
	if _, ok := registeredNets[params.Net]; ok {
		return ErrDuplicateNet
	}
	registeredNets[params.Net] = struct{}{}
	if params.ProvaAddrID != 0 {
		provaAddrIDs[params.ProvaAddrID] = struct{}{}
	}
	hdPrivToPubKeyIDs[params.HDPrivateKeyID] = params.HDPublicKeyID[:]
	return nil
}

// mustRegister performs the same function as Register except it panics if there
// is an error.  This should only be called from package init functions.
func mustRegister(params *Params) {
	if err := Register(params); err != nil {
		panic("failed to register network: " + err.Error())
	}
}

// IsPubKeyHashAddrID returns whether the id is an identifier known to prefix a
// pay-to-pubkey-hash address on any default or registered network.  This is
// used when decoding an address string into a specific address type.  It is up
// to the caller to check both this and IsScriptHashAddrID and decide whether an
// address is a pubkey hash address, script hash address, neither, or
// undeterminable (if both return true).
func IsPubKeyHashAddrID(id byte) bool {
	_, ok := pubKeyHashAddrIDs[id]
	return ok
}

// IsScriptHashAddrID returns whether the id is an identifier known to prefix a
// pay-to-script-hash address on any default or registered network.  This is
// used when decoding an address string into a specific address type.  It is up
// to the caller to check both this and IsPubKeyHashAddrID and decide whether an
// address is a pubkey hash address, script hash address, neither, or
// undeterminable (if both return true).
func IsScriptHashAddrID(id byte) bool {
	_, ok := scriptHashAddrIDs[id]
	return ok
}

// IsProvaAddrID returns whether the id is an identifier known to prefix a
// standard Prova address on any default or registered network.  This is
// used when decoding an address string into a specific address type.
func IsProvaAddrID(id byte) bool {
	_, ok := provaAddrIDs[id]
	return ok
}

// HDPrivateKeyToPublicKeyID accepts a private hierarchical deterministic
// extended key id and returns the associated public key id.  When the provided
// id is not registered, the ErrUnknownHDKeyID error will be returned.
func HDPrivateKeyToPublicKeyID(id []byte) ([]byte, error) {
	if len(id) != 4 {
		return nil, ErrUnknownHDKeyID
	}

	var key [4]byte
	copy(key[:], id)
	pubBytes, ok := hdPrivToPubKeyIDs[key]
	if !ok {
		return nil, ErrUnknownHDKeyID
	}

	return pubBytes, nil
}

// newHashFromStr converts the passed big-endian hex string into a
// chainhash.Hash.  It only differs from the one available in chainhash in that
// it panics on an error since it will only (and must only) be called with
// hard-coded, and therefore known good, hashes.
func newHashFromStr(hexStr string) *chainhash.Hash {
	hash, err := chainhash.NewHashFromStr(hexStr)
	if err != nil {
		// Ordinarily I don't like panics in library code since it
		// can take applications down without them having a chance to
		// recover which is extremely annoying, however an exception is
		// being made in this case because the only way this can panic
		// is if there is an error in the hard-coded hashes.  Thus it
		// will only ever potentially panic on init and therefore is
		// 100% predictable.
		panic(err)
	}
	return hash
}

// powLimitFromStr returns a pow limit based on a difficulty hex value.
func powLimitFromStr(hexStr string) *big.Int {
	limit := big.NewInt(0)
	limit.SetString(hexStr, 16)
	return limit
}

func init() {
	// Register all default networks when the package is initialized.
	mustRegister(&MainNetParams)
	mustRegister(&TestNetParams)
	mustRegister(&RegressionNetParams)
	mustRegister(&SimNetParams)
}
