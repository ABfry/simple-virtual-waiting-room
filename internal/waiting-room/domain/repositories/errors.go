package repositories

import "errors"

// 共通エラー
var (
	ErrNotFound = errors.New("repository: not found")
)
