package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDirHash(t *testing.T) {
	dir := "./examples/manifests/"
	expected := "9a8a081ad93c9739557035140b5dabbe2075c273"

	hash, err := getDirHash(dir)
	assert.Nil(t, err)
	assert.Equal(t, expected, hash)
}
