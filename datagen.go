package main

// generateObjectData returns a deterministic byte slice for the given
// object number, key, and size using a XorShift64 PRNG.
//
// Seed derivation: uint64(objnum)*0x9e3779b97f4a7c15 ^ uint64(key)<<32
// This ensures that different (objnum, key) pairs produce distinct seeds,
// and therefore distinct output byte sequences.
func generateObjectData(objnum int64, key uint8, size int) []byte {
	seed := uint64(objnum)*0x9e3779b97f4a7c15 ^ uint64(key)<<32
	state := seed | 1 // guard against all-zero state
	buf := make([]byte, size)
	for i := range buf {
		state ^= state << 13
		state ^= state >> 7
		state ^= state << 17
		buf[i] = byte(state)
	}
	return buf
}
