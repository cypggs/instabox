// Package id generates InstaBox-style prefixed ULIDs.
package id

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"time"
)

// ulid-like: 26 Crockford base32 chars
const encode = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

func ulid() string {
	var b [16]byte
	binary.BigEndian.PutUint64(b[0:8], uint64(time.Now().UnixMilli()))
	if _, err := rand.Read(b[8:]); err != nil {
		panic(err)
	}
	var out [26]byte
	// 48 bits time + 80 bits random
	v := uint64(b[0])<<40 | uint64(b[1])<<32 | uint64(b[2])<<24 | uint64(b[3])<<16 | uint64(b[4])<<8 | uint64(b[5])
	out[0] = encode[(v>>45)&31]
	out[1] = encode[(v>>40)&31]
	out[2] = encode[(v>>35)&31]
	out[3] = encode[(v>>30)&31]
	out[4] = encode[(v>>25)&31]
	out[5] = encode[(v>>20)&31]
	out[6] = encode[(v>>15)&31]
	out[7] = encode[(v>>10)&31]
	out[8] = encode[(v>>5)&31]
	out[9] = encode[v&31]
	// remaining 80 bits (b[6..15]) -> 16 chars
	idx := 10
	var acc uint32
	nacc := 0
	for i := 6; i < 16; i++ {
		acc = acc<<8 | uint32(b[i])
		nacc += 8
		for nacc >= 5 {
			nacc -= 5
			out[idx] = encode[(acc>>nacc)&31]
			idx++
		}
	}
	if nacc > 0 {
		out[idx] = encode[(acc<<(5-nacc))&31]
	}
	return string(out[:])
}

func Machine() string   { return "m_" + ulid() }
func Snapshot() string  { return "sn_" + ulid() }
func Operation() string { return "op_" + ulid() }
func Org() string       { return fmt.Sprintf("org_%032x", randBytes(16)) }
func Port() string      { return "pv-" + ulid() }
func Exec() string      { return "ex_" + ulid() }
func Key() string       { return "key_" + ulid() }
func Job() string       { return "job_" + ulid() }

func randBytes(n int) []byte {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return b
}

// Token returns a random token with the given prefix.
func Token(prefix string, n int) string {
	b := randBytes(n)
	s := make([]byte, n)
	for i, v := range b {
		s[i] = encode[v&31]
	}
	return prefix + string(s)
}
