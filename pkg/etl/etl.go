package etl

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	corev1 "github.com/AudiusProject/audiusd/pkg/api/core/v1"
	etlv1 "github.com/AudiusProject/audiusd/pkg/api/etl/v1"
	"github.com/AudiusProject/audiusd/pkg/etl/db"
	"github.com/AudiusProject/audiusd/pkg/etl/location"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
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
		etl.logger.Errorf("error creating location service: %v", err)
		return fmt.Errorf("error creating location service: %v", err)
	}
	etl.logger.Infof("location service initialized successfully")
	etl.locationDB = locationDB

	// Initialize materialized view refresher
	etl.mvRefresher = NewMaterializedViewRefresher(etl.pool, etl.logger)

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

	ctx := context.Background()
	g, _ := errgroup.WithContext(ctx)

	// Start materialized view refresher in errgroup
	g.Go(func() error {
		return etl.mvRefresher.Start(ctx)
	})

	g.Go(func() error {
		if err := etl.indexBlocks(); err != nil {
			return fmt.Errorf("error indexing blocks: %v", err)
		}

		return nil
	})

	return g.Wait()
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

		res, err := etl.core.GetStatus(context.Background(), connect.NewRequest(&corev1.GetStatusRequest{}))
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
		block, err := etl.core.GetBlock(context.Background(), connect.NewRequest(&corev1.GetBlockRequest{
			Height: nextHeight,
		}))
		if err != nil {
			etl.logger.Errorf("error getting block %d: %v", nextHeight, err)
			continue
		}

		if block.Msg.Block.Height < 0 {
			continue
		}

		// Insert block first
		blockRecord, err := etl.db.InsertBlock(context.Background(), db.InsertBlockParams{
			ProposerAddress: block.Msg.Block.Proposer,
			BlockHeight:     block.Msg.Block.Height,
			BlockTime:       pgtype.Timestamp{Time: block.Msg.Block.Timestamp.AsTime(), Valid: true},
		})
		if err != nil {
			etl.logger.Errorf("error inserting block %d: %v", nextHeight, err)
			continue
		}

		// Process transactions
		txs := block.Msg.Block.Transactions
		for index, tx := range txs {
			txType := ""

			// Insert transaction record first
			transactionRecord, err := etl.db.InsertTransaction(context.Background(), db.InsertTransactionParams{
				TxHash:  tx.Hash,
				BlockID: blockRecord.ID,
				TxIndex: int32(index),
				TxType:  "", // We'll update this after determining the type
			})
			if err != nil {
				// If insertion failed due to conflict, try to get the existing transaction
				if strings.Contains(err.Error(), "no rows") {
					// Transaction already exists, get it
					existingTx, getErr := etl.db.GetTransaction(context.Background(), tx.Hash)
					if getErr != nil {
						etl.logger.Errorf("error getting existing transaction %s: %v", tx.Hash, getErr)
						continue
					}
					// Check if it's in the same block - if so, skip processing
					if existingTx.BlockHeight == block.Msg.Block.Height {
						etl.logger.Debugf("transaction %s already processed in block %d, skipping", tx.Hash, block.Msg.Block.Height)
						continue
					}
					// If it's in a different block, we need to create a new record for this block
					// This should not happen with our composite unique constraint, but log it
					etl.logger.Warn("transaction exists in different block", "tx_hash", tx.Hash, "existing_block", existingTx.BlockHeight, "current_block", block.Msg.Block.Height)
					continue
				}
				etl.logger.Errorf("error inserting transaction %s: %v", tx.Hash, err)
				continue
			}

			// Helper function to get or create address
			getOrCreateAddress := func(address string) (int32, error) {
				addressID, err := etl.db.GetOrCreateAddress(context.Background(), db.GetOrCreateAddressParams{
					Address:          address,
					FirstSeenBlockID: pgtype.Int4{Int32: blockRecord.ID, Valid: true},
				})
				if err != nil {
					return 0, err
				}
				return addressID, nil
			}

			switch signedTx := tx.Transaction.Transaction.(type) {
			case *corev1.SignedTransaction_Plays:
				txType = TxTypePlay
				for _, play := range signedTx.Plays.GetPlays() {
					addressID, err := getOrCreateAddress(play.UserId)
					if err != nil {
						etl.logger.Errorf("error getting/creating address for play: %v", err)
						continue
					}

					_, err = etl.db.InsertPlay(context.Background(), db.InsertPlayParams{
						TransactionID: transactionRecord.ID,
						AddressID:     addressID,
						TrackID:       play.TrackId,
						City:          pgtype.Text{String: play.City, Valid: play.City != ""},
						Region:        pgtype.Text{String: play.Region, Valid: play.Region != ""},
						Country:       pgtype.Text{String: play.Country, Valid: play.Country != ""},
						PlayedAt:      pgtype.Timestamp{Time: play.Timestamp.AsTime(), Valid: true},
					})
					if err != nil {
						etl.logger.Errorf("error inserting play: %v", err)
						continue
					}

					// TODO: persist lat long in db, only supported in streams
					// check if city, region, country are not empty and if there are subscribers
					if play.City != "" && play.Region != "" && play.Country != "" && etl.playPubsub.HasSubscribers(PlayTopic) {
						latLong, err := etl.locationDB.GetLatLong(context.Background(), play.City, play.Region, play.Country)
						if err == nil {
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
						}
					}
				}

			case *corev1.SignedTransaction_ManageEntity:
				txType = TxTypeManageEntity
				me := signedTx.ManageEntity

				addressID, err := getOrCreateAddress(me.GetSigner())
				if err != nil {
					etl.logger.Errorf("error getting/creating address for manage entity: %v", err)
					continue
				}

				signerAddressID, err := getOrCreateAddress(me.GetSigner())
				if err != nil {
					etl.logger.Errorf("error getting/creating signer address for manage entity: %v", err)
					continue
				}

				_, err = etl.db.InsertManageEntity(context.Background(), db.InsertManageEntityParams{
					TransactionID:   transactionRecord.ID,
					AddressID:       addressID,
					EntityType:      me.GetEntityType(),
					EntityID:        me.GetEntityId(),
					Action:          me.GetAction(),
					Metadata:        pgtype.Text{String: me.GetMetadata(), Valid: me.GetMetadata() != ""},
					Signature:       me.GetSignature(),
					SignerAddressID: signerAddressID,
					Nonce:           me.GetNonce(),
				})
				if err != nil {
					etl.logger.Errorf("error inserting manage entity: %v", err)
					continue
				}

			case *corev1.SignedTransaction_ValidatorRegistration:
				txType = TxTypeValidatorRegistrationLegacy
				vr := signedTx.ValidatorRegistration

				_, err = etl.db.InsertValidatorRegistrationLegacy(context.Background(), db.InsertValidatorRegistrationLegacyParams{
					TransactionID: transactionRecord.ID,
					Endpoint:      vr.Endpoint,
					CometAddress:  vr.CometAddress,
					EthBlock:      vr.EthBlock,
					NodeType:      vr.NodeType,
					SpID:          vr.SpId,
					PubKey:        vr.PubKey,
					Power:         vr.Power,
				})
				if err != nil {
					etl.logger.Errorf("error inserting legacy validator registration: %v", err)
					continue
				}

			case *corev1.SignedTransaction_SlaRollup:
				txType = TxTypeSlaRollup
				sr := signedTx.SlaRollup

				slaRollup, err := etl.db.InsertSlaRollup(context.Background(), db.InsertSlaRollupParams{
					TransactionID: transactionRecord.ID,
					Timestamp:     pgtype.Timestamp{Time: sr.Timestamp.AsTime(), Valid: true},
					BlockStart:    sr.BlockStart,
					BlockEnd:      sr.BlockEnd,
				})
				if err != nil {
					etl.logger.Errorf("error inserting SLA rollup: %v", err)
					continue
				}

				// Insert SLA node reports
				for _, report := range sr.Reports {
					addressID, err := getOrCreateAddress(report.Address)
					if err != nil {
						etl.logger.Errorf("error getting/creating address for SLA report: %v", err)
						continue
					}

					_, err = etl.db.InsertSlaNodeReport(context.Background(), db.InsertSlaNodeReportParams{
						SlaRollupID:       slaRollup.ID,
						AddressID:         addressID,
						NumBlocksProposed: report.NumBlocksProposed,
					})
					if err != nil {
						etl.logger.Errorf("error inserting SLA node report: %v", err)
						continue
					}
				}

			case *corev1.SignedTransaction_ValidatorDeregistration:
				txType = TxTypeValidatorMisbehaviorDeregistration
				vd := signedTx.ValidatorDeregistration

				_, err = etl.db.InsertValidatorMisbehaviorDeregistration(context.Background(), db.InsertValidatorMisbehaviorDeregistrationParams{
					TransactionID: transactionRecord.ID,
					CometAddress:  vd.CometAddress,
					PubKey:        vd.PubKey,
				})
				if err != nil {
					etl.logger.Errorf("error inserting validator misbehavior deregistration: %v", err)
					continue
				}

			case *corev1.SignedTransaction_StorageProof:
				txType = TxTypeStorageProof
				sp := signedTx.StorageProof

				addressID, err := getOrCreateAddress(sp.Address)
				if err != nil {
					etl.logger.Errorf("error getting/creating address for storage proof: %v", err)
					continue
				}

				_, err = etl.db.InsertStorageProof(context.Background(), db.InsertStorageProofParams{
					TransactionID:   transactionRecord.ID,
					Height:          sp.Height,
					AddressID:       addressID,
					ProverAddresses: sp.ProverAddresses,
					Cid:             sp.Cid,
					ProofSignature:  sp.ProofSignature,
				})
				if err != nil {
					etl.logger.Errorf("error inserting storage proof: %v", err)
					continue
				}

			case *corev1.SignedTransaction_StorageProofVerification:
				txType = TxTypeStorageProofVerification
				spv := signedTx.StorageProofVerification

				_, err = etl.db.InsertStorageProofVerification(context.Background(), db.InsertStorageProofVerificationParams{
					TransactionID: transactionRecord.ID,
					Height:        spv.Height,
					Proof:         spv.Proof,
				})
				if err != nil {
					etl.logger.Errorf("error inserting storage proof verification: %v", err)
					continue
				}

			case *corev1.SignedTransaction_Attestation:
				at := signedTx.Attestation
				if at.GetValidatorRegistration() != nil {
					txType = TxTypeValidatorRegistration
					vr := at.GetValidatorRegistration()

					addressID, err := getOrCreateAddress(vr.DelegateWallet)
					if err != nil {
						etl.logger.Errorf("error getting/creating address for validator registration: %v", err)
						continue
					}

					_, err = etl.db.InsertValidatorRegistration(context.Background(), db.InsertValidatorRegistrationParams{
						TransactionID: transactionRecord.ID,
						AddressID:     addressID,
						Endpoint:      vr.Endpoint,
						CometAddress:  vr.CometAddress,
						EthBlock:      fmt.Sprintf("%d", vr.EthBlock),
						NodeType:      vr.NodeType,
						Spid:          vr.SpId,
						CometPubkey:   vr.PubKey,
						VotingPower:   vr.Power,
					})
					if err != nil {
						etl.logger.Errorf("error inserting validator registration: %v", err)
						continue
					}
				}
				if at.GetValidatorDeregistration() != nil {
					txType = TxTypeValidatorDeregistration
					vd := at.GetValidatorDeregistration()

					_, err = etl.db.InsertValidatorDeregistration(context.Background(), db.InsertValidatorDeregistrationParams{
						TransactionID: transactionRecord.ID,
						CometAddress:  vd.CometAddress,
						CometPubkey:   vd.PubKey,
					})
					if err != nil {
						etl.logger.Errorf("error inserting validator deregistration: %v", err)
						continue
					}
				}

			case *corev1.SignedTransaction_Release:
				txType = TxTypeRelease
				// Convert the release message to JSON for storage
				releaseData, err := protojson.Marshal(signedTx.Release)
				if err != nil {
					etl.logger.Errorf("error marshaling release data: %v", err)
					continue
				}

				_, err = etl.db.InsertRelease(context.Background(), db.InsertReleaseParams{
					TransactionID: transactionRecord.ID,
					ReleaseData:   releaseData,
				})
				if err != nil {
					etl.logger.Errorf("error inserting release: %v", err)
					continue
				}
			}

			// Update transaction type if we determined it
			if txType != "" {
				err = etl.db.UpdateTransactionType(context.Background(), db.UpdateTransactionTypeParams{
					ID:     transactionRecord.ID,
					TxType: txType,
				})
				if err != nil {
					etl.logger.Errorf("error updating transaction type: %v", err)
				}
			}
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
