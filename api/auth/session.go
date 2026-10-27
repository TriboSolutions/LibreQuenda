package auth

import "context"

// Session is the verified identity of the current request.
type Session struct {
	Subject     string // Zitadel "sub"
	AccessToken string
}

type sessionKey struct{}

func WithSession(ctx context.Context, session Session) context.Context {
	return context.WithValue(ctx, sessionKey{}, session)
}

func SessionFromContext(ctx context.Context) (Session, bool) {
	var session, ok = ctx.Value(sessionKey{}).(Session)
	return session, ok
}
