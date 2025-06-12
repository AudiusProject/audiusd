package etl

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/AudiusProject/audiusd/pkg/api/core/v1"
	etlv1 "github.com/AudiusProject/audiusd/pkg/api/etl/v1"
	"github.com/AudiusProject/audiusd/pkg/etl/db"
	"github.com/AudiusProject/audiusd/pkg/etl/location"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	TxTypePlay                               = "play"
	TxTypeManageEntity                       = "manage_entity"
	TxTypeValidatorRegistration              = "validator_registration"
	TxTypeValidatorDeregistration            = "validator_deregistration"
	TxTypeValidatorRegistrationLegacy        = "validator_registration_legacy"
	TxTypeSlaRollup                          = "sla_rollup"
	TxTypeValidatorMisbehaviorDeregistration = "validator_misbehavior_deregistration"
	TxTypeStorageProof                       = "storage_proof"
	TxTypeStorageProofVerification           = "storage_proof_verification"
	TxTypeRelease                            = "release"
)

func (etl *ETLService) Run() error {
	dbUrl := etl.dbURL
	if dbUrl == "" {
		return fmt.Errorf("dbUrl environment variable not set")
	}

	err := db.RunMigrations(etl.logger, dbUrl, etl.runDownMigrations)
	if err != nil {
		return fmt.Errorf("error running migrations: %v", err)
	}

	pgConfig, err := pgxpool.ParseConfig(dbUrl)
	if err != nil {
		return fmt.Errorf("error parsing database config: %v", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), pgConfig)
	if err != nil {
		return fmt.Errorf("error creating database pool: %v", err)
	}

	etl.pool = pool
	etl.db = db.New(pool)

	locationDB, err := location.NewLocationService()
	if err != nil {
		return fmt.Errorf("error creating location service: %v", err)
	}
	etl.locationDB = locationDB

	// Initialize chain ID from core service
	err = etl.InitializeChainID(context.Background())
	if err != nil {
		etl.logger.Errorf("error initializing chain ID: %v", err)
	}

	etl.logger.Infof("starting etl service")

	if etl.checkReadiness {
		err = etl.awaitReadiness()
		if err != nil {
			etl.logger.Errorf("error awaiting readiness: %v", err)
		}
	}

	err = etl.indexBlocks()
	if err != nil {
		return fmt.Errorf("indexer crashed: %v", err)
	}

	return nil
}

func (etl *ETLService) awaitReadiness() error {
	etl.logger.Infof("awaiting readiness")
	attempts := 0

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		attempts++
		if attempts > 60 {
			return fmt.Errorf("timed out waiting for readiness")
		}

		res, err := etl.core.GetStatus(context.Background(), connect.NewRequest(&v1.GetStatusRequest{}))
		if err != nil {
			continue
		}

		if res.Msg.Ready {
			return nil
		}
	}

	return nil
}

func (etl *ETLService) indexBlocks() error {
	for {
		// Get the latest indexed block height
		latestHeight, err := etl.db.GetLatestIndexedBlock(context.Background())
		if err != nil {
			// If no records exist, start from block 1
			if errors.Is(err, pgx.ErrNoRows) {
				if etl.startingBlockHeight > 0 {
					// Start from block 1 (nextHeight will be 1)
					latestHeight = etl.startingBlockHeight - 1
				} else {
					// Start from block 1 (nextHeight will be 1)
					latestHeight = 0
				}
			} else {
				etl.logger.Errorf("error getting latest indexed block: %v", err)
				continue
			}
		}

		// Get the next block
		nextHeight := latestHeight + 1
		block, err := etl.core.GetBlock(context.Background(), connect.NewRequest(&v1.GetBlockRequest{
			Height: nextHeight,
		}))
		if err != nil {
			etl.logger.Errorf("error getting block %d: %v", nextHeight, err)
			continue
		}

		if block.Msg.Block.Height < 0 {
			continue
		}

		_, err = etl.db.InsertBlock(context.Background(), db.InsertBlockParams{
			ProposerAddress: block.Msg.Block.Proposer,
			BlockHeight:     block.Msg.Block.Height,
			BlockTime:       pgtype.Timestamp{Time: block.Msg.Block.Timestamp.AsTime(), Valid: true},
		})
		if err != nil {
			etl.logger.Errorf("error inserting block %d: %v", nextHeight, err)
			continue
		}

		txs := block.Msg.Block.Transactions
		for index, tx := range txs {
			txType := ""

			switch signedTx := tx.Transaction.Transaction.(type) {
			case *v1.SignedTransaction_Plays:
				txType = TxTypePlay
				for _, play := range signedTx.Plays.GetPlays() {
					etl.db.InsertPlay(context.Background(), db.InsertPlayParams{
						Address:     play.UserId,
						TrackID:     play.TrackId,
						City:        play.City,
						Region:      play.Region,
						Country:     play.Country,
						PlayedAt:    pgtype.Timestamp{Time: play.Timestamp.AsTime(), Valid: true},
						BlockHeight: block.Msg.Block.Height,
						TxHash:      tx.Hash,
					})

					// TODO: persist lat long in db, only supported in streams
					go func() {
						// check if city, region, country are not empty
						if play.City == "" || play.Region == "" || play.Country == "" {
							return
						}

						latLong, err := etl.locationDB.GetLatLong(context.Background(), play.City, play.Region, play.Country)
						if err != nil {
							return
						}

						etl.playPubsub.Publish(context.Background(), PlayTopic, &etlv1.TrackPlay{
							Address:   play.UserId,
							TrackId:   play.TrackId,
							City:      play.City,
							Region:    play.Region,
							Country:   play.Country,
							PlayedAt:  play.Timestamp,
							Latitude:  latLong.Latitude,
							Longitude: latLong.Longitude,
						})
					}()
				}
			case *v1.SignedTransaction_ManageEntity:
				txType = TxTypeManageEntity
				me := signedTx.ManageEntity
				etl.db.InsertManageEntity(context.Background(), db.InsertManageEntityParams{
					Address:     me.GetSigner(),
					EntityType:  me.GetEntityType(),
					EntityID:    me.GetEntityId(),
					Action:      me.GetAction(),
					Metadata:    pgtype.Text{String: me.GetMetadata(), Valid: true},
					Signature:   me.GetSignature(),
					Signer:      me.GetSigner(),
					Nonce:       me.GetNonce(),
					BlockHeight: block.Msg.Block.Height,
					TxHash:      tx.Hash,
				})
			case *v1.SignedTransaction_ValidatorRegistration:
				txType = TxTypeValidatorRegistrationLegacy
				vr := signedTx.ValidatorRegistration
				etl.db.InsertValidatorRegistrationLegacy(context.Background(), db.InsertValidatorRegistrationLegacyParams{
					Endpoint:     vr.Endpoint,
					CometAddress: vr.CometAddress,
					EthBlock:     vr.EthBlock,
					NodeType:     vr.NodeType,
					SpID:         vr.SpId,
					PubKey:       vr.PubKey,
					Power:        vr.Power,
					BlockHeight:  block.Msg.Block.Height,
					TxHash:       tx.Hash,
				})
			case *v1.SignedTransaction_SlaRollup:
				txType = TxTypeSlaRollup
				sr := signedTx.SlaRollup
				slaRollup, err := etl.db.InsertSlaRollup(context.Background(), db.InsertSlaRollupParams{
					Timestamp:   pgtype.Timestamp{Time: sr.Timestamp.AsTime(), Valid: true},
					BlockStart:  sr.BlockStart,
					BlockEnd:    sr.BlockEnd,
					BlockHeight: block.Msg.Block.Height,
					TxHash:      tx.Hash,
				})
				if err != nil {
					etl.logger.Errorf("error inserting SLA rollup: %v", err)
					continue
				}
				// Insert SLA node reports
				for _, report := range sr.Reports {
					etl.db.InsertSlaNodeReport(context.Background(), db.InsertSlaNodeReportParams{
						SlaRollupID:       slaRollup.ID,
						Address:           report.Address,
						NumBlocksProposed: report.NumBlocksProposed,
						BlockHeight:       block.Msg.Block.Height,
						TxHash:            tx.Hash,
					})
				}
			case *v1.SignedTransaction_ValidatorDeregistration:
				txType = TxTypeValidatorMisbehaviorDeregistration
				vd := signedTx.ValidatorDeregistration
				etl.db.InsertValidatorMisbehaviorDeregistration(context.Background(), db.InsertValidatorMisbehaviorDeregistrationParams{
					CometAddress: vd.CometAddress,
					PubKey:       vd.PubKey,
					BlockHeight:  block.Msg.Block.Height,
					TxHash:       tx.Hash,
				})
			case *v1.SignedTransaction_StorageProof:
				txType = TxTypeStorageProof
				sp := signedTx.StorageProof
				etl.db.InsertStorageProof(context.Background(), db.InsertStorageProofParams{
					Height:          sp.Height,
					Address:         sp.Address,
					ProverAddresses: sp.ProverAddresses,
					Cid:             sp.Cid,
					ProofSignature:  sp.ProofSignature,
					BlockHeight:     block.Msg.Block.Height,
					TxHash:          tx.Hash,
				})
			case *v1.SignedTransaction_StorageProofVerification:
				txType = TxTypeStorageProofVerification
				spv := signedTx.StorageProofVerification
				etl.db.InsertStorageProofVerification(context.Background(), db.InsertStorageProofVerificationParams{
					Height:      spv.Height,
					Proof:       spv.Proof,
					BlockHeight: block.Msg.Block.Height,
					TxHash:      tx.Hash,
				})
			case *v1.SignedTransaction_Attestation:
				at := signedTx.Attestation
				if at.GetValidatorRegistration() != nil {
					txType = TxTypeValidatorRegistration
					vr := at.GetValidatorRegistration()
					etl.db.InsertValidatorRegistration(context.Background(), db.InsertValidatorRegistrationParams{
						Address:      block.Msg.Block.Proposer,
						Endpoint:     vr.Endpoint,
						CometAddress: vr.CometAddress,
						EthBlock:     fmt.Sprintf("%d", vr.EthBlock),
						NodeType:     vr.NodeType,
						Spid:         vr.SpId,
						CometPubkey:  vr.PubKey,
						VotingPower:  vr.Power,
						BlockHeight:  block.Msg.Block.Height,
						TxHash:       tx.Hash,
					})
				}
				if at.GetValidatorDeregistration() != nil {
					txType = TxTypeValidatorDeregistration
					vd := at.GetValidatorDeregistration()
					etl.db.InsertValidatorDeregistration(context.Background(), db.InsertValidatorDeregistrationParams{
						CometAddress: vd.CometAddress,
						CometPubkey:  vd.PubKey,
						BlockHeight:  block.Msg.Block.Height,
						TxHash:       tx.Hash,
					})
				}
			case *v1.SignedTransaction_Release:
				txType = TxTypeRelease
				// Convert the release message to JSON for storage
				releaseData, err := protojson.Marshal(signedTx.Release)
				if err != nil {
					etl.logger.Errorf("error marshaling release data: %v", err)
					continue
				}
				etl.db.InsertRelease(context.Background(), db.InsertReleaseParams{
					ReleaseData: releaseData,
					BlockHeight: block.Msg.Block.Height,
					TxHash:      tx.Hash,
				})
			}

			etl.db.InsertTransaction(context.Background(), db.InsertTransactionParams{
				TxHash:      tx.Hash,
				BlockHeight: block.Msg.Block.Height,
				Index:       int64(index),
				TxType:      txType,
			})
		}

		go func() {
			if err == nil {
				etl.blockPubsub.Publish(context.Background(), BlockTopic, &etlv1.Block{
					Height:    block.Msg.Block.Height,
					Proposer:  block.Msg.Block.Proposer,
					Timestamp: block.Msg.Block.Timestamp,
				})
			}
		}()

		if etl.endingBlockHeight > 0 && block.Msg.Block.Height >= etl.endingBlockHeight {
			etl.logger.Infof("ending block height reached, stopping etl service")
			return nil
		}
	}
}
