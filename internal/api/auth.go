package api

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

// ===== ЛР-4: аутентификация (JWT) и авторизация, отозванные токены — в Redis (blacklist) =====

const (
	jwtSecret      = "breakeven_point_secret" // в реальном проекте — из переменной окружения
	tokenTTL       = time.Hour
	bearerPrefix   = "Bearer "
	blacklistKey   = "blacklist:"
	redisAddr      = "localhost:6379"
	redisPassword  = "redispassword"
	ctxUserID      = "user_id"
	ctxTokenID     = "token_jti"
	ctxTokenExpire = "token_exp"
)

// JWTClaims — содержимое токена: стандартные поля (iss, iat, exp, jti) + id и роль пользователя.
// jti — короткий уникальный ID токена: по нему токен попадает в blacklist Redis
// (ключ blacklist:<jti> короткий и читается в redis-cli, в отличие от всего JWT).
type JWTClaims struct {
	jwt.RegisteredClaims
	UserID int    `json:"user_id"`
	Role   string `json:"role"`
}

// newJTI — случайный ID токена из 8 байт (16 hex-символов)
func newJTI() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func NewRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{Addr: redisAddr, Password: redisPassword})
}

// Login godoc
// @Summary      Аутентификация пользователя
// @Description  Проверяет логин и пароль (bcrypt) и выдаёт JWT. Доступно всем.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body body LoginRequest true "Логин и пароль"
// @Success      200 {object} LoginResponse
// @Failure      400 "неверный формат запроса"
// @Failure      401 "неверный логин или пароль"
// @Router       /api/users/login [post]
func (a *API) Login(ctx *gin.Context) {
	var body LoginRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}
	u, err := a.Repo.GetUserByUsername(body.Username)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(body.Password)) != nil {
		ctx.Status(http.StatusUnauthorized)
		return
	}
	now := time.Now()
	claims := JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "breakeven_point",
			ID:        newJTI(), // jti
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(tokenTTL)),
		},
		UserID: int(u.ID),
		Role:   u.Role,
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(jwtSecret))
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}
	ctx.JSON(http.StatusOK, LoginResponse{AccessToken: token, TokenType: "Bearer", ExpiresIn: int(tokenTTL.Seconds())})
}

// Logout godoc
// @Summary      Деавторизация пользователя
// @Description  Кладёт jti текущего JWT в blacklist Redis до окончания срока его действия. Доступно создателю.
// @Tags         users
// @Security     BearerAuth
// @Success      200 "токен отозван"
// @Failure      401 "нет токена или токен недействителен"
// @Router       /api/users/logout [post]
func (a *API) Logout(ctx *gin.Context) {
	jti := ctx.GetString(ctxTokenID)
	exp, _ := ctx.Get(ctxTokenExpire)
	ttl := time.Until(exp.(time.Time))
	if ttl > 0 {
		// ключ blacklist:<jti>, значение — id пользователя: в redis-cli видно, чей токен отозван
		if err := a.RDB.Set(ctx.Request.Context(), blacklistKey+jti, currentUserID(ctx), ttl).Err(); err != nil {
			ctx.Status(http.StatusInternalServerError)
			return
		}
	}
	ctx.Status(http.StatusOK)
}

// parseToken — достаёт JWT из заголовка Authorization, проверяет подпись и срок, затем blacklist по jti
func (a *API) parseToken(ctx *gin.Context) (*JWTClaims, bool) {
	header := ctx.GetHeader("Authorization")
	if !strings.HasPrefix(header, bearerPrefix) {
		return nil, false
	}
	raw := strings.TrimPrefix(header, bearerPrefix)
	claims := &JWTClaims{}
	tok, err := jwt.ParseWithClaims(raw, claims, func(*jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || !tok.Valid {
		return nil, false
	}
	if n, _ := a.RDB.Exists(ctx.Request.Context(), blacklistKey+claims.ID).Result(); n > 0 {
		return nil, false // токен отозван через logout
	}
	return claims, true
}

// WithAuth — middleware для методов создателя: без действующего токена 401
func (a *API) WithAuth(ctx *gin.Context) {
	claims, ok := a.parseToken(ctx)
	if !ok {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	ctx.Set(ctxUserID, claims.UserID)
	ctx.Set(ctxTokenID, claims.ID)
	ctx.Set(ctxTokenExpire, claims.ExpiresAt.Time)
	ctx.Next()
}

// WithOptionalAuth — для чтения: гость проходит, а с токеном известен пользователь (is_mine, is_liked)
func (a *API) WithOptionalAuth(ctx *gin.Context) {
	if claims, ok := a.parseToken(ctx); ok {
		ctx.Set(ctxUserID, claims.UserID)
	}
	ctx.Next()
}

// currentUserID заменяет функцию-singleton из ЛР-3: пользователь берётся из JWT, 0 — гость
func currentUserID(ctx *gin.Context) int {
	return ctx.GetInt(ctxUserID)
}
