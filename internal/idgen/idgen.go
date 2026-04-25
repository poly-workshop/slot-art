package idgen

import (
	"crypto/rand"
	"fmt"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const cdkeyCharset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func randFromCharset(n int, chars string) string {
	b := make([]byte, n)
	rand.Read(b)
	for i := range b {
		b[i] = chars[b[i]%byte(len(chars))]
	}
	return string(b)
}

func randString(n int) string { return randFromCharset(n, charset) }

func NewUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func NewTaskID() string           { return randString(16) }
func NewRefID() string            { return "ref_" + randString(8) }
func NewImageID() string          { return "img_" + randString(8) }
func NewCreditID() string         { return "crt_" + randString(12) }
func NewCDKeyID() string          { return "key_" + randString(12) }
func NewPromptTemplateID() string { return "ptpl_" + randString(12) }

func NewCDKeyPlain() string {
	return fmt.Sprintf("SA-%s-%s-%s-%s",
		randFromCharset(4, cdkeyCharset),
		randFromCharset(4, cdkeyCharset),
		randFromCharset(4, cdkeyCharset),
		randFromCharset(4, cdkeyCharset),
	)
}
