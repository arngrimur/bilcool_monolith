package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const LocalStackInternalPort = "4566/tcp"

type AwsLocalCloud struct {
	CancelFunc context.CancelFunc
	Ctx        context.Context
	LocalStack *testcontainers.DockerContainer
	provider   *testcontainers.DockerProvider
}

func SetupLocalCloud(t *testing.T, services string) *AwsLocalCloud {
	t.Helper()
	t.Setenv("AWS_ACCESS_KEY_ID", "test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	t.Setenv("AWS_DEFAULT_REGION", "eu-north-1")
	localCloud := &AwsLocalCloud{}
	localCloud.Ctx, localCloud.CancelFunc = context.WithCancel(context.Background())
	var err error = nil
	localCloud.LocalStack, err = testcontainers.Run(
		localCloud.Ctx, "localstack/localstack:4",
		testcontainers.WithExposedPorts(LocalStackInternalPort),
		testcontainers.WithEnv(map[string]string{"SERVICES": services}),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort(LocalStackInternalPort),
			wait.ForLog("Ready."),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	return localCloud
}

func (s *AwsLocalCloud) TearDown(t *testing.T) {
	go s.CancelFunc()
	testcontainers.CleanupContainer(t, s.LocalStack)
}

func (a *AwsLocalCloud) CreateConfig(t *testing.T) aws.Config {
	mappedPort, err := a.LocalStack.MappedPort(a.Ctx, LocalStackInternalPort)
	require.NoError(t, err)

	provider, err := testcontainers.NewDockerProvider()
	require.NoError(t, err)
	defer provider.Close()

	host, err := provider.DaemonHost(a.Ctx)
	require.NoError(t, err)

	awsCfg, err := config.LoadDefaultConfig(a.Ctx,
		config.WithRegion("eu-north-1"),
		config.WithBaseEndpoint("http://"+host+":"+mappedPort.Port()),
	)
	require.NoError(t, err)

	return awsCfg

}
