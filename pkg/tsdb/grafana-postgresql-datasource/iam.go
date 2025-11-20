package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds/rdsutils"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

const (
	awsAuth              = "aws"
	TokenRefreshWindow   = 2 * time.Minute
	DefaultTokenLifeTime = 15 * time.Minute
	RefreshRetryPeriod   = 10 * time.Second
)

var iamLogger = log.New()

func generateIAMToken(ctx context.Context, sess *session.Session, region, dbUser, dbEndpoint string) (string, error) {
	if sess == nil {
		return "", fmt.Errorf("aws session is not configured")
	}
	if region == "" {
		if sess.Config.Region != nil && *sess.Config.Region != "" {
			region = *sess.Config.Region
		} else {
			return "", fmt.Errorf("aws region is not specified")
		}
	}
	endpoint, err := url.Parse(dbEndpoint)
	if err != nil {
		return "", fmt.Errorf("invalid db endpoint: %w", err)
	}
	token, err := rdsutils.BuildAuthToken(endpoint.Host, region, dbUser, sess.Config.Credentials)
	if err != nil {
		return "", fmt.Errorf("failed to build auth token: %w", err)
	}
	return token, nil
}

type AWSCredentials struct {
	Profile string `json:"profile"`
	Region  string `json:"region"`
}

type AWSDatesourceSettings struct {
	AuthType    string `json:"authType"`
	Profile     string `json:"profile"`
	Region      string `json:"region"`
	AssumeRole  string `json:"assumeRole"`
	RoleARN     string `json:"roleArn"`
	ExternalID  string `json:"externalId"`
	SessionName string `json:"sessionName"`
}

type iamManager struct {
	locker *locker
	cache  sync.Map
}

type iamAuth struct {
	token    string
	cancel   context.CancelFunc
	awsSess  *session.Session
	settings *backend.DataSourceInstanceSettings
}

func newIAMManager() *iamManager {
	return &iamManager{
		locker: newLocker(),
		cache:  sync.Map{},
	}
}

func (m *iamManager) getIAMAuthToken(ctx context.Context, settings *backend.DataSourceInstanceSettings) (string, error) {
	key := settings.UID
	m.locker.RLock(key)
	item, ok := m.cache.Load(key)
	m.locker.RUnlock(key)
	if ok {
		return item.(*iamAuth).token, nil
	}
	m.locker.Lock(key)
	defer m.locker.Unlock(key)

	item, ok = m.cache.Load(key)
	if ok {
		return item.(*iamAuth).token, nil
	}
	auth, err := m.newIAMAuth(ctx, settings)
	if err != nil {
		return "", err
	}
	m.cache.Store(key, auth)
	return auth.token, nil
}

func (m *iamManager) newIAMAuth(ctx context.Context, settings *backend.DataSourceInstanceSettings) (*iamAuth, error) {
	println("generating new IAM auth")
	awsSettings := &AWSDatesourceSettings{}
	if err := json.Unmarshal(settings.JSONData, awsSettings); err != nil {
		return nil, fmt.Errorf("could not unmarshal aws settings: %w", err)
	}
	sess, err := newAWSSession(awsSettings)
	if err != nil {
		return nil, err
	}
	token, err := generateIAMToken(ctx, sess, awsSettings.Region, settings.User, settings.URL)
	if err != nil {
		return nil, err
	}
	cancellableCtx, cancel := context.WithCancel(ctx)
	auth := &iamAuth{
		token:    token,
		cancel:   cancel,
		awsSess:  sess,
		settings: settings,
	}
	go auth.refreshToken(cancellableCtx, DefaultTokenLifeTime, TokenRefreshWindow, settings)
	return auth, nil
}

func (auth *iamAuth) refreshToken(ctx context.Context, tokenLifeTime time.Duration, tokenRefreshWindow time.Duration, settings *backend.DataSourceInstanceSettings) {
	ticker := time.NewTicker(tokenLifeTime - tokenRefreshWindow)
	defer ticker.Stop()
	for {
		select {
		case <- ctx.Done():
			return
		case <- ticker.C:
			awsSettings := &AWSDatesourceSettings{}
			if err := json.Unmarshal(auth.settings.JSONData, awsSettings); err != nil {
				iamLogger.Error("failed to unmarshal aws settings", "error", err)
				continue
			}
			token, err := generateIAMToken(ctx, auth.awsSess, awsSettings.Region, auth.settings.User, auth.settings.URL)
			if err != nil {
				iamLogger.Error("failed to refresh iam token", "error", err)
				ticker.Reset(RefreshRetryPeriod)
				continue
			}
			println("token: %w", token)
			settings.DecryptedSecureJSONData["password"] = token
			auth.token = token
			ticker.Reset(tokenLifeTime - tokenRefreshWindow)
			iamLogger.Debug("successfully refreshed iam token")
		}
	}
}

func isIAMAuth(settings *backend.DataSourceInstanceSettings) bool {
	if settings == nil || settings.JSONData == nil {
		return false
	}
	var jsonData struct {
		AuthType string `json:"authType"`
	}
	if err := json.Unmarshal(settings.JSONData, &jsonData); err != nil {
		return false
	}
	return jsonData.AuthType == awsAuth
}

func newAWSSession(settings *AWSDatesourceSettings) (*session.Session, error) {
	config := aws.NewConfig()
	if settings.Region != "" {
		config.WithRegion(settings.Region)
	}
	creds := credentials.NewChainCredentials([]credentials.Provider{
		&credentials.EnvProvider{},
		&credentials.SharedCredentialsProvider{Profile: settings.Profile},
	})
	config.WithCredentials(creds)
	sess, err := session.NewSession(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}
	return sess, nil
}

func (m *iamManager) Dispose(uid string) {
	m.locker.Lock(uid)
	defer m.locker.Unlock(uid)
	if item, ok := m.cache.Load(uid); ok {
		item.(*iamAuth).cancel()
		m.cache.Delete(uid)
	}
}
