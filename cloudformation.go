package main

import (
	"context"
	"errors"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/rs/zerolog/log"
	"time"
)

/*
EKS can have more than 1 CloudFormation stacks which can be dependent.
AWS API will silently do nothing if parent stack will be deleted before child.
The function will try to delete all the stacks, sleep for a while and check stacks again.
*/
func deleteCloudFormation(ctx context.Context, client *cloudformation.Client, clusterName string) error {
	stacks, err := listCloudFormationStacks(ctx, client, clusterName)
	if err != nil {
		return err
	}
	if len(stacks) == 0 {
		return nil
	}
	for _, stack := range stacks {
		_, err := client.DeleteStack(ctx, &cloudformation.DeleteStackInput{
			StackName: stack.StackName,
		})
		if err != nil {
			return err
		}
	}
	// Give AWS a bit of time to update the statuses
	time.Sleep(time.Second * 10)
	// Check statuses one more time
	stacks, err = listCloudFormationStacks(ctx, client, clusterName)
	if err == nil && len(stacks) > 0 {
		err = errors.New("CloudFormation failed")
	}
	log.Err(err).
		Str("Name", clusterName).
		Msg("CloudFormation")
	return err
}

var clusterTags = newStringSet("alpha.eksctl.io/cluster-name", "eksctl.cluster.k8s.io/v1alpha1/cluster-name")

func listCloudFormationStacks(ctx context.Context, client *cloudformation.Client, clusterName string) ([]types.Stack, error) {
	result := make([]types.Stack, 0)
	var nextToken *string
	for {
		res, err := client.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, err
		}
		for _, stack := range res.Stacks {
			if stack.StackStatus == types.StackStatusDeleteInProgress ||
				stack.StackStatus == types.StackStatusDeleteFailed ||
				stack.StackStatus == types.StackStatusDeleteComplete {
				continue
			}
			for _, tag := range stack.Tags {
				if clusterTags.contains(*tag.Key) && *tag.Value == clusterName {
					result = append(result, stack)
					break
				}
			}
		}
		if res.NextToken == nil {
			return result, nil
		}
		nextToken = res.NextToken
	}
}
