package handlers

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/assaidy/vodzilla/internals/services"
	media_service "github.com/assaidy/vodzilla/internals/services/media"
	notification_service "github.com/assaidy/vodzilla/internals/services/notification"
	reaction_service "github.com/assaidy/vodzilla/internals/services/reaction"
	social_service "github.com/assaidy/vodzilla/internals/services/social"
	user_service "github.com/assaidy/vodzilla/internals/services/user"
	video_service "github.com/assaidy/vodzilla/internals/services/video"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pressly/goose/v3"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	_ "github.com/lib/pq"
)

var (
	testDB       *sql.DB
	testRedis    *redis.Client
	testHandler  *Handler
	testMailer   *mockMailer
	testServices *services.Manager
)

func TestMain(m *testing.M) {
	os.Setenv("EMAIL_FROM", "test@test.com")
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:18-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		panic(err)
	}
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic(err)
	}

	testDB, err = sql.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		panic(err)
	}

	svcs := []string{"user", "video", "media", "reaction", "social", "notification"}
	for _, svc := range svcs {
		dir := "../services/" + svc + "/db/migrations"
		goose.SetTableName(svc + "_goose_db_version")
		if err := goose.Up(testDB, dir, goose.WithAllowMissing()); err != nil {
			panic(err)
		}
	}

	redisReq := testcontainers.ContainerRequest{
		Image:        "redis:8.6-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("* Ready to accept connections tcp"),
	}
	redisContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: redisReq,
		Started:          true,
	})
	if err != nil {
		panic(err)
	}
	redisPort, err := redisContainer.MappedPort(ctx, "6379")
	if err != nil {
		panic(err)
	}

	testRedis = redis.NewClient(&redis.Options{Addr: "localhost:" + redisPort.Port()})

	minioReq := testcontainers.ContainerRequest{
		Image:        "minio/minio:latest",
		ExposedPorts: []string{"9000/tcp"},
		Cmd:          []string{"server", "/data"},
		WaitingFor:   wait.ForLog("API:"),
	}
	minioContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: minioReq,
		Started:          true,
	})
	if err != nil {
		panic(err)
	}
	minioPort, err := minioContainer.MappedPort(ctx, "9000")
	if err != nil {
		panic(err)
	}

	minioCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider("minioadmin", "minioadmin", ""),
		),
		config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(
				func(service, region string, options ...any) (aws.Endpoint, error) {
					return aws.Endpoint{
						URL:               "http://localhost:" + minioPort.Port(),
						HostnameImmutable: true,
					}, nil
				},
			),
		),
	)
	if err != nil {
		panic(err)
	}
	s3Client := s3.NewFromConfig(minioCfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	for _, bucket := range []string{"videos", "avatars", "thumbnails"} {
		_, err := s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
			Bucket: aws.String(bucket),
		})
		if err != nil {
			panic(err)
		}
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	testMailer = newMockMailer()

	userService := user_service.New(testDB, testRedis, s3Client, testMailer, logger.WithGroup("user service"))
	videoService := video_service.New(testDB, testRedis, logger.WithGroup("video service"))
	mediaService := media_service.New(testDB, testRedis, s3Client, logger.WithGroup("media service"))
	reactionService := reaction_service.New(testDB, testRedis, logger.WithGroup("reaction service"))
	socialService := social_service.New(testDB, testRedis, logger.WithGroup("social service"))
	notificationService := notification_service.New(testDB, testRedis, logger.WithGroup("notification service"))

	testServices = services.NewManager(logger.WithGroup("service manager"))
	testServices.Add("user service", userService)
	testServices.Add("video service", videoService)
	testServices.Add("media service", mediaService)
	testServices.Add("reaction service", reactionService)
	testServices.Add("social service", socialService)
	testServices.Add("notification service", notificationService)

	testServices.StartAll()

	testHandler = New(
		logger.WithGroup("handler"),
		testRedis,
		userService,
		videoService,
		mediaService,
		reactionService,
		socialService,
		notificationService,
	)

	code := m.Run()

	testServices.StopAll()

	testDB.Close()
	testRedis.Close()

	pgContainer.Terminate(ctx)
	redisContainer.Terminate(ctx)
	minioContainer.Terminate(ctx)
	os.Exit(code)
}
