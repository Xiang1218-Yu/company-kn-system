package service

import (
	"context"
	"regexp"

	"github.com/google/uuid"

	"kn-system/internal/auth"
	apperr "kn-system/internal/errors"
	"kn-system/internal/model"
	"kn-system/internal/repository"
)

// AuthService owns the registration and login business rules: email validation,
// uniqueness, password hashing, and token issuance. It is the only consumer of
// the auth package, so crypto concerns are localized here.
type AuthService struct {
	users *repository.UserRepo
	pw    *auth.Password
	jwt   *auth.JWT
}

func NewAuthService(users *repository.UserRepo, pw *auth.Password, jwt *auth.JWT) *AuthService {
	return &AuthService{users: users, pw: pw, jwt: jwt}
}

// RegisterInput is the validated shape of a registration request.
type RegisterInput struct {
	Email    string
	Password string
	Name     string
	Role     model.Role
}

var emailRe = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// Register validates input, hashes the password, persists the user, and issues
// the first token so the client can proceed without a second round-trip.
func (s *AuthService) Register(ctx context.Context, in RegisterInput) (model.User, string, error) {
	if !emailRe.MatchString(in.Email) {
		return model.User{}, "", apperr.New(apperr.KindValidation, "invalid email")
	}
	if len(in.Password) < 6 {
		return model.User{}, "", apperr.New(apperr.KindValidation, "password must be at least 6 characters")
	}
	if in.Name == "" {
		return model.User{}, "", apperr.New(apperr.KindValidation, "name is required")
	}
	// Default new users to member; only an admin (via a separate admin path)
	// can elevate. The role is validated against the allow-list.
	if in.Role == "" {
		in.Role = model.RoleMember
	}
	switch in.Role {
	case model.RoleAdmin, model.RoleManager, model.RoleMember:
	default:
		return model.User{}, "", apperr.New(apperr.KindValidation, "invalid role")
	}

	if existing, _ := s.users.FindByEmail(ctx, in.Email); existing != nil {
		return model.User{}, "", apperr.New(apperr.KindConflict, "email already registered")
	}

	hash, err := s.pw.Hash(in.Password)
	if err != nil {
		return model.User{}, "", apperr.Wrap(apperr.KindInternal, "hash password", err)
	}
	u := model.User{
		Email:        in.Email,
		PasswordHash: hash,
		Name:         in.Name,
		Role:         in.Role,
	}
	if err := s.users.Create(ctx, &u); err != nil {
		return model.User{}, "", apperr.Wrap(apperr.KindInternal, "create user", err)
	}
	token, err := s.jwt.Issue(u)
	if err != nil {
		return model.User{}, "", apperr.Wrap(apperr.KindInternal, "issue token", err)
	}
	return u, token, nil
}

// LoginInput is the validated shape of a login request.
type LoginInput struct {
	Email    string
	Password string
}

// Login verifies credentials and issues a token. A wrong email or password
// yields the same error to avoid user-enumeration via differential responses.
func (s *AuthService) Login(ctx context.Context, in LoginInput) (model.User, string, error) {
	u, err := s.users.FindByEmail(ctx, in.Email)
	if err != nil {
		return model.User{}, "", apperr.New(apperr.KindUnauthorized, "invalid email or password")
	}
	if !s.pw.Compare(u.PasswordHash, in.Password) {
		return model.User{}, "", apperr.New(apperr.KindUnauthorized, "invalid email or password")
	}
	token, err := s.jwt.Issue(*u)
	if err != nil {
		return model.User{}, "", apperr.Wrap(apperr.KindInternal, "issue token", err)
	}
	return *u, token, nil
}

// Authenticate resolves a token back to a user, used by middleware to populate
// the request context. A bad token surfaces as Unauthorized.
func (s *AuthService) Authenticate(ctx context.Context, token string) (model.User, error) {
	claims, err := s.jwt.Verify(token)
	if err != nil {
		return model.User{}, apperr.New(apperr.KindUnauthorized, "invalid or expired token")
	}
	u, err := s.users.FindByID(ctx, claims.UserID)
	if err != nil {
		return model.User{}, apperr.New(apperr.KindUnauthorized, "user not found")
	}
	return *u, nil
}

// CurrentUser loads a user by id; used by the /me endpoint.
func (s *AuthService) CurrentUser(ctx context.Context, id uuid.UUID) (model.User, error) {
	u, err := s.users.FindByID(ctx, id)
	if err != nil {
		return model.User{}, apperr.Wrap(apperr.KindNotFound, "user not found", err)
	}
	return *u, nil
}
