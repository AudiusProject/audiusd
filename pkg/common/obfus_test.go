package common_test

import (
	"testing"

	"github.com/AudiusProject/audiusd/pkg/common"
	"github.com/stretchr/testify/assert"
)

func TestObfuscate(t *testing.T) {
	obfuscated := common.Obfuscate("test")
	assert.NotEmpty(t, obfuscated)
}

func TestDeobfuscate(t *testing.T) {
	obfuscated := common.Obfuscate("test")
	deobfuscated, err := common.Deobfuscate(obfuscated)
	assert.NoError(t, err)
	assert.Equal(t, "test", deobfuscated)
}
