# aws-delete-vpc

Delete Virtual Private Clouds in Amazon Web Services.

## Motivation

VPCs can only be deleted when all of their dependent resources are deleted, and
AWS does not provide any tools to do this automatically.

## Installation

To install the latest `aws-delete-vpc` release run:

```console
$ go install github.com/isovalent/aws-delete-vpc@latest
```

## Build

To build `aws-delete-vpc` from source please run the following locally:

```console
$ go build .
```

## Usage

Syntax:

```console
$ aws-delete-vpc -vpc-id=$VPC_ID
```

or

```console
$ aws-delete-vpc -cluster-name=$CLUSTER_NAME
```

This will attempt to delete the specified VPC and its dependent resources.
Several attempts may be needed due to limitations of the AWS API.

If the optional `-cluster-name` flag is passed then the VPC ID will be
discovered automatically and any EKS cluster with the same name deleted after
the VPC is deleted.

## Known limitations

Currently the program is unable to identify AutoScalingGroups associated with
the VPC unassisted. Instead, it looks for and deletes AutoScalingGroups with the
tag key and value specified by the `autoscaling-tag-key` and
`autoscaling-tag-label` command line flags.

Many AWS API calls return incorrect values that prevent the program from
operating correctly. Known problems include:

* `DeleteVpc` will return a `DependencyViolation` error when there are no
  dependent resources, but the VPC will eventually be deleted.

* `InstanceTerminatedWaiter`s return that an instance has terminated before it
  has actually terminated, meaning that deleting related resources (e.g.
  NetworkInterfaces) will fail.

* There is no API to wait for a NetworkInterface to be detached.

Some resources (e.g. InternetGateways, NetworkInterfaces, and VpnGateways) must
be detached before they can be deleted. If the program is interrupted between
detachment and deletion these resources will not be deleted the next time the
program is run.

Future: may lose resources if program is interrupted between detach and delete.

## References

* [I tried to delete my Amazon VPC, and I received a dependency error. How can I delete my Amazon VPC?](https://aws.amazon.com/premiumsupport/knowledge-center/troubleshoot-dependency-error-delete-vpc/)
* [add --all-dependencies option to ec2 delete-vpc](https://github.com/aws/aws-cli/issues/1721)

## License

Apache-2.0
