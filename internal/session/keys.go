package session

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
)

func CanonicalPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	return resolved, nil
}

func Key(canonicalPath string) string {
	sum := sha256.Sum256([]byte(canonicalPath))
	return hex.EncodeToString(sum[:])[:16]
}

func URLFor(key string, port int) string {
	return URLForHost(key, "127.0.0.1", port)
}

func URLForHost(key, host string, port int) string {
	if host == "" {
		host = "127.0.0.1"
	}
	return "http://" + host + ":" + itoa(port) + "/session/" + key
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
