package etl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"connectrpc.com/connect"
	corev1 "github.com/AudiusProject/audiusd/pkg/api/core/v1"
	"github.com/AudiusProject/audiusd/pkg/etl/db"
	"github.com/AudiusProject/audiusd/pkg/etl/location"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
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

	// Initialize pubsub instances
	etl.blockPubsub = NewPubsub[*db.EtlBlock]()
	etl.playPubsub = NewPubsub[*db.EtlPlay]()

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
	g, gCtx := errgroup.WithContext(ctx)

	// Start materialized view refresher in errgroup
	g.Go(func() error {
		return etl.mvRefresher.Start(gCtx)
	})

	// Start PostgreSQL notification listener
	g.Go(func() error {
		return etl.startPgNotifyListener(gCtx)
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
		err = etl.db.InsertBlock(context.Background(), db.InsertBlockParams{
			ProposerAddress: block.Msg.Block.Proposer,
			BlockHeight:     block.Msg.Block.Height,
			BlockTime:       pgtype.Timestamp{Time: block.Msg.Block.Timestamp.AsTime(), Valid: true},
		})
		if err != nil {
			etl.logger.Errorf("error inserting block %d: %v", nextHeight, err)
			continue
		}

		var wg sync.WaitGroup
		wg.Add(len(block.Msg.Block.Transactions))

		for index := range block.Msg.Block.Transactions {
			go func(block *corev1.Block, index int) {
				defer wg.Done()

				tx := block.Transactions[index]
				insertTxParams := db.InsertTransactionParams{
					TxHash:      tx.Hash,
					BlockHeight: block.Height,
					TxIndex:     int32(index),
					TxType:      "", // We'll update this after determining the type
					CreatedAt:   pgtype.Timestamp{Time: block.Timestamp.AsTime(), Valid: true},
				}

				switch signedTx := tx.Transaction.Transaction.(type) {
				case *corev1.SignedTransaction_Plays:
					insertTxParams.TxType = TxTypePlay
					// Process plays with batch insert
					plays := signedTx.Plays.GetPlays()
					if len(plays) > 0 {
						// Prepare batch insert parameters
						userIDs := make([]string, len(plays))
						trackIDs := make([]string, len(plays))
						cities := make([]string, len(plays))
						regions := make([]string, len(plays))
						countries := make([]string, len(plays))
						playedAts := make([]pgtype.Timestamp, len(plays))
						blockHeights := make([]int64, len(plays))
						txHashes := make([]string, len(plays))
						listenedAts := make([]pgtype.Timestamp, len(plays))
						recordedAts := make([]pgtype.Timestamp, len(plays))

						for i, play := range plays {
							userIDs[i] = play.UserId
							trackIDs[i] = play.TrackId
							cities[i] = play.City
							regions[i] = play.Region
							countries[i] = play.Country
							playedAts[i] = pgtype.Timestamp{Time: play.Timestamp.AsTime(), Valid: true}
							blockHeights[i] = block.Height
							txHashes[i] = tx.Hash
							listenedAts[i] = pgtype.Timestamp{Time: play.Timestamp.AsTime(), Valid: true}
							recordedAts[i] = pgtype.Timestamp{Time: block.Timestamp.AsTime(), Valid: true}
						}

						// Batch insert all plays
						err = etl.db.InsertPlays(context.Background(), db.InsertPlaysParams{
							Column1:  userIDs,
							Column2:  trackIDs,
							Column3:  cities,
							Column4:  regions,
							Column5:  countries,
							Column6:  playedAts,
							Column7:  blockHeights,
							Column8:  txHashes,
							Column9:  listenedAts,
							Column10: recordedAts,
						})
						if err != nil {
							etl.logger.Errorf("error batch inserting plays: %v", err)
						}

						/**
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
						*/
					}

				case *corev1.SignedTransaction_ManageEntity:
					insertTxParams.TxType = TxTypeManageEntity
					me := signedTx.ManageEntity

					// Insert address first
					err := etl.db.InsertAddress(context.Background(), db.InsertAddressParams{
						Address:              me.GetSigner(),
						PubKey:               nil,
						FirstSeenBlockHeight: pgtype.Int8{Int64: block.Height, Valid: true},
						CreatedAt:            pgtype.Timestamp{Time: block.Timestamp.AsTime(), Valid: true},
					})
					if err != nil {
						etl.logger.Errorf("error inserting address %s: %v", me.GetSigner(), err)
					}

					err = etl.db.InsertManageEntity(context.Background(), db.InsertManageEntityParams{
						Address:     me.GetSigner(),
						EntityType:  me.GetEntityType(),
						EntityID:    me.GetEntityId(),
						Action:      me.GetAction(),
						Metadata:    pgtype.Text{String: me.GetMetadata(), Valid: me.GetMetadata() != ""},
						Signature:   me.GetSignature(),
						Signer:      me.GetSigner(),
						Nonce:       me.GetNonce(),
						BlockHeight: block.Height,
						TxHash:      tx.Hash,
						CreatedAt:   pgtype.Timestamp{Time: block.Timestamp.AsTime(), Valid: true},
					})
					if err != nil {
						etl.logger.Errorf("error inserting manage entity %s: %v", me.GetSigner(), err)
					}

				case *corev1.SignedTransaction_ValidatorRegistration:
					insertTxParams.TxType = TxTypeValidatorRegistrationLegacy
					// Legacy validator registration - no specific table insert needed
				case *corev1.SignedTransaction_ValidatorDeregistration:
					insertTxParams.TxType = TxTypeValidatorMisbehaviorDeregistration
					vd := signedTx.ValidatorDeregistration
					err = etl.db.InsertValidatorMisbehaviorDeregistration(context.Background(), db.InsertValidatorMisbehaviorDeregistrationParams{
						CometAddress: vd.CometAddress,
						PubKey:       vd.PubKey,
						BlockHeight:  block.Height,
						TxHash:       tx.Hash,
						CreatedAt:    pgtype.Timestamp{Time: block.Timestamp.AsTime(), Valid: true},
					})
					if err != nil {
						etl.logger.Errorf("error inserting validator misbehavior deregistration: %v", err)
					}
				case *corev1.SignedTransaction_SlaRollup:
					insertTxParams.TxType = TxTypeSlaRollup
					sr := signedTx.SlaRollup
					err = etl.db.InsertSlaRollup(context.Background(), db.InsertSlaRollupParams{
						BlockStart:     sr.BlockStart,
						BlockEnd:       sr.BlockEnd,
						BlockHeight:    block.Height,
						ValidatorCount: int32(len(sr.Reports)),
						BlockQuota:     0, // Set default or calculate
						TxHash:         tx.Hash,
						CreatedAt:      pgtype.Timestamp{Time: block.Timestamp.AsTime(), Valid: true},
					})
					if err != nil {
						etl.logger.Errorf("error inserting SLA rollup: %v", err)
					}

					// Insert SLA node reports
					for _, report := range sr.Reports {
						err = etl.db.InsertSlaNodeReport(context.Background(), db.InsertSlaNodeReportParams{
							SlaRollupID:        1, // This would need to be the actual rollup ID
							Address:            report.Address,
							NumBlocksProposed:  report.NumBlocksProposed,
							ChallengesReceived: 0, // Set default
							ChallengesFailed:   0, // Set default
							BlockHeight:        block.Height,
							TxHash:             tx.Hash,
							CreatedAt:          pgtype.Timestamp{Time: block.Timestamp.AsTime(), Valid: true},
						})
						if err != nil {
							etl.logger.Errorf("error inserting SLA node report: %v", err)
						}
					}
				case *corev1.SignedTransaction_StorageProof:
					insertTxParams.TxType = TxTypeStorageProof
					sp := signedTx.StorageProof
					err = etl.db.InsertStorageProof(context.Background(), db.InsertStorageProofParams{
						Height:          sp.Height,
						Address:         sp.Address,
						ProverAddresses: sp.ProverAddresses,
						Cid:             sp.Cid,
						ProofSignature:  sp.ProofSignature,
						BlockHeight:     block.Height,
						TxHash:          tx.Hash,
						CreatedAt:       pgtype.Timestamp{Time: block.Timestamp.AsTime(), Valid: true},
					})
					if err != nil {
						etl.logger.Errorf("error inserting storage proof: %v", err)
					}
				case *corev1.SignedTransaction_StorageProofVerification:
					insertTxParams.TxType = TxTypeStorageProofVerification
					spv := signedTx.StorageProofVerification
					err = etl.db.InsertStorageProofVerification(context.Background(), db.InsertStorageProofVerificationParams{
						Height:      spv.Height,
						Proof:       spv.Proof,
						BlockHeight: block.Height,
						TxHash:      tx.Hash,
						CreatedAt:   pgtype.Timestamp{Time: block.Timestamp.AsTime(), Valid: true},
					})
					if err != nil {
						etl.logger.Errorf("error inserting storage proof verification: %v", err)
					}
				case *corev1.SignedTransaction_Attestation:
					at := signedTx.Attestation
					if vr := at.GetValidatorRegistration(); vr != nil {
						insertTxParams.TxType = TxTypeValidatorRegistration
						err = etl.db.InsertValidatorRegistration(context.Background(), db.InsertValidatorRegistrationParams{
							Address:      vr.DelegateWallet,
							Endpoint:     vr.Endpoint,
							CometAddress: vr.CometAddress,
							EthBlock:     fmt.Sprintf("%d", vr.EthBlock),
							NodeType:     vr.NodeType,
							Spid:         vr.SpId,
							CometPubkey:  vr.PubKey,
							VotingPower:  vr.Power,
							BlockHeight:  block.Height,
							TxHash:       tx.Hash,
						})
						if err != nil {
							etl.logger.Errorf("error inserting validator registration: %v", err)
						}
						// insert RegisteredValidator record
						err = etl.db.RegisterValidator(context.Background(), db.RegisterValidatorParams{
							Address:        vr.DelegateWallet,
							Endpoint:       vr.Endpoint,
							CometAddress:   vr.CometAddress,
							NodeType:       vr.NodeType,
							Spid:           vr.SpId,
							VotingPower:    vr.Power,
							Status:         "active",
							RegisteredAt:   block.Height,
							DeregisteredAt: pgtype.Int8{Valid: false},
							CreatedAt:      pgtype.Timestamp{Time: block.Timestamp.AsTime(), Valid: true},
							UpdatedAt:      pgtype.Timestamp{Time: block.Timestamp.AsTime(), Valid: true},
						})
						if err != nil {
							etl.logger.Errorf("error registering validator: %v", err)
						}
					}
					if vd := at.GetValidatorDeregistration(); vd != nil {
						insertTxParams.TxType = TxTypeValidatorDeregistration
						err = etl.db.InsertValidatorDeregistration(context.Background(), db.InsertValidatorDeregistrationParams{
							CometAddress: vd.CometAddress,
							CometPubkey:  vd.PubKey,
							BlockHeight:  block.Height,
							TxHash:       tx.Hash,
						})
						if err != nil {
							etl.logger.Errorf("error inserting validator deregistration: %v", err)
						}
						// insert DeregisteredValidator record
						err = etl.db.DeregisterValidator(context.Background(), db.DeregisterValidatorParams{
							DeregisteredAt: pgtype.Int8{Int64: block.Height, Valid: true},
							UpdatedAt:      pgtype.Timestamp{Time: block.Timestamp.AsTime(), Valid: true},
							Status:         "deregistered",
							CometAddress:   vd.CometAddress,
						})
						if err != nil {
							etl.logger.Errorf("error deregistering validator: %v", err)
						}
					}
				}

				err = etl.db.InsertTransaction(context.Background(), insertTxParams)
				if err != nil {
					etl.logger.Errorf("error inserting transaction %s: %v", tx.Hash, err)
					return
				}

			}(block.Msg.Block, index)
		}

		wg.Wait()

		// TODO: use pgnotify to publish block and play events to pubsub

		if etl.endingBlockHeight > 0 && block.Msg.Block.Height >= etl.endingBlockHeight {
			etl.logger.Infof("ending block height reached, stopping etl service")
			return nil
		}
	}
}

func (etl *ETLService) startPgNotifyListener(ctx context.Context) error {
	conn, err := pgx.Connect(ctx, etl.dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect for notifications: %w", err)
	}
	defer conn.Close(ctx)

	// Listen to both channels
	_, err = conn.Exec(ctx, "LISTEN new_block")
	if err != nil {
		return fmt.Errorf("failed to listen to new_block: %w", err)
	}

	_, err = conn.Exec(ctx, "LISTEN new_plays")
	if err != nil {
		return fmt.Errorf("failed to listen to new_plays: %w", err)
	}

	for {
		notification, err := conn.WaitForNotification(ctx)
		if err != nil {
			return fmt.Errorf("error waiting for notification: %w", err)
		}

		switch notification.Channel {
		case "new_block":
			block := &db.EtlBlock{}
			err = json.Unmarshal([]byte(notification.Payload), block)
			if err != nil {
				etl.logger.Errorf("error unmarshalling block: %v", err)
				continue
			}
			if etl.blockPubsub.HasSubscribers(BlockTopic) {
				etl.blockPubsub.Publish(context.Background(), BlockTopic, block)
			}
		case "new_plays":
			play := &db.EtlPlay{}
			err = json.Unmarshal([]byte(notification.Payload), play)
			if err != nil {
				etl.logger.Errorf("error unmarshalling play: %v", err)
				continue
			}
			if etl.playPubsub.HasSubscribers(PlayTopic) {
				etl.playPubsub.Publish(context.Background(), PlayTopic, play)
			}
		}
	}
}
