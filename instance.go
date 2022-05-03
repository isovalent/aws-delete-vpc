package main

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/rs/zerolog/log"
)

func instanceIds(reservations []types.Reservation) []string {
	var instanceIds []string
	for _, reservation := range reservations {
		for _, instance := range reservation.Instances {
			if instance.InstanceId != nil {
				instanceIds = append(instanceIds, *instance.InstanceId)
			}
		}
	}
	return instanceIds
}

func listReservations(ctx context.Context, client *ec2.Client, vpcId string) ([]types.Reservation, error) {
	input := ec2.DescribeInstancesInput{
		Filters: ec2VpcFilter(vpcId),
	}
	var reservations []types.Reservation
	for {
		output, err := client.DescribeInstances(ctx, &input)
		if err != nil {
			return nil, err
		}
		reservations = append(reservations, output.Reservations...)
		if output.NextToken == nil {
			return reservations, nil
		}
		input.NextToken = output.NextToken
	}
}

func terminateInstancesInReservations(ctx context.Context, client *ec2.Client, reservations []types.Reservation) error {
	// Find all non-terminated Instances.
	var nonTerminatedInstanceIds []string
	for _, reservation := range reservations {
		for _, instance := range reservation.Instances {
			if instance.InstanceId == nil {
				continue
			}
			if instance.State != nil && instance.State.Name == types.InstanceStateNameTerminated {
				continue
			}
			nonTerminatedInstanceIds = append(nonTerminatedInstanceIds, *instance.InstanceId)
		}
	}

	// If all Instances are terminated then we are done.
	if len(nonTerminatedInstanceIds) == 0 {
		return nil
	}

	// Terminate all non-terminated Instances.
	terminateInstancesOutput, err := client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: nonTerminatedInstanceIds,
	})
	log.Err(err).
		Strs("InstanceIds", nonTerminatedInstanceIds).
		Msg("TerminateInstances")
	if err != nil {
		return err
	}

	// Find all terminating Instances.
	var terminatingInstanceIds []string
	for _, terminatingInstance := range terminateInstancesOutput.TerminatingInstances {
		if terminatingInstance.InstanceId == nil {
			continue
		}
		if terminatingInstance.CurrentState != nil && terminatingInstance.CurrentState.Name == types.InstanceStateNameTerminated {
			continue
		}
		terminatingInstanceIds = append(terminatingInstanceIds, *terminatingInstance.InstanceId)
	}

	// If there are no terminating Instances then we are done.
	if len(terminatingInstanceIds) == 0 {
		return nil
	}

	// Wait for all terminating Instances to terminate.
	instanceTerminatedWaiter := ec2.NewInstanceTerminatedWaiter(client)
	log.Info().
		Strs("InstanceIds", terminatingInstanceIds).
		Msg("InstanceTerminatedWaiter.Wait")
	err = instanceTerminatedWaiter.Wait(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: terminatingInstanceIds,
	}, instanceTerminatedWaiterMaxDuration)
	log.Err(err).
		Msg("InstanceTerminatedWaiter.Wait")
	if err != nil {
		return err
	}

	return nil
}
