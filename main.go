package main

// FIXME Delete CloudFormation resources?

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	instanceTerminatedWaiterMaxDuration = 5 * time.Minute
)

var (
	vpcId = flag.String("vpc-id", "", "VPC ID")
	tries = flag.Int("tries", 1, "tries")
)

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
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

	for try := 0; try < *tries; try++ {
		err := deleteVpcDependencies(ctx, clients, *vpcId)
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
