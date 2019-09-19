package blockchain

import "github.com/opacey/dmgd/wire"

// IsGenerationShareRateLimited determines if block generation is rate
// limited due to a consumption of the permitted block share.
func IsGenerationShareRateLimited(validatePubKey wire.BlockValidatingPubKey, prevPubKeys []wire.BlockValidatingPubKey, maxBlocks int, prospectiveInclusion bool, lastValidatePubKey wire.BlockValidatingPubKey) bool {
	blockCount := 0
	// For prospectiveInclusion the lastValidatePubKey is already present
	// in a valid chain.
	if prospectiveInclusion && validatePubKey == lastValidatePubKey {
		blockCount += 1
	}
	// Iterate through the ids, summing towards the mined blocks count
	for _, key := range prevPubKeys {
		if key == validatePubKey {
			blockCount += 1
		}
		if blockCount >= maxBlocks {
			return true
		}
	}
	return false
}
