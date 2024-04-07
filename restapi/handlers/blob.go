package handlers

import (
	"github.com/bnb-chain/blob-syncer/models"
	"github.com/bnb-chain/blob-syncer/restapi/operations/blob"
	"github.com/bnb-chain/blob-syncer/service"
	"github.com/bnb-chain/blob-syncer/types"
	"github.com/go-openapi/runtime/middleware"
)

func HandleGetBlobSidecars() func(params blob.GetBlobSidecarsByBlockNumParams) middleware.Responder {
	return func(params blob.GetBlobSidecarsByBlockNumParams) middleware.Responder {
		sidecars, err := service.BlobSvc.GetBlobSidecars(params.BlockNum)
		if err != nil {
			return blob.NewGetBlobSidecarsByBlockNumInternalServerError().WithPayload(types.InternalErrorWithError(err))
		}

		payload := models.GetBlobSideCarsResponse{
			Data: sidecars,
		}
		return blob.NewGetBlobSidecarsByBlockNumOK().WithPayload(&payload)
	}
}
