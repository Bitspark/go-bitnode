package util

import (
	"strings"
	"testing"
)

func TestRandomString(t *testing.T) {
	rs := RandomString("x", 10)
	if rs != strings.Repeat("x", 10) {
		t.Fatal(rs)
	}
}

func TestIsAlphanumeric(t *testing.T) {
	if !IsAlphanumeric("123") {
		t.Fatal()
	}
	if !IsAlphanumeric("abc") {
		t.Fatal()
	}
	if !IsAlphanumeric("ABC") {
		t.Fatal()
	}
	if !IsAlphanumeric("1aZ") {
		t.Fatal()
	}

	if IsAlphanumeric(" ") {
		t.Fatal()
	}
	if IsAlphanumeric("abc.d") {
		t.Fatal()
	}
	if IsAlphanumeric("abc-d") {
		t.Fatal()
	}
	if IsAlphanumeric("abc_d") {
		t.Fatal()
	}
}

func TestCheckString(t *testing.T) {
	if ok := CheckString(CharsAlphaLowerNum, "Test123", true); ok {
		t.Fatal()
	}
	if ok := CheckString(CharsAlphaLowerNum, "Test123", false); !ok {
		t.Fatal()
	}
	if ok := CheckString(CharsAlphaLowerNum, "12test", true); !ok {
		t.Fatal()
	}
	if ok := CheckString(CharsAlphaLowerNum, "test", true); !ok {
		t.Fatal()
	}
}
