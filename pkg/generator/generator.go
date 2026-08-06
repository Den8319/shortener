// Package generator реализует генерацию криптостойких коротких идентификаторов URL.
package generator

import (
	"crypto/rand"
	"math/big"
)

// alphabet — набор символов для генерации коротких URL (base64url-совместимый).
const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"

// GenerateShort генерирует случайную строку длины n из символов alphabet.
// Использует crypto/rand для криптостойкой генерации.
func GenerateShort(n int) (string, error) {
	result := make([]byte, n)

	for i := range result {
		num, err := rand.Int(rand.Reader, big.NewInt(64))
		if err != nil {
			return "", err
		}
		result[i] = alphabet[num.Int64()]
	}
	return string(result), nil
}
