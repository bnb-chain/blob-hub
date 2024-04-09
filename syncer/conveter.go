package syncer

import (
	"encoding/json"
	"strings"

	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	v1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

// ToBlockAndExecutionPayloadDeneb extract beacon-signed block and execution payload from GetBlockV2Response
func ToBlockAndExecutionPayloadDeneb(blockResp *structs.GetBlockV2Response) (*ethpb.BeaconBlockDeneb, *v1.ExecutionPayloadDeneb, error) {
	sbb := &structs.SignedBeaconBlockDeneb{Message: &structs.BeaconBlockDeneb{}}
	err := json.Unmarshal(blockResp.Data.Message, sbb.Message)
	if err != nil {
		return nil, nil, err
	}
	sbb.Signature = blockResp.Data.Signature
	signedBeaconBlockDeneb, err := sbb.ToConsensus()
	if err != nil {
		return nil, nil, err
	}
	return signedBeaconBlockDeneb.GetBlock(), signedBeaconBlockDeneb.GetBlock().GetBody().GetExecutionPayload(), nil
}

func SplitByComma(str string) []string {
	str = strings.TrimSpace(str)
	strArr := strings.Split(str, ",")
	var trimStr []string
	for _, item := range strArr {
		if len(strings.TrimSpace(item)) > 0 {
			trimStr = append(trimStr, strings.TrimSpace(item))
		}
	}
	return trimStr
}

func JoinWithComma(slice []string) string {
	return strings.Join(slice, ",")
}
