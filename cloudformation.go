package main

import (
	"context"
	"errors"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

/*
EKS can have more than 1 CloudFormation stacks which can be dependent.
AWS API will silently do nothing if parent stack will be deleted before child.
The function will try to delete stacks (with retries) till one of the DELETE statuses is reached.
*/
func deleteCloudFormation(ctx context.Context, client *cloudformation.Client, clusterName string) error {
	const retry = 5
	for i := 0; i < retry; i++ {
		stacks, err := listCloudFormationStacks(ctx, client, clusterName)
		if err != nil {
			return err
		}
		stackNames := make([]*string, 0)
		for _, stack := range stacks {
			if stack.StackStatus != types.StackStatusDeleteInProgress &&
				stack.StackStatus != types.StackStatusDeleteFailed &&
				stack.StackStatus != types.StackStatusDeleteComplete {
				stackNames = append(stackNames, stack.StackName)
			}
		}
		if len(stackNames) == 0 {
			return nil
		}
		for _, name := range stackNames {
			_, err := client.DeleteStack(ctx, &cloudformation.DeleteStackInput{
				StackName: name,
			})
			if err != nil {
				return err
			}
		}
	}
	return errors.New("CloudFormation failed")
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
