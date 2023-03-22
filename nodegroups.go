package main

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"
)

func deleteClusterNodeGroups(ctx context.Context, client *eks.Client, cluster *types.Cluster) error {
	nodeGroups, err := listClusterNodeGroups(ctx, client, cluster)
	if err != nil {
		return err
	}
	for _, group := range nodeGroups {
		_, err = client.DeleteNodegroup(ctx, &eks.DeleteNodegroupInput{
			ClusterName:   cluster.Name,
			NodegroupName: &group,
		})
		if err != nil {
			return err
		}
	}
	return err
}

func listClusterNodeGroups(ctx context.Context, client *eks.Client, cluster *types.Cluster) ([]string, error) {
	var nextToken *string
	result := make([]string, 0)
	for {
		groups, err := client.ListNodegroups(ctx, &eks.ListNodegroupsInput{
			ClusterName: cluster.Name,
			NextToken:   nextToken,
		})
		if err != nil {
			return nil, err
		}
		result = append(result, groups.Nodegroups...)
		if len(groups.Nodegroups) == 0 || groups.NextToken == nil {
			break
		}
		nextToken = groups.NextToken
	}
	return result, nil
}
