package security

import (
	"fmt"
	"leaderboard/config"
	"testing"

	"github.com/spf13/viper"
)

func TestAes(t *testing.T) {
	viper.Set(config.AESKey, "DEADBEEFDEADBEEF")
	encrypted := []byte{41, 227, 27, 224, 152, 128, 38, 20, 232, 121, 76, 71, 36, 167, 25, 251, 110, 248, 71, 248, 247, 8, 190, 119, 125, 223, 139, 236, 50, 233, 107, 245}
	expected := "howdy there boys and girls"
	actual, err := decryptWithAES(encrypted)
	if err != nil {
		t.Error("unexpected error")
	}

	fmt.Println([]byte(actual))

	if actual != expected {
		t.Errorf("got %s!, wanted %s!", actual, expected)
	}
}
