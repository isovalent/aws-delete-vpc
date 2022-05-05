package main

// FIXME Delete CloudFormation resources?
// FIXME Delete CloudWatch log groups?

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	instanceTerminatedWaiterMaxDuration = 5 * time.Minute
)

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	excludeResources := newStringSet()
	includeResources := newStringSet(
		"AutoScalingGroups",
		"InternetGateways",
		"LoadBalancers",
		"NatGateways",
		"NetworkAcls",
		"NetworkInterfaces",
		"Reservations",
		"RouteTables",
		"SecurityGroups",
		"Subnets",
		"VpcPeeringConnections",
		"VpnGateways",
	)

	autoScalingTagKey := flag.String("autoscaling-tag-key", "", "AutoScaling tag key")
	autoScalingTagValue := flag.String("autoscaling-tag-value", "owned", `AutoScaling tag value (default "owner")`)
	flag.Var(excludeResources, "exclude", "resource types to exclude (default none)")
	flag.Var(includeResources, "include", "resource types to include (default all)")
	retryInterval := flag.Duration("retry-interval", 1*time.Minute, "Re-try interval")
	tries := flag.Int("tries", 3, "tries")
	vpcId := flag.String("vpc-id", "", "VPC ID")

	flag.Parse()
	if *vpcId == "" {
		return errors.New("VPC ID not set")
	}

	ctx := context.Background()

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	config, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}
	clients := newClientsFromConfig(config)

	deleted, err := tryDeleteVpc(ctx, clients.ec2, *vpcId)
	log.Err(err).
		Bool("deleted", deleted).
		Str("vpcId", *vpcId).
		Msg("tryDeleteVpc")
	switch {
	case err != nil:
		return err
	case deleted:
		return nil
	}

	resources := includeResources.subtract(excludeResources)
	// FIXME Find an alternative way to detect AutoScalingGroups associated with
	// the VPC.
	var autoScalingFilters []types.Filter
	if *autoScalingTagKey != "" && *autoScalingTagValue != "" {
		autoScalingFilters = []types.Filter{
			{
				Name:   aws.String("tag-key"),
				Values: []string{*autoScalingTagKey},
			},
			{
				Name:   aws.String("tag-value"),
				Values: []string{*autoScalingTagValue},
			},
		}
	}

	for try := 0; try < *tries; try++ {
		if try != 0 {
			log.Info().
				Dur("duration", *retryInterval).
				Msg("Sleep")
			time.Sleep(*retryInterval)
		}

		err := deleteVpcDependencies(ctx, clients, *vpcId, resources, autoScalingFilters)
		log.Err(err).
			Str("vpcId", *vpcId).
			Msg("deleteVpcDependencies")

		deleted, err := tryDeleteVpc(ctx, clients.ec2, *vpcId)
		log.Err(err).
			Bool("deleted", deleted).
			Str("vpcId", *vpcId).
			Msg("tryDeleteVpc")
		if deleted {
			return nil
		}
	}

	return errors.New("failed")
}
