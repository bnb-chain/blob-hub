package external

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
)

const quickNode = "https://docs-demo.quiknode.pro"

func TestGetBlockHeight(t *testing.T) {

	beaconClient, err := NewBeaconClient(quickNode, time.Second*3)
	if err != nil {
		panic(err)
	}
	sideCars, err := beaconClient.GetBlob(context.Background(), 8779233)
	if err != nil {
		panic(err)
	}

	for _, sc := range sideCars {
		fmt.Println(sc.Index)
		fmt.Println(sc.KzgProof)
		fmt.Println(sc.KzgCommitment)
		fmt.Println(sc.SignedBeaconBlockHeader.Message.BodyRoot)
		fmt.Println(sc.SignedBeaconBlockHeader.Message.StateRoot)
		fmt.Println(sc.SignedBeaconBlockHeader.Message.ParentRoot)
		fmt.Println(sc.CommitmentInclusionProof)
		fmt.Println(sc.Blob[:300])
	}
}

func TestGetBlock(t *testing.T) {
	beaconClient, err := NewBeaconClient(quickNode, time.Second*3)
	if err != nil {
		panic(err)
	}

	resp, err := beaconClient.GetLatestBlock(context.Background())
	if err != nil {
		panic(err)
	}

	resp, err = beaconClient.GetBlock(context.Background(), 8762217)
	if err != nil {
		panic(err)
	}

	switch resp.Version {
	case version.String(version.Deneb):
		sbb := &structs.SignedBeaconBlockDeneb{Message: &structs.BeaconBlockDeneb{}}
		err = json.Unmarshal(resp.Data.Message, sbb.Message)
		if err != nil {
			panic(err)
		}
		sbb.Signature = resp.Data.Signature
		signedBeaconBlockDeneb, err := sbb.ToConsensus()
		if err != nil {
			panic(err)
		}
		fmt.Println("eth1data " + signedBeaconBlockDeneb.GetBlock().GetBody().GetEth1Data().String())
		fmt.Printf("eth block number %d\n", signedBeaconBlockDeneb.GetBlock().GetBody().GetExecutionPayload().GetBlockNumber())
	}
}

func TestGetHeader(t *testing.T) {
	beaconClient, err := NewBeaconClient(quickNode, time.Second*3)
	if err != nil {
		panic(err)
	}

	resp, err := beaconClient.GetHeader(context.Background(), 8777552)
	if err != nil {
		panic(err)
	}
	fmt.Println(resp.Data.Root)
}
