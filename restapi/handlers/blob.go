package handlers

import (
	"encoding/hex"
	"fmt"

	"github.com/bnb-chain/blob-hub/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/go-openapi/runtime/middleware"

	"github.com/bnb-chain/blob-hub/models"
	"github.com/bnb-chain/blob-hub/restapi/operations/blob"
	"github.com/bnb-chain/blob-hub/service"
	"github.com/bnb-chain/blob-hub/util"
)

func HandleGetBlobSidecars() func(params blob.GetBlobSidecarsByBlockNumParams) middleware.Responder {
	return func(params blob.GetBlobSidecarsByBlockNumParams) middleware.Responder {

		blockID := params.BlockID
		indices := params.Indices
		var root []byte
		switch blockID {
		case "genesis", "finalized":
			return blob.NewGetBlobSidecarsByBlockNumBadRequest().WithPayload(service.BadRequestWithError(fmt.Errorf("block identifier not supported, only <slot> and <block root>")))
		default:
			var (
				err      error
				sidecars []*models.Sidecar
			)

			indicesInx := make([]int64, 0)
			for _, idx := range indices {
				i, err := util.StringToInt64(idx)
				if err != nil {
					return blob.NewGetBlobSidecarsByBlockNumBadRequest().WithPayload(service.BadRequestWithError(err))
				}
				indicesInx = append(indicesInx, i)
			}

			root, err = hexutil.Decode(blockID)
			if err == nil {
				if len(root) != types.RootLength {
					return blob.NewGetBlobSidecarsByBlockNumBadRequest().WithPayload(service.BadRequestWithError(fmt.Errorf("invalid block root of length %d", len(root))))
				}
				sidecars, err = service.BlobSvc.GetBlobSidecarsByRoot(hex.EncodeToString(root), indicesInx)
				if err != nil {
					return blob.NewGetBlobSidecarsByBlockNumInternalServerError().WithPayload(service.InternalErrorWithError(err))
				}
			} else {
				slot, err := util.StringToUint64(blockID)
				if err != nil {
					return blob.NewGetBlobSidecarsByBlockNumBadRequest().WithPayload(service.BadRequestWithError(err))
				}
				sidecars, err = service.BlobSvc.GetBlobSidecarsBySlot(slot, indicesInx)
				if err != nil {
					return blob.NewGetBlobSidecarsByBlockNumInternalServerError().WithPayload(service.InternalErrorWithError(err))
				}
			}
			payload := models.GetBlobSideCarsResponse{
				Data: sidecars,
			}
			return blob.NewGetBlobSidecarsByBlockNumOK().WithPayload(&payload)
		}
	}
}
