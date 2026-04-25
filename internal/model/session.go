package model

type Session struct {
	UID         string `json:"uid"`
	Fingerprint string `json:"fingerprint"`
	CreatedAt   int64  `json:"created_at"`
	LastSeen    int64  `json:"last_seen"`
}
