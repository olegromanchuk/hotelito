package localstacktest

import (
	"context"
	"fmt"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/localstack"
	"os"
	"sync"
	"sync/atomic"
)

var (
	once                sync.Once
	localstackContainer testcontainers.Container
	testPackagesCounter int32
	ctx                 context.Context
)

func StartLocalStack() error {
	once.Do(func() {
		var err error
		ctx = context.Background() //not really used. We rely on CI/CD to clean up containers

		localstackContainer, err = localstack.RunContainer(ctx,
			testcontainers.WithImage("localstack/localstack:1.4.0"),
		)
		if err != nil {
			panic(err)
		}

		host, err := localstackContainer.Host(ctx)
		if err != nil {
			fmt.Println("Error fetching container host:", err)
			return
		}

		port, err := localstackContainer.MappedPort(ctx, "4566")
		if err != nil {
			fmt.Println("Error fetching container port:", err)
			return
		}

		fmt.Printf("🔥🔥🔥 Localstack is running on %s:%s 🔥🔥🔥\n", host, port.Port())
		// Now you can connect to LocalStack on this host and port.

		//set env vars with host and port of localstack. Will be used later in tests
		os.Setenv("LOCALSTACK_HOST", host)
		os.Setenv("LOCALSTACK_PORT", string(port))

	})
	atomic.AddInt32(&testPackagesCounter, 1)
	return nil
}

func StopLocalStack() error {
	newCounterValue := atomic.AddInt32(&testPackagesCounter, -1)
	if newCounterValue == 0 {
		return localstackContainer.Terminate(ctx)
	}
	return nil
}
