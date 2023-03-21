package main

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/rs/zerolog/log"
	"go.uber.org/multierr"
)

func deleteInternetGateways(ctx context.Context, client *ec2.Client, vpcId string, internetGateways []types.InternetGateway) (errs error) {
	for _, internetGateway := range internetGateways {
		if internetGateway.InternetGatewayId == nil {
			continue
		}

		// Detach the InternetGateway from the VPC.
		var internetGatewayErrs error
		for _, internetGatewayAttachment := range internetGateway.Attachments {
			state := internetGatewayAttachment.State
			if state == types.AttachmentStatusDetaching || state == types.AttachmentStatusDetached {
				continue
			}
			if internetGatewayAttachment.VpcId == nil || *internetGatewayAttachment.VpcId != vpcId {
				continue
			}
			_, err := client.DetachInternetGateway(ctx, &ec2.DetachInternetGatewayInput{
				InternetGatewayId: internetGateway.InternetGatewayId,
				VpcId:             internetGatewayAttachment.VpcId,
			})
			log.Err(err).
				Str("InternetGatewayId", *internetGateway.InternetGatewayId).
				Str("VpcId", *internetGatewayAttachment.VpcId).
				Msg("DetachInternetGateway")
			internetGatewayErrs = multierr.Append(internetGatewayErrs, err)
		}
		errs = multierr.Append(errs, internetGatewayErrs)
		if internetGatewayErrs != nil {
			continue
		}

		// Delete the InternetGateway.
		_, err := client.DeleteInternetGateway(ctx, &ec2.DeleteInternetGatewayInput{
			InternetGatewayId: internetGateway.InternetGatewayId,
		})
		log.Err(err).
			Str("InternetGatewayId", *internetGateway.InternetGatewayId).
			Msg("DeleteInternetGateway")
		errs = multierr.Append(errs, err)
	}
	return
}

func internetGatewayIds(internetGateways []types.InternetGateway) []string {
	internetGatewayIds := make([]string, 0, len(internetGateways))
	for _, internetGateway := range internetGateways {
		if internetGateway.InternetGatewayId != nil {
			internetGatewayIds = append(internetGatewayIds, *internetGateway.InternetGatewayId)
		}
	}
	return internetGatewayIds
}

func listInternetGateways(ctx context.Context, client *ec2.Client, vpcId string) ([]types.InternetGateway, error) {
	input := ec2.DescribeInternetGatewaysInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("attachment.vpc-id"),
				Values: []string{vpcId},
			},
		},
	}
	var internetGateways []types.InternetGateway

	paginator := ec2.NewDescribeInternetGatewaysPaginator(client, &input)
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		internetGateways = append(internetGateways, output.InternetGateways...)
	}
	return internetGateways, nil
}
