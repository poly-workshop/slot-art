package model

type CreditStatus string

const (
	CreditActive   CreditStatus = "active"
	CreditConsumed CreditStatus = "consumed"
	CreditExpired  CreditStatus = "expired"
)

type Credit struct {
	CreditID            string       `json:"credit_id"`
	UID                 string       `json:"uid"`
	Status              CreditStatus `json:"status"`
	ImageCount          int          `json:"image_count"`
	RemainingImageCount int          `json:"remaining_image_count"`
	Source              string       `json:"source"`
	SourceRef           string       `json:"source_ref"`
	CreatedAt           int64        `json:"created_at"`
	ExpiresAt           int64        `json:"expires_at,omitempty"`
	ConsumedAt          int64        `json:"consumed_at,omitempty"`
}
