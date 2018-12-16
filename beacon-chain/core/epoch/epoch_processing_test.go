package epoch

import (
	"bytes"
	"reflect"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestEpochAttestations_ok(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}

	var pendingAttestations []*pb.PendingAttestationRecord
	for i := uint64(0); i < params.BeaconConfig().EpochLength*2; i++ {
		pendingAttestations = append(pendingAttestations, &pb.PendingAttestationRecord{
			Data: &pb.AttestationData{
				Slot: i,
			},
		})
	}

	state := &pb.BeaconState{LatestAttestations: pendingAttestations}

	tests := []struct {
		stateSlot            uint64
		firstAttestationSlot uint64
	}{
		{
			stateSlot:            10,
			firstAttestationSlot: 0,
		},
		{
			stateSlot:            63,
			firstAttestationSlot: 0,
		},
		{
			stateSlot:            64,
			firstAttestationSlot: 64 - params.BeaconConfig().EpochLength,
		}, {
			stateSlot:            127,
			firstAttestationSlot: 127 - params.BeaconConfig().EpochLength,
		}, {
			stateSlot:            128,
			firstAttestationSlot: 128 - params.BeaconConfig().EpochLength,
		},
	}

	for _, tt := range tests {
		state.Slot = tt.stateSlot

		if Attestations(state)[0].Data.Slot != tt.firstAttestationSlot {
			t.Errorf(
				"Result slot was an unexpected value. Wanted %d, got %d",
				tt.firstAttestationSlot,
				Attestations(state)[0].Data.Slot,
			)
		}
	}
}

func TestEpochBoundaryAttestations(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}

	epochAttestations := []*pb.PendingAttestationRecord{
		{Data: &pb.AttestationData{JustifiedBlockRootHash32: []byte{0}, JustifiedSlot: 0}},
		{Data: &pb.AttestationData{JustifiedBlockRootHash32: []byte{1}, JustifiedSlot: 1}},
		{Data: &pb.AttestationData{JustifiedBlockRootHash32: []byte{2}, JustifiedSlot: 2}},
		{Data: &pb.AttestationData{JustifiedBlockRootHash32: []byte{3}, JustifiedSlot: 3}},
	}

	var latestBlockRootHash [][]byte
	for i := uint64(0); i < params.BeaconConfig().EpochLength; i++ {
		latestBlockRootHash = append(latestBlockRootHash, []byte{byte(i)})
	}

	state := &pb.BeaconState{
		LatestAttestations:     epochAttestations,
		Slot:                   params.BeaconConfig().EpochLength,
		LatestBlockRootHash32S: [][]byte{},
	}

	epochBoundaryAttestation, err := BoundaryAttestations(state, epochAttestations)
	if err == nil {
		t.Fatalf("EpochBoundaryAttestations should have failed with empty block root hash")
	}

	state.LatestBlockRootHash32S = latestBlockRootHash
	epochBoundaryAttestation, err = BoundaryAttestations(state, epochAttestations)
	if err != nil {
		t.Fatalf("EpochBoundaryAttestations failed: %v", err)
	}

	if epochBoundaryAttestation[0].GetData().JustifiedSlot != 0 {
		t.Errorf("Wanted justified slot 0 for epoch boundary attestation, got: %d", epochBoundaryAttestation[0].Data.JustifiedSlot)
	}

	if !bytes.Equal(epochBoundaryAttestation[0].GetData().JustifiedBlockRootHash32, []byte{0}) {
		t.Errorf("Wanted justified block hash [0] for epoch boundary attestation, got: %v",
			epochBoundaryAttestation[0].Data.JustifiedBlockRootHash32)
	}
}

func TestBoundaryAttestingBalance(t *testing.T) {
	attesters := []*pb.ValidatorRecord{
		{Balance: 25 * 1e9},
		{Balance: 26 * 1e9},
		{Balance: 32 * 1e9},
		{Balance: 33 * 1e9},
		{Balance: 100 * 1e9},
	}
	attestedBalances := BoundaryAttestingBalance(attesters)

	// 25 + 26 + 32 + 32 + 32 = 147
	if attestedBalances != 147*1e9 {
		t.Errorf("Incorrect attested balances. Wanted: %f, got: %d", 147*1e9, attestedBalances)
	}
}

func TestBoundaryAttesters(t *testing.T) {
	var validators []*pb.ValidatorRecord

	for i := 0; i < 100; i++ {
		validators = append(validators, &pb.ValidatorRecord{Pubkey: []byte{byte(i)}})
	}

	state := &pb.BeaconState{ValidatorRegistry: validators}

	boundaryAttesters := BoundaryAttesters(state, []uint32{5, 2, 87, 42, 99, 0})

	expectedBoundaryAttesters := []*pb.ValidatorRecord{
		{Pubkey: []byte{byte(5)}},
		{Pubkey: []byte{byte(2)}},
		{Pubkey: []byte{byte(87)}},
		{Pubkey: []byte{byte(42)}},
		{Pubkey: []byte{byte(99)}},
		{Pubkey: []byte{byte(0)}},
	}

	if !reflect.DeepEqual(expectedBoundaryAttesters, boundaryAttesters) {
		t.Errorf("Incorrect boundary attesters. Wanted: %v, got: %v", expectedBoundaryAttesters, boundaryAttesters)
	}
}

func TestBoundaryAttesterIndices(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}
	var committeeIndices []uint32
	for i := uint32(0); i < 10; i++ {
		committeeIndices = append(committeeIndices, i)
	}
	var shardAndCommittees []*pb.ShardAndCommitteeArray
	for i := uint64(0); i < params.BeaconConfig().EpochLength*2; i++ {
		shardAndCommittees = append(shardAndCommittees, &pb.ShardAndCommitteeArray{
			ArrayShardAndCommittee: []*pb.ShardAndCommittee{
				{Shard: 100, Committee: committeeIndices},
			},
		})
	}

	state := &pb.BeaconState{
		ShardAndCommitteesAtSlots: shardAndCommittees,
		Slot:                      5,
	}

	boundaryAttestations := []*pb.PendingAttestationRecord{
		{Data: &pb.AttestationData{Slot: 2, Shard: 100}, ParticipationBitfield: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}}, // returns index 7
		{Data: &pb.AttestationData{Slot: 2, Shard: 100}, ParticipationBitfield: []byte{7, 7, 7, 7, 7, 7, 7, 7, 7, 7}},  // returns indices 5,6,7
		{Data: &pb.AttestationData{Slot: 2, Shard: 100}, ParticipationBitfield: []byte{10, 0, 0, 0, 0, 0, 0, 0, 0, 0}}, // returns indices 4,6
	}

	attesterIndices, err := BoundaryAttesterIndices(state, boundaryAttestations)
	if err != nil {
		t.Fatalf("Failed to run BoundaryAttesterIndices: %v", err)
	}

	if !reflect.DeepEqual(attesterIndices, []uint32{7, 5, 6, 4}) {
		t.Errorf("Incorrect boundary attester indices. Wanted: %v, got: %v", []uint32{7, 5, 6, 4}, attesterIndices)
	}
}
