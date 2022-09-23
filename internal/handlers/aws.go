package handlers

import (
	"github.com/kubefirst/kubefirst/internal/aws"
	"github.com/kubefirst/kubefirst/internal/flagset"
)

type AwsHandler struct {
	HostedZone string
	CLIFlags   flagset.DestroyFlags
}

func NewAwsHandler(hostedZone string, cliFlags flagset.DestroyFlags) AwsHandler {
	return AwsHandler{
		HostedZone: hostedZone,
		CLIFlags:   cliFlags,
	}
}

func (handler AwsHandler) HostedZoneDelete() error {

	// get hosted zone id
	hostedZoneId, err := aws.Route53GetHostedZoneId(handler.HostedZone)
	if err != nil {
		return err
	}

	// handles TXT records
	txtRecords, err := aws.Route53ListTXTRecords(hostedZoneId)
	if err != nil {
		return err
	}
	err = aws.Route53DeleteTXTRecords(
		hostedZoneId,
		handler.HostedZone,
		handler.CLIFlags.HostedZoneKeepBase,
		txtRecords,
	)
	if err != nil {
		return err
	}

	// handles A records
	//aRecords, err := aws.Route53ListARecords(hostedZoneId)
	//if err != nil {
	//	return err
	//}
	//err = aws.Route53DeleteARecords(hostedZoneId, aRecords)
	//if err != nil {
	//	return err
	//}
	//
	//// deletes full hosted zone, at this point there is only a SOA and a NS record, and deletion will succeed
	//if !handler.CLIFlags.HostedZoneKeepBase {
	//	err := aws.Route53DeleteHostedZone(hostedZoneId, handler.HostedZone)
	//	if err != nil {
	//		return err
	//	}
	//}
	return nil
}
