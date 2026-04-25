package service

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/poly-workshop/slot-art/gen/go/slot/v1"
	"github.com/poly-workshop/slot-art/internal/idgen"
	"github.com/poly-workshop/slot-art/internal/model"
	"github.com/poly-workshop/slot-art/internal/store"
)

type AdminService struct {
	pb.UnimplementedAdminServiceServer
	deps Deps
}

func NewAdminService(deps Deps) *AdminService {
	return &AdminService{deps: deps}
}

func (s *AdminService) GetDashboard(ctx context.Context, _ *pb.GetDashboardRequest) (*pb.GetDashboardResponse, error) {
	pending, _ := s.deps.Store.CountTasksByStatus(ctx, model.TaskPending)
	processing, _ := s.deps.Store.CountTasksByStatus(ctx, model.TaskProcessing)
	completed, _ := s.deps.Store.CountTasksByStatus(ctx, model.TaskCompleted)
	failed, _ := s.deps.Store.CountTasksByStatus(ctx, model.TaskFailed)
	created, _ := s.deps.Store.CountCDKeysByStatus(ctx, model.CDKeyCreated)
	redeemed, _ := s.deps.Store.CountCDKeysByStatus(ctx, model.CDKeyRedeemed)
	revoked, _ := s.deps.Store.CountCDKeysByStatus(ctx, model.CDKeyRevoked)
	templates, _ := s.deps.Store.ListPromptTemplates(ctx, false, 1000)
	sessions, _ := s.deps.Store.ListSessions(ctx, 1000)
	return &pb.GetDashboardResponse{
		PendingTasks:          int32(pending),
		ProcessingTasks:       int32(processing),
		CompletedTasks:        int32(completed),
		FailedTasks:           int32(failed),
		ActiveSessions:        int32(len(sessions)),
		ActivePromptTemplates: int32(len(templates)),
		CdkeysCreated:         int32(created),
		CdkeysRedeemed:        int32(redeemed),
		CdkeysRevoked:         int32(revoked),
	}, nil
}

func (s *AdminService) GenerateCDKeys(ctx context.Context, req *pb.GenerateCDKeysRequest) (*pb.GenerateCDKeysResponse, error) {
	if req.GetCount() <= 0 || req.GetCount() > 500 {
		return nil, status.Error(codes.InvalidArgument, "count must be between 1 and 500")
	}
	imageCount := int(req.GetImageCount())
	if imageCount <= 0 {
		imageCount = 1
	}
	maxImages := s.deps.Cfg.OpenAIImage2.MaxImages
	if maxImages <= 0 {
		maxImages = 4
	}
	if imageCount > maxImages {
		return nil, status.Errorf(codes.InvalidArgument, "image_count must be <= %d", maxImages)
	}

	batchID := req.GetBatchId()
	if batchID == "" {
		batchID = "batch_" + idgen.NewTaskID()
	}
	now := time.Now().Unix()
	createdBy := "admin"
	items := make([]*model.CDKey, 0, req.GetCount())
	resp := &pb.GenerateCDKeysResponse{Cdkeys: make([]*pb.GeneratedCDKey, 0, req.GetCount())}
	for range req.GetCount() {
		plain := idgen.NewCDKeyPlain()
		cdkey := &model.CDKey{
			CDKeyID:    idgen.NewCDKeyID(),
			KeyHash:    store.HashCDKey(plain),
			MaskedKey:  store.MaskCDKey(plain),
			BatchID:    batchID,
			Status:     model.CDKeyCreated,
			ImageCount: imageCount,
			ExpiresAt:  req.GetExpiresAtUnix(),
			CreatedBy:  createdBy,
			CreatedAt:  now,
			Note:       req.GetNote(),
		}
		items = append(items, cdkey)
		resp.Cdkeys = append(resp.Cdkeys, &pb.GeneratedCDKey{Cdkey: toProtoCDKey(cdkey), PlaintextKey: plain})
	}
	if err := s.deps.Store.CreateCDKeys(ctx, items); err != nil {
		return nil, status.Errorf(codes.Internal, "create cdkeys: %v", err)
	}
	return resp, nil
}

func (s *AdminService) ListCDKeys(ctx context.Context, req *pb.ListCDKeysRequest) (*pb.ListCDKeysResponse, error) {
	items, err := s.deps.Store.ListCDKeys(ctx, fromProtoCDKeyStatus(req.GetStatus()), req.GetBatchId(), pageSize(req.GetPage(), 100))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list cdkeys: %v", err)
	}
	out := make([]*pb.CDKey, 0, len(items))
	for _, item := range items {
		out = append(out, toProtoCDKey(item))
	}
	return &pb.ListCDKeysResponse{Cdkeys: out, Page: pageResponse(len(out))}, nil
}

func (s *AdminService) GetCDKey(ctx context.Context, req *pb.GetCDKeyRequest) (*pb.GetCDKeyResponse, error) {
	cdkey, err := s.deps.Store.GetCDKey(ctx, req.GetCdkeyId())
	if err != nil {
		return nil, mapStoreError(err)
	}
	return &pb.GetCDKeyResponse{Cdkey: toProtoCDKey(cdkey)}, nil
}

func (s *AdminService) RevokeCDKey(ctx context.Context, req *pb.RevokeCDKeyRequest) (*pb.RevokeCDKeyResponse, error) {
	cdkey, err := s.deps.Store.RevokeCDKey(ctx, req.GetCdkeyId())
	if err != nil {
		return nil, mapStoreError(err)
	}
	return &pb.RevokeCDKeyResponse{Cdkey: toProtoCDKey(cdkey)}, nil
}

func (s *AdminService) ListTasks(ctx context.Context, req *pb.ListTasksRequest) (*pb.ListTasksResponse, error) {
	items, err := s.deps.Store.ListAllTasks(ctx, req.GetUid(), fromProtoTaskStatus(req.GetStatus()), pageSize(req.GetPage(), 100))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list tasks: %v", err)
	}
	out := make([]*pb.AdminTask, 0, len(items))
	for _, item := range items {
		out = append(out, toProtoAdminTask(item))
	}
	return &pb.ListTasksResponse{Tasks: out, Page: pageResponse(len(out))}, nil
}

func (s *AdminService) GetAdminTask(ctx context.Context, req *pb.GetAdminTaskRequest) (*pb.GetAdminTaskResponse, error) {
	t, err := s.deps.Store.GetTask(ctx, req.GetTaskId())
	if err != nil {
		return nil, mapStoreError(err)
	}
	return &pb.GetAdminTaskResponse{Task: toProtoAdminTask(t)}, nil
}

func (s *AdminService) ListSessions(ctx context.Context, req *pb.ListSessionsRequest) (*pb.ListSessionsResponse, error) {
	items, err := s.deps.Store.ListSessions(ctx, pageSize(req.GetPage(), 100))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list sessions: %v", err)
	}
	out := make([]*pb.Session, 0, len(items))
	for _, item := range items {
		out = append(out, toProtoSession(item))
	}
	return &pb.ListSessionsResponse{Sessions: out, Page: pageResponse(len(out))}, nil
}

func (s *AdminService) ListCredits(ctx context.Context, req *pb.ListCreditsRequest) (*pb.ListCreditsResponse, error) {
	items, err := s.deps.Store.ListAllCredits(ctx, req.GetUid(), fromProtoCreditStatus(req.GetStatus()), pageSize(req.GetPage(), 100))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list credits: %v", err)
	}
	out := make([]*pb.Credit, 0, len(items))
	for _, item := range items {
		out = append(out, toProtoCredit(item))
	}
	return &pb.ListCreditsResponse{Credits: out, Page: pageResponse(len(out))}, nil
}

func (s *AdminService) ListPromptTemplates(ctx context.Context, req *pb.ListPromptTemplatesRequest) (*pb.ListPromptTemplatesResponse, error) {
	items, err := s.deps.Store.ListPromptTemplates(ctx, req.GetIncludeDisabled(), pageSize(req.GetPage(), 100))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list prompt templates: %v", err)
	}
	out := make([]*pb.AdminPromptTemplate, 0, len(items))
	for _, item := range items {
		out = append(out, toProtoAdminPromptTemplate(item))
	}
	return &pb.ListPromptTemplatesResponse{Templates: out, Page: pageResponse(len(out))}, nil
}

func (s *AdminService) GetPromptTemplate(ctx context.Context, req *pb.GetPromptTemplateRequest) (*pb.GetPromptTemplateResponse, error) {
	tmpl, err := s.deps.Store.GetPromptTemplate(ctx, req.GetTemplateId())
	if err != nil {
		return nil, mapStoreError(err)
	}
	return &pb.GetPromptTemplateResponse{Template: toProtoAdminPromptTemplate(tmpl)}, nil
}

func (s *AdminService) CreatePromptTemplate(ctx context.Context, req *pb.CreatePromptTemplateRequest) (*pb.CreatePromptTemplateResponse, error) {
	tmpl := fromProtoAdminPromptTemplate(req.GetTemplate())
	if tmpl == nil {
		return nil, status.Error(codes.InvalidArgument, "template is required")
	}
	if tmpl.TemplateID == "" {
		tmpl.TemplateID = idgen.NewPromptTemplateID()
	}
	if err := validatePromptTemplate(tmpl); err != nil {
		return nil, err
	}
	if err := s.deps.Store.CreatePromptTemplate(ctx, tmpl); err != nil {
		return nil, status.Errorf(codes.Internal, "create prompt template: %v", err)
	}
	created, err := s.deps.Store.GetPromptTemplate(ctx, tmpl.TemplateID)
	if err != nil {
		return nil, mapStoreError(err)
	}
	return &pb.CreatePromptTemplateResponse{Template: toProtoAdminPromptTemplate(created)}, nil
}

func (s *AdminService) UpdatePromptTemplate(ctx context.Context, req *pb.UpdatePromptTemplateRequest) (*pb.UpdatePromptTemplateResponse, error) {
	tmpl := fromProtoAdminPromptTemplate(req.GetTemplate())
	if tmpl == nil || tmpl.TemplateID == "" {
		return nil, status.Error(codes.InvalidArgument, "template_id is required")
	}
	if err := validatePromptTemplate(tmpl); err != nil {
		return nil, err
	}
	if err := s.deps.Store.UpdatePromptTemplate(ctx, tmpl); err != nil {
		return nil, mapStoreError(err)
	}
	updated, err := s.deps.Store.GetPromptTemplate(ctx, tmpl.TemplateID)
	if err != nil {
		return nil, mapStoreError(err)
	}
	return &pb.UpdatePromptTemplateResponse{Template: toProtoAdminPromptTemplate(updated)}, nil
}

func (s *AdminService) SetPromptTemplateEnabled(ctx context.Context, req *pb.SetPromptTemplateEnabledRequest) (*pb.SetPromptTemplateEnabledResponse, error) {
	tmpl, err := s.deps.Store.SetPromptTemplateEnabled(ctx, req.GetTemplateId(), req.GetEnabled())
	if err != nil {
		return nil, mapStoreError(err)
	}
	return &pb.SetPromptTemplateEnabledResponse{Template: toProtoAdminPromptTemplate(tmpl)}, nil
}

func (s *AdminService) DeletePromptTemplate(ctx context.Context, req *pb.DeletePromptTemplateRequest) (*pb.DeletePromptTemplateResponse, error) {
	if req.GetTemplateId() == "" {
		return nil, status.Error(codes.InvalidArgument, "template_id is required")
	}
	if err := s.deps.Store.DeletePromptTemplate(ctx, req.GetTemplateId()); err != nil {
		return nil, status.Errorf(codes.Internal, "delete prompt template: %v", err)
	}
	return &pb.DeletePromptTemplateResponse{}, nil
}
