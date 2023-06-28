package util

import (
	"math/rand"
	"regexp"
	"strings"
	"time"
)

var Rand *rand.Rand

const CharsAlphaNum = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
const CharsAlphaLowerNum = "abcdefghijklmnopqrstuvwxyz0123456789"
const CharsAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
const CharsHex = "0123456789abcdef"
const CharsDigits = "0123456789"

func init() {
	Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func RandomString(alphabet string, length int) string {
	chars := []byte(alphabet)
	str := make([]byte, length)
	for i := 0; i < length; i++ {
		str[i] = chars[Rand.Int()%len(chars)]
	}
	return string(str)
}

func CheckString(alphabet string, str string, sensitive bool) bool {
	if !sensitive {
		alphabet = strings.ToLower(alphabet)
		str = strings.ToLower(str)
	}
	for _, r := range str {
		if !strings.ContainsAny(alphabet, string(r)) {
			return false
		}
	}
	return true
}

func IsAlphanumeric(s string) bool {
	match, _ := regexp.MatchString("^[a-zA-Z0-9]+$", s)
	return match
}
