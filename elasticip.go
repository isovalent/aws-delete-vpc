package main

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/rs/zerolog/log"
	"go.uber.org/multierr"
)

func releaseElasticIps(ctx context.Context, client *ec2.Client, addresses []types.Address) (errs error) {
	for _, address := range addresses {
		_, err := client.ReleaseAddress(ctx, &ec2.ReleaseAddressInput{
			AllocationId: address.AllocationId,
		})
		log.Err(err).
			Str("AllocationId", *address.AllocationId).
			Str("PublicIp", *address.PublicIp).
			Msg("ReleaseAddress")
		errs = multierr.Append(errs, err)
	}
	return
}

func listElasticIps(ctx context.Context, client *ec2.Client, filters []types.Filter) ([]types.Address, error) {
	output, err := client.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{
		Filters: filters,
	})
	if err != nil {
		return nil, err
	}
	return output.Addresses, nil
}

func publicIps(addresses []types.Address) []string {
	publicIps := make([]string, 0, len(addresses))
	for _, address := range addresses {
		if address.PublicIp != nil {
			publicIps = append(publicIps, *address.PublicIp)
		}
	}
	return publicIps
}
