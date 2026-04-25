package model

type RefImg struct {
	RefID       string `json:"ref_id"`
	UID         string `json:"uid"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Data        []byte `json:"data,omitempty"`
	FilePath    string `json:"file_path,omitempty"`
	UploadedAt  int64  `json:"uploaded_at"`
}
