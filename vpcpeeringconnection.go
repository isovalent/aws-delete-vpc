package main

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/rs/zerolog/log"
	"go.uber.org/multierr"
)

func deleteVpcPeeringConnections(ctx context.Context, client *ec2.Client, vpcId string, vpcPeeringConnections []types.VpcPeeringConnection) (errs error) {
	for _, vpcPeeringConnection := range vpcPeeringConnections {
		if vpcPeeringConnection.VpcPeeringConnectionId == nil {
			continue
		}
		isAccepter := vpcPeeringConnection.AccepterVpcInfo != nil &&
			vpcPeeringConnection.AccepterVpcInfo.VpcId != nil &&
			*vpcPeeringConnection.AccepterVpcInfo.VpcId == vpcId
		isRequester := vpcPeeringConnection.RequesterVpcInfo != nil &&
			vpcPeeringConnection.RequesterVpcInfo.VpcId != nil &&
			*vpcPeeringConnection.RequesterVpcInfo.VpcId == vpcId
		if !isAccepter && !isRequester {
			continue
		}

		_, err := client.DeleteVpcPeeringConnection(ctx, &ec2.DeleteVpcPeeringConnectionInput{
			VpcPeeringConnectionId: vpcPeeringConnection.VpcPeeringConnectionId,
		})
		log.Err(err).
			Str("VpcPeeringConnectionId", *vpcPeeringConnection.VpcPeeringConnectionId).
			Msg("DeleteVpcPeeringConnection")
		errs = multierr.Append(errs, err)
	}
	return
}

func listVpcPeeringConnections(ctx context.Context, client *ec2.Client, vpcId string) ([]types.VpcPeeringConnection, error) {
	var vpcPeeringConnections []types.VpcPeeringConnection
	for _, name := range []string{
		"accepter-vpc-info.vpc-id",
		"requester-vpc-info.vpc-id",
	} {
		input := ec2.DescribeVpcPeeringConnectionsInput{
			Filters: []types.Filter{
				{
					Name:   aws.String(name),
					Values: []string{vpcId},
				},
			},
		}
		paginator := ec2.NewDescribeVpcPeeringConnectionsPaginator(client, &input)
		for paginator.HasMorePages() {
			output, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			vpcPeeringConnections = append(vpcPeeringConnections, output.VpcPeeringConnections...)
		}
	}
	return vpcPeeringConnections, nil
}

func vpcPeeringConnectionIds(vpcPeeringConnections []types.VpcPeeringConnection) []string {
	vpcPeeringConnectionIds := make([]string, 0, len(vpcPeeringConnections))
	for _, vpcPeeringConnection := range vpcPeeringConnections {
		if vpcPeeringConnection.VpcPeeringConnectionId != nil {
			vpcPeeringConnectionIds = append(vpcPeeringConnectionIds, *vpcPeeringConnection.VpcPeeringConnectionId)
		}
	}
	return vpcPeeringConnectionIds
}
