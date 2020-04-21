package security

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"leaderboard/config"

	"github.com/spf13/viper"
)

var (
	//possible valid security values
	securityValues = []string{"aes", "stupid", "none", ""}
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
func decryptValues(score, name, checksum []byte) (decrScore, decrName, decrChecksum string, progErr error) {
	var err error
	decrScore, err = decryptWithAES(score)
	if err != nil {
		return "", "", "", err
	}

	decrName, err = decryptWithAES(name)
	if err != nil {
		return "", "", "", err
	}

	decrChecksum, err = decryptWithAES(checksum)
	if err != nil {
		return "", "", "", err
	}

	return decrScore, decrName, decrChecksum, nil
}

// decrypt the given byte value
func decryptWithAES(in []byte) (string, error) {
	key := viper.GetString(config.AESKey)
	err := ensureAESKeyLength(key)
	if err != nil {
		return "", err
	}
	c, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
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

//make sure an aes key is the appropriate length
func ensureAESKeyLength(key string) error {
	lenKey := len(key)
	switch lenKey {
	case 16, 24, 32:
		return nil
	default:
		return fmt.Errorf("aes key is of length %d, must be of length 16, 24, or 32", len(key))
	}
}

func ValidateSecurityType() {
	//validate stuff
	//ensure aes key length requirements
	err := ensureAESKeyLength(viper.GetString(config.AESKey))
	if err != nil {
		panic(err)
	}

	//ensure security is a valid value
	sec := viper.GetString(config.SecurityType)
	validSec := false
	for _, v := range securityValues {
		if v == sec {
			validSec = true
			break
		}
	}
	if !validSec {
		panic(fmt.Sprintf("invalid value '%s' for 'security', must be one of [%v]", sec, securityValues))
	}
}

// ValidateRequestParams will validate that the given request values are acceptable given the current security setting
func ValidateRequestParams(score string, name string, checksum string) error {
	securityType := viper.GetString(config.SecurityType)

	switch securityType {
	case "none", "":
		//no security
	case "stupid":
		//hacky checksum check: do not use this security measure. only here for backwards compatibility
		err := validateDumbChecksum(score, name, checksum)
		if err != nil {
			return fmt.Errorf("stupid checksum invalid: %w", err)
		}
	case "aes":
		//validate using aes encryption
		var err error
		score, name, checksum, err = decryptValues([]byte(score), []byte(name), []byte(checksum))
		if err != nil {
			return fmt.Errorf("couldn't decrypt aes: %w", err)
		}
		if checksum != viper.GetString(config.AESChecksum) {
			return fmt.Errorf("aes checksum '%s' invalid", checksum)
		}
	default:
		//invalid security value set
		return fmt.Errorf("invalid value for 'security': %s", securityType)
	}
	return nil
}
