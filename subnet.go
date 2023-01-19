package main

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/rs/zerolog/log"
	"go.uber.org/multierr"
)

func deleteSubnets(ctx context.Context, client *ec2.Client, vpcId string, subnets []types.Subnet) (errs error) {
	for _, subnet := range subnets {
		if subnet.SubnetId == nil {
			continue
		}
		if subnet.VpcId == nil || *subnet.VpcId != vpcId {
			continue
		}

		_, err := client.DeleteSubnet(ctx, &ec2.DeleteSubnetInput{
			SubnetId: subnet.SubnetId,
		})
		log.Err(err).
			Str("SubnetId", *subnet.SubnetId).
			Msg("DeleteSubnet")
		errs = multierr.Append(errs, err)
	}
	return
}

func listSubnets(ctx context.Context, client *ec2.Client, vpcId string) ([]types.Subnet, error) {
	input := ec2.DescribeSubnetsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{vpcId},
			},
		},
	}
	var subnets []types.Subnet
	paginator := ec2.NewDescribeSubnetsPaginator(client, &input)
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		subnets = append(subnets, output.Subnets...)
	}
	return subnets, nil
}

func subnetIds(subnets []types.Subnet) []string {
	subnetIds := make([]string, 0, len(subnets))
	for _, subnet := range subnets {
		if subnet.SubnetId != nil {
			subnetIds = append(subnetIds, *subnet.SubnetId)
		}
	}
	return subnetIds
}
