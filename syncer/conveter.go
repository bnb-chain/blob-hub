package syncer

import (
	"encoding/json"

	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	v1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

// ToBlockAndExecutionPayloadElectra extract beacon-signed block and execution payload from GetBlockV2Response
func ToBlockAndExecutionPayloadElectra(blockResp *structs.GetBlockV2Response) (*ethpb.BeaconBlockElectra, *v1.ExecutionPayloadElectra, error) {
	sbb := &structs.SignedBeaconBlockElectra{Message: &structs.BeaconBlockElectra{}}
	err := json.Unmarshal(blockResp.Data.Message, sbb.Message)
	if err != nil {
		return nil, nil, err
	}
	sbb.Signature = blockResp.Data.Signature
	signedBeaconBlockElectra, err := sbb.ToConsensus()
	if err != nil {
		return nil, nil, err
	}
	return signedBeaconBlockElectra.GetBlock(), signedBeaconBlockElectra.GetBlock().GetBody().GetExecutionPayload(), nil
}
