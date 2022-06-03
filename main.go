package main

// FIXME Delete CloudFormation resources?
// FIXME Delete CloudWatch log groups?

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	autoscalingtypes "github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	ekstypes "github.com/aws/aws-sdk-go-v2/service/eks/types"
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
		"Clusters",
		"ElasticIps",
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
	clusterName := flag.String("cluster-name", "", "cluster name")
	flag.Var(excludeResources, "exclude", "resource types to exclude (default none)")
	flag.Var(includeResources, "include", "resource types to include (default all)")
	retryInterval := flag.Duration("retry-interval", 1*time.Minute, "Re-try interval")
	tries := flag.Int("tries", 3, "tries")
	vpcId := flag.String("vpc-id", "", "VPC ID")

	flag.Parse()

	ctx := context.Background()

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	config, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}
	clients := newClientsFromConfig(config)

	var cluster *ekstypes.Cluster
	if *clusterName != "" {
		cluster, err = listCluster(ctx, clients.eks, *clusterName)
		// if err != nil { V
		// 	var operationErr *smithy.OperationError
		// 	if !errors.As(err, &operationErr) || operationErr.
		// }
		// Ignore ResourceNotFoundExceptions in case the cluster has already been deleted.
		var resourceNotFoundExceptionErr *ekstypes.ResourceNotFoundException
		if err != nil && !errors.As(err, &resourceNotFoundExceptionErr) {
			return err
		}

		// Retrieve the VPC ID from the cluster if it is not already known.
		if *vpcId == "" && cluster != nil && cluster.ResourcesVpcConfig != nil && cluster.ResourcesVpcConfig.VpcId != nil {
			vpcId = cluster.ResourcesVpcConfig.VpcId
		}

		// Retrieve the VPC ID by name if it is not already known. This assumes
		// that the VPC has a tag with key Name and value equal to the cluster
		// name.
		if *vpcId == "" {
			switch vpcs, err := findVpcsByName(ctx, clients.ec2, *clusterName); {
			case err != nil:
				return err
			case len(vpcs) == 0:
				// Do nothing.
			case len(vpcs) == 1:
				vpcId = vpcs[0].VpcId
			default:
				return fmt.Errorf("multiple VPCs with cluster name %q: %s", *clusterName, strings.Join(vpcIds(vpcs), ", "))
			}
		}
	}

	resources := includeResources.subtract(excludeResources)

	// By default, use the tag k8s.io/cluster/$CLUSTER_NAME=owned to identify
	// AutoScalingGroups.
	//
	// FIXME Find an alternative way to detect AutoScalingGroups associated with
	// the VPC.
	if *autoScalingTagKey == "" && *autoScalingTagValue == "owned" && *clusterName != "" {
		autoScalingTagKey = aws.String("k8s.io/cluster/" + *clusterName)
	}
	var autoScalingFilters []autoscalingtypes.Filter
	if *autoScalingTagKey != "" && *autoScalingTagValue != "" {
		autoScalingFilters = []autoscalingtypes.Filter{
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

	if *vpcId == "" {
		return errors.New("VPC ID not set")
	}

	deleted, err := tryDeleteVpc(ctx, clients.ec2, *vpcId)
	log.Err(err).
		Bool("deleted", deleted).
		Str("vpcId", *vpcId).
		Msg("tryDeleteVpc")
	switch {
	case err != nil:
		return err
	case deleted:
		if resources.contains("Clusters") {
			if cluster != nil {
				if err := deleteCluster(ctx, clients.eks, cluster); err != nil {
					return err
				}
			}
		}
		return nil
	}

	for try := 0; try < *tries; try++ {
		if try != 0 {
			log.Info().
				Dur("duration", *retryInterval).
				Msg("Sleep")
			time.Sleep(*retryInterval)
		}

		err := deleteVpcDependencies(ctx, clients, *clusterName, *vpcId, resources, autoScalingFilters)
		log.Err(err).
			Str("vpcId", *vpcId).
			Msg("deleteVpcDependencies")

		deleted, err := tryDeleteVpc(ctx, clients.ec2, *vpcId)
		log.Err(err).
			Bool("deleted", deleted).
			Str("vpcId", *vpcId).
			Msg("tryDeleteVpc")
		if deleted {
			if resources.contains("Clusters") {
				if cluster != nil {
					if err := deleteCluster(ctx, clients.eks, cluster); err != nil {
						return err
					}
				}
			}
			return nil
		}
	}

	return errors.New("failed")
}
