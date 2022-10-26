package handlers

import (
	"github.com/kubefirst/kubefirst/internal/aws"
)

// AwsHandler provides base data for Aws Handler methods.
type AwsHandler struct {
	hostedZone         string
	hostedZoneKeepBase bool
}

// NewAwsHandler creates a new Aws Handler object.
func NewAwsHandler(hostedZone string, hostedZoneKeepBase bool) AwsHandler {
	return AwsHandler{
		hostedZone:         hostedZone,
		hostedZoneKeepBase: hostedZoneKeepBase,
	}
}

// HostedZoneDelete deletes Hosted Zone data based on CLI flags. There are two possibilities to this handler, completely
// delete a hosted zone, or delete all hosted zone records except the base ones (SOA, NS and TXT liveness).
func (handler AwsHandler) HostedZoneDelete() error {

	// get hosted zone id
	hostedZoneId, err := aws.Route53GetHostedZoneId(handler.hostedZone)
	if err != nil {
		return err
	}

	// TXT records
	txtRecords, err := aws.Route53ListTXTRecords(hostedZoneId)
	if err != nil {
		return err
	}
	err = aws.Route53DeleteTXTRecords(
		hostedZoneId,
		handler.hostedZone,
		handler.hostedZoneKeepBase,
		txtRecords,
	)
	if err != nil {
		return err
	}

	// A records
	aRecords, err := aws.Route53ListARecords(hostedZoneId)
	if err != nil {
		return err
	}
	err = aws.Route53DeleteARecords(hostedZoneId, aRecords)
	if err != nil {
		return err
	}

	// deletes full hosted zone, at this point there is only a SOA and a NS record, and deletion will succeed
	if !handler.hostedZoneKeepBase {
		err := aws.Route53DeleteHostedZone(hostedZoneId, handler.hostedZone)
		if err != nil {
			return err
		}
	}
	return nil
}
