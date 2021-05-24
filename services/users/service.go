package users

import (
	"context"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/anBertoli/snap-vault/pkg/store"
	"github.com/anBertoli/snap-vault/pkg/validator"
)

type UsersService struct {
	Store store.Store
}

func (us *UsersService) RegisterUser(ctx context.Context, name, email, password string) (store.User, store.Keys, string, error) {

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

	keys, err := us.Store.Keys.New(user.ID)
	if err != nil {
		return store.User{}, store.Keys{}, "", err
	}
	err = us.Store.Permissions.ReplaceForKey(keys.ID, store.PermissionMain)
	if err != nil {
		return store.User{}, store.Keys{}, "", err
	}

	activationToken, err := us.Store.Tokens.New(user.ID, time.Hour*24, store.ScopeActivation)
	if err != nil {
		return store.User{}, store.Keys{}, "", err
	}

	err = us.Store.Stats.InsertForUser(user.ID)
	if err != nil {
		return store.User{}, store.Keys{}, "", err
	}

	return user, keys, activationToken.Plain, nil
}

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

func (us *UsersService) GenKeyRecoveryToken(ctx context.Context, email, password string) (string, error) {

	user, err := us.Store.Users.GetForEmail(email)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrRecordNotFound):
			return "", store.ErrUnauthenticated
		default:
			return "", err
		}
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		switch {
		case err == bcrypt.ErrMismatchedHashAndPassword:
			return "", store.ErrUnauthenticated
		default:
			return "", err
		}
	}

	recoverKeysToken, err := us.Store.Tokens.New(user.ID, time.Hour*3, store.ScopeRecoverMainKeys)
	if err != nil {
		return "", err
	}

	return recoverKeysToken.Plain, nil
}

func (us *UsersService) RecoverMainKey(ctx context.Context, token string) (store.Keys, error) {

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

	key, err := us.Store.Keys.New(user.ID)
	if err != nil {
		return store.Keys{}, err
	}
	err = us.Store.Permissions.ReplaceForKey(key.ID, store.PermissionMain)
	if err != nil {
		return store.Keys{}, err
	}

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

func (us *UsersService) ListUserKeys(ctx context.Context) ([]KeysList, error) {
	auth := store.ContextGetAuth(ctx)

	keys, err := us.Store.Keys.GetAllForUser(auth.User.ID)
	if err != nil {
		return nil, err
	}

	if !auth.Permissions.Include(store.PermissionMain) {
		for _, k := range keys {
			if k.ID == auth.Keys.ID {
				keys = []store.Keys{k}
			}
		}
	}

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

func (us *UsersService) AddUserKey(ctx context.Context, permissions store.Permissions) (store.Keys, error) {
	auth := store.ContextGetAuth(ctx)

	keys, err := us.Store.Keys.New(auth.User.ID)
	if err != nil {
		return store.Keys{}, err
	}

	err = us.Store.Permissions.ReplaceForKey(keys.ID, permissions...)
	if err != nil {
		return store.Keys{}, err
	}

	return keys, nil
}

func (us *UsersService) EditUserKey(ctx context.Context, keyID int64, permissions store.Permissions) (store.Keys, store.Permissions, error) {
	auth := store.ContextGetAuth(ctx)

	var targetKeys *store.Keys
	userKeys, err := us.Store.Keys.GetAllForUser(auth.User.ID)
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

	oldPermissions, err := us.Store.Permissions.GetAllForKey(targetKeys.AuthKeyHash, true)
	if err != nil {
		return store.Keys{}, store.Permissions{}, err
	}
	if oldPermissions.Include(store.PermissionMain) {
		return store.Keys{}, store.Permissions{}, ErrMainKeysEdit
	}

	err = us.Store.Permissions.ReplaceForKey(targetKeys.ID, permissions...)
	if err != nil {
		return store.Keys{}, store.Permissions{}, err
	}

	return *targetKeys, permissions, nil
}

func (us *UsersService) DeleteUserKey(ctx context.Context, keyID int64) error {
	auth := store.ContextGetAuth(ctx)

	var targetKeys *store.Keys
	userKeys, err := us.Store.Keys.GetAllForUser(auth.User.ID)
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

func (us *UsersService) GetStats(ctx context.Context) (store.Stats, error) {
	auth := store.ContextGetAuth(ctx)

	stats, err := us.Store.Stats.GetForUser(auth.User.ID)
	if err != nil {
		return store.Stats{}, err
	}

	return stats, nil
}
