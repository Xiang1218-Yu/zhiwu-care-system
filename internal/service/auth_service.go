package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"plant-diary/internal/model"
	"plant-diary/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("邮箱或密码错误")
	ErrEmailExists        = errors.New("邮箱已注册")
)

type AuthService struct {
	users  *repository.UserRepository
	secret []byte
	expire time.Duration
}

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func NewAuthService(users *repository.UserRepository, secret string, expireHours int) *AuthService {
	return &AuthService{users: users, secret: []byte(secret), expire: time.Duration(expireHours) * time.Hour}
}

func (s *AuthService) Register(name, email, password string) (*model.User, string, error) {
	name, email = strings.TrimSpace(name), strings.ToLower(strings.TrimSpace(email))
	if name == "" || email == "" || len(password) < 6 {
		return nil, "", errors.New("姓名、邮箱不能为空，密码至少 6 位")
	}
	existing, err := s.users.FindByEmail(email)
	if err != nil {
		return nil, "", err
	}
	if existing != nil {
		return nil, "", ErrEmailExists
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", fmt.Errorf("hash password: %w", err)
	}
	user := &model.User{ID: uuid.NewString(), Name: name, Email: email, PasswordHash: string(hash)}
	if err := s.users.Create(user); err != nil {
		return nil, "", err
	}
	token, err := s.issue(user.ID)
	return user, token, err
}

func (s *AuthService) Login(email, password string) (*model.User, string, error) {
	user, err := s.users.FindByEmail(strings.ToLower(strings.TrimSpace(email)))
	if err != nil || user == nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return nil, "", ErrInvalidCredentials
	}
	token, err := s.issue(user.ID)
	return user, token, err
}

func (s *AuthService) ParseToken(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil {
		return "", err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid || claims.UserID == "" {
		return "", errors.New("invalid token")
	}
	return claims.UserID, nil
}

func (s *AuthService) issue(userID string) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: "plant-diary", Subject: userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.expire)),
		},
	})
	return token.SignedString(s.secret)
}
