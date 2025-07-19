package server

import (
	"context"

	"github.com/AudiusProject/audiusd/pkg/api/core/v1beta1"
	"github.com/AudiusProject/audiusd/pkg/api/ddex/v1beta2"
	"github.com/AudiusProject/audiusd/pkg/common"
	"github.com/AudiusProject/audiusd/pkg/core/db"
)

func (s *CoreService) buildTxReceipt(ctx context.Context, incomingTx *v1beta1.Transaction, block *db.CoreBlock) (*v1beta1.TransactionReceipt, error) {
	receipt := new(v1beta1.TransactionReceipt)
	receipt.MessageReceipts = make([]*v1beta1.MessageReceipt, len(incomingTx.Envelope.Messages))

	txHash, err := common.ToTxHash(incomingTx.Envelope)
	if err != nil {
		return nil, err
	}

	receipt.TxHash = txHash
	receipt.Height = block.Height
	receipt.Timestamp = block.CreatedAt.Time.Unix()
	receipt.Responder = s.core.config.ProposerAddress
	receipt.Proposer = block.Proposer
	receipt.EnvelopeInfo = &v1beta1.EnvelopeReceiptInfo{
		ChainId:      s.core.config.GenesisFile.ChainID,
		Expiration:   incomingTx.Envelope.Header.Expiration,
		Nonce:        incomingTx.Envelope.Header.Nonce,
		MessageCount: int32(len(incomingTx.Envelope.Messages)),
	}

	// message receipts
	// iterate over incomingTx.Envelope.Messages, switch on message type, and build the receipt for each message
	for i, message := range incomingTx.Envelope.Messages {
		switch message.GetMessage().(type) {
		case *v1beta1.Message_Ern:
			// Query the database for the ERN address and related addresses
			ernAddress, err := s.core.db.GetERNAddressByTxHash(ctx, txHash)
			if err != nil {
				return nil, err
			}

			// Get release addresses for this ERN
			releaseAddresses, err := s.core.db.GetReleaseAddressesByERNAddress(ctx, ernAddress)
			if err != nil {
				return nil, err
			}

			// Get sound recording addresses for this ERN
			soundRecordingAddresses, err := s.core.db.GetSoundRecordingAddressesByERNAddress(ctx, ernAddress)
			if err != nil {
				return nil, err
			}

			// Build the NewReleaseMessageAck
			ack := &v1beta2.ElectronicReleaseNotificationAck{
				ReleaseAddress: &v1beta2.ElectronicReleaseNotificationAck_Address{
					Index:   0, // Main ERN address index
					Address: ernAddress,
				},
				SoundRecordingAddresses: make([]*v1beta2.ElectronicReleaseNotificationAck_Address, len(soundRecordingAddresses)),
				ImageAddresses:          []*v1beta2.ElectronicReleaseNotificationAck_Address{
					// TODO: Add image addresses if needed - for now empty
				},
			}

			// Populate sound recording addresses
			for j, addr := range soundRecordingAddresses {
				ack.SoundRecordingAddresses[j] = &v1beta2.ElectronicReleaseNotificationAck_Address{
					Index:   uint32(j),
					Address: addr,
				}
			}

			// Add release addresses to party addresses for now (since releases are also addressable entities)
			for j, addr := range releaseAddresses {
				ack.PartyAddresses = append(ack.PartyAddresses, &v1beta2.ElectronicReleaseNotificationAck_Address{
					Index:   uint32(j),
					Address: addr,
				})
			}

			receipt.MessageReceipts[i] = &v1beta1.MessageReceipt{
				MessageIndex: int32(i),
				Result: &v1beta1.MessageReceipt_ErnAck{
					ErnAck: ack,
				},
			}
		}
	}

	return receipt, nil
}
