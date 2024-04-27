package storage

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/yezzey-gp/aws-sdk-go/aws"
	"github.com/yezzey-gp/aws-sdk-go/aws/credentials"
	"github.com/yezzey-gp/aws-sdk-go/aws/defaults"
	"github.com/yezzey-gp/aws-sdk-go/aws/session"
	"github.com/yezzey-gp/aws-sdk-go/service/s3"
	"github.com/yezzey-gp/yproxy/config"
	"github.com/yezzey-gp/yproxy/pkg/ylogger"

	"golang.org/x/sync/semaphore"
)

type SessionPool interface {
	GetSession(ctx context.Context) (*s3.S3, error)
}

type S3SessionPool struct {
	cnf *config.Storage

	sem *semaphore.Weighted
}

func NewSessionPool(cnf *config.Storage) SessionPool {
	return &S3SessionPool{
		cnf: cnf,
		sem: semaphore.NewWeighted(cnf.StorageConcurrency),
	}
}

// TODO : unit tests
func (sp *S3SessionPool) createSession() (*session.Session, error) {
	s, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	provider := &credentials.StaticProvider{Value: credentials.Value{
		AccessKeyID:     sp.cnf.AccessKeyId,
		SecretAccessKey: sp.cnf.SecretAccessKey,
	}}

	ylogger.Zero.Debug().Str("endpoint", sp.cnf.StorageEndpoint).Msg("acquire external storage session")

	providers := make([]credentials.Provider, 0)
	providers = append(providers, provider)
	providers = append(providers, defaults.CredProviders(s.Config, defaults.Handlers())...)
	newCredentials := credentials.NewCredentials(&credentials.ChainProvider{
		VerboseErrors: aws.BoolValue(s.Config.CredentialsChainVerboseErrors),
		Providers:     providers,
	})

	s.Config.WithRegion(sp.cnf.StorageRegion)

	s.Config.WithEndpoint(sp.cnf.StorageEndpoint)

	s.Config.WithCredentials(newCredentials)
	return s, err
}

func (s *S3SessionPool) GetSession(ctx context.Context) (*s3.S3, error) {
	fmt.Printf("get session 1\n")
	s.sem.Acquire(ctx, 1)
	fmt.Printf("get session 2\n")
	defer s.sem.Release(1)

	sess, err := s.createSession()
	fmt.Printf("get session 3\n")
	if err != nil {
		fmt.Printf("get session 4\n")
		return nil, errors.Wrap(err, "failed to create new session")
	}
	fmt.Printf("get session 5\n")
	return s3.New(sess), nil
}
