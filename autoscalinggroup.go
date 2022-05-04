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

func listAutoScalingGroups(ctx context.Context, client *autoscaling.Client, filters []types.Filter) ([]types.AutoScalingGroup, error) {
	input := autoscaling.DescribeAutoScalingGroupsInput{
		Filters: filters,
	}
	var autoScalingGroups []types.AutoScalingGroup
	for {
		output, err := client.DescribeAutoScalingGroups(ctx, &input)
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
