package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/AudiusProject/audiusd/pkg/common"
	"github.com/AudiusProject/audiusd/pkg/core/gen/core_proto"
	"github.com/jackc/pgx/v5"
	"google.golang.org/protobuf/proto"
)

func (s *Server) isValidReleaseTx(ctx context.Context, tx *core_proto.SignedTransaction) error {
	ern := tx.GetRelease()
	tx.Signature
	if ern == nil {
		return errors.New("Empty release in signed tx")
	}

	bodyBytes, err := proto.Marshal(ern)
	if err != nil {
		return fmt.Errorf("could not marshal release tx body into bytes: %v", err)
	}
	pubkey, addr, err := common.EthRecover(tx.GetSignature(), bodyBytes)
	if err != nil {
		return fmt.Errorf("could not recover release tx signer: %v", err)
	}

	// check tx was signed by trusted party
	node, err := s.db.GetRegisteredNodeByEthAddress(ctx, addr)
	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New("Invalid release tx signer")
	} else if err != nil {
		return fmt.Errorf("could not check signer of release tx: %v", err)
	}

	if ern.ReleaseHeader == nil {
		return errors.New("Empty release header")
	}
	if ern.ReleaseHeader.Sender == nil {
		return errors.New("Empty release sender")
	}
	if !bytes.Equal(pubkey, ern.ReleaseHeader.Sender.PubKey) {
		return errors.New("Sender and signer do not match")
	}

	if ern.ReleaseList == nil || len(ern.ReleaseList) == 0 {
		return errors.New("Empty release list")
	}
	if ern.ReleaseList == nil || len(ern.ReleaseList) == 0 {
		return errors.New("Empty release list")
	}
	if ern.ResourceList == nil || len(ern.ResourceList) == 0 {
		return errors.New("Empty resource list")
	}

	for _, resource := range ern.ResourceList {
		if resource.GetImage() != nil {
			img := resource.GetImage()
		} else if resource.GetSoundRecording() != nil {
		} else {
			s.logger.Warningf("Unsupported resource type %v", resource.GetResource())
		}
	}

	return nil
}

func (s *Server) finalizeRelease(ctx context.Context, tx *core_proto.SignedTransaction, txHash string) (*core_proto.SignedTransaction, error) {
	return tx, nil
}
