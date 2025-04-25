package sdk

import (
	"crypto/ecdsa"
	"errors"

	"github.com/AudiusProject/audiusd/pkg/common"
)

func (s *AudiusdSDK) SetPrivKey(privKey *ecdsa.PrivateKey) {
	s.privKey = privKey
}

func (s *AudiusdSDK) Sign(msg []byte) (string, error) {
	if s.privKey == nil {
		return "", errors.New("private key not set")
	}

	signature, err := common.EthSign(s.privKey, msg)
	if err != nil {
		return "", err
	}

	return signature, nil
}

func (s *AudiusdSDK) RecoverSigner(msg []byte, signature string) (string, error) {
	_, address, err := common.EthRecover(signature, msg)
	if err != nil {
		return "", err
	}

	return address, nil
}
