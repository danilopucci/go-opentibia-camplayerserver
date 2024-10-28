package crypt

import "encoding/binary"

func ExpandXteaKey(k [4]uint32) [64]uint32 {
	const delta = 0x9E3779B9
	var expanded [64]uint32

	sum := uint32(0)
	nextSum := sum + delta

	for i := 0; i < len(expanded); i += 2 {
		expanded[i] = sum + k[sum&3]
		expanded[i+1] = nextSum + k[(nextSum>>11)&3]
		sum = nextSum
		nextSum += delta
	}

	return expanded
}

func XteaEncrypt(data []byte, k [64]uint32) {
	for i := 0; i < len(k); i += 2 {
		for offset := 0; offset < len(data); offset += 8 {

			// Handle left and right parts of the block
			left := binary.LittleEndian.Uint32(data[offset : offset+4])
			right := binary.LittleEndian.Uint32(data[offset+4 : offset+8])

			// XTEA encryption round
			left += ((right << 4) ^ (right >> 5)) + right ^ k[i]
			right += ((left << 4) ^ (left >> 5)) + left ^ k[i+1]

			// Store encrypted result back into data slice
			binary.LittleEndian.PutUint32(data[offset:offset+4], left)
			binary.LittleEndian.PutUint32(data[offset+4:offset+8], right)
		}
	}
}

func XteaDecrypt(data []byte, k [64]uint32) {
	for i := len(k) - 2; i >= 0; i -= 2 {
		for offset := 0; offset < len(data); offset += 8 {

			// Handle left and right parts of the block
			left := binary.LittleEndian.Uint32(data[offset : offset+4])
			right := binary.LittleEndian.Uint32(data[offset+4 : offset+8])

			// XTEA decryption round
			right -= ((left << 4) ^ (left >> 5)) + left ^ k[i+1]
			left -= ((right << 4) ^ (right >> 5)) + right ^ k[i]

			// Store decrypted result back into data slice
			binary.LittleEndian.PutUint32(data[offset:offset+4], left)
			binary.LittleEndian.PutUint32(data[offset+4:offset+8], right)
		}
	}
}
