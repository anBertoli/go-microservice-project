package users

import (
	"context"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/anBertoli/snap-vault/pkg/auth"
	"github.com/anBertoli/snap-vault/pkg/store"
	"github.com/anBertoli/snap-vault/pkg/validator"
)

// The UsersService retrieves and save users data and user statistics in a relation database.
type UsersService struct {
	Store store.Store
}

// Register a new user into the system and generate 'main' keys for the user. The
// user must be activated before using any other parts of the application. The
// auth key and the token must be delivered somehow to the user, since we don't
// store the plain text versions.
func (us *UsersService) RegisterUser(ctx context.Context, name, email, password string) (store.User, store.Keys, string, error) {

	// Hash the password. The plain text password is not stored in the db.
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return store.User{}, store.Keys{}, "", err
	}
	user := store.User{
		Name:         name,
		Email:        email,
		Password:     password,
		PasswordHash: string(hash),
	}

	user, err = us.Store.Users.Insert(user)
	if err != nil {
		return store.User{}, store.Keys{}, "", err
	}

	// Create new keys for the user with 'main' permissions. The plan version
	// of the key is returned to the caller and not stored anywhere.
	keys, err := us.Store.Keys.New(user.ID)
	if err != nil {
		return store.User{}, store.Keys{}, "", err
	}
	err = us.Store.Permissions.ReplaceForKey(keys.ID, store.PermissionMain)
	if err != nil {
		return store.User{}, store.Keys{}, "", err
	}

	// Create an activation token that must be delivered in some form to
	// the user. This is responsibility of the caller.
	activationToken, err := us.Store.Tokens.New(user.ID, time.Hour*24, store.ScopeActivation)
	if err != nil {
		return store.User{}, store.Keys{}, "", err
	}

	// Initialize stats for the user.
	err = us.Store.Stats.InitStatsForUser(user.ID)
	if err != nil {
		return store.User{}, store.Keys{}, "", err
	}

	return user, keys, activationToken.Plain, nil
}

// Regenerate the activation token for a specific user. Old activations tokens
// are deleted.
func (us *UsersService) RegenerateActivationToken(ctx context.Context, email, password string) (store.User, string, error) {
	user, err := us.Store.Users.GetForEmail(email)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrRecordNotFound):
			return store.User{}, "", auth.ErrUnauthenticated
		default:
			return store.User{}, "", err
		}
	}

	// Hash the password and compare it with the has we have stored into the db.
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		switch {
		case err == bcrypt.ErrMismatchedHashAndPassword:
			return store.User{}, "", auth.ErrUnauthenticated
		default:
			return store.User{}, "", err
		}
	}

	// No need to regenerate an activation token if the user was already activated.
	if user.Activated {
		return store.User{}, "", ErrAlreadyActive
	}

	// Delete all old activation tokens and recreate a new one.
	err = us.Store.Tokens.DeleteAllForUser(store.ScopeActivation, user.ID)
	if err != nil {
		return store.User{}, "", err
	}
	token, err := us.Store.Tokens.New(user.ID, time.Hour*24, store.ScopeActivation)
	if err != nil {
		return store.User{}, "", err
	}

	return user, token.Plain, nil
}

// Activate the user using the generated activation token.
func (us *UsersService) ActivateUser(ctx context.Context, token string) (store.User, error) {

	user, err := us.Store.Users.GetForToken(store.ScopeActivation, token)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrRecordNotFound):
			v := validator.New()
			v.AddError("token", "invalid or expired activation token")
			return store.User{}, v
		default:
			return store.User{}, err
		}
	}

	user.Activated = true

	user, err = us.Store.Users.Update(user)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrRecordNotFound):
			return store.User{}, store.ErrEditConflict
		default:
			return store.User{}, err
		}
	}

	// Delete all tokens with activation scope.
	err = us.Store.Tokens.DeleteAllForUser(store.ScopeActivation, user.ID)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrRecordNotFound): // ok
		default:
			return store.User{}, err
		}
	}

	return user, nil
}

// Authenticate the user with email and password and generate a token used to
// recover the main auth key for the account. The token must be delivered
// somehow to the user, since we don't store the plain text version.
func (us *UsersService) GenKeyRecoveryToken(ctx context.Context, email, password string) (string, error) {
	user, err := us.Store.Users.GetForEmail(email)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrRecordNotFound):
			return "", auth.ErrUnauthenticated
		default:
			return "", err
		}
	}

	// Hash the password and compare it with the has we have stored into the db.
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		switch {
		case err == bcrypt.ErrMismatchedHashAndPassword:
			return "", auth.ErrUnauthenticated
		default:
			return "", err
		}
	}

	// Create a new key recovery token.
	recoverKeysToken, err := us.Store.Tokens.New(user.ID, time.Hour*3, store.ScopeRecoverMainKeys)
	if err != nil {
		return "", err
	}

	return recoverKeysToken.Plain, nil
}

// Regenerate the main auth key using the provided key recovery token. The auth key
// must be delivered somehow to the user, since we don't store the plain text version.
func (us *UsersService) RegenerateMainKey(ctx context.Context, token string) (store.Keys, error) {

	user, err := us.Store.Users.GetForToken(store.ScopeRecoverMainKeys, token)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrRecordNotFound):
			v := validator.New()
			v.AddError("token", "invalid or expired activation token")
			return store.Keys{}, v
		default:
			return store.Keys{}, err
		}
	}

	// Retrieve all the user auth keys and search the main one. If found, delete it.
	keys, err := us.Store.Keys.GetAllForUser(user.ID)
	if err != nil {
		return store.Keys{}, nil
	}
	for _, key := range keys {
		perms, err := us.Store.Permissions.GetAllForKey(key.AuthKeyHash, true)
		if err != nil {
			return store.Keys{}, err
		}
		if !perms.Include(store.PermissionMain) {
			continue
		}
		err = us.Store.Keys.DeleteKey(key.ID, user.ID)
		if err != nil {
			return store.Keys{}, err
		}
	}

	// Regenerate the auth key. The plan text version of the key is returned
	// and must be delivered to the user, since it is not store anywhere.
	key, err := us.Store.Keys.New(user.ID)
	if err != nil {
		return store.Keys{}, err
	}
	err = us.Store.Permissions.ReplaceForKey(key.ID, store.PermissionMain)
	if err != nil {
		return store.Keys{}, err
	}

	// Clean recovery key tokens for the user.
	err = us.Store.Tokens.DeleteAllForUser(store.ScopeRecoverMainKeys, user.ID)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrRecordNotFound): // ok
		default:
			return store.Keys{}, err
		}
	}

	return key, nil
}

// List all the auth keys of the user.
func (us *UsersService) ListUserKeys(ctx context.Context) ([]KeysList, error) {
	authData := auth.MustContextGetAuth(ctx)

	keys, err := us.Store.Keys.GetAllForUser(authData.User.ID)
	if err != nil {
		return nil, err
	}

	// Enrich the response with keys permissions.
	var keysList []KeysList
	for _, key := range keys {
		permissions, err := us.Store.Permissions.GetAllForKey(key.AuthKeyHash, true)
		if err != nil {
			return nil, err
		}
		keysList = append(keysList, KeysList{
			AuthKeyID:   key.ID,
			CreatedAt:   key.CreatedAt,
			Permissions: permissions,
		})
	}

	return keysList, nil
}

type KeysList struct {
	AuthKeyID   int64             `json:"auth_key_id"`
	CreatedAt   time.Time         `json:"created_at"`
	Permissions store.Permissions `json:"permissions"`
}

// Create a new auth key for the authenticated user with the provided permissions.
func (us *UsersService) AddUserKey(ctx context.Context, permissions store.Permissions) (store.Keys, error) {
	authData := auth.MustContextGetAuth(ctx)

	keys, err := us.Store.Keys.New(authData.User.ID)
	if err != nil {
		return store.Keys{}, err
	}

	err = us.Store.Permissions.ReplaceForKey(keys.ID, permissions...)
	if err != nil {
		return store.Keys{}, err
	}

	return keys, nil
}

// Edit an existing auth key for the authenticated user with the provided permissions.
// The main auth key for the account cannot be edited.
func (us *UsersService) EditUserKey(ctx context.Context, keyID int64, permissions store.Permissions) (store.Keys, store.Permissions, error) {
	authData := auth.MustContextGetAuth(ctx)

	// Search the specified auth key.
	var targetKeys *store.Keys
	userKeys, err := us.Store.Keys.GetAllForUser(authData.User.ID)
	if err != nil {
		return store.Keys{}, store.Permissions{}, err
	}
	for _, uk := range userKeys {
		if uk.ID == keyID {
			targetKeys = &uk
		}
	}
	if targetKeys == nil {
		return store.Keys{}, store.Permissions{}, store.ErrRecordNotFound
	}

	// Avoid editing the main auth key.
	oldPermissions, err := us.Store.Permissions.GetAllForKey(targetKeys.AuthKeyHash, true)
	if err != nil {
		return store.Keys{}, store.Permissions{}, err
	}
	if oldPermissions.Include(store.PermissionMain) {
		return store.Keys{}, store.Permissions{}, ErrMainKeysEdit
	}

	// Replace old permissions with new permissions.
	err = us.Store.Permissions.ReplaceForKey(targetKeys.ID, permissions...)
	if err != nil {
		return store.Keys{}, store.Permissions{}, err
	}

	return *targetKeys, permissions, nil
}

// Delete an existing auth key for the authenticated user. The main auth key for the account
// cannot be deleted.
func (us *UsersService) DeleteUserKey(ctx context.Context, keyID int64) error {
	authData := auth.MustContextGetAuth(ctx)

	var targetKeys *store.Keys
	userKeys, err := us.Store.Keys.GetAllForUser(authData.User.ID)
	if err != nil {
		return err
	}
	for _, uk := range userKeys {
		if uk.ID == keyID {
			targetKeys = &uk
		}
	}
	if targetKeys == nil {
		return store.ErrRecordNotFound
	}

	// Avoid deleting the main auth key.
	oldPermissions, err := us.Store.Permissions.GetAllForKey(targetKeys.AuthKeyHash, true)
	if err != nil {
		return err
	}
	if oldPermissions.Include(store.PermissionMain) {
		return ErrMainKeysEdit
	}

	err = us.Store.Keys.DeleteKey(targetKeys.ID, targetKeys.UserID)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrRecordNotFound):
			return store.ErrEditConflict
		default:
			return err
		}
	}

	return nil
}

// Retrieve the authentication data about the authenticated user.
func (us *UsersService) GetMe(ctx context.Context) (auth.Auth, error) {
	return auth.ContextGetAuth(ctx)
}

// Retrieve statistics about the user.
func (us *UsersService) GetStats(ctx context.Context) (store.Stats, error) {
	authData := auth.MustContextGetAuth(ctx)

	stats, err := us.Store.Stats.GetForUser(authData.User.ID)
	if err != nil {
		return store.Stats{}, err
	}

	return stats, nil
}
