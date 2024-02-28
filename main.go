package main

import (
	"context"
	"fmt"

	goGrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/access/grpc"
	"github.com/rs/zerolog"
)

var evmEventTypes = []string{
	"evm.BlockExecuted",
	"evm.TransactionExecuted",
}

func main() {
	logger := zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Logger()

	flowClient, err := grpc.NewBaseClient(
		grpc.PreviewnetHost,
		goGrpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	latestBlockHeader, err := flowClient.GetLatestBlockHeader(ctx, true)
	if err != nil {
		panic(err)
	}

	data, errChan, initErr := flowClient.SubscribeEventsByBlockHeight(
		ctx,
		latestBlockHeader.Height, // change height to any value
		flow.EventFilter{
			EventTypes: evmEventTypes,
		},
		grpc.WithHeartbeatInterval(20),
	)

	if initErr != nil {
		logger.Error().Msgf("could not subscribe to events: %v", initErr)
	}

	for {
		select {
		case <-ctx.Done():
			return

		case response, ok := <-data:
			if !ok {
				if ctx.Err() != nil {
					return
				}
				logger.Error().Msg("subscription closed - reconnecting")
				return
			}

			logger.Info().Msgf(
				"block height: %d, hash: %s, events: %d",
				response.Height,
				response.BlockID,
				len(response.Events),
			)

			for _, event := range response.Events {
				if event.Type == "evm.TransactionExecuted" {
					fmt.Println("evm.TransactionExecuted: ", event.Value)
				}
				if event.Type == "evm.BlockExecuted" {
					fmt.Println("evm.BlockExecuted: ", event.Value)
				}
			}

		case err, ok := <-errChan:
			if !ok {
				if ctx.Err() != nil {
					return
				}
				return // unexpected close
			}

			logger.Error().Msgf("ERROR: %v", err)
			return
		}
	}
}
