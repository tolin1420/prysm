package operations

import (
	"context"
	"path"
	"testing"

	"github.com/golang/snappy"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/spectest/utils"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func RunProposerSlashingTest(t *testing.T, config string) {
	require.NoError(t, utils.SetConfig(t, config))
	testFolders, testsFolderPath := utils.TestFolders(t, config, "bellatrix", "operations/proposer_slashing/pyspec_tests")
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			folderPath := path.Join(testsFolderPath, folder.Name())
			proposerSlashingFile, err := util.BazelFileBytes(folderPath, "proposer_slashing.ssz_snappy")
			require.NoError(t, err)
			proposerSlashingSSZ, err := snappy.Decode(nil /* dst */, proposerSlashingFile)
			require.NoError(t, err, "Failed to decompress")
			proposerSlashing := &ethpb.ProposerSlashing{}
			require.NoError(t, proposerSlashing.UnmarshalSSZ(proposerSlashingSSZ), "Failed to unmarshal")

			body := &ethpb.BeaconBlockBodyMerge{ProposerSlashings: []*ethpb.ProposerSlashing{proposerSlashing}}
			RunBlockOperationTest(t, folderPath, body, func(ctx context.Context, s state.BeaconState, b block.SignedBeaconBlock) (state.BeaconState, error) {
				return blocks.ProcessProposerSlashings(ctx, s, b.Block().Body().ProposerSlashings(), validators.SlashValidator)
			})
		})
	}
}
