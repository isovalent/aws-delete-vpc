package main

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/aws/smithy-go"
	"github.com/rs/zerolog/log"
	"go.uber.org/multierr"
)

type clients struct {
	autoscaling          *autoscaling.Client
	ec2                  *ec2.Client
	elasticloadbalancing *elasticloadbalancing.Client
}

func newClientsFromConfig(config aws.Config) *clients {
	return &clients{
		autoscaling:          autoscaling.NewFromConfig(config),
		ec2:                  ec2.NewFromConfig(config),
		elasticloadbalancing: elasticloadbalancing.NewFromConfig(config),
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
func deleteVpcDependencies(ctx context.Context, clients *clients, vpcId string) (errs error) {
	if true {
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

	if true {
		if autoScalingGroups, err := listAutoScalingGroups(ctx, clients.autoscaling, vpcId); err != nil {
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

	if true {
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

	if true {
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
					Strs("networkInterfaces", networkInterfaceIds(networkInterfaces)).
					Msg("deleteNetworkInterfaces")
				errs = multierr.Append(errs, err)
			}
		}
	}

	if true {
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

	if true {
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

	if true {
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

	if true {
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

	if true {
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

	if true {
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

	err := deleteVpc(ctx, clients.ec2, vpcId)
	log.Err(err).
		Str("vpcId", vpcId).
		Msg("deleteVpc")
	errs = multierr.Append(errs, err)

	return
}

func ec2VpcFilter(vpcId string) []types.Filter {
	return []types.Filter{
		{
			Name:   aws.String("vpc-id"),
			Values: []string{vpcId},
		},
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
