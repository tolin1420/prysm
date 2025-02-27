package slasher

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/slasher"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestServer_IsSlashableAttestation_SlashingFound(t *testing.T) {
	mockSlasher := &slasher.MockSlashingChecker{
		AttesterSlashingFound: true,
	}
	s := Server{SlashingChecker: mockSlasher}
	ctx := context.Background()
	slashing, err := s.IsSlashableAttestation(ctx, &ethpb.IndexedAttestation{})
	require.NoError(t, err)
	require.Equal(t, true, len(slashing.AttesterSlashings) > 0)
}

func TestServer_IsSlashableAttestation_SlashingNotFound(t *testing.T) {
	mockSlasher := &slasher.MockSlashingChecker{
		AttesterSlashingFound: false,
	}
	s := Server{SlashingChecker: mockSlasher}
	ctx := context.Background()
	slashing, err := s.IsSlashableAttestation(ctx, &ethpb.IndexedAttestation{})
	require.NoError(t, err)
	require.Equal(t, true, len(slashing.AttesterSlashings) == 0)
}

func TestServer_IsSlashableBlock_SlashingFound(t *testing.T) {
	mockSlasher := &slasher.MockSlashingChecker{
		ProposerSlashingFound: true,
	}
	s := Server{SlashingChecker: mockSlasher}
	ctx := context.Background()
	slashing, err := s.IsSlashableBlock(ctx, &ethpb.SignedBeaconBlockHeader{})
	require.NoError(t, err)
	require.Equal(t, true, len(slashing.ProposerSlashings) > 0)
}

func TestServer_IsSlashableBlock_SlashingNotFound(t *testing.T) {
	mockSlasher := &slasher.MockSlashingChecker{
		ProposerSlashingFound: false,
	}
	s := Server{SlashingChecker: mockSlasher}
	ctx := context.Background()
	slashing, err := s.IsSlashableBlock(ctx, &ethpb.SignedBeaconBlockHeader{})
	require.NoError(t, err)
	require.Equal(t, true, len(slashing.ProposerSlashings) == 0)
}
