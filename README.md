# aws-delete-vpc

Delete Virtual Private Clouds in Amazon Web Services.

## Motivation

VPCs can only be deleted when all of their dependent resources are deleted, and
AWS does not provide any tools to do this automatically.

## Usage

Syntax:

```console
$ export AWS_PROFILE=xxxxxxxx
$ export AWS_REGION=xx-xxxx-x
$ export VPC_ID=vpc-xxxxxxxxxxxxxxxxx
$ export CLUSTER_NAME=xxxxxxxx
$ aws-delete-vpc -vpc-id=$VPC_ID -autoscaling-tag-key=k8s.io/cluster/$CLUSTER_NAME -autoscaling-tag-value=owner
```

This will attempt to delete the specified VPC and its dependent resources.
Several attempts may be needed due to limitations of the AWS API.

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

Future: may lose resources if program is interrupted between detach and delete.

## References

* [I tried to delete my Amazon VPC, and I received a dependency error. How can I delete my Amazon VPC?](https://aws.amazon.com/premiumsupport/knowledge-center/troubleshoot-dependency-error-delete-vpc/)

## License

Apache-2.0