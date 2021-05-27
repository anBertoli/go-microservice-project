package users

import (
	"context"
	"errors"
	"time"

	"github.com/anBertoli/snap-vault/pkg/store"
)

// Public interface for the users service. The service is exposed
// via transport-specific adapters, e.g. the JSON-HTTP api.
type Service interface {
	RegisterUser(ctx context.Context, name, email, password string) (store.User, store.Keys, string, error)
	ActivateUser(ctx context.Context, token string) (store.User, error)
	GenKeyRecoveryToken(ctx context.Context, email, password string) (string, error)
	RecoverMainKey(ctx context.Context, token string) (store.Keys, error)

	ListUserKeys(ctx context.Context) ([]KeysList, error)
	AddUserKey(ctx context.Context, permissions store.Permissions) (store.Keys, error)
	EditUserKey(ctx context.Context, keyID int64, permissions store.Permissions) (store.Keys, store.Permissions, error)
	DeleteUserKey(ctx context.Context, keyID int64) error

	GetStats(ctx context.Context) (store.Stats, error)
}

type KeysList struct {
	AuthKeyID   int64             `json:"auth_key_id"`
	CreatedAt   time.Time         `json:"created_at"`
	Permissions store.Permissions `json:"permissions"`
}

var ErrMainKeysEdit = errors.New("main keys not editable")

// This checks makes sure that all service implementation remain
// valid while we refactor our code.
var _ Service = &UsersService{}
var _ Service = &AuthMiddleware{}
var _ Service = &ValidationMiddleware{}
