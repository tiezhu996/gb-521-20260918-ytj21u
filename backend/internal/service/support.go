package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"mine-ventilation-network-simulator/backend/internal/constants"
	"mine-ventilation-network-simulator/backend/internal/model"
	"mine-ventilation-network-simulator/backend/internal/repository"
	"mine-ventilation-network-simulator/backend/pkg/api"
)

type Actor struct {
	ID        uint   `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"display_name"`
	Role      string `json:"role"`
	RequestID string `json:"-"`
}

func (a Actor) Audit(action, entity string) repository.AuditRecord {
	return repository.AuditRecord{
		RequestID: a.RequestID,
		ActorID:   a.ID, ActorEmail: a.Email,
		Action: action, EntityType: entity,
	}
}

type LoginResult struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      Actor     `json:"user"`
}

type AuditQuery struct {
	Page       int
	PageSize   int
	EntityType string
	Action     string
	Actor      string
	Status     string
	From       *time.Time
	To         *time.Time
}

type authClaims struct {
	Role        string `json:"role"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	jwt.RegisteredClaims
}

type SupportService struct {
	repo      *repository.SupportRepository
	jwtSecret []byte
	jwtTTL    time.Duration
}

func NewSupportService(repo *repository.SupportRepository, jwtSecret string, jwtTTL time.Duration) *SupportService {
	return &SupportService{repo: repo, jwtSecret: []byte(jwtSecret), jwtTTL: jwtTTL}
}

func (s *SupportService) Ready(ctx context.Context) error {
	if err := s.repo.Ready(ctx); err != nil {
		return api.Internal(fmt.Errorf("readiness check: %w", err))
	}
	return nil
}

func (s *SupportService) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	user, err := s.repo.FindUserByEmail(ctx, strings.TrimSpace(email))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, api.Unauthorized("邮箱或密码不正确")
		}
		return nil, api.Internal(fmt.Errorf("authenticate user: %w", err))
	}
	if !user.Active || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return nil, api.Unauthorized("邮箱或密码不正确")
	}
	now := time.Now().UTC()
	expiresAt := now.Add(s.jwtTTL)
	claims := authClaims{
		Role: user.Role, Email: user.Email, DisplayName: user.DisplayName,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "mine-ventilation-network-simulator",
			Subject:   strconv.FormatUint(uint64(user.ID), 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
	if err != nil {
		return nil, api.Internal(fmt.Errorf("sign access token: %w", err))
	}
	actor := Actor{ID: user.ID, Email: user.Email, Name: user.DisplayName, Role: user.Role}
	return &LoginResult{Token: token, ExpiresAt: expiresAt, User: actor}, nil
}

func (s *SupportService) AuthenticateToken(ctx context.Context, raw string) (Actor, error) {
	claims := &authClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method %s", token.Method.Alg())
		}
		return s.jwtSecret, nil
	}, jwt.WithIssuer("mine-ventilation-network-simulator"), jwt.WithExpirationRequired())
	if err != nil || !token.Valid {
		return Actor{}, api.Unauthorized("登录状态已失效，请重新登录")
	}
	id, err := strconv.ParseUint(claims.Subject, 10, 64)
	if err != nil || id == 0 || !constants.ValidRole(claims.Role) {
		return Actor{}, api.Unauthorized("登录凭据内容无效")
	}
	user, err := s.repo.FindUserByID(ctx, uint(id))
	if err != nil || !user.Active || user.Role != claims.Role {
		return Actor{}, api.Unauthorized("账号不可用或权限已变更")
	}
	return Actor{ID: user.ID, Email: user.Email, Name: user.DisplayName, Role: user.Role}, nil
}

func (s *SupportService) ListAudits(ctx context.Context, query AuditQuery) ([]model.AuditEvent, int64, int, int, error) {
	page, pageSize := normalizePage(query.Page, query.PageSize)
	items, total, err := s.repo.ListAudits(ctx, page, pageSize, query.EntityType, query.Action, query.Actor, query.Status, query.From, query.To)
	if err != nil {
		return nil, 0, page, pageSize, api.Internal(fmt.Errorf("list audit events: %w", err))
	}
	return items, total, page, pageSize, nil
}

func normalizePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 25
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func mapRepositoryError(err error, entity string) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, gorm.ErrRecordNotFound):
		return api.NotFound(entity)
	case errors.Is(err, gorm.ErrDuplicatedKey):
		return api.Conflict("DUPLICATE_RESOURCE", entity+"已存在或操作已完成")
	case errors.Is(err, repository.ErrVersionConflict):
		return api.Conflict("VERSION_CONFLICT", "数据已被其他操作者更新，请刷新后重试")
	default:
		return api.Internal(fmt.Errorf("repository %s: %w", entity, err))
	}
}
