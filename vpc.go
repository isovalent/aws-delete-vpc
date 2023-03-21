package main

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/aws/smithy-go"
	"github.com/rs/zerolog/log"
	"go.uber.org/multierr"
)

type clients struct {
	autoscaling          *autoscaling.Client
	ec2                  *ec2.Client
	elasticloadbalancing *elasticloadbalancing.Client
	eks                  *eks.Client
}

func newClientsFromConfig(config aws.Config) *clients {
	return &clients{
		autoscaling:          autoscaling.NewFromConfig(config),
		ec2:                  ec2.NewFromConfig(config),
		elasticloadbalancing: elasticloadbalancing.NewFromConfig(config),
		eks:                  eks.NewFromConfig(config),
	}
}

func deleteVpc(ctx context.Context, client *ec2.Client, vpcId string) error {
	input := ec2.DeleteVpcInput{
		VpcId: aws.String(vpcId),
	}
	_, err := client.DeleteVpc(ctx, &input)
	log.Err(err).
		Str("vpcId", vpcId).
		Msg("DeleteVpc")
	return err
}

// deleteVpcDependencies tries to delete all dependencies of the VPC with ID
// vpcId. It accumulates errors.
func deleteVpcDependencies(ctx context.Context, clients *clients, clusterName string, vpcId string, resources stringSet, autoScalingTagKey *string, autoScalingTagValue *string) (errs error) {
	if resources.contains("VpcPeeringConnections") {
		if vpcPeeringConnections, err := listVpcPeeringConnections(ctx, clients.ec2, vpcId); err != nil {
			log.Err(err).Msg("listVpcPeeringConnections")
			errs = multierr.Append(errs, err)
		} else {
			log.Info().
				Strs("vpcPeeringConnectionIds", vpcPeeringConnectionIds(vpcPeeringConnections)).
				Msg("listVpcPeeringConnections")
			if len(vpcPeeringConnections) > 0 {
				err := deleteVpcPeeringConnections(ctx, clients.ec2, vpcId, vpcPeeringConnections)
				log.Err(err).
					Strs("vpcPeeringConnectionIds", vpcPeeringConnectionIds(vpcPeeringConnections)).
					Msg("deleteVpcPeeringConnections")
				errs = multierr.Append(errs, err)
			}
		}
	}

	if resources.contains("LoadBalancers") {
		if loadBalancerDescriptions, err := listLoadBalancers(ctx, clients.elasticloadbalancing, vpcId); err != nil {
			log.Err(err).Msg("listLoadBalancers")
			errs = multierr.Append(errs, err)
		} else {
			log.Info().
				Strs("loadBalancerNames", loadBalancerNames(loadBalancerDescriptions)).
				Msg("listLoadBalancers")
			if len(loadBalancerDescriptions) > 0 {
				err := deleteLoadBalancers(ctx, clients.elasticloadbalancing, loadBalancerDescriptions)
				log.Err(err).
					Strs("loadBalancerNames", loadBalancerNames(loadBalancerDescriptions)).
					Msg("deleteLoadBalancers")
				errs = multierr.Append(errs, err)
			}
		}
	}

	if resources.contains("AutoScalingGroups") {
		if autoScalingGroups, err := listAutoScalingGroups(ctx, clients.autoscaling, clusterName, autoScalingTagKey, autoScalingTagValue); err != nil {
			log.Err(err).
				Msg("listAutoScalingGroups")
			errs = multierr.Append(errs, err)
		} else {
			log.Info().
				Strs("autoScalingGroupNames", autoScalingGroupNames(autoScalingGroups)).
				Msg("listAutoScalingGroups")
			if len(autoScalingGroups) > 0 {
				err := deleteAutoScalingGroups(ctx, clients.autoscaling, clients.ec2, autoScalingGroups)
				log.Err(err).
					Strs("autoScalingGroupNames", autoScalingGroupNames(autoScalingGroups)).
					Msg("deleteAutoScalingGroups")
				errs = multierr.Append(errs, err)
			}
		}
	}

	if resources.contains("Reservations") {
		if reservations, err := listReservations(ctx, clients.ec2, vpcId); err != nil {
			log.Err(err).Msg("listReservations")
			errs = multierr.Append(errs, err)
		} else {
			log.Info().
				Strs("instanceIds", instanceIds(reservations)).
				Msg("listReservations")
			if len(reservations) > 0 {
				err := terminateInstancesInReservations(ctx, clients.ec2, reservations)
				log.Err(err).
					Strs("instanceIds", instanceIds(reservations)).
					Msg("terminateInstancesInReservations")
				errs = multierr.Append(errs, err)
			}
		}
	}

	if resources.contains("NetworkAcls") {
		if networkAcls, err := listNonDefaultNetworkAcls(ctx, clients.ec2, vpcId); err != nil {
			log.Err(err).
				Msg("listNetworkAcls")
			errs = multierr.Append(errs, err)
		} else {
			log.Info().
				Strs("networkAcls", networkAclIds(networkAcls)).
				Msg("listNetworkAcls")
			if len(networkAcls) > 0 {
				err := deleteNetworkAcls(ctx, clients.ec2, vpcId, networkAcls)
				log.Err(err).
					Strs("networkAcls", networkAclIds(networkAcls)).
					Msg("deleteNetworkAcls")
				errs = multierr.Append(errs, err)
			}
		}
	}

	if resources.contains("NetworkInterfaces") {
		if networkInterfaces, err := listNetworkInterfaces(ctx, clients.ec2, vpcId); err != nil {
			log.Err(err).
				Msg("listNetworkInterfaces")
			errs = multierr.Append(errs, err)
		} else {
			log.Info().
				Strs("networkInterfaceIds", networkInterfaceIds(networkInterfaces)).
				Msg("listNetworkInterfaces")
			if len(networkInterfaces) > 0 {
				err := deleteNetworkInterfaces(ctx, clients.ec2, networkInterfaces)
				log.Err(err).
					Strs("networkInterfaceIds", networkInterfaceIds(networkInterfaces)).
					Msg("deleteNetworkInterfaces")
				errs = multierr.Append(errs, err)
			}
		}
	}

	if resources.contains("NatGateways") {
		if natGateways, err := listNatGateways(ctx, clients.ec2, vpcId); err != nil {
			log.Err(err).
				Msg("listNatGateways")
			errs = multierr.Append(errs, err)
		} else {
			log.Info().
				Strs("natGatewayIds", natGatewayIds(natGateways)).
				Msg("listNatGateways")
			if len(natGateways) > 0 {
				err := deleteNatGateways(ctx, clients.ec2, natGateways)
				log.Err(err).
					Strs("natGatewayIds", natGatewayIds(natGateways)).
					Msg("deleteNatGateways")
				errs = multierr.Append(errs, err)
			}
		}
	}

	if resources.contains("InternetGateways") {
		if internetGateways, err := listInternetGateways(ctx, clients.ec2, vpcId); err != nil {
			log.Err(err).
				Msg("listInternetGateways")
		} else {
			log.Info().
				Strs("internetGatewayIds", internetGatewayIds(internetGateways)).
				Msg("listInternetGateways")
			if len(internetGateways) > 0 {
				err := deleteInternetGateways(ctx, clients.ec2, vpcId, internetGateways)
				log.Err(err).
					Strs("internetGatewayIds", internetGatewayIds(internetGateways)).
					Msg("deleteInternetGateways")
				errs = multierr.Append(errs, err)
			}
		}
	}

	if resources.contains("Subnets") {
		if subnets, err := listSubnets(ctx, clients.ec2, vpcId); err != nil {
			log.Err(err).
				Msg("listSubnets")
		} else {
			log.Info().
				Strs("subnetIds", subnetIds(subnets)).
				Msg("listSubnets")
			if len(subnets) > 0 {
				err := deleteSubnets(ctx, clients.ec2, vpcId, subnets)
				log.Err(err).
					Strs("subnetIds", subnetIds(subnets)).
					Msg("deleteSubnets")
				errs = multierr.Append(errs, err)
			}
		}
	}

	if resources.contains("SecurityGroups") {
		if securityGroups, err := listNonDefaultSecurityGroups(ctx, clients.ec2, vpcId); err != nil {
			log.Err(err).
				Msg("listNonDefaultSecurityGroups")
		} else {
			log.Info().
				Strs("securityGroupIds", securityGroupIds(securityGroups)).
				Msg("listNonDefaultSecurityGroups")
			if len(securityGroups) > 0 {
				err := deleteSecurityGroups(ctx, clients.ec2, vpcId, securityGroups)
				log.Err(err).
					Strs("securityGroupIds", securityGroupIds(securityGroups)).
					Msg("deleteSecurityGroups")
				errs = multierr.Append(errs, err)
			}
		}
	}

	if resources.contains("RouteTables") {
		if routeTables, err := listRouteTables(ctx, clients.ec2, vpcId); err != nil {
			log.Err(err).
				Msg("listRouteTables")
		} else {
			log.Info().
				Strs("routeTableIds", routeTableIds(routeTables)).
				Msg("listRouteTables")
			if len(routeTables) > 0 {
				err := deleteRouteTables(ctx, clients.ec2, vpcId, routeTables)
				log.Err(err).
					Strs("routeTableIds", routeTableIds(routeTables)).
					Msg("deleteRouteTables")
				errs = multierr.Append(errs, err)
			}
		}
	}

	if resources.contains("VpnGateways") {
		if vpnGateways, err := listVpnGateways(ctx, clients.ec2, vpcId); err != nil {
			log.Err(err).
				Msg("listVpnGateways")
			errs = multierr.Append(errs, err)
		} else {
			log.Info().
				Strs("vpnGatewayIds", vpnGatewayIds(vpnGateways)).
				Msg("listVpnGateways")
			if len(vpnGateways) > 0 {
				err := deleteVpnGateways(ctx, clients.ec2, vpcId, vpnGateways)
				log.Err(err).
					Strs("vpnGatewayIds", vpnGatewayIds(vpnGateways)).
					Msg("deleteVpnGateways")
				errs = multierr.Append(errs, err)
			}
		}
	}

	if resources.contains("ElasticIps") && clusterName != "" {
		filters := []ec2types.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: []string{clusterName + "*"},
			},
		}
		if addresses, err := listElasticIps(ctx, clients.ec2, filters); err != nil {
			log.Err(err).
				Msg("listElasticIps")
		} else {
			log.Info().
				Strs("publicIps", publicIps(addresses)).
				Msg("listElasticIps")
			if len(addresses) > 0 {
				err := releaseElasticIps(ctx, clients.ec2, addresses)
				log.Err(err).
					Strs("publicIps", publicIps(addresses)).
					Msg("deleteElasticIps")
				errs = multierr.Append(errs, err)
			}
		}
	}

	err := deleteVpc(ctx, clients.ec2, vpcId)
	log.Err(err).
		Str("vpcId", vpcId).
		Msg("deleteVpc")
	errs = multierr.Append(errs, err)

	return
}

func ec2VpcFilter(vpcId string) []ec2types.Filter {
	return []ec2types.Filter{
		{
			Name:   aws.String("vpc-id"),
			Values: []string{vpcId},
		},
	}
}

func findVpcsByName(ctx context.Context, client *ec2.Client, name string) ([]ec2types.Vpc, error) {
	input := ec2.DescribeVpcsInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: []string{name},
			},
		},
	}
	var vpcs []ec2types.Vpc
	for {
		output, err := client.DescribeVpcs(ctx, &input)
		if err != nil {
			return nil, err
		}
		vpcs = append(vpcs, output.Vpcs...)
		if output.NextToken == nil {
			return vpcs, nil
		}
		input.NextToken = output.NextToken
	}
}

// tryDeleteVpc tries to delete the VPC with ID vpcId. It returns a boolean
// indicating if the VPC was deleted and any error. If the VPC was not deleted
// and the error is nil then the VPC has dependencies that must be deleted
// first.
func tryDeleteVpc(ctx context.Context, client *ec2.Client, vpcId string) (bool, error) {
	err := deleteVpc(ctx, client, vpcId)
	if err == nil {
		return true, nil
	}
	if genericAPIError := (*smithy.GenericAPIError)(nil); errors.As(err, &genericAPIError) {
		switch genericAPIError.ErrorCode() {
		case "InvalidVpcID.NotFound":
			return true, nil
		case "DependencyViolation":
			return false, nil
		}
	}
	return false, err
}

func vpcIds(vpcs []ec2types.Vpc) []string {
	vpcIds := make([]string, 0, len(vpcs))
	for _, vpc := range vpcs {
		if vpc.VpcId != nil {
			vpcIds = append(vpcIds, *vpc.VpcId)
		}
	}
	return vpcIds
}
