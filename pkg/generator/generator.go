package generator

import (
	"crypto/rand"
	"math/big"
)


const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"

func GenerateShort(n int) (string , error) {
	result := make([]byte, n)

	for i :=range result {
		num, err := rand.Int(rand.Reader , big.NewInt(64))
		 if err != nil {
			return "" ,err
		 }
		 result[i] = alphabet[num.Int64()]
	}
	return string(result) ,nil
}


