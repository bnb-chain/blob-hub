package handlers

import (
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/go-openapi/runtime/middleware"

	"github.com/bnb-chain/blob-hub/models"
	"github.com/bnb-chain/blob-hub/restapi/operations/blob"
	"github.com/bnb-chain/blob-hub/service"
	"github.com/bnb-chain/blob-hub/types"
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
					return blob.NewGetBlobSidecarsByBlockNumInternalServerError().WithPayload(service.InternalError())
				}
			} else {
				slot, err := util.StringToUint64(blockID)
				if err != nil {
					return blob.NewGetBlobSidecarsByBlockNumBadRequest().WithPayload(service.BadRequestWithError(err))
				}
				sidecars, err = service.BlobSvc.GetBlobSidecarsByBlockNumOrSlot(slot, indicesInx)
				if err != nil {
					return blob.NewGetBlobSidecarsByBlockNumInternalServerError().WithPayload(service.InternalError())
				}
			}
			payload := models.GetBlobSideCarsResponse{
				Data: sidecars,
			}
			return blob.NewGetBlobSidecarsByBlockNumOK().WithPayload(&payload)
		}
	}
}

func HandleGetBSCBlobSidecars() func(params blob.GetBSCBlobSidecarsByBlockNumParams) middleware.Responder {
	return func(params blob.GetBSCBlobSidecarsByBlockNumParams) middleware.Responder {

		rpcRequest := params.Body
		if rpcRequest.Params == nil {
			return blob.NewGetBSCBlobSidecarsByBlockNumOK().WithPayload(
				&models.RPCResponse{
					ID:      rpcRequest.ID,
					Jsonrpc: rpcRequest.Jsonrpc,
					Error: &models.RPCError{
						Code:    -32600,
						Message: "Invalid request",
					},
				},
			)
		}

		switch rpcRequest.Method {
		case "eth_getBlobSidecars":
			blockNum, err := util.HexToUint64(rpcRequest.Params[0])
			if err != nil {
				return blob.NewGetBSCBlobSidecarsByBlockNumOK().WithPayload(
					&models.RPCResponse{
						ID:      rpcRequest.ID,
						Jsonrpc: rpcRequest.Jsonrpc,
						Error: &models.RPCError{
							Code:    -32602,
							Message: "invalid argument",
						},
					},
				)
			}
			sidecars, err := service.BlobSvc.GetBlobSidecarsByBlockNumOrSlot(blockNum, nil)
			if err != nil {
				return blob.NewGetBlobSidecarsByBlockNumInternalServerError().WithPayload(service.InternalError())
			}
			// group sidecars by tx hash
			bscTxSidecars := make(map[string]*models.BSCBlobTxSidecar)
			for _, sidecar := range sidecars {
				txSidecar, ok := bscTxSidecars[sidecar.TxHash]
				if !ok {
					txSidecar = &models.BSCBlobTxSidecar{
						BlobSidecar: &models.BSCBlobSidecar{},
						TxHash:      sidecar.TxHash,
					}
					bscTxSidecars[sidecar.TxHash] = txSidecar
				}
				txSidecar.BlobSidecar.Blobs = append(txSidecar.BlobSidecar.Blobs, sidecar.Blob)
				txSidecar.BlobSidecar.Commitments = append(txSidecar.BlobSidecar.Commitments, sidecar.KzgCommitment)
				txSidecar.BlobSidecar.Proofs = append(txSidecar.BlobSidecar.Proofs, sidecar.KzgProof)
				txSidecar.TxIndex = util.Int64ToHex(sidecar.TxIndex)
				txSidecar.BlockNumber = rpcRequest.Params[0]
			}
			// convert txSidecars to array
			txSidecarsArr := make([]*models.BSCBlobTxSidecar, 0)
			for _, txSidecar := range bscTxSidecars {
				txSidecarsArr = append(txSidecarsArr, txSidecar)
			}
			response := &models.RPCResponse{
				ID:      rpcRequest.ID,
				Jsonrpc: rpcRequest.Jsonrpc,
				Result:  txSidecarsArr,
			}
			return blob.NewGetBSCBlobSidecarsByBlockNumOK().WithPayload(response)
		default:
			return blob.NewGetBSCBlobSidecarsByBlockNumOK().WithPayload(
				&models.RPCResponse{
					ID:      rpcRequest.ID,
					Jsonrpc: rpcRequest.Jsonrpc,
					Error: &models.RPCError{
						Code:    -32601,
						Message: "method not supported",
					},
				},
			)
		}
	}
}
