package localstacktest

import (
	"context"
	"fmt"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/localstack"
	"io"
	"os"
	"sync"
)

var (
	once                sync.Once
	localStackContainer testcontainers.Container
)

func StartLocalStack(ctx context.Context) error {
	var err error
	once.Do(func() {

		ctx := context.Background()

		localstackContainer, err := localstack.RunContainer(ctx,
			testcontainers.WithImage("localstack/localstack:1.4.0"),
		)
		if err != nil {
			panic(err)
		}

		logs, _ := localstackContainer.Logs(ctx)
		fmt.Printf("🧚 ")
		io.Copy(os.Stdout, logs)

		// Clean up the container
		defer func() {
			if err := localstackContainer.Terminate(ctx); err != nil {
				panic(err)
			}
		}()

		//req := testcontainers.ContainerRequest{
		//	Image:        "localstack/localstack",
		//	ExposedPorts: []string{"4566/tcp"},
		//	WaitingFor:   wait.ForLog("Ready."),
		//	Env: map[string]string{
		//		"SERVICES":       "s3,dynamodb",
		//		"DEFAULT_REGION": "us-east-1",
		//		"DEBUG":          "1",
		//		"DATA_DIR":       "/tmp/localstack/data",
		//	},
		//}
		//
		//localStackContainer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		//	ContainerRequest: req,
		//	Started:          true,
		//})
	})
	return err
}

func StopLocalStack(ctx context.Context) error {
	return localStackContainer.Terminate(ctx)
}
