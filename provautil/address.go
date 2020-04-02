// Copyright (c) 2013, 2014 The btcsuite developers
// Copyright (c) 2017 BitGo
// Copyright (c) 2019 Tranquility Node Ltd
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package provautil

import (
	"encoding/binary"
	"encoding/hex"
	"errors"

	"github.com/pyx-partners/dmgd/btcec"
	"github.com/pyx-partners/dmgd/chaincfg"
	"github.com/pyx-partners/dmgd/provautil/base58"
	"github.com/btcsuite/golangcrypto/ripemd160"
)

var (
	// ErrChecksumMismatch describes an error where decoding failed due
	// to a bad checksum.
	ErrChecksumMismatch = errors.New("checksum mismatch")

	// ErrUnknownAddressType describes an error where an address can not
	// decoded as a specific address type due to the string encoding
	// begining with an identifier byte unknown to any standard or
	// registered (via chaincfg.Register) network.
	ErrUnknownAddressType = errors.New("unknown address type")

	// ErrAddressCollision describes an error where an address can not
	// be uniquely determined as either a pay-to-pubkey-hash or
	// pay-to-script-hash address since the leading identifier is used for
	// describing both address kinds, but for different networks.  Rather
	// than assuming or defaulting to one or the other, this error is
	// returned and the caller must decide how to decode the address.
	ErrAddressCollision = errors.New("address collision")
)

func encodeProvaAddress(keyIDs []btcec.KeyID, hash160 []byte, netID byte) string {
	data := make([]byte, len(keyIDs)*btcec.KeyIDSize+ripemd160.Size)
	copy(data[0:], hash160)
	offset := ripemd160.Size
	for _, keyID := range keyIDs {
		binary.LittleEndian.PutUint32(data[offset:], uint32(keyID))
		offset += btcec.KeyIDSize
	}
	return base58.CheckEncode(data, netID)
}

// TODO(prova): Modify this interface to handle only Prova-form addresses. No need
// to retain the old interface / address types.
//
// Address is an interface type for any type of destination a transaction
// output may spend to.  This includes pay-to-pubkey (P2PK), pay-to-pubkey-hash
// (P2PKH), and pay-to-script-hash (P2SH).  Address is designed to be generic
// enough that other kinds of addresses may be added in the future without
// changing the decoding and encoding API.
type Address interface {
	// String returns the string encoding of the transaction output
	// destination.
	//
	// Please note that String differs subtly from EncodeAddress: String
	// will return the value as a string without any conversion, while
	// EncodeAddress may convert destination types (for example,
	// converting pubkeys to P2PKH addresses) before encoding as a
	// payment address string.
	String() string

	// EncodeAddress returns the string encoding of the payment address
	// associated with the Address value.  See the comment on String
	// for how this method differs from String.
	EncodeAddress() string

	// ScriptAddress returns the raw bytes of the address to be used
	// when inserting the address into a txout's script.
	ScriptAddress() []byte

	// ScriptKeyIDs returns the key ids to be used when inserting the
	// address into a txout's script.
	ScriptKeyIDs() []btcec.KeyID

	// IsForNet returns whether or not the address is associated with the
	// passed bitcoin network.
	IsForNet(*chaincfg.Params) bool
}

// DecodeAddress decodes the string encoding of an address and returns
// the Address if addr is a valid encoding for a known address type.
//
// The bitcoin network the address is associated with is extracted if possible.
// When the address does not encode the network, such as in the case of a raw
// public key, the address will be associated with the passed defaultNet.
func DecodeAddress(addr string, defaultNet *chaincfg.Params) (Address, error) {
	// Switch on decoded length to determine the type.
	decoded, netID, err := base58.CheckDecode(addr)
	if err != nil {
		if err == base58.ErrChecksum {
			return nil, ErrChecksumMismatch
		}
		return nil, errors.New("decoded address is of unknown format")
	}

	if chaincfg.IsProvaAddrID(netID) {
		decodedLen := len(decoded)
		mininumKeyIdsCount := 2
		maximumKeyIdsCount := 19
		if decodedLen < ripemd160.Size+(mininumKeyIdsCount*btcec.KeyIDSize) {
			return nil, errors.New("decoded address is of unknown size")
		}
		if decodedLen > ripemd160.Size+(maximumKeyIdsCount*btcec.KeyIDSize) {
			return nil, errors.New("decoded address exceeds maximum size")
		}
		if (decodedLen-ripemd160.Size)%btcec.KeyIDSize != 0 {
			return nil, errors.New("decoded address has invalid size")
		}
		return newAddressProvaFromBytes(decoded, netID)
	}

	return nil, errors.New("decoded address is of unknown size")
}

// AddressProva is a standard n-1 of n Prova address with n-1 keyids
type AddressProva struct {
	keyIDs []btcec.KeyID
	hash   [ripemd160.Size]byte
	netID  byte
}

// NewAddressProva returns a new AddressProva.  pkHash mustbe 20
// bytes.
func NewAddressProva(pkHash []byte, keyIDs []btcec.KeyID, net *chaincfg.Params) (*AddressProva, error) {
	return newAddressProva(pkHash, keyIDs, net.ProvaAddrID)
}

// newAddressProva is the internal API to create an Prova address
// with a known leading identifier byte for a network, rather than looking
// it up through its parameters.  This is useful when creating a new address
// structure from a string encoding where the identifer byte is already
// known.
func newAddressProva(pkHash []byte, keyIDs []btcec.KeyID, netID byte) (*AddressProva, error) {
	// Check for a valid pubkey hash length.
	if len(pkHash) != ripemd160.Size {
		return nil, errors.New("pkHash must be 20 bytes")
	}
	// Check for the allowable range of keyid counts.
	if len(keyIDs) < 2 {
		return nil, errors.New("keyIDs must have length at least 2")
	}
	if len(keyIDs) > 19 {
		return nil, errors.New("keyIDs must have length at most 19")
	}

	addr := &AddressProva{netID: netID}
	copy(addr.hash[:], pkHash)
	numKeyIDs := len(keyIDs)
	addr.keyIDs = make([]btcec.KeyID, numKeyIDs, numKeyIDs)
	copy(addr.keyIDs, keyIDs)
	return addr, nil
}

// newAddressProvaFromBytes is the internal API to create an Prova address
// directly from the encoded bytes
//
// Note: this function assumes that the data is well formed
func newAddressProvaFromBytes(data []byte, netID byte) (*AddressProva, error) {
	keyIDs := []btcec.KeyID{}
	keyIDSize := btcec.KeyIDSize
	offset := ripemd160.Size

	for i := offset; i <= len(data)-keyIDSize; i += keyIDSize {
		id := btcec.KeyIDFromAddressBuffer(data[i : i+keyIDSize])
		keyIDs = append(keyIDs, id)
	}
	return newAddressProva(data[0:offset], keyIDs, netID)
}

// EncodeAddress returns the string encoding of an Prova address.
// Part of the Address interface.
func (a *AddressProva) EncodeAddress() string {
	return encodeProvaAddress(a.keyIDs[:], a.hash[:], a.netID)
}

// ScriptAddress returns the bytes to be included in a txout script for an Prova address.
// Part of the Address interface.
func (a *AddressProva) ScriptAddress() []byte {
	return a.hash[:]
}

// ScriptKeyIDs returns the key ids to be included in a txout script for an Prova address.
func (a *AddressProva) ScriptKeyIDs() []btcec.KeyID {
	return a.keyIDs[:]
}

// IsForNet returns whether or not the Prova address is associated
// with the passed bitcoin network.
func (a *AddressProva) IsForNet(net *chaincfg.Params) bool {
	return a.netID == net.ProvaAddrID
}

// String returns a human-readable string for the Prova address type.
// This is equivalent to calling EncodeAddress, but is provided so the type can
// be used as a fmt.Stringer.
func (a *AddressProva) String() string {
	return a.EncodeAddress()
}

// AddressPubKeyHash is an Address for a pay-to-pubkey-hash (P2PKH)
// transaction.
type AddressPubKeyHash struct {
	hash  [ripemd160.Size]byte
	netID byte
}

// newAddressPubKeyHash is the internal API to create a pubkey hash address
// with a known leading identifier byte for a network, rather than looking
// it up through its parameters.  This is useful when creating a new address
// structure from a string encoding where the identifer byte is already
// known.
func newAddressPubKeyHash(pkHash []byte, netID byte) (*AddressPubKeyHash, error) {
	// Check for a valid pubkey hash length.
	if len(pkHash) != ripemd160.Size {
		return nil, errors.New("pkHash must be 20 bytes")
	}

	addr := &AddressPubKeyHash{netID: netID}
	copy(addr.hash[:], pkHash)
	return addr, nil
}

// ScriptAddress returns the bytes to be included in a txout script to pay
// to a pubkey hash.  Part of the Address interface.
func (a *AddressPubKeyHash) ScriptAddress() []byte {
	return a.hash[:]
}

// ScriptKeyIDs returns the key ids to be included in a txout script
// todo(ben): should this be part of the Address interface?
func (a *AddressPubKeyHash) ScriptKeyIDs() []btcec.KeyID {
	return make([]btcec.KeyID, 0)
}

// Hash160 returns the underlying array of the pubkey hash.  This can be useful
// when an array is more appropiate than a slice (for example, when used as map
// keys).
func (a *AddressPubKeyHash) Hash160() *[ripemd160.Size]byte {
	return &a.hash
}

// AddressScriptHash is an Address for a pay-to-script-hash (P2SH)
// transaction.
type AddressScriptHash struct {
	hash  [ripemd160.Size]byte
	netID byte
}

// newAddressScriptHashFromHash is the internal API to create a script hash
// address with a known leading identifier byte for a network, rather than
// looking it up through its parameters.  This is useful when creating a new
// address structure from a string encoding where the identifer byte is already
// known.
func newAddressScriptHashFromHash(scriptHash []byte, netID byte) (*AddressScriptHash, error) {
	// Check for a valid script hash length.
	if len(scriptHash) != ripemd160.Size {
		return nil, errors.New("scriptHash must be 20 bytes")
	}

	addr := &AddressScriptHash{netID: netID}
	copy(addr.hash[:], scriptHash)
	return addr, nil
}

// ScriptAddress returns the bytes to be included in a txout script to pay
// to a script hash.  Part of the Address interface.
func (a *AddressScriptHash) ScriptAddress() []byte {
	return a.hash[:]
}

// ScriptKeyIDs returns the key ids to be included in a txout script
// todo(ben): should this be part of the Address interface?
func (a *AddressScriptHash) ScriptKeyIDs() []btcec.KeyID {
	return make([]btcec.KeyID, 0)
}

// Hash160 returns the underlying array of the script hash.  This can be useful
// when an array is more appropiate than a slice (for example, when used as map
// keys).
func (a *AddressScriptHash) Hash160() *[ripemd160.Size]byte {
	return &a.hash
}

// PubKeyFormat describes what format to use for a pay-to-pubkey address.
type PubKeyFormat int

const (
	// PKFUncompressed indicates the pay-to-pubkey address format is an
	// uncompressed public key.
	PKFUncompressed PubKeyFormat = iota

	// PKFCompressed indicates the pay-to-pubkey address format is a
	// compressed public key.
	PKFCompressed

	// PKFHybrid indicates the pay-to-pubkey address format is a hybrid
	// public key.
	PKFHybrid
)

// AddressPubKey is an Address for a pay-to-pubkey transaction.
type AddressPubKey struct {
	pubKeyFormat PubKeyFormat
	pubKey       *btcec.PublicKey
	pubKeyHashID byte
}

// NewAddressPubKey returns a new AddressPubKey which represents a pay-to-pubkey
// address.  The serializedPubKey parameter must be a valid pubkey and can be
// uncompressed, compressed, or hybrid.
func NewAddressPubKey(serializedPubKey []byte, net *chaincfg.Params) (*AddressPubKey, error) {
	pubKey, err := btcec.ParsePubKey(serializedPubKey, btcec.S256())
	if err != nil {
		return nil, err
	}

	// Set the format of the pubkey.  This probably should be returned
	// from btcec, but do it here to avoid API churn.  We already know the
	// pubkey is valid since it parsed above, so it's safe to simply examine
	// the leading byte to get the format.
	pkFormat := PKFUncompressed
	switch serializedPubKey[0] {
	case 0x02, 0x03:
		pkFormat = PKFCompressed
	case 0x06, 0x07:
		pkFormat = PKFHybrid
	}

	return &AddressPubKey{
		pubKeyFormat: pkFormat,
		pubKey:       pubKey,
	}, nil
}

// serialize returns the serialization of the public key according to the
// format associated with the address.
func (a *AddressPubKey) serialize() []byte {
	switch a.pubKeyFormat {
	default:
		fallthrough
	case PKFUncompressed:
		return a.pubKey.SerializeUncompressed()

	case PKFCompressed:
		return a.pubKey.SerializeCompressed()

	case PKFHybrid:
		return a.pubKey.SerializeHybrid()
	}
}

// ScriptAddress returns the bytes to be included in a txout script to pay
// to a public key.  Setting the public key format will affect the output of
// this function accordingly.  Part of the Address interface.
func (a *AddressPubKey) ScriptAddress() []byte {
	return a.serialize()
}

// ScriptKeyIDs returns the key ids to be included in a txout script
// todo(ben): should this be part of the Address interface?
func (a *AddressPubKey) ScriptKeyIDs() []btcec.KeyID {
	return make([]btcec.KeyID, 0)
}

// String returns the hex-encoded human-readable string for the pay-to-pubkey
// address.  This is not the same as calling EncodeAddress.
func (a *AddressPubKey) String() string {
	return hex.EncodeToString(a.serialize())
}

// Format returns the format (uncompressed, compressed, etc) of the
// pay-to-pubkey address.
func (a *AddressPubKey) Format() PubKeyFormat {
	return a.pubKeyFormat
}

// SetFormat sets the format (uncompressed, compressed, etc) of the
// pay-to-pubkey address.
func (a *AddressPubKey) SetFormat(pkFormat PubKeyFormat) {
	a.pubKeyFormat = pkFormat
}

// AddressPubKeyHash returns the pay-to-pubkey address converted to a
// pay-to-pubkey-hash address.  Note that the public key format (uncompressed,
// compressed, etc) will change the resulting address.  This is expected since
// pay-to-pubkey-hash is a hash of the serialized public key which obviously
// differs with the format.  At the time of this writing, most Bitcoin addresses
// are pay-to-pubkey-hash constructed from the uncompressed public key.
func (a *AddressPubKey) AddressPubKeyHash() *AddressPubKeyHash {
	addr := &AddressPubKeyHash{netID: a.pubKeyHashID}
	copy(addr.hash[:], Hash160(a.serialize()))
	return addr
}

// PubKey returns the underlying public key for the address.
func (a *AddressPubKey) PubKey() *btcec.PublicKey {
	return a.pubKey
}
