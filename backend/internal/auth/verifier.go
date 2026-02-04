package auth

import "context"

type Claims struct {
	Subject string
	Email   string
}

type Verifier interface {
	Verify(ctx context.Context, token string) (Claims, error)
}
