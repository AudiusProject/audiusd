package sdk

import (
	"context"
	"errors"
	"fmt"
	"time"

	"crypto/ecdsa"

	"github.com/AudiusProject/audiusd/pkg/common"
	ccommon "github.com/AudiusProject/audiusd/pkg/core/common"
	"github.com/AudiusProject/audiusd/pkg/core/gen/core_openapi/protocol"
	"github.com/AudiusProject/audiusd/pkg/core/gen/core_proto"
	adx "github.com/AudiusProject/audiusd/pkg/core/gen/core_proto/audiusddex/v1beta1"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
)

type Sdk struct {
	logger      Logger
	useHttps    bool
	privKeyPath string
	privKey     *ecdsa.PrivateKey

	retries int
	delay   time.Duration
}

func defaultSdk() *Sdk {
	return &Sdk{
		logger:  NewNoOpLogger(),
		retries: 10,
		delay:   3 * time.Second,
	}
}

func initSdk(sdk *Sdk) error {
	ctx := context.Background()
	// TODO: add default environment here if not set

	// TODO: add node selection logic here, based on environment, if endpoint not configured

	g, ctx := errgroup.WithContext(ctx)

	if sdk.privKeyPath != "" {
		g.Go(func() error {
			key, err := common.LoadPrivateKey(sdk.privKeyPath)
			if err != nil {
				return err
			}
			sdk.privKey = key
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		sdk.logger.Error("init sdk error", "error", err)
		return err
	}

	return nil
}

type ReleaseResult struct {
	TxHash  string
	TrackID string
}

func (sdk *Sdk) ReleaseTrack(cid, title, genre string) (*ReleaseResult, error) {
	if sdk.privKey == nil {
		return nil, errors.New("No private key set, cannot release track")
	}
	trackId := uuid.NewString()
	ern := &adx.NewReleaseMessage{
		ReleaseHeader: &adx.ReleaseHeader{
			Sender: &adx.Party{
				PartyId: "audius_sdk",
				PubKey:  crypto.CompressPubkey(&sdk.privKey.PublicKey),
			},
		},
		ResourceList: []*adx.Resource{
			&adx.Resource{
				ResourceReference: "AT1",
				Resource: &adx.Resource_SoundRecording{
					SoundRecording: &adx.SoundRecording{
						Cid: cid,
						Id: &adx.SoundRecordingId{
							Isrc: uuid.NewString(),
						},
					},
				},
			},
		},
		ReleaseList: []*adx.Release{
			&adx.Release{
				Release: &adx.Release_TrackRelease{
					TrackRelease: &adx.TrackRelease{
						ReleaseId: &adx.ReleaseId{
							Isrc: trackId,
						},
						ReleaseResourceReference: "AT1",
						Title:                    title,
						Genre:                    genre,
					},
				},
			},
		},
	}

	ernBytes, err := proto.Marshal(ern)
	if err != nil {
		return nil, fmt.Errorf("failure to marshal ern: %v", err)
	}

	sig, err := common.EthSign(sdk.privKey, ernBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to sign message: %w", err)
	}

	tx := &core_proto.SignedTransaction{
		Signature: sig,
		RequestId: uuid.NewString(),
		Transaction: &core_proto.SignedTransaction_Release{
			Release: ern,
		},
	}
	sendParams := protocol.NewProtocolSendTransactionParams()
	sendParams.SetTransaction(ccommon.SignedTxProtoIntoSignedTxOapi(tx))
	res, err := sdk.ProtocolSendTransaction(sendParams)
	if err != nil {
		return nil, fmt.Errorf("ern failed: %w", err)
	}

	return &ReleaseResult{TrackID: trackId, TxHash: res.Payload.Txhash}, nil
}
