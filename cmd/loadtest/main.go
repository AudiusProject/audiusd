package main

import (
	"log"
	"time"

	"github.com/AudiusProject/audiusd/pkg/core/gen/core_openapi/protocol"
	"github.com/AudiusProject/audiusd/pkg/core/gen/models"
	"github.com/AudiusProject/audiusd/pkg/core/sdk"
	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"
)

func main() {
	sdk, err := sdk.NewSdk(sdk.WithOapiendpoint("node1.audiusd.devnet"), sdk.WithUsehttps(true))
	if err != nil {
		log.Fatal(err)
	}

	for {
		plays := []*models.ProtocolTrackPlay{
			{
				UserID:    uuid.NewString(),
				TrackID:   uuid.NewString(),
				Timestamp: strfmt.DateTime(time.Now()),
				Signature: uuid.NewString(),
				City:      uuid.NewString(),
				Region:    uuid.NewString(),
				Country:   uuid.NewString(),
			},
			{
				UserID:    uuid.NewString(),
				TrackID:   uuid.NewString(),
				Timestamp: strfmt.DateTime(time.Now()),
				Signature: uuid.NewString(),
				City:      uuid.NewString(),
				Region:    uuid.NewString(),
				Country:   uuid.NewString(),
			},
			{
				UserID:    uuid.NewString(),
				TrackID:   uuid.NewString(),
				Timestamp: strfmt.DateTime(time.Now()),
				Signature: uuid.NewString(),
				City:      uuid.NewString(),
				Region:    uuid.NewString(),
				Country:   uuid.NewString(),
			},
			{
				UserID:    uuid.NewString(),
				TrackID:   uuid.NewString(),
				Timestamp: strfmt.DateTime(time.Now()),
				Signature: uuid.NewString(),
				City:      uuid.NewString(),
				Region:    uuid.NewString(),
				Country:   uuid.NewString(),
			},
			{
				UserID:    uuid.NewString(),
				TrackID:   uuid.NewString(),
				Timestamp: strfmt.DateTime(time.Now()),
				Signature: uuid.NewString(),
				City:      uuid.NewString(),
				Region:    uuid.NewString(),
				Country:   uuid.NewString(),
			},
			{
				UserID:    uuid.NewString(),
				TrackID:   uuid.NewString(),
				Timestamp: strfmt.DateTime(time.Now()),
				Signature: uuid.NewString(),
				City:      uuid.NewString(),
				Region:    uuid.NewString(),
				Country:   uuid.NewString(),
			},
			{
				UserID:    uuid.NewString(),
				TrackID:   uuid.NewString(),
				Timestamp: strfmt.DateTime(time.Now()),
				Signature: uuid.NewString(),
				City:      uuid.NewString(),
				Region:    uuid.NewString(),
				Country:   uuid.NewString(),
			},
			{
				UserID:    uuid.NewString(),
				TrackID:   uuid.NewString(),
				Timestamp: strfmt.DateTime(time.Now()),
				Signature: uuid.NewString(),
				City:      uuid.NewString(),
				Region:    uuid.NewString(),
				Country:   uuid.NewString(),
			},
		}

		signedTransaction := &models.ProtocolSignedTransaction{
			Signature: uuid.NewString(),
			RequestID: uuid.NewString(),
			Plays: &models.ProtocolTrackPlays{
				Plays: plays,
			},
		}

		sendParams := protocol.NewProtocolSendTransactionParams()
		sendParams.SetTransaction(signedTransaction)

		sdk.ProtocolSendTransaction(sendParams)
	}
}
