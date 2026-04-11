package ids

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func New(prefix string) string {
	return fmt.Sprintf("%s_%s", prefix, randomHex(12))
}

func Token(prefix string) string {
	return fmt.Sprintf("%s_%s", prefix, randomHex(24))
}

func randomHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}

	return hex.EncodeToString(buf)
}
