package repo

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/bytedance/sonic"
	"github.com/dongwlin/legero-backend/internal/model"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

var (
	ErrTokenExpired        = errors.New("token expired")
	ErrTokenNotFound       = errors.New("token not found")
	ErrInvalidToken        = errors.New("invalid token format")
	ErrTooManyActiveTokens = errors.New("too many active tokens")
	ErrRenewFailed         = errors.New("token renew failed")
	ErrTokenStale          = errors.New("stale token data")
)

type Token interface {
	Create(ctx context.Context, token *model.Token) (*model.Token, error)
	GetByUserID(ctx context.Context, userID uint64) ([]*model.Token, error)
	GetByToken(ctx context.Context, token string) (*model.Token, error)
	Delete(ctx context.Context, token string) (uint64, error)
}

type TokenImpl struct {
	rdb            *redis.Client
	autoRenew      bool
	renewThreshold time.Duration
	renewDuration  time.Duration
}

func (t *TokenImpl) tokenKey(token string) string {
	return "token:" + token
}

func (t *TokenImpl) userTokenKey(userID uint64) string {
	return "user_tokens:" + strconv.FormatUint(userID, 10)
}

// Create implements Token.
func (t *TokenImpl) Create(ctx context.Context, token *model.Token) (*model.Token, error) {

	expiration := time.Until(token.ExpiredAt)

	if expiration < time.Minute {
		return nil, fmt.Errorf("%w: expiration too soon", ErrInvalidToken)
	}

	token.CreatedAt = time.Now()
	data, err := sonic.Marshal(token)
	if err != nil {
		return nil, fmt.Errorf("%w: marshal failed", ErrInvalidToken)
	}

	userTokenKey := t.userTokenKey(token.UserID)

	txn := func(tx *redis.Tx) error {

		count, err := tx.SCard(ctx, userTokenKey).Result()
		if err != nil && err != redis.Nil {
			return err
		}
		if count > 10 {
			return ErrTooManyActiveTokens
		}

		_, err = tx.TxPipelined(ctx, func(p redis.Pipeliner) error {
			p.Set(ctx, t.tokenKey(token.AccessToken), data, expiration)
			p.SAdd(ctx, userTokenKey, token.AccessToken)
			return nil
		})
		return err
	}

	err = t.rdb.Watch(ctx, txn, userTokenKey)
	if err != nil {
		return nil, err
	}

	return token, nil
}

// GetByToken implements Token.
func (t *TokenImpl) GetByToken(ctx context.Context, tokenStr string) (*model.Token, error) {

	key := t.tokenKey(tokenStr)
	data, err := t.rdb.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrTokenNotFound
		}

		return nil, err
	}

	var token model.Token
	if err := sonic.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("%w: invalid formt", ErrInvalidToken)
	}

	now := time.Now()
	if now.After(token.ExpiredAt) {
		go t.asyncCleanup(context.Background(), token.UserID, token.AccessToken)
		return nil, fmt.Errorf("%w: expired at %s", ErrTokenExpired, token.ExpiredAt)
	}

	if t.autoRenew && time.Until(token.ExpiredAt) < t.renewThreshold {
		newExpiry := now.Add(t.renewDuration)
		if err := t.renewToken(ctx, &token, newExpiry); err != nil {
			log.Error().
				Err(err).
				Msg("failed to renew token")
		}
	}

	return &token, nil
}

func (t *TokenImpl) renewToken(ctx context.Context, token *model.Token, newExpiry time.Time) error {

	key := t.tokenKey(token.AccessToken)
	return t.rdb.Watch(ctx, func(tx *redis.Tx) error {

		currentData, err := tx.Get(ctx, key).Bytes()
		if err != nil {
			return err
		}

		var currentToken model.Token
		if err := sonic.Unmarshal(currentData, &currentToken); err != nil {
			return err
		}
		if currentToken.ExpiredAt.Equal(token.ExpiredAt) {
			return fmt.Errorf("%w: expiration time mismatch", ErrTokenStale)
		}

		token.ExpiredAt = newExpiry
		newData, err := sonic.Marshal(token)
		if err != nil {
			return err
		}

		_, err = tx.TxPipelined(ctx, func(p redis.Pipeliner) error {
			p.Set(ctx, key, newData, newExpiry.Sub(time.Now()))
			return nil
		})
		return err
	}, key)
}

// GetByUserID implements Token.
func (t *TokenImpl) GetByUserID(ctx context.Context, userID uint64) ([]*model.Token, error) {

	userTokenKey := t.userTokenKey(userID)
	tokens, err := t.rdb.SMembers(ctx, userTokenKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}

		return nil, err
	}

	pipe := t.rdb.Pipeline()
	cmds := make(map[string]*redis.StringCmd)
	for _, token := range tokens {
		cmds[token] = pipe.Get(ctx, t.tokenKey(token))
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, err
	}

	validTokens := make([]*model.Token, 0, len(tokens))
	for _, token := range tokens {
		data, err := cmds[token].Bytes()
		if err != nil {
			if err == redis.Nil {
				go t.asyncCleanup(ctx, userID, token)
				continue
			}

			return nil, err
		}

		var t model.Token
		if err := sonic.Unmarshal(data, &t); err != nil {
			continue
		}

		if time.Now().Before(t.ExpiredAt) {
			validTokens = append(validTokens, &t)
		}
	}

	return validTokens, nil
}

// Delete implements Token.
func (t *TokenImpl) Delete(ctx context.Context, tokenStr string) (uint64, error) {

	token, err := t.GetByToken(ctx, tokenStr)
	if err != nil {
		if errors.Is(err, ErrTokenNotFound) {
			return 0, nil
		}

		return 0, err
	}

	userTokenKey := t.userTokenKey(token.UserID)
	_, err = t.rdb.TxPipelined(ctx, func(p redis.Pipeliner) error {
		p.Del(ctx, t.tokenKey(token.AccessToken))
		p.SRem(ctx, userTokenKey, token.AccessToken)
		return nil
	})
	if err != nil {
		return 0, err
	}

	return token.UserID, nil
}

func (t *TokenImpl) asyncCleanup(ctx context.Context, userID uint64, token string) {

	_, err := t.rdb.TxPipelined(ctx, func(p redis.Pipeliner) error {
		p.Del(ctx, t.tokenKey(token))
		p.SRem(ctx, t.userTokenKey(userID), token)
		return nil
	})

	if err != nil {
		log.Error().
			Err(err).
			Msg("failed to async cleanup exipred token")
	}
}

func NewToken(rdb *redis.Client) Token {
	return &TokenImpl{
		rdb: rdb,
	}
}
