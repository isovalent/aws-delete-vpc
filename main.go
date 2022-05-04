package main

// FIXME Delete CloudFormation resources?

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
	includeResources := newStringSet(
		"AutoScalingGroups",
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
	excludeResources := newStringSet()

	autoScalingTagKey := flag.String("autoscaling-tag-key", "", "AutoScaling tag key")
	autoScalingTagValue := flag.String("autoscaling-tag-value", "owned", `AutoScaling tag value (default "owner")`)
	flag.Var(includeResources, "include", "resource types to include (default all)")
	flag.Var(excludeResources, "exclude", "resource types to exclude (default none)")
	vpcId := flag.String("vpc-id", "", "VPC ID")
	tries := flag.Int("tries", 1, "tries")

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
	autoScalingFilters := []types.Filter{
		{
			Name:   aws.String("tag-key"),
			Values: []string{*autoScalingTagKey},
		},
		{
			Name:   aws.String("tag-value"),
			Values: []string{*autoScalingTagValue},
		},
	}

	for try := 0; try < *tries; try++ {
		err := deleteVpcDependencies(ctx, clients, *vpcId, resources, autoScalingFilters)
		log.Err(err).
			Str("vpcId", *vpcId).
			Msg("deleteVpcDependencies")
		if err != nil {
			continue
		}

		deleted, err := tryDeleteVpc(ctx, clients.ec2, *vpcId)
		log.Err(err).
			Bool("deleted", deleted).
			Str("vpcId", *vpcId).
			Msg("tryDeleteVpc")
		switch {
		case err != nil:
			continue
		case deleted:
			return nil
		}
	}

	return errors.New("failed")
}
