// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// ProtocolStorageProofVerification protocol storage proof verification
//
// swagger:model protocolStorageProofVerification
type ProtocolStorageProofVerification struct {

	// height
	Height string `json:"height,omitempty"`

	// proof
	// Format: byte
	Proof strfmt.Base64 `json:"proof,omitempty"`
}

// Validate validates this protocol storage proof verification
func (m *ProtocolStorageProofVerification) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this protocol storage proof verification based on context it is used
func (m *ProtocolStorageProofVerification) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *ProtocolStorageProofVerification) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *ProtocolStorageProofVerification) UnmarshalBinary(b []byte) error {
	var res ProtocolStorageProofVerification
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
