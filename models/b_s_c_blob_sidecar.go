// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// BSCBlobSidecar b s c blob sidecar
//
// swagger:model BSCBlobSidecar
type BSCBlobSidecar struct {

	// blobs
	Blobs []string `json:"blobs"`

	// commitments
	Commitments []string `json:"commitments"`

	// proofs
	Proofs []string `json:"proofs"`
}

// Validate validates this b s c blob sidecar
func (m *BSCBlobSidecar) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this b s c blob sidecar based on context it is used
func (m *BSCBlobSidecar) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *BSCBlobSidecar) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *BSCBlobSidecar) UnmarshalBinary(b []byte) error {
	var res BSCBlobSidecar
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
