package main

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/rs/zerolog/log"
	"go.uber.org/multierr"
)

func deleteRouteTables(ctx context.Context, client *ec2.Client, vpcId string, routeTables []types.RouteTable) (errs error) {
	for _, routeTable := range routeTables {
		if routeTable.RouteTableId == nil {
			continue
		}
		if routeTable.VpcId == nil || *routeTable.VpcId != vpcId {
			continue
		}

		// FIXME remove this un-needed code
		/*
			var routeErrs error
			for _, route := range routeTable.Routes {
				_, err := client.DeleteRoute(ctx, &ec2.DeleteRouteInput{
					RouteTableId:             routeTable.RouteTableId,
					DestinationCidrBlock:     route.DestinationCidrBlock,
					DestinationIpv6CidrBlock: route.DestinationIpv6CidrBlock,
					DestinationPrefixListId:  route.DestinationPrefixListId,
				})
				log.Err(err).
					Str("RouteTableId", *routeTable.RouteTableId).
					Msg("DeleteRoute")
				routeErrs = multierr.Append(routeErrs, err)
			}
			errs = multierr.Append(errs, routeErrs)
			if routeErrs != nil {
				continue
			}
		*/

		_, err := client.DeleteRouteTable(ctx, &ec2.DeleteRouteTableInput{
			RouteTableId: routeTable.RouteTableId,
		})
		log.Err(err).
			Str("RouteTableId", *routeTable.RouteTableId).
			Msg("DeleteRouteTable")
		errs = multierr.Append(errs, err)
	}
	return
}

func listRouteTables(ctx context.Context, client *ec2.Client, vpcId string) ([]types.RouteTable, error) {
	input := ec2.DescribeRouteTablesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{vpcId},
			},
		},
	}
	var routeTables []types.RouteTable
	for {
		output, err := client.DescribeRouteTables(ctx, &input)
		if err != nil {
			return nil, err
		}
		routeTables = append(routeTables, output.RouteTables...)
		if output.NextToken == nil {
			return routeTables, nil
		}
		input.NextToken = output.NextToken
	}
}

func routeTableIds(routeTables []types.RouteTable) []string {
	routeTableIds := make([]string, 0, len(routeTables))
	for _, routeTable := range routeTables {
		if routeTable.RouteTableId != nil {
			routeTableIds = append(routeTableIds, *routeTable.RouteTableId)
		}
	}
	return routeTableIds
}
