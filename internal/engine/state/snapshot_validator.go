package state

import (
	"fmt"

	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/contracts"
)

func ValidateSnapshot(cfg config.Config, snapshot contracts.Snapshot, nowMs int64) error {
	if err := snapshot.Validate(); err != nil {
		return err
	}
	if snapshot.Metadata.SnapshotHash == "" {
		return fmt.Errorf("snapshot metadata snapshot_hash missing")
	}
	expected, err := snapshot.Hash()
	if err != nil {
		return err
	}
	if snapshot.Metadata.SnapshotHash != expected {
		return fmt.Errorf("snapshot metadata snapshot_hash mismatch")
	}
	if err := validateTimestamps(cfg, snapshot, nowMs); err != nil {
		return err
	}
	return nil
}

func validateTimestamps(cfg config.Config, snapshot contracts.Snapshot, nowMs int64) error {
	minTs := nowMs - int64(cfg.RestStaleMsPause)
	maxTs := nowMs + int64(cfg.TimeSyncRecvWindowMs)
	for _, ts := range snapshotTimestamps(snapshot) {
		if ts == 0 {
			return fmt.Errorf("snapshot timestamp missing")
		}
		if ts < minTs {
			return fmt.Errorf("snapshot timestamp too old")
		}
		if ts > maxTs {
			return fmt.Errorf("snapshot timestamp too new")
		}
	}
	return nil
}

func snapshotTimestamps(snapshot contracts.Snapshot) []int64 {
	timestamps := []int64{
		snapshot.Metadata.CreatedTsMs,
		snapshot.Metadata.ExchangeTimeMs,
		snapshot.Metadata.LocalReceivedMs,
		snapshot.Market24h.SourceTsMs,
		snapshot.ReturnsSeries.ComputedTsMs,
	}
	return timestamps
}
