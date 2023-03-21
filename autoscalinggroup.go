package main

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/rs/zerolog/log"
	"go.uber.org/multierr"
)

func autoScalingGroupNames(autoScalingGroups []types.AutoScalingGroup) []string {
	autoScalingGroupNames := make([]string, 0, len(autoScalingGroups))
	for _, autoScalingGroup := range autoScalingGroups {
		if autoScalingGroup.AutoScalingGroupName != nil {
			autoScalingGroupNames = append(autoScalingGroupNames, *autoScalingGroup.AutoScalingGroupName)
		}
	}
	return autoScalingGroupNames
}

func deleteAutoScalingGroups(ctx context.Context, client *autoscaling.Client, ec2Client *ec2.Client, autoScalingGroups []types.AutoScalingGroup) (errs error) {
	for _, autoScalingGroup := range autoScalingGroups {
		if autoScalingGroup.AutoScalingGroupName == nil {
			continue
		}

		// Resize the AutoScalingGroup to zero if not already zero.
		if (autoScalingGroup.DesiredCapacity != nil && *autoScalingGroup.DesiredCapacity != 0) ||
			(autoScalingGroup.MaxSize != nil && *autoScalingGroup.MaxSize != 0) ||
			(autoScalingGroup.MinSize != nil && *autoScalingGroup.MinSize != 0) {
			_, err := client.UpdateAutoScalingGroup(ctx, &autoscaling.UpdateAutoScalingGroupInput{
				AutoScalingGroupName: autoScalingGroup.AutoScalingGroupName,
				DesiredCapacity:      aws.Int32(0),
				MaxSize:              aws.Int32(0),
				MinSize:              aws.Int32(0),
			})
			log.Err(err).
				Str("AutoScalingGroupName", *autoScalingGroup.AutoScalingGroupName).
				Msg("UpdateAutoScalingGroup")
			errs = multierr.Append(errs, err)
		}

		// Wait for any Instances to terminate.
		instanceIds := make([]string, 0, len(autoScalingGroup.Instances))
		for _, instance := range autoScalingGroup.Instances {
			if instance.InstanceId != nil {
				instanceIds = append(instanceIds, *instance.InstanceId)
			}
		}
		if len(instanceIds) > 0 {
			instanceTerminatedWaiter := ec2.NewInstanceTerminatedWaiter(ec2Client)
			log.Info().
				Strs("InstanceIds", instanceIds).
				Msg("InstanceTerminatedWaiter.Wait")
			err := instanceTerminatedWaiter.Wait(ctx, &ec2.DescribeInstancesInput{
				InstanceIds: instanceIds,
			}, instanceTerminatedWaiterMaxDuration)
			log.Err(err).
				Msg("InstanceTerminatedWaiter.Wait")
			errs = multierr.Append(errs, err)
			if err != nil {
				continue
			}
		}

		// Delete the AutoScalingGroup.
		_, err := client.DeleteAutoScalingGroup(ctx, &autoscaling.DeleteAutoScalingGroupInput{
			AutoScalingGroupName: autoScalingGroup.AutoScalingGroupName,
		})
		log.Err(err).
			Str("AutoScalingGroupName", *autoScalingGroup.AutoScalingGroupName).
			Msg("DeleteAutoScalingGroup")
		errs = multierr.Append(errs, err)
	}
	return
}

func listAutoScalingGroups(ctx context.Context, client *autoscaling.Client, clusterName string, paramTagKey *string, paramTagValue *string) ([]types.AutoScalingGroup, error) {
	autoScalingGroups := make([]types.AutoScalingGroup, 0)
	filters := autoScalingFilters(clusterName, paramTagKey, paramTagValue)
	groupNames := newStringSet()
	for _, filter := range filters {
		groups, err := describeAutoScalingGroups(ctx, client, filter)
		if err != nil {
			return nil, err
		}
		for _, group := range groups {
			if groupNames.contains(*group.AutoScalingGroupARN) {
				continue
			}
			autoScalingGroups = append(autoScalingGroups, group)
			_ = groupNames.Set(*group.AutoScalingGroupARN)
		}
	}
	return autoScalingGroups, nil
}

func autoScalingFilters(clusterName string, paramTagKey *string, paramTagValue *string) [][]types.Filter {
	filters := make([][]types.Filter, 0)
	if clusterName != "" {
		filters = append(filters, []types.Filter{
			{
				Name:   aws.String("tag-key"),
				Values: []string{"k8s.io/cluster-autoscaler/" + clusterName, "kubernetes.io/cluster/" + clusterName, "k8s.io/cluster/" + clusterName},
			},
			{
				Name:   aws.String("tag-value"),
				Values: []string{"owned"},
			},
		})
		filters = append(filters, []types.Filter{
			{
				Name:   aws.String("tag-key"),
				Values: []string{"eks:cluster-name"},
			},
			{
				Name:   aws.String("tag-value"),
				Values: []string{clusterName},
			},
		})
	}
	if *paramTagKey != "" && *paramTagValue != "" {
		filters = append(filters, []types.Filter{
			{
				Name:   aws.String("tag-key"),
				Values: []string{*paramTagKey},
			},
			{
				Name:   aws.String("tag-value"),
				Values: []string{*paramTagValue},
			},
		})
	}
	return filters
}

func describeAutoScalingGroups(ctx context.Context, client *autoscaling.Client, filters []types.Filter) ([]types.AutoScalingGroup, error) {
	var autoScalingGroups []types.AutoScalingGroup
	for {
		input := &autoscaling.DescribeAutoScalingGroupsInput{
			Filters: filters,
		}
		output, err := client.DescribeAutoScalingGroups(ctx, input)
		if err != nil {
			return nil, err
		}
		autoScalingGroups = append(autoScalingGroups, output.AutoScalingGroups...)
		if output.NextToken == nil {
			return autoScalingGroups, nil
		}
		input.NextToken = output.NextToken
	}
}
