package main

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/rs/zerolog/log"
	"go.uber.org/multierr"
)

func deleteNatGateways(ctx context.Context, client *ec2.Client, natGateways []types.NatGateway) (errs error) {
	for _, natGateway := range natGateways {
		if natGateway.NatGatewayId == nil {
			continue
		}
		_, err := client.DeleteNatGateway(ctx, &ec2.DeleteNatGatewayInput{
			NatGatewayId: natGateway.NatGatewayId,
		})
		log.Err(err).
			Str("NatGatewayId", *natGateway.NatGatewayId).
			Msg("DeleteNatGateway")
		errs = multierr.Append(errs, err)
	}
	return
}

func listNatGateways(ctx context.Context, client *ec2.Client, vpcId string) ([]types.NatGateway, error) {
	input := ec2.DescribeNatGatewaysInput{
		Filter: ec2VpcFilter(vpcId),
	}
	var natGateways []types.NatGateway
	for {
		output, err := client.DescribeNatGateways(ctx, &input)
		if err != nil {
			return nil, err
		}
		natGateways = append(natGateways, output.NatGateways...)
		if output.NextToken == nil {
			return natGateways, nil
		}
		input.NextToken = output.NextToken
	}
}

func natGatewayIds(natGateways []types.NatGateway) []string {
	natGatewayIds := make([]string, 0, len(natGateways))
	for _, natGateway := range natGateways {
		if natGateway.NatGatewayId != nil {
			natGatewayIds = append(natGatewayIds, *natGateway.NatGatewayId)
		}
	}
	return natGatewayIds
}
