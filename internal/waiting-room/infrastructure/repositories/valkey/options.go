package valkey

import "strings"

type options struct {
	namespace string
}

// Valkeyのオプション
type Option func(*options)

// Valkeyに保存するキーへ名前空間を付与
func WithNamespace(namespace string) Option {
	return func(o *options) {
		o.namespace = namespace
	}
}

func applyOptions(opts []Option) options {
	cfg := options{}
	for _, opt := range opts {
		opt(&cfg)
	}
	cfg.namespace = normalizeNamespace(cfg.namespace)
	return cfg
}

func normalizeNamespace(ns string) string {
	ns = strings.TrimSpace(ns)
	if ns == "" {
		return ""
	}
	if strings.HasSuffix(ns, ":") {
		return ns
	}
	return ns + ":"
}
