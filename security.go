package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/spf13/viper"
)

//dumb hacky checksum validation. do not use this security measure. only here for backwards compatibility.
func validateDumbChecksum(score, name, checksum string) error {
	if len(checksum) != 33 {
		return errors.New("invalid checksum: wrong length")
	}

	extraChar := checksum[9]
	if extraChar != 'a' {
		return errors.New("invalid checksum: missing char")
	}

	incomingHash := checksum[:9] + checksum[10:]

	md5 := md5.Sum([]byte(name + score))
	realHash := hex.EncodeToString(md5[:])

	if !bytes.Equal([]byte(realHash), []byte(incomingHash)) {
		fmt.Println(incomingHash)
		fmt.Println(realHash)
		return errors.New("invalid checksum: no match")
	}

	return nil
}

// decrypt the given byte values
func decryptValues(score, name []byte) (decrScore, decrName string, progErr error) {
	var err error
	decrScore, err = decryptWithAES(score)
	if err != nil {
		return "", "", err
	}

	decrName, err = decryptWithAES(name)
	if err != nil {
		return "", "", err
	}

	return decrScore, decrName, nil
}

// decrypt the given byte value
func decryptWithAES(in []byte) (result string, err error) {
	key := viper.GetString("aes_key")
	c, err := aes.NewCipher([]byte(key))
	if err != nil {
		fmt.Printf("Key is of length %d, must be of length 16, 24, or 32", len(key))
	}
	out := make([]byte, len(in))

	ensureBlockSize(&in, aes.BlockSize)
	ensureBlockSize(&out, aes.BlockSize)

	decr := cipher.NewCBCDecrypter(c, []byte(key))
	decr.CryptBlocks(out, in)

	//remove null chars
	out = bytes.Trim(out, "\x00")

	return string(out), nil
}

func ensureBlockSize(arr *[]byte, size int) {
	remainder := len(*arr) % size
	if remainder != 0 {
		newIn := make([]byte, len(*arr)+(size-remainder))
		copy(newIn, *arr)
		*arr = newIn
	}
}
