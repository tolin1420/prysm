package initialsync

import (
	"context"
	"fmt"
	"sort"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	p2pTypes "github.com/prysmaticlabs/prysm/beacon-chain/p2p/types"
	"github.com/prysmaticlabs/prysm/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	p2ppb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/time/slots"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// forkData represents alternative chain path supported by a given peer.
// Blocks are stored in an ascending slot order. The first block is guaranteed to have parent
// either in DB or initial sync cache.
type forkData struct {
	peer   peer.ID
	blocks []block.SignedBeaconBlock
}

// nonSkippedSlotAfter checks slots after the given one in an attempt to find a non-empty future slot.
// For efficiency only one random slot is checked per epoch, so returned slot might not be the first
// non-skipped slot. This shouldn't be a problem, as in case of adversary peer, we might get incorrect
// data anyway, so code that relies on this function must be robust enough to re-request, if no progress
// is possible with a returned value.
func (f *blocksFetcher) nonSkippedSlotAfter(ctx context.Context, slot types.Slot) (types.Slot, error) {
	ctx, span := trace.StartSpan(ctx, "initialsync.nonSkippedSlotAfter")
	defer span.End()

	headEpoch, targetEpoch, peers := f.calculateHeadAndTargetEpochs()
	log.WithFields(logrus.Fields{
		"start":       slot,
		"headEpoch":   headEpoch,
		"targetEpoch": targetEpoch,
	}).Debug("Searching for non-skipped slot")

	// Exit early if no peers with epoch higher than our known head are found.
	if targetEpoch <= headEpoch {
		return 0, errSlotIsTooHigh
	}

	// Transform peer list to avoid eclipsing (filter, shuffle, trim).
	peers = f.filterPeers(ctx, peers, peersPercentagePerRequest)
	return f.nonSkippedSlotAfterWithPeersTarget(ctx, slot, peers, targetEpoch)
}

// nonSkippedSlotWithPeersTarget traverse peers (supporting a given target epoch), in an attempt
// to find non-skipped slot among returned blocks.
func (f *blocksFetcher) nonSkippedSlotAfterWithPeersTarget(
	ctx context.Context, slot types.Slot, peers []peer.ID, targetEpoch types.Epoch,
) (types.Slot, error) {
	// Exit early if no peers are ready.
	if len(peers) == 0 {
		return 0, errNoPeersAvailable
	}

	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	pidInd := 0

	fetch := func(pid peer.ID, start types.Slot, count, step uint64) (types.Slot, error) {
		req := &p2ppb.BeaconBlocksByRangeRequest{
			StartSlot: start,
			Count:     count,
			Step:      step,
		}
		blocks, err := f.requestBlocks(ctx, req, pid)
		if err != nil {
			return 0, err
		}
		if len(blocks) > 0 {
			for _, block := range blocks {
				if block.Block().Slot() > slot {
					return block.Block().Slot(), nil
				}
			}
		}
		return 0, nil
	}

	// Start by checking several epochs fully, w/o resorting to random sampling.
	start := slot + 1
	end := start + nonSkippedSlotsFullSearchEpochs*slotsPerEpoch
	for ind := start; ind < end; ind += slotsPerEpoch {
		nextSlot, err := fetch(peers[pidInd%len(peers)], ind, uint64(slotsPerEpoch), 1)
		if err != nil {
			return 0, err
		}
		if nextSlot > slot {
			return nextSlot, nil
		}
		pidInd++
	}

	// Quickly find the close enough epoch where a non-empty slot definitely exists.
	// Only single random slot per epoch is checked - allowing to move forward relatively quickly.
	slot += nonSkippedSlotsFullSearchEpochs * slotsPerEpoch
	upperBoundSlot, err := slots.EpochStart(targetEpoch + 1)
	if err != nil {
		return 0, err
	}
	for ind := slot + 1; ind < upperBoundSlot; ind += (slotsPerEpoch * slotsPerEpoch) / 2 {
		start := ind.Add(uint64(f.rand.Intn(int(slotsPerEpoch))))
		nextSlot, err := fetch(peers[pidInd%len(peers)], start, uint64(slotsPerEpoch/2), uint64(slotsPerEpoch))
		if err != nil {
			return 0, err
		}
		pidInd++
		if nextSlot > slot && upperBoundSlot >= nextSlot {
			upperBoundSlot = nextSlot
			break
		}
	}

	// Epoch with non-empty slot is located. Check all slots within two nearby epochs.
	if upperBoundSlot > slotsPerEpoch {
		upperBoundSlot -= slotsPerEpoch
	}
	upperBoundSlot, err = slots.EpochStart(slots.ToEpoch(upperBoundSlot))
	if err != nil {
		return 0, err
	}
	nextSlot, err := fetch(peers[pidInd%len(peers)], upperBoundSlot, uint64(slotsPerEpoch*2), 1)
	if err != nil {
		return 0, err
	}
	s, err := slots.EpochStart(targetEpoch + 1)
	if err != nil {
		return 0, err
	}
	if nextSlot < slot || s < nextSlot {
		return 0, errors.New("invalid range for non-skipped slot")
	}
	return nextSlot, nil
}

// findFork queries all peers that have higher head slot, in an attempt to find
// ones that feature blocks from alternative branches. Once found, peer is further queried
// to find common ancestor slot. On success, all obtained blocks and peer is returned.
func (f *blocksFetcher) findFork(ctx context.Context, slot types.Slot) (*forkData, error) {
	ctx, span := trace.StartSpan(ctx, "initialsync.findFork")
	defer span.End()

	// Safe-guard, since previous epoch is used when calculating.
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	if slot < slotsPerEpoch*2 {
		return nil, fmt.Errorf("slot is too low to backtrack, min. expected %d", slotsPerEpoch*2)
	}

	// The current slot's epoch must be after the finalization epoch,
	// triggering backtracking on earlier epochs is unnecessary.
	finalizedEpoch := f.chain.FinalizedCheckpt().Epoch
	epoch := slots.ToEpoch(slot)
	if epoch <= finalizedEpoch {
		return nil, errors.New("slot is not after the finalized epoch, no backtracking is necessary")
	}
	// Update slot to the beginning of the current epoch (preserve original slot for comparison).
	slot, err := slots.EpochStart(epoch)
	if err != nil {
		return nil, err
	}

	// Select peers that have higher head slot, and potentially blocks from more favourable fork.
	// Exit early if no peers are ready.
	_, peers := f.p2p.Peers().BestNonFinalized(1, epoch+1)
	if len(peers) == 0 {
		return nil, errNoPeersAvailable
	}
	f.rand.Shuffle(len(peers), func(i, j int) {
		peers[i], peers[j] = peers[j], peers[i]
	})

	// Query all found peers, stop on peer with alternative blocks, and try backtracking.
	for i, pid := range peers {
		log.WithFields(logrus.Fields{
			"peer": pid,
			"step": fmt.Sprintf("%d/%d", i+1, len(peers)),
		}).Debug("Searching for alternative blocks")
		fork, err := f.findForkWithPeer(ctx, pid, slot)
		if err != nil {
			log.WithFields(logrus.Fields{
				"peer":  pid,
				"error": err.Error(),
			}).Debug("No alternative blocks found for peer")
			continue
		}
		return fork, nil
	}
	return nil, errNoPeersWithAltBlocks
}

// findForkWithPeer loads some blocks from a peer in an attempt to find alternative blocks.
func (f *blocksFetcher) findForkWithPeer(ctx context.Context, pid peer.ID, slot types.Slot) (*forkData, error) {
	// Safe-guard, since previous epoch is used when calculating.
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	if slot < slotsPerEpoch*2 {
		return nil, fmt.Errorf("slot is too low to backtrack, min. expected %d", slotsPerEpoch*2)
	}

	// Locate non-skipped slot, supported by a given peer (can survive long periods of empty slots).
	// When searching for non-empty slot, start an epoch earlier - for those blocks we
	// definitely have roots. So, spotting a fork will be easier. It is not a problem if unknown
	// block of the current fork is found: we are searching for forks when FSMs are stuck, so
	// being able to progress on any fork is good.
	pidState, err := f.p2p.Peers().ChainState(pid)
	if err != nil {
		return nil, fmt.Errorf("cannot obtain peer's status: %w", err)
	}
	nonSkippedSlot, err := f.nonSkippedSlotAfterWithPeersTarget(
		ctx, slot-slotsPerEpoch, []peer.ID{pid}, slots.ToEpoch(pidState.HeadSlot))
	if err != nil {
		return nil, fmt.Errorf("cannot locate non-empty slot for a peer: %w", err)
	}

	// Request blocks starting from the first non-empty slot.
	req := &p2ppb.BeaconBlocksByRangeRequest{
		StartSlot: nonSkippedSlot,
		Count:     uint64(slotsPerEpoch.Mul(2)),
		Step:      1,
	}
	blocks, err := f.requestBlocks(ctx, req, pid)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch blocks: %w", err)
	}

	// Traverse blocks, and if we've got one that doesn't have parent in DB, backtrack on it.
	for i, block := range blocks {
		parentRoot := bytesutil.ToBytes32(block.Block().ParentRoot())
		if !f.db.HasBlock(ctx, parentRoot) && !f.chain.HasInitSyncBlock(parentRoot) {
			log.WithFields(logrus.Fields{
				"peer": pid,
				"slot": block.Block().Slot(),
				"root": fmt.Sprintf("%#x", parentRoot),
			}).Debug("Block with unknown parent root has been found")
			// Backtrack only if the first block is diverging,
			// otherwise we already know the common ancestor slot.
			if i == 0 {
				// Backtrack on a root, to find a common ancestor from which we can resume syncing.
				fork, err := f.findAncestor(ctx, pid, block)
				if err != nil {
					return nil, fmt.Errorf("failed to find common ancestor: %w", err)
				}
				return fork, nil
			}
			return &forkData{peer: pid, blocks: blocks}, nil
		}
	}
	return nil, errors.New("no alternative blocks exist within scanned range")
}

// findAncestor tries to figure out common ancestor slot that connects a given root to known block.
func (f *blocksFetcher) findAncestor(ctx context.Context, pid peer.ID, b block.SignedBeaconBlock) (*forkData, error) {
	outBlocks := []block.SignedBeaconBlock{b}
	for i := uint64(0); i < backtrackingMaxHops; i++ {
		parentRoot := bytesutil.ToBytes32(outBlocks[len(outBlocks)-1].Block().ParentRoot())
		if f.db.HasBlock(ctx, parentRoot) || f.chain.HasInitSyncBlock(parentRoot) {
			// Common ancestor found, forward blocks back to processor.
			sort.Slice(outBlocks, func(i, j int) bool {
				return outBlocks[i].Block().Slot() < outBlocks[j].Block().Slot()
			})
			return &forkData{
				peer:   pid,
				blocks: outBlocks,
			}, nil
		}
		// Request block's parent.
		req := &p2pTypes.BeaconBlockByRootsReq{parentRoot}
		blocks, err := f.requestBlocksByRoot(ctx, req, pid)
		if err != nil {
			return nil, err
		}
		if len(blocks) == 0 {
			break
		}
		outBlocks = append(outBlocks, blocks[0])
	}
	return nil, errors.New("no common ancestor found")
}

// bestFinalizedSlot returns the highest finalized slot of the majority of connected peers.
func (f *blocksFetcher) bestFinalizedSlot() types.Slot {
	finalizedEpoch, _ := f.p2p.Peers().BestFinalized(
		params.BeaconConfig().MaxPeersToSync, f.chain.FinalizedCheckpt().Epoch)
	return params.BeaconConfig().SlotsPerEpoch.Mul(uint64(finalizedEpoch))
}

// bestNonFinalizedSlot returns the highest non-finalized slot of enough number of connected peers.
func (f *blocksFetcher) bestNonFinalizedSlot() types.Slot {
	headEpoch := slots.ToEpoch(f.chain.HeadSlot())
	targetEpoch, _ := f.p2p.Peers().BestNonFinalized(flags.Get().MinimumSyncPeers*2, headEpoch)
	return params.BeaconConfig().SlotsPerEpoch.Mul(uint64(targetEpoch))
}

// calculateHeadAndTargetEpochs return node's current head epoch, along with the best known target
// epoch. For the latter peers supporting that target epoch are returned as well.
func (f *blocksFetcher) calculateHeadAndTargetEpochs() (headEpoch, targetEpoch types.Epoch, peers []peer.ID) {
	if f.mode == modeStopOnFinalizedEpoch {
		headEpoch = f.chain.FinalizedCheckpt().Epoch
		targetEpoch, peers = f.p2p.Peers().BestFinalized(params.BeaconConfig().MaxPeersToSync, headEpoch)
	} else {
		headEpoch = slots.ToEpoch(f.chain.HeadSlot())
		targetEpoch, peers = f.p2p.Peers().BestNonFinalized(flags.Get().MinimumSyncPeers, headEpoch)
	}
	return headEpoch, targetEpoch, peers
}
