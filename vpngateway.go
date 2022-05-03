package main

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/rs/zerolog/log"
	"go.uber.org/multierr"
)

func deleteVpnGateways(ctx context.Context, client *ec2.Client, vpcId string, vpnGateways []types.VpnGateway) (errs error) {
	for _, vpnGateway := range vpnGateways {
		if vpnGateway.VpnGatewayId == nil {
			continue
		}

		var vpcAttachmentErrs error
		for _, vpcAttachment := range vpnGateway.VpcAttachments {
			state := vpcAttachment.State
			if state == types.AttachmentStatusDetached || state == types.AttachmentStatusDetaching {
				continue
			}
			if vpcAttachment.VpcId == nil || *vpcAttachment.VpcId != vpcId {
				continue
			}

			_, err := client.DetachVpnGateway(ctx, &ec2.DetachVpnGatewayInput{
				VpcId:        vpcAttachment.VpcId,
				VpnGatewayId: vpnGateway.VpnGatewayId,
			})
			log.Err(err).
				Str("VpcId", *vpcAttachment.VpcId).
				Str("VpnGatewayId", *vpnGateway.VpnGatewayId).
				Msg("DetachVpnGateway")
			vpcAttachmentErrs = multierr.Append(vpcAttachmentErrs, err)
		}
		if vpcAttachmentErrs != nil {
			continue
		}

		_, err := client.DeleteVpnGateway(ctx, &ec2.DeleteVpnGatewayInput{
			VpnGatewayId: vpnGateway.VpnGatewayId,
		})
		log.Err(err).
			Str("VpnGatewayId", *vpnGateway.VpnGatewayId).
			Msg("DeleteVpnGateway")
		errs = multierr.Append(errs, err)
	}
	return
}

func listVpnGateways(ctx context.Context, client *ec2.Client, vpcId string) ([]types.VpnGateway, error) {
	output, err := client.DescribeVpnGateways(ctx, &ec2.DescribeVpnGatewaysInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("attachment.vpc-id"),
				Values: []string{vpcId},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return output.VpnGateways, nil
}

func vpnGatewayIds(vpnGateways []types.VpnGateway) []string {
	vpnGatewayIds := make([]string, 0, len(vpnGateways))
	for _, vpnGateway := range vpnGateways {
		if vpnGateway.VpnGatewayId != nil {
			vpnGatewayIds = append(vpnGatewayIds, *vpnGateway.VpnGatewayId)
		}
	}
	return vpnGatewayIds
}
