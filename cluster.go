package main

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/rs/zerolog/log"
)

func deleteCluster(ctx context.Context, client *eks.Client, cluster *types.Cluster) error {
	if err := deleteClusterNodeGroups(ctx, client, cluster); err != nil {
		log.Err(err).
			Str("Name", *cluster.Name).
			Msg("DeleteClusterNodeGroups")
		return err
	}
	_, err := client.DeleteCluster(ctx, &eks.DeleteClusterInput{
		Name: cluster.Name,
	})
	log.Err(err).
		Str("Name", *cluster.Name).
		Msg("DeleteCluster")
	return err
}

func listCluster(ctx context.Context, client *eks.Client, clusterName string) (*types.Cluster, error) {
	output, err := client.DescribeCluster(ctx, &eks.DescribeClusterInput{
		Name: aws.String(clusterName),
	})
	if err != nil {
		return nil, err
	}
	return output.Cluster, nil
}
