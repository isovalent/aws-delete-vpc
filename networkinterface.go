package main

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/rs/zerolog/log"
	"go.uber.org/multierr"
)

func deleteNetworkInterfaces(ctx context.Context, client *ec2.Client, networkInterfaces []types.NetworkInterface) (errs error) {
	for _, networkInterface := range networkInterfaces {
		if networkInterface.NetworkInterfaceId == nil {
			continue
		}

		// Detach the NetworkInterface.
		if networkInterface.Attachment != nil && networkInterface.Attachment.AttachmentId != nil {
			_, err := client.DetachNetworkInterface(ctx, &ec2.DetachNetworkInterfaceInput{
				AttachmentId: networkInterface.Attachment.AttachmentId,
			})
			log.Err(err).
				Str("AttachmentId", *networkInterface.Attachment.AttachmentId).
				Msg("DetachNetworkInterface")
			if err != nil {
				errs = multierr.Append(errs, err)
				continue
			}

			// FIXME wait for detachment somehow
		}

		// Delete the NetworkInterface.
		_, err := client.DeleteNetworkInterface(ctx, &ec2.DeleteNetworkInterfaceInput{
			NetworkInterfaceId: networkInterface.NetworkInterfaceId,
		})
		log.Err(err).
			Str("NetworkInterfaceId", *networkInterface.NetworkInterfaceId).
			Msg("DeleteNetworkInterface")
		errs = multierr.Append(errs, err)
	}
	return
}

func listNetworkInterfaces(ctx context.Context, client *ec2.Client, vpcId string) ([]types.NetworkInterface, error) {
	input := ec2.DescribeNetworkInterfacesInput{
		Filters: ec2VpcFilter(vpcId),
	}
	var networkInterfaces []types.NetworkInterface
	for {
		output, err := client.DescribeNetworkInterfaces(ctx, &input)
		if err != nil {
			return nil, err
		}
		networkInterfaces = append(networkInterfaces, output.NetworkInterfaces...)
		if output.NextToken == nil {
			return networkInterfaces, nil
		}
		input.NextToken = output.NextToken
	}
}

func networkInterfaceIds(networkInterfaces []types.NetworkInterface) []string {
	networkInterfaceIds := make([]string, 0, len(networkInterfaces))
	for _, networkInterface := range networkInterfaces {
		if networkInterface.NetworkInterfaceId != nil {
			networkInterfaceIds = append(networkInterfaceIds, *networkInterface.NetworkInterfaceId)
		}
	}
	return networkInterfaceIds
}
