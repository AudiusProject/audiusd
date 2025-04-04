// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// ProtocolDeregistrationAttestationResponse protocol deregistration attestation response
//
// swagger:model protocolDeregistrationAttestationResponse
type ProtocolDeregistrationAttestationResponse struct {

	// deregistration
	Deregistration *ProtocolValidatorDeregistration `json:"deregistration,omitempty"`

	// signature
	Signature string `json:"signature,omitempty"`
}

// Validate validates this protocol deregistration attestation response
func (m *ProtocolDeregistrationAttestationResponse) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateDeregistration(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *ProtocolDeregistrationAttestationResponse) validateDeregistration(formats strfmt.Registry) error {
	if swag.IsZero(m.Deregistration) { // not required
		return nil
	}

	if m.Deregistration != nil {
		if err := m.Deregistration.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("deregistration")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("deregistration")
			}
			return err
		}
	}

	return nil
}

// ContextValidate validate this protocol deregistration attestation response based on the context it is used
func (m *ProtocolDeregistrationAttestationResponse) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateDeregistration(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *ProtocolDeregistrationAttestationResponse) contextValidateDeregistration(ctx context.Context, formats strfmt.Registry) error {

	if m.Deregistration != nil {

		if swag.IsZero(m.Deregistration) { // not required
			return nil
		}

		if err := m.Deregistration.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("deregistration")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("deregistration")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *ProtocolDeregistrationAttestationResponse) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *ProtocolDeregistrationAttestationResponse) UnmarshalBinary(b []byte) error {
	var res ProtocolDeregistrationAttestationResponse
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
