package access

import "context"

type clientIPKey struct{}

// WithClientIP — адрес клиента запроса (transport кладёт его для ограничения
// частоты входа и события security.auth.failed, AD-15).
func WithClientIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, clientIPKey{}, ip)
}

// ClientIPFrom — адрес клиента; нет — пусто.
func ClientIPFrom(ctx context.Context) string {
	s, _ := ctx.Value(clientIPKey{}).(string)
	return s
}
