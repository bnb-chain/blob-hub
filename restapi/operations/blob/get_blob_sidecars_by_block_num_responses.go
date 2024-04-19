// Code generated by go-swagger; DO NOT EDIT.

package blob

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/runtime"

	"github.com/bnb-chain/blob-syncer/models"
)

// GetBlobSidecarsByBlockNumOKCode is the HTTP code returned for type GetBlobSidecarsByBlockNumOK
const GetBlobSidecarsByBlockNumOKCode int = 200

/*
GetBlobSidecarsByBlockNumOK successful operation

swagger:response getBlobSidecarsByBlockNumOK
*/
type GetBlobSidecarsByBlockNumOK struct {

	/*
	  In: Body
	*/
	Payload *models.GetBlobSideCarsResponse `json:"body,omitempty"`
}

// NewGetBlobSidecarsByBlockNumOK creates GetBlobSidecarsByBlockNumOK with default headers values
func NewGetBlobSidecarsByBlockNumOK() *GetBlobSidecarsByBlockNumOK {

	return &GetBlobSidecarsByBlockNumOK{}
}

// WithPayload adds the payload to the get blob sidecars by block num o k response
func (o *GetBlobSidecarsByBlockNumOK) WithPayload(payload *models.GetBlobSideCarsResponse) *GetBlobSidecarsByBlockNumOK {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get blob sidecars by block num o k response
func (o *GetBlobSidecarsByBlockNumOK) SetPayload(payload *models.GetBlobSideCarsResponse) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetBlobSidecarsByBlockNumOK) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(200)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// GetBlobSidecarsByBlockNumBadRequestCode is the HTTP code returned for type GetBlobSidecarsByBlockNumBadRequest
const GetBlobSidecarsByBlockNumBadRequestCode int = 400

/*
GetBlobSidecarsByBlockNumBadRequest Bad Request

swagger:response getBlobSidecarsByBlockNumBadRequest
*/
type GetBlobSidecarsByBlockNumBadRequest struct {

	/*
	  In: Body
	*/
	Payload *models.Error `json:"body,omitempty"`
}

// NewGetBlobSidecarsByBlockNumBadRequest creates GetBlobSidecarsByBlockNumBadRequest with default headers values
func NewGetBlobSidecarsByBlockNumBadRequest() *GetBlobSidecarsByBlockNumBadRequest {

	return &GetBlobSidecarsByBlockNumBadRequest{}
}

// WithPayload adds the payload to the get blob sidecars by block num bad request response
func (o *GetBlobSidecarsByBlockNumBadRequest) WithPayload(payload *models.Error) *GetBlobSidecarsByBlockNumBadRequest {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get blob sidecars by block num bad request response
func (o *GetBlobSidecarsByBlockNumBadRequest) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetBlobSidecarsByBlockNumBadRequest) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(400)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// GetBlobSidecarsByBlockNumNotFoundCode is the HTTP code returned for type GetBlobSidecarsByBlockNumNotFound
const GetBlobSidecarsByBlockNumNotFoundCode int = 404

/*
GetBlobSidecarsByBlockNumNotFound blob not found

swagger:response getBlobSidecarsByBlockNumNotFound
*/
type GetBlobSidecarsByBlockNumNotFound struct {

	/*
	  In: Body
	*/
	Payload *models.Error `json:"body,omitempty"`
}

// NewGetBlobSidecarsByBlockNumNotFound creates GetBlobSidecarsByBlockNumNotFound with default headers values
func NewGetBlobSidecarsByBlockNumNotFound() *GetBlobSidecarsByBlockNumNotFound {

	return &GetBlobSidecarsByBlockNumNotFound{}
}

// WithPayload adds the payload to the get blob sidecars by block num not found response
func (o *GetBlobSidecarsByBlockNumNotFound) WithPayload(payload *models.Error) *GetBlobSidecarsByBlockNumNotFound {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get blob sidecars by block num not found response
func (o *GetBlobSidecarsByBlockNumNotFound) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetBlobSidecarsByBlockNumNotFound) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(404)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// GetBlobSidecarsByBlockNumInternalServerErrorCode is the HTTP code returned for type GetBlobSidecarsByBlockNumInternalServerError
const GetBlobSidecarsByBlockNumInternalServerErrorCode int = 500

/*
GetBlobSidecarsByBlockNumInternalServerError internal server error

swagger:response getBlobSidecarsByBlockNumInternalServerError
*/
type GetBlobSidecarsByBlockNumInternalServerError struct {

	/*
	  In: Body
	*/
	Payload *models.Error `json:"body,omitempty"`
}

// NewGetBlobSidecarsByBlockNumInternalServerError creates GetBlobSidecarsByBlockNumInternalServerError with default headers values
func NewGetBlobSidecarsByBlockNumInternalServerError() *GetBlobSidecarsByBlockNumInternalServerError {

	return &GetBlobSidecarsByBlockNumInternalServerError{}
}

// WithPayload adds the payload to the get blob sidecars by block num internal server error response
func (o *GetBlobSidecarsByBlockNumInternalServerError) WithPayload(payload *models.Error) *GetBlobSidecarsByBlockNumInternalServerError {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get blob sidecars by block num internal server error response
func (o *GetBlobSidecarsByBlockNumInternalServerError) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetBlobSidecarsByBlockNumInternalServerError) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(500)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}
