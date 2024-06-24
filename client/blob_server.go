package client

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/bnb-chain/blob-hub/models"
	blobproto "github.com/bnb-chain/blob-hub/proto"
	"github.com/bnb-chain/blob-hub/service"
	"github.com/bnb-chain/blob-hub/types"
	"github.com/bnb-chain/blob-hub/util"
)

type BlobServer struct {
	blobproto.UnimplementedBlobServiceServer
	service.Blob
}

func (s *BlobServer) GetBlobSidecars(ctx context.Context, req *blobproto.GetBlobSidecarsRequest) (*blobproto.GetBlobSidecarsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("invalid request")
	}
	blockID := req.GetBlockId()
	indices := req.GetIndices()

	var root []byte
	switch blockID {
	case "genesis", "finalized":
		return nil, fmt.Errorf("block identifier not supported, only <slot> and <block root>")
	default:
		var (
			err      error
			sidecars []*models.Sidecar
		)

		indicesInx := make([]int64, 0)
		for _, idx := range indices {
			i, err := util.StringToInt64(idx)
			if err != nil {
				return nil, err
			}
			indicesInx = append(indicesInx, i)
		}
		root, err = hexutil.Decode(blockID)
		if err == nil {
			if len(root) != types.RootLength {
				return nil, fmt.Errorf("invalid block root of length %d", len(root))
			}
			sidecars, err = service.BlobSvc.GetBlobSidecarsByRoot(hex.EncodeToString(root), indicesInx)
			if err != nil {
				return nil, err
			}
		} else {
			blockNumOrSlot, err := util.StringToUint64(blockID)
			if err != nil {
				return nil, err
			}
			sidecars, err = service.BlobSvc.GetBlobSidecarsByBlockNumOrSlot(blockNumOrSlot, indicesInx)
			if err != nil {
				return nil, err
			}
		}
		data := make([]*blobproto.SideCar, 0)
		for _, sc := range sidecars {
			data = append(
				data,
				&blobproto.SideCar{
					Blob:                        sc.Blob,
					Index:                       sc.Index,
					KzgCommitment:               sc.KzgCommitment,
					KzgCommitmentInclusionProof: sc.KzgCommitmentInclusionProof,
					KzgProof:                    sc.KzgProof,
					SignedBlockHeader: &blobproto.SignedBeaconBlockHeader{
						Message: &blobproto.BeaconBlockHeader{
							BodyRoot:      sc.SignedBlockHeader.Message.BodyRoot,
							ParentRoot:    sc.SignedBlockHeader.Message.ParentRoot,
							StateRoot:     sc.SignedBlockHeader.Message.StateRoot,
							Slot:          sc.SignedBlockHeader.Message.Slot,
							ProposerIndex: sc.SignedBlockHeader.Message.ProposerIndex,
						},
						Signature: sc.SignedBlockHeader.Signature,
					},
				},
			)
		}
		return &blobproto.GetBlobSidecarsResponse{
			Data: data,
		}, nil
	}

}
