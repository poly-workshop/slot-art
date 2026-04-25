package service

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/poly-workshop/slot-art/gen/go/slot/v1"
	"github.com/poly-workshop/slot-art/internal/config"
	"github.com/poly-workshop/slot-art/internal/model"
	"github.com/poly-workshop/slot-art/internal/store"
	taskqueue "github.com/poly-workshop/slot-art/internal/task"
)

const openAIImage2ProviderName = "openai_image2"

var placeholderRE = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_]+)\s*\}\}`)

type Deps struct {
	Cfg      *config.Config
	Store    *store.Store
	Enqueuer *taskqueue.Enqueuer
	Log      *slog.Logger
}

func uidFromContext(ctx context.Context) (string, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", false
	}
	for _, key := range []string{"x-slot-uid", "x-slot-user", "uid"} {
		vals := md.Get(key)
		if len(vals) > 0 && vals[0] != "" {
			return vals[0], true
		}
	}
	return "", false
}

func requireUID(ctx context.Context) (string, error) {
	uid, ok := uidFromContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing X-Slot-UID metadata")
	}
	return uid, nil
}

func authorized(ctx context.Context, token string) bool {
	if token == "" {
		return false
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return false
	}
	for _, v := range md.Get("authorization") {
		if strings.TrimSpace(v) == "Bearer "+token || strings.TrimSpace(v) == token {
			return true
		}
	}
	for _, v := range md.Get("x-admin-token") {
		if strings.TrimSpace(v) == token {
			return true
		}
	}
	return false
}

func AdminUnaryInterceptor(token string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if strings.HasPrefix(info.FullMethod, "/slot.v1.AdminService/") && !authorized(ctx, token) {
			return nil, status.Error(codes.Unauthenticated, "admin authorization required")
		}
		return handler(ctx, req)
	}
}

func AdminHTTPMiddleware(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token == "" || !httpAuthorized(r, token) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"admin authorization required"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func httpAuthorized(r *http.Request, token string) bool {
	if strings.TrimSpace(r.Header.Get("X-Admin-Token")) == token {
		return true
	}
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	return auth == "Bearer "+token || auth == token
}

func ts(unix int64) *timestamppb.Timestamp {
	if unix <= 0 {
		return nil
	}
	return timestamppb.New(time.Unix(unix, 0))
}

func pageSize(req *pb.PageRequest, fallback int) int {
	if req != nil && req.GetPageSize() > 0 {
		return int(req.GetPageSize())
	}
	return fallback
}

func pageResponse(total int) *pb.PageResponse {
	return &pb.PageResponse{TotalSize: int32(total)}
}

func toProtoCredit(c *model.Credit) *pb.Credit {
	if c == nil {
		return nil
	}
	return &pb.Credit{
		CreditId:            c.CreditID,
		Uid:                 c.UID,
		Status:              toProtoCreditStatus(c.Status),
		ImageCount:          int32(c.ImageCount),
		RemainingImageCount: int32(c.RemainingImageCount),
		Source:              c.Source,
		SourceRef:           c.SourceRef,
		CreatedAt:           ts(c.CreatedAt),
		ExpiresAt:           ts(c.ExpiresAt),
		ConsumedAt:          ts(c.ConsumedAt),
	}
}

func toProtoCreditStatus(status model.CreditStatus) pb.CreditStatus {
	switch status {
	case model.CreditActive:
		return pb.CreditStatus_CREDIT_STATUS_ACTIVE
	case model.CreditConsumed:
		return pb.CreditStatus_CREDIT_STATUS_CONSUMED
	case model.CreditExpired:
		return pb.CreditStatus_CREDIT_STATUS_EXPIRED
	default:
		return pb.CreditStatus_CREDIT_STATUS_UNSPECIFIED
	}
}

func fromProtoCreditStatus(status pb.CreditStatus) model.CreditStatus {
	switch status {
	case pb.CreditStatus_CREDIT_STATUS_ACTIVE:
		return model.CreditActive
	case pb.CreditStatus_CREDIT_STATUS_CONSUMED:
		return model.CreditConsumed
	case pb.CreditStatus_CREDIT_STATUS_EXPIRED:
		return model.CreditExpired
	default:
		return ""
	}
}

func toProtoCDKey(c *model.CDKey) *pb.CDKey {
	if c == nil {
		return nil
	}
	return &pb.CDKey{
		CdkeyId:       c.CDKeyID,
		MaskedKey:     c.MaskedKey,
		KeyHash:       c.KeyHash,
		BatchId:       c.BatchID,
		Status:        toProtoCDKeyStatus(c.Status),
		ImageCount:    int32(c.ImageCount),
		ExpiresAt:     ts(c.ExpiresAt),
		RedeemedByUid: c.RedeemedByUID,
		RedeemedAt:    ts(c.RedeemedAt),
		CreatedBy:     c.CreatedBy,
		CreatedAt:     ts(c.CreatedAt),
		Note:          c.Note,
	}
}

func toProtoCDKeyStatus(status model.CDKeyStatus) pb.CDKeyStatus {
	switch status {
	case model.CDKeyCreated:
		return pb.CDKeyStatus_CD_KEY_STATUS_CREATED
	case model.CDKeyRedeemed:
		return pb.CDKeyStatus_CD_KEY_STATUS_REDEEMED
	case model.CDKeyRevoked:
		return pb.CDKeyStatus_CD_KEY_STATUS_REVOKED
	case model.CDKeyExpired:
		return pb.CDKeyStatus_CD_KEY_STATUS_EXPIRED
	default:
		return pb.CDKeyStatus_CD_KEY_STATUS_UNSPECIFIED
	}
}

func fromProtoCDKeyStatus(status pb.CDKeyStatus) model.CDKeyStatus {
	switch status {
	case pb.CDKeyStatus_CD_KEY_STATUS_CREATED:
		return model.CDKeyCreated
	case pb.CDKeyStatus_CD_KEY_STATUS_REDEEMED:
		return model.CDKeyRedeemed
	case pb.CDKeyStatus_CD_KEY_STATUS_REVOKED:
		return model.CDKeyRevoked
	case pb.CDKeyStatus_CD_KEY_STATUS_EXPIRED:
		return model.CDKeyExpired
	default:
		return ""
	}
}

func toProtoTaskStatus(status model.TaskStatus) pb.TaskStatus {
	switch status {
	case model.TaskPending:
		return pb.TaskStatus_TASK_STATUS_PENDING
	case model.TaskProcessing:
		return pb.TaskStatus_TASK_STATUS_PROCESSING
	case model.TaskCompleted:
		return pb.TaskStatus_TASK_STATUS_COMPLETED
	case model.TaskFailed:
		return pb.TaskStatus_TASK_STATUS_FAILED
	default:
		return pb.TaskStatus_TASK_STATUS_UNSPECIFIED
	}
}

func fromProtoTaskStatus(status pb.TaskStatus) model.TaskStatus {
	switch status {
	case pb.TaskStatus_TASK_STATUS_PENDING:
		return model.TaskPending
	case pb.TaskStatus_TASK_STATUS_PROCESSING:
		return model.TaskProcessing
	case pb.TaskStatus_TASK_STATUS_COMPLETED:
		return model.TaskCompleted
	case pb.TaskStatus_TASK_STATUS_FAILED:
		return model.TaskFailed
	default:
		return ""
	}
}

func toProtoFieldType(t model.PromptTemplateFieldType) pb.PromptTemplateFieldType {
	switch t {
	case model.PromptTemplateFieldText:
		return pb.PromptTemplateFieldType_PROMPT_TEMPLATE_FIELD_TYPE_TEXT
	case model.PromptTemplateFieldTextarea:
		return pb.PromptTemplateFieldType_PROMPT_TEMPLATE_FIELD_TYPE_TEXTAREA
	case model.PromptTemplateFieldSelect:
		return pb.PromptTemplateFieldType_PROMPT_TEMPLATE_FIELD_TYPE_SELECT
	case model.PromptTemplateFieldNumber:
		return pb.PromptTemplateFieldType_PROMPT_TEMPLATE_FIELD_TYPE_NUMBER
	default:
		return pb.PromptTemplateFieldType_PROMPT_TEMPLATE_FIELD_TYPE_UNSPECIFIED
	}
}

func fromProtoFieldType(t pb.PromptTemplateFieldType) model.PromptTemplateFieldType {
	switch t {
	case pb.PromptTemplateFieldType_PROMPT_TEMPLATE_FIELD_TYPE_TEXT:
		return model.PromptTemplateFieldText
	case pb.PromptTemplateFieldType_PROMPT_TEMPLATE_FIELD_TYPE_TEXTAREA:
		return model.PromptTemplateFieldTextarea
	case pb.PromptTemplateFieldType_PROMPT_TEMPLATE_FIELD_TYPE_SELECT:
		return model.PromptTemplateFieldSelect
	case pb.PromptTemplateFieldType_PROMPT_TEMPLATE_FIELD_TYPE_NUMBER:
		return model.PromptTemplateFieldNumber
	default:
		return model.PromptTemplateFieldText
	}
}

func toProtoField(f model.PromptTemplateField) *pb.PromptTemplateField {
	return &pb.PromptTemplateField{
		Name:        f.Name,
		Label:       f.Label,
		Type:        toProtoFieldType(f.Type),
		Required:    f.Required,
		MaxLength:   int32(f.MaxLength),
		Options:     append([]string(nil), f.Options...),
		Placeholder: f.Placeholder,
		HelpText:    f.HelpText,
	}
}

func fromProtoField(f *pb.PromptTemplateField) model.PromptTemplateField {
	if f == nil {
		return model.PromptTemplateField{}
	}
	return model.PromptTemplateField{
		Name:        f.GetName(),
		Label:       f.GetLabel(),
		Type:        fromProtoFieldType(f.GetType()),
		Required:    f.GetRequired(),
		MaxLength:   int(f.GetMaxLength()),
		Options:     append([]string(nil), f.GetOptions()...),
		Placeholder: f.GetPlaceholder(),
		HelpText:    f.GetHelpText(),
	}
}

func toProtoPromptTemplateSummary(t *model.PromptTemplate) *pb.PromptTemplateSummary {
	if t == nil {
		return nil
	}
	fields := make([]*pb.PromptTemplateField, 0, len(t.Fields))
	for _, field := range t.Fields {
		fields = append(fields, toProtoField(field))
	}
	return &pb.PromptTemplateSummary{
		TemplateId:  t.TemplateID,
		Name:        t.Name,
		Description: t.Description,
		Category:    t.Category,
		Version:     int32(t.Version),
		Fields:      fields,
	}
}

func toProtoAdminPromptTemplate(t *model.PromptTemplate) *pb.AdminPromptTemplate {
	if t == nil {
		return nil
	}
	fields := make([]*pb.PromptTemplateField, 0, len(t.Fields))
	for _, field := range t.Fields {
		fields = append(fields, toProtoField(field))
	}
	return &pb.AdminPromptTemplate{
		TemplateId:   t.TemplateID,
		Name:         t.Name,
		Description:  t.Description,
		Category:     t.Category,
		Version:      int32(t.Version),
		Enabled:      t.Enabled,
		TemplateBody: t.TemplateBody,
		Fields:       fields,
		Source:       t.Source,
		CreatedAt:    ts(t.CreatedAt),
		UpdatedAt:    ts(t.UpdatedAt),
	}
}

func fromProtoAdminPromptTemplate(t *pb.AdminPromptTemplate) *model.PromptTemplate {
	if t == nil {
		return nil
	}
	fields := make([]model.PromptTemplateField, 0, len(t.GetFields()))
	for _, field := range t.GetFields() {
		fields = append(fields, fromProtoField(field))
	}
	return &model.PromptTemplate{
		TemplateID:   t.GetTemplateId(),
		Name:         t.GetName(),
		Description:  t.GetDescription(),
		Category:     t.GetCategory(),
		Version:      int(t.GetVersion()),
		Enabled:      t.GetEnabled(),
		TemplateBody: t.GetTemplateBody(),
		Fields:       fields,
		Source:       t.GetSource(),
	}
}

func toProtoPublicTask(t *model.Task, result *model.Result) *pb.Task {
	if t == nil {
		return nil
	}
	out := &pb.Task{
		TaskId:              t.TaskID,
		Status:              toProtoTaskStatus(t.Status),
		TemplateId:          t.TemplateID,
		TemplateName:        t.TemplateName,
		ImageCountRequested: int32(t.ImageCountRequested),
		ImageCountGenerated: int32(t.ImageCountGenerated),
		ErrorMessage:        t.ErrorMessage,
		CreatedAt:           ts(t.CreatedAt),
		UpdatedAt:           ts(t.UpdatedAt),
		CompletedAt:         ts(t.CompletedAt),
	}
	if result != nil {
		images := make([]*pb.ResultImage, 0, len(result.Images))
		for _, img := range result.Images {
			images = append(images, &pb.ResultImage{
				ImageId:     img.ImageID,
				DownloadUrl: img.URL,
				Width:       int32(img.Width),
				Height:      int32(img.Height),
				Format:      img.Format,
			})
		}
		out.Result = &pb.TaskResult{Images: images, GeneratedAt: ts(result.GeneratedAt)}
	}
	return out
}

func toProtoAdminTask(t *model.Task) *pb.AdminTask {
	if t == nil {
		return nil
	}
	return &pb.AdminTask{
		TaskId:              t.TaskID,
		Uid:                 t.UID,
		Status:              toProtoTaskStatus(t.Status),
		TemplateId:          t.TemplateID,
		TemplateVersion:     int32(t.TemplateVersion),
		SlotValues:          cloneStringMap(t.SlotValues),
		RenderedPrompt:      t.RenderedPrompt,
		ImageCountRequested: int32(t.ImageCountRequested),
		ImageCountGenerated: int32(t.ImageCountGenerated),
		ErrorMessage:        t.ErrorMessage,
		CreatedAt:           ts(t.CreatedAt),
		UpdatedAt:           ts(t.UpdatedAt),
		CompletedAt:         ts(t.CompletedAt),
	}
}

func toProtoSession(s *model.Session) *pb.Session {
	if s == nil {
		return nil
	}
	return &pb.Session{
		Uid:         s.UID,
		Fingerprint: s.Fingerprint,
		CreatedAt:   ts(s.CreatedAt),
		LastSeen:    ts(s.LastSeen),
	}
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func renderPrompt(t *model.PromptTemplate, values map[string]string) (string, error) {
	if t == nil {
		return "", status.Error(codes.NotFound, "prompt template not found")
	}
	if !t.Enabled {
		return "", status.Error(codes.FailedPrecondition, "prompt template is disabled")
	}
	fieldByName := make(map[string]model.PromptTemplateField, len(t.Fields))
	for _, field := range t.Fields {
		if strings.TrimSpace(field.Name) == "" {
			return "", status.Error(codes.InvalidArgument, "template field name is required")
		}
		fieldByName[field.Name] = field
		value := strings.TrimSpace(values[field.Name])
		if field.Required && value == "" {
			return "", status.Errorf(codes.InvalidArgument, "field %s is required", field.Name)
		}
		if field.MaxLength > 0 && len([]rune(value)) > field.MaxLength {
			return "", status.Errorf(codes.InvalidArgument, "field %s exceeds max length", field.Name)
		}
		if value != "" && field.Type == model.PromptTemplateFieldSelect && len(field.Options) > 0 && !contains(field.Options, value) {
			return "", status.Errorf(codes.InvalidArgument, "field %s has invalid option", field.Name)
		}
		if value != "" && field.Type == model.PromptTemplateFieldNumber {
			if _, err := strconv.ParseFloat(value, 64); err != nil {
				return "", status.Errorf(codes.InvalidArgument, "field %s must be a number", field.Name)
			}
		}
	}
	var renderErr error
	rendered := placeholderRE.ReplaceAllStringFunc(t.TemplateBody, func(match string) string {
		parts := placeholderRE.FindStringSubmatch(match)
		if len(parts) != 2 {
			return ""
		}
		name := parts[1]
		if _, ok := fieldByName[name]; !ok {
			renderErr = status.Errorf(codes.InvalidArgument, "template references unknown field %s", name)
			return ""
		}
		return values[name]
	})
	if renderErr != nil {
		return "", renderErr
	}
	return strings.TrimSpace(rendered), nil
}

func contains(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}

func mapStoreError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, store.ErrNotFound):
		return status.Error(codes.NotFound, "resource not found")
	case errors.Is(err, store.ErrCDKeyRedeemed):
		return status.Error(codes.FailedPrecondition, "cdkey already redeemed")
	case errors.Is(err, store.ErrCDKeyRevoked):
		return status.Error(codes.FailedPrecondition, "cdkey revoked")
	case errors.Is(err, store.ErrCDKeyExpired):
		return status.Error(codes.FailedPrecondition, "cdkey expired")
	case errors.Is(err, store.ErrCreditAlreadyConsumed):
		return status.Error(codes.FailedPrecondition, "credit already consumed")
	default:
		return status.Errorf(codes.Internal, "%v", err)
	}
}

func validatePromptTemplate(t *model.PromptTemplate) error {
	if t == nil {
		return status.Error(codes.InvalidArgument, "template is required")
	}
	if strings.TrimSpace(t.Name) == "" {
		return status.Error(codes.InvalidArgument, "template name is required")
	}
	if strings.TrimSpace(t.TemplateBody) == "" {
		return status.Error(codes.InvalidArgument, "template body is required")
	}
	seen := make(map[string]struct{}, len(t.Fields))
	for _, field := range t.Fields {
		if strings.TrimSpace(field.Name) == "" {
			return status.Error(codes.InvalidArgument, "field name is required")
		}
		if _, ok := seen[field.Name]; ok {
			return status.Errorf(codes.InvalidArgument, "duplicate field %s", field.Name)
		}
		seen[field.Name] = struct{}{}
	}
	return nil
}
