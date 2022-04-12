package middlewares

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/golang-jwt/jwt"
)

func AuthMiddleware(
	next func(http.ResponseWriter, *http.Request),
) func(http.ResponseWriter, *http.Request) {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {

		fmt.Println("hello from the auth middleware")
		token := r.Header.Get("Authorization")
		if token == "" {
			// http.Error(rw, "Token missing", 40s1)
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(401)
			json.NewEncoder(rw).Encode(map[string]string{"message": "token missing"})
			return
		}
		fmt.Printf("token: %v\n", token)

		tokenData, err := jwt.Parse(
			token,
			func(t *jwt.Token) (interface{}, error) { return os.Getenv("MAIN_SECRET_TOKEN"), nil },
		)
		fmt.Printf("tokenData: %v\n", tokenData)
		if err != nil {
			if err == jwt.ErrSignatureInvalid {
				rw.WriteHeader(http.StatusUnauthorized)
				return
			}
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		if !tokenData.Valid {
			rw.WriteHeader(http.StatusUnauthorized)
			return
		}

		claims, ok := tokenData.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(rw, "Error token", 400)
		}

		ownerId := claims["id"].(string)
		req := r.WithContext(context.WithValue(r.Context(), "ownerId", ownerId))
		next(rw, req)
	})
}
