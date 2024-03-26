package checksum

import (
	"crypto/md5"
	"encoding/hex"
)

func Calculate(data []byte) string {
	hashes := md5.New()
	hashes.Write(data)
	return hex.EncodeToString(hashes.Sum(nil))
}
