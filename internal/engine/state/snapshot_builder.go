package state

import (
	"fmt"

	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/contracts"
)

const (
	defaultSlippageEntryMakerBps = 1
	defaultSlippageEntryTakerBps = 3
	defaultSlippageExitTakerBps  = 2
)

func BuildSnapshot(cfg config.Config, snapshot contracts.Snapshot, nowMs int64) (contracts.Snapshot, error) {
	if snapshot.Metadata.CreatedTsMs == 0 {
		snapshot.Metadata.CreatedTsMs = nowMs
	}
	if snapshot.Metadata.LocalReceivedMs == 0 {
		snapshot.Metadata.LocalReceivedMs = nowMs
	}
	if snapshot.CostInputs.SlippageEntryMakerBps == 0 &&
		snapshot.CostInputs.SlippageEntryTakerBps == 0 &&
		snapshot.CostInputs.SlippageExitTakerBps == 0 {
		snapshot.CostInputs.SlippageEntryMakerBps = defaultSlippageEntryMakerBps
		snapshot.CostInputs.SlippageEntryTakerBps = defaultSlippageEntryTakerBps
		snapshot.CostInputs.SlippageExitTakerBps = defaultSlippageExitTakerBps
	}
	hash, err := snapshot.Hash()
	if err != nil {
		return contracts.Snapshot{}, err
	}
	if hash == "" {
		return contracts.Snapshot{}, fmt.Errorf("snapshot hash missing")
	}
	snapshot.Metadata.SnapshotHash = hash
	if err := ValidateSnapshot(cfg, snapshot, nowMs); err != nil {
		return contracts.Snapshot{}, err
	}
	return snapshot, nil
}
