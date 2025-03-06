package server

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/AudiusProject/audiusd/pkg/core/gen/core_proto"
	"github.com/cometbft/cometbft/types"
	"google.golang.org/protobuf/proto"
)

func (s *Server) startSubscriber() error {
	<-s.awaitRpcReady

	node := s.node
	eb := node.EventBus()

	if eb == nil {
		return errors.New("event bus not ready")
	}

	subscriberID := "block-cache-subscriber"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	query := types.EventQueryNewBlock
	subscription, err := eb.Subscribe(ctx, subscriberID, query)
	if err != nil {
		return fmt.Errorf("failed to subscribe to NewBlock events: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Stopping block event subscription")
			return nil
		case msg := <-subscription.Out():
			blockEvent := msg.Data().(types.EventDataNewBlock)

			s.updateLatestBlock(&blockEvent)
			s.broadcastPubsubEvents(&blockEvent)

		case err := <-subscription.Canceled():
			s.logger.Errorf("Subscription cancelled: %v", err)
			return nil
		}
	}
}

func (s *Server) updateLatestBlock(blockEvent *types.EventDataNewBlock) {
	blockHeight := blockEvent.Block.Height
	atomic.StoreInt64(&s.cache.currentHeight, blockHeight)
}

func (s *Server) broadcastPubsubEvents(blockEvent *types.EventDataNewBlock) {
	ctx := context.Background()

	height := blockEvent.Block.Height
	txs, err := s.db.GetBlockTransactions(ctx, height)
	if err != nil {
		s.logger.Errorf("block %d not available for pubsub broadcast: %v", height, err)
		return
	}

	for _, tx := range txs {
		txbytes := tx.Transaction
		var tx core_proto.SignedTransaction
		err := proto.Unmarshal(txbytes, &tx)
		if err != nil {
			continue
		}

		if plays := tx.GetPlays(); plays != nil {
			for _, play := range plays.GetPlays() {
				s.playsPubsub.Publish(ctx, play.TrackId, play)
			}
		}
	}
}
