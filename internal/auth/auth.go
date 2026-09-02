package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"context"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

func GenerateToken(userID string) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Println("JWT_SECRET is not set")

		return "", errors.New("JWT_SECRET is not set")
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userID": userID,
	})
	return token.SignedString([]byte(secret))
}

func GenerateAdminToken() (string, error) {
	secret := os.Getenv("ADMIN_JWT_SECRET")
	if secret == "" {
		return "", errors.New("ADMIN_JWT_SECRET is not set")
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"role": "admin",
	})
	return token.SignedString([]byte(secret))
}

func AdminAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authz := r.Header.Get("Authorization")
		if !strings.HasPrefix(authz, "Bearer ") {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		parsed, err := jwt.Parse(strings.TrimPrefix(authz, "Bearer "), func(t *jwt.Token) (interface{}, error) {
			return []byte(os.Getenv("ADMIN_JWT_SECRET")), nil
		})
		if err != nil || parsed == nil || !parsed.Valid {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		claims, ok := parsed.Claims.(jwt.MapClaims)
		if !ok || claims["role"] != "admin" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func SeedSampleKey(ctx context.Context, rdb *redis.Client) string {
	raw := os.Getenv("SAMPLE_API_KEY")
	if raw == "" {
		raw = "demo-key"
	}
	sum := sha256.Sum256([]byte(raw))
	redisKey := "apikey:" + hex.EncodeToString(sum[:])
	if err := rdb.Set(ctx, redisKey, "1", 0).Err(); err != nil {
		log.Fatal(err)
	}
	log.Printf("sample API key: %s  (header X-API-Key)", raw)
	return raw
}

func HashKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func New(rdb *redis.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authz := r.Header.Get("Authorization")
			if strings.HasPrefix(authz, "Bearer ") {
				token := strings.TrimPrefix(authz, "Bearer ")
				parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
					if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
						return nil, errors.New("invalid token")
					}
					return []byte(os.Getenv("JWT_SECRET")), nil
				},
				)
				if err != nil {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				if parsed == nil || !parsed.Valid {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				if _, ok := parsed.Claims.(jwt.MapClaims); !ok {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}

				next.ServeHTTP(w, r)
				return
			}
			raw := r.Header.Get("X-API-Key")
			if raw == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			sum := sha256.Sum256([]byte(raw))
			hexKey := hex.EncodeToString(sum[:])
			n, err := rdb.Exists(r.Context(), "apikey:"+hexKey).Result()
			if err != nil || n == 0 {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
