// Package store provides PebbleDB-backed message persistence for AgentChat.
// This is the L1 cache required by fgt-sdk conventions.
package store

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/biodoia/agentchat/pkg/types"
	"github.com/cockroachdb/pebble"
)

// MessageStore persists messages in PebbleDB.
type MessageStore struct {
	db  *pebble.DB
	log *slog.Logger
}

// New opens or creates a PebbleDB message store.
func New(dataDir string, log *slog.Logger) (*MessageStore, error) {
	dbPath := filepath.Join(dataDir, "messages.db")
	db, err := pebble.Open(dbPath, &pebble.Options{})
	if err != nil {
		return nil, fmt.Errorf("store: open pebble: %w", err)
	}
	log.Info("PebbleDB opened", "path", dbPath)
	return &MessageStore{db: db, log: log}, nil
}

// Close closes the database.
func (s *MessageStore) Close() error {
	return s.db.Close()
}

// key returns a PebbleDB key for a message: group/ts/id
func key(msg *types.Message) []byte {
	return []byte(fmt.Sprintf("%s/%s/%s", msg.Group, msg.TS.Format(time.RFC3339Nano), msg.ID))
}

// Save persists a message.
func (s *MessageStore) Save(msg *types.Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("store: marshal: %w", err)
	}
	return s.db.Set(key(msg), data, pebble.Sync)
}

// SaveBatch persists multiple messages atomically.
func (s *MessageStore) SaveBatch(msgs []*types.Message) error {
	batch := s.db.NewBatch()
	defer batch.Close()
	for _, msg := range msgs {
		data, err := json.Marshal(msg)
		if err != nil {
			return err
		}
		batch.Set(key(msg), data, nil)
	}
	return batch.Commit(pebble.Sync)
}

// LoadRecent returns the last N messages for a group, newest first.
func (s *MessageStore) LoadRecent(group string, limit int) ([]*types.Message, error) {
	prefix := []byte(group + "/")
	iter, err := s.db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: append(prefix, 0xFF),
	})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	// Collect all keys (sorted), then take the last N
	var all []*types.Message
	for iter.First(); iter.Valid(); iter.Next() {
		val, err := iter.ValueAndErr()
		if err != nil {
			return nil, err
		}
		var msg types.Message
		if err := json.Unmarshal(val, &msg); err != nil {
			continue
		}
		all = append(all, &msg)
	}

	// Take last N (reverse to newest first)
	if limit > 0 && len(all) > limit {
		all = all[len(all)-limit:]
	}

	// Reverse for newest first
	for i, j := 0, len(all)-1; i < j; i, j = i+1, j-1 {
		all[i], all[j] = all[j], all[i]
	}

	return all, nil
}

// LoadSince returns messages for a group since a given time.
func (s *MessageStore) LoadSince(group string, since time.Time) ([]*types.Message, error) {
	prefix := []byte(group + "/")
	startKey := []byte(fmt.Sprintf("%s/%s", group, since.Format(time.RFC3339Nano)))

	iter, err := s.db.NewIter(&pebble.IterOptions{
		LowerBound: startKey,
		UpperBound: append(prefix, 0xFF),
	})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var msgs []*types.Message
	for iter.First(); iter.Valid(); iter.Next() {
		val, err := iter.ValueAndErr()
		if err != nil {
			return nil, err
		}
		var msg types.Message
		if err := json.Unmarshal(val, &msg); err != nil {
			continue
		}
		msgs = append(msgs, &msg)
	}
	return msgs, nil
}
