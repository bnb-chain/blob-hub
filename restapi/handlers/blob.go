package handlers

import (
	"github.com/bnb-chain/blob-syncer/models"
	"github.com/bnb-chain/blob-syncer/restapi/operations/blob"
	"github.com/bnb-chain/blob-syncer/service"
	"github.com/go-openapi/runtime/middleware"
)

func HandleGetBlobSidecars() func(params blob.GetBlobSidecarsByBlockNumParams) middleware.Responder {
	return func(params blob.GetBlobSidecarsByBlockNumParams) middleware.Responder {
		sidecars, err := service.BlobSvc.GetBlobSidecars(params.BlockNum)
		code, message := Error(err)
		payload := models.GetBlobSideCarsResponse{
			Code:    code,
			Message: message,
		}
		if err == nil {
			payload.Data = &models.GetBlobSideCarsResponseData{
				Sidecar: sidecars,
			}
		}
		return blob.NewGetBlobSidecarsByBlockNumOK().WithPayload(&payload)
	}
}
