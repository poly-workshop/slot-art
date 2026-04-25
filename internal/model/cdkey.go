package model

type CDKeyStatus string

const (
	CDKeyCreated  CDKeyStatus = "created"
	CDKeyRedeemed CDKeyStatus = "redeemed"
	CDKeyRevoked  CDKeyStatus = "revoked"
	CDKeyExpired  CDKeyStatus = "expired"
)

type CDKey struct {
	CDKeyID       string      `json:"cdkey_id"`
	KeyHash       string      `json:"key_hash"`
	MaskedKey     string      `json:"masked_key"`
	BatchID       string      `json:"batch_id,omitempty"`
	Status        CDKeyStatus `json:"status"`
	ImageCount    int         `json:"image_count"`
	ExpiresAt     int64       `json:"expires_at,omitempty"`
	RedeemedByUID string      `json:"redeemed_by_uid,omitempty"`
	RedeemedAt    int64       `json:"redeemed_at,omitempty"`
	CreatedBy     string      `json:"created_by,omitempty"`
	CreatedAt     int64       `json:"created_at"`
	Note          string      `json:"note,omitempty"`
}
