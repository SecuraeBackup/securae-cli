package cmd

import (
	"testing"
)


func TestHashEncryptionKey(t *testing.T) {
	key := "nMncUq8SsU7uz3cMucmFmgvUGXZ8LiBm8qx93hzrh6k="
	want := "XDWErHXbj7CKDan2Qw4wjQ=="

	hash, _ := hashEncryptionKey(key)
	if want != hash {
		t.Errorf("Result was incorrect, got: %s, want: %s.", hash, want)
	}
}

func TestHashEncryptionKeyEmpty(t *testing.T) {
	_, err := hashEncryptionKey("")
	if err == nil {
		t.Errorf("An error must be raised when passing an empty key")
	}
}


