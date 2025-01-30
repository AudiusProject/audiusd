package server

import (
	"fmt"
	"time"

	"github.com/cockroachdb/pebble"
)

func (s *Server) startPebbleCompactor() error {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	opts := &pebble.Options{
		ReadOnly: false,
	}

	for range ticker.C {
		if err := func() error {
			db, err := pebble.Open(s.cometbftConfig.BaseConfig.DBPath+"/state.db", opts)
			if err != nil {
				return fmt.Errorf("could not open pebbledb: %v", err)
			}
			defer db.Close()

			start := []byte{}
			end := []byte{}

			iter, err := db.NewIter(nil)
			if err != nil {
				return err
			}
			defer iter.Close()

			if iter.First() {
				start = append(start, iter.Key()...)
			}
			if iter.Last() {
				end = append(end, iter.Key()...)
			}

			if err := db.Compact(nil, nil, true); err != nil {
				return err
			}
			s.logger.Info("manual pebble compaction succeeded")
			return nil
		}(); err != nil {
			s.logger.Errorf("manual pebble compaction failed: %v", err)
		}
	}
	return nil
}
