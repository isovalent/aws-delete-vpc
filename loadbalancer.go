package main

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing/types"
	"github.com/rs/zerolog/log"
	"go.uber.org/multierr"
)

func deleteLoadBalancers(ctx context.Context, client *elasticloadbalancing.Client, loadBalancerDescriptions []types.LoadBalancerDescription) (errs error) {
	for _, loadBalancerDescription := range loadBalancerDescriptions {
		if loadBalancerDescription.LoadBalancerName == nil {
			continue
		}
		_, err := client.DeleteLoadBalancer(ctx, &elasticloadbalancing.DeleteLoadBalancerInput{
			LoadBalancerName: loadBalancerDescription.LoadBalancerName,
		})
		log.Err(err).
			Str("LoadBalancerName", *loadBalancerDescription.LoadBalancerName).
			Msg("DeleteLoadBalancer")
		errs = multierr.Append(errs, err)
	}
	return
}

func listLoadBalancers(ctx context.Context, client *elasticloadbalancing.Client, vpcId string) ([]types.LoadBalancerDescription, error) {
	input := elasticloadbalancing.DescribeLoadBalancersInput{}
	var loadBalancerDescriptions []types.LoadBalancerDescription
	for {
		output, err := client.DescribeLoadBalancers(ctx, &input)
		if err != nil {
			return nil, err
		}
		for _, loadBalancerDescription := range output.LoadBalancerDescriptions {
			if loadBalancerDescription.VPCId == nil || *loadBalancerDescription.VPCId != vpcId {
				continue
			}
			loadBalancerDescriptions = append(loadBalancerDescriptions, loadBalancerDescription)
		}
		if output.NextMarker == nil {
			return loadBalancerDescriptions, nil
		}
		input.Marker = output.NextMarker
	}
}

func loadBalancerNames(loadBalancerDescriptions []types.LoadBalancerDescription) []string {
	loadBalancerNames := make([]string, 0, len(loadBalancerDescriptions))
	for _, loadBalancerDescription := range loadBalancerDescriptions {
		if loadBalancerDescription.LoadBalancerName != nil {
			loadBalancerNames = append(loadBalancerNames, *loadBalancerDescription.LoadBalancerName)
		}
	}
	return loadBalancerNames
}
