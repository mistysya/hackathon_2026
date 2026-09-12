// Package profile implements fixture-backed employee profile enrichment.
package profile

import "errors"

var (
	// ErrEmployeeNotFound indicates that the requested employee does not exist.
	ErrEmployeeNotFound = errors.New("profile employee not found")
	// ErrEnrichmentFailed indicates that enrichment output could not be produced
	// as a valid profile after the permitted validator retry.
	ErrEnrichmentFailed = errors.New("profile enrichment failed")
)
