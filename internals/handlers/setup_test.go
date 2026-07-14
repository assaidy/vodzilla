package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/assaidy/vodzilla/internals/services"
	media_service "github.com/assaidy/vodzilla/internals/services/media"
	notification_service "github.com/assaidy/vodzilla/internals/services/notification"
	reaction_service "github.com/assaidy/vodzilla/internals/services/reaction"
	social_service "github.com/assaidy/vodzilla/internals/services/social"
	user_service "github.com/assaidy/vodzilla/internals/services/user"
	video_service "github.com/assaidy/vodzilla/internals/services/video"
	"github.com/assaidy/vodzilla/internals/utils/mailer"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	_ "github.com/lib/pq"
)

var (
	testDb       *sql.DB
	testRedis    *redis.Client
	testHandler  *Handler
	testMailer   *mockMailer
	testServices *services.Manager
)

type mockMailer struct {
	mu       sync.Mutex
	messages []mailer.Message
}

func newMockMailer() *mockMailer {
	return &mockMailer{}
}

func (m *mockMailer) Send(_ context.Context, msg mailer.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
	return nil
}

func (m *mockMailer) LastMessage() (mailer.Message, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.messages) == 0 {
		return mailer.Message{}, false
	}
	return m.messages[len(m.messages)-1], true
}

func (m *mockMailer) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = nil
}

type testSession struct {
	Cookies   []*http.Cookie
	CsrfToken string
}

type testUser struct {
	ID       uuid.UUID
	Username string
	Session  *testSession
}

func newTestApp(t *testing.T) *fiber.App {
	t.Helper()

	app := fiber.New(fiber.Config{})
	testHandler.RegisterRoutes(app)
	return app
}

func testRequest(t *testing.T, app *fiber.App, method, path string, body any, session *testSession) *http.Response {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		require.NoError(t, err)
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, path, bodyReader)
	require.NoError(t, err)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if session != nil {
		for _, c := range session.Cookies {
			req.AddCookie(c)
		}
		if session.CsrfToken != "" {
			req.Header.Set("X-CSRF-Token", session.CsrfToken)
		}
	}

	resp, err := app.Test(req)
	require.NoError(t, err)
	return resp
}

func parseResponse(t *testing.T, resp *http.Response) (int, fiber.Map) {
	t.Helper()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	defer resp.Body.Close()

	if len(body) == 0 {
		return resp.StatusCode, nil
	}

	var data fiber.Map
	if err := json.Unmarshal(body, &data); err != nil {
		return resp.StatusCode, nil
	}

	return resp.StatusCode, data
}

func parseArrayResponse(t *testing.T, resp *http.Response) (int, []any) {
	t.Helper()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	defer resp.Body.Close()

	var data []any
	if err := json.Unmarshal(body, &data); err != nil {
		return resp.StatusCode, nil
	}

	return resp.StatusCode, data
}

func extractSession(resp *http.Response) *testSession {
	session := &testSession{}
	for _, c := range resp.Cookies() {
		session.Cookies = append(session.Cookies, c)
		if c.Name == "csrf_token" {
			session.CsrfToken = c.Value
		}
	}
	return session
}

var hrefRe = regexp.MustCompile(`href="([^"]+)"`)

func extractVerificationToken() string {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		msg, ok := testMailer.LastMessage()
		if ok {
			m := hrefRe.FindStringSubmatch(msg.Body)
			if len(m) > 1 {
				u, err := url.Parse(m[1])
				if err == nil {
					if tok := u.Query().Get("token"); tok != "" {
						return tok
					}
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return ""
}

func assertKind(t *testing.T, status int, data fiber.Map, expectedStatus int, expectedKind string) bool {
	t.Helper()

	if status != expectedStatus {
		t.Errorf("status: want %d, got %d", expectedStatus, status)
		return false
	}

	if expectedKind != "" {
		kind, _ := data["kind"].(string)
		if kind != expectedKind {
			t.Errorf("kind: want %s, got %s", expectedKind, kind)
			return false
		}
	}

	return true
}

func createVerifiedUser(t *testing.T, app *fiber.App, email, password, name, username string) *testUser {
	t.Helper()

	resp := testRequest(t, app, http.MethodPost, "/auth/register", fiber.Map{
		"email":    email,
		"password": password,
		"name":     name,
		"username": username,
	}, nil)
	status, _ := parseResponse(t, resp)
	require.Equal(t, 200, status, "register failed")

	testMailer.Clear()

	resp = testRequest(t, app, http.MethodPost, "/auth/verification_email", fiber.Map{
		"email":   email,
		"baseUrl": "http://localhost/auth/verification_email/verify",
	}, nil)
	status, _ = parseResponse(t, resp)
	require.Equal(t, 200, status, "send verification email failed")

	token := extractVerificationToken()
	require.NotEmpty(t, token, "no verification token in redis")

	resp = testRequest(t, app, http.MethodGet, "/auth/verification_email/verify?token="+token, nil, nil)
	status, _ = parseResponse(t, resp)
	require.Equal(t, 200, status, "verify email failed")

	resp = testRequest(t, app, http.MethodPost, "/auth/login", fiber.Map{
		"email":    email,
		"password": password,
	}, nil)
	status, _ = parseResponse(t, resp)
	require.Equal(t, 200, status, "login failed")

	session := extractSession(resp)

	resp = testRequest(t, app, http.MethodGet, "/profiles", nil, session)
	status, data := parseResponse(t, resp)
	require.Equal(t, 200, status, "get profile failed")

	idStr, _ := data["id"].(string)
	userID, err := uuid.Parse(idStr)
	require.NoError(t, err, "parse user id failed")

	return &testUser{
		ID:       userID,
		Username: username,
		Session:  session,
	}
}

func createVerifiedVideo(t *testing.T, ownerID uuid.UUID, title, description string) uuid.UUID {
	t.Helper()

	videoID := uuid.Must(uuid.NewV7())
	_, err := testDb.Exec(`
		INSERT INTO video_service.videos (id, owner_id, title, description, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`, videoID, ownerID, title, description)
	require.NoError(t, err)

	objectKey := uuid.Must(uuid.NewV7()).String()
	_, err = testDb.Exec(`
		INSERT INTO media_service.videos (id, object_key)
		VALUES ($1, $2)
	`, videoID, objectKey)
	require.NoError(t, err)

	return videoID
}

func resetDb(t *testing.T) {
	t.Helper()

	if err := goose.SetDialect("postgres"); err != nil {
		require.NoError(t, err)
	}

	svcs := []string{"user", "video", "media", "reaction", "social", "notification"}
	for _, svc := range svcs {
		dir := "../services/" + svc + "/db/migrations"
		tableName := svc + "_goose_db_version"
		goose.SetTableName(tableName)

		var tableExists bool
		if err := testDb.QueryRow(
			"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = $1)",
			tableName,
		).Scan(&tableExists); err == nil && tableExists {
			if err := goose.Reset(testDb, dir); err != nil {
				require.NoError(t, err)
			}
		}

		if err := goose.Up(testDb, dir, goose.WithAllowMissing()); err != nil {
			require.NoError(t, err)
		}
	}

	slog.Info("database reset complete")
	testRedis.FlushAll(context.Background())
	testMailer.Clear()
}

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

	testDb, err = sql.Open("postgres", connStr)
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
		if err := goose.Up(testDb, dir, goose.WithAllowMissing()); err != nil {
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

	userService := user_service.New(testDb, testRedis, s3Client, testMailer, logger.WithGroup("user service"))
	videoService := video_service.New(testDb, testRedis, logger.WithGroup("video service"))
	mediaService := media_service.New(testDb, testRedis, s3Client, logger.WithGroup("media service"))
	reactionService := reaction_service.New(testDb, testRedis, logger.WithGroup("reaction service"))
	socialService := social_service.New(testDb, testRedis, logger.WithGroup("social service"))
	notificationService := notification_service.New(testDb, testRedis, logger.WithGroup("notification service"))

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

	testDb.Close()
	testRedis.Close()

	pgContainer.Terminate(ctx)
	redisContainer.Terminate(ctx)
	minioContainer.Terminate(ctx)
	os.Exit(code)
}
