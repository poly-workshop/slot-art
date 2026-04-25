package service

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/poly-workshop/slot-art/gen/go/slot/v1"
	"github.com/poly-workshop/slot-art/internal/idgen"
	"github.com/poly-workshop/slot-art/internal/model"
	"github.com/poly-workshop/slot-art/internal/store"
	taskqueue "github.com/poly-workshop/slot-art/internal/task"
)

type SlotArtService struct {
	pb.UnimplementedSlotArtServiceServer
	deps Deps
}

func NewSlotArtService(deps Deps) *SlotArtService {
	return &SlotArtService{deps: deps}
}

func (s *SlotArtService) InitSession(ctx context.Context, req *pb.InitSessionRequest) (*pb.InitSessionResponse, error) {
	if uid, ok := uidFromContext(ctx); ok {
		if err := s.deps.Store.TouchSession(ctx, uid, s.deps.Cfg.SessionTTL); err == nil {
			return &pb.InitSessionResponse{Uid: uid, FingerprintValid: true}, nil
		}
	}

	uid := idgen.NewUID()
	sess, err := s.deps.Store.CreateSession(ctx, uid, req.GetFingerprint(), s.deps.Cfg.SessionTTL)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create session: %v", err)
	}
	return &pb.InitSessionResponse{Uid: sess.UID, FingerprintValid: true}, nil
}

func (s *SlotArtService) GetBootstrap(ctx context.Context, _ *pb.GetBootstrapRequest) (*pb.GetBootstrapResponse, error) {
	uid, err := requireUID(ctx)
	if err != nil {
		return nil, err
	}

	credits, err := s.deps.Store.ListCredits(ctx, uid, 10)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list credits: %v", err)
	}
	var active *pb.Credit
	for _, credit := range credits {
		if credit.Status == model.CreditActive && credit.RemainingImageCount > 0 {
			active = toProtoCredit(credit)
			break
		}
	}

	templates, err := s.deps.Store.ListPromptTemplates(ctx, false, 100)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list prompt templates: %v", err)
	}
	summaries := make([]*pb.PromptTemplateSummary, 0, len(templates))
	for _, tmpl := range templates {
		summaries = append(summaries, toProtoPromptTemplateSummary(tmpl))
	}

	maxRefs := int32(0)
	if s.deps.Cfg.OpenAIImage2.SupportsRefs {
		maxRefs = 4
	}
	return &pb.GetBootstrapResponse{
		ActiveCredit:           active,
		PromptTemplates:        summaries,
		MaxReferenceImages:     maxRefs,
		MaxReferenceImageBytes: 10 << 20,
	}, nil
}

func (s *SlotArtService) RedeemCDKey(ctx context.Context, req *pb.RedeemCDKeyRequest) (*pb.RedeemCDKeyResponse, error) {
	uid, err := requireUID(ctx)
	if err != nil {
		return nil, err
	}
	if store.NormalizeCDKey(req.GetKey()) == "" {
		return nil, status.Error(codes.InvalidArgument, "cdkey is required")
	}

	now := time.Now().Unix()
	credit := &model.Credit{
		CreditID:  idgen.NewCreditID(),
		UID:       uid,
		Status:    model.CreditActive,
		Source:    "cdkey",
		CreatedAt: now,
		ExpiresAt: now + int64(s.deps.Cfg.CreditTTL.Seconds()),
	}
	cdkey, err := s.deps.Store.RedeemCDKey(ctx, req.GetKey(), uid, credit, s.deps.Cfg.CreditTTL)
	if err != nil {
		return nil, mapStoreError(err)
	}
	credit.ImageCount = cdkey.ImageCount
	credit.RemainingImageCount = cdkey.ImageCount
	credit.SourceRef = cdkey.CDKeyID
	return &pb.RedeemCDKeyResponse{Credit: toProtoCredit(credit)}, nil
}

func (s *SlotArtService) UploadReferenceImage(ctx context.Context, req *pb.UploadReferenceImageRequest) (*pb.UploadReferenceImageResponse, error) {
	uid, err := requireUID(ctx)
	if err != nil {
		return nil, err
	}
	if !s.deps.Cfg.OpenAIImage2.SupportsRefs {
		return nil, status.Error(codes.FailedPrecondition, "reference images are disabled for openai_image2")
	}
	if len(req.GetData()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "image data is required")
	}
	if len(req.GetData()) > 10<<20 {
		return nil, status.Error(codes.InvalidArgument, "image too large")
	}
	contentType := req.GetContentType()
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	now := time.Now().Unix()
	img := &model.RefImg{
		RefID:       idgen.NewRefID(),
		UID:         uid,
		Filename:    req.GetFilename(),
		ContentType: contentType,
		SizeBytes:   int64(len(req.GetData())),
		Data:        req.GetData(),
		UploadedAt:  now,
	}
	if err := s.deps.Store.CreateRefImg(ctx, img, s.deps.Cfg.RefImgTTL); err != nil {
		return nil, status.Errorf(codes.Internal, "store reference image: %v", err)
	}
	return &pb.UploadReferenceImageResponse{ReferenceImage: &pb.ReferenceImage{
		ReferenceImageId: img.RefID,
		Filename:         img.Filename,
		ContentType:      img.ContentType,
		SizeBytes:        img.SizeBytes,
		ExpiresAt:        ts(now + int64(s.deps.Cfg.RefImgTTL.Seconds())),
	}}, nil
}

func (s *SlotArtService) CreateTask(ctx context.Context, req *pb.CreateTaskRequest) (*pb.CreateTaskResponse, error) {
	uid, err := requireUID(ctx)
	if err != nil {
		return nil, err
	}
	if req.GetCreditId() == "" {
		return nil, status.Error(codes.InvalidArgument, "credit_id is required")
	}
	if req.GetTemplateId() == "" {
		return nil, status.Error(codes.InvalidArgument, "template_id is required")
	}
	if len(req.GetReferenceImageIds()) > 0 && !s.deps.Cfg.OpenAIImage2.SupportsRefs {
		return nil, status.Error(codes.FailedPrecondition, "reference images are disabled for openai_image2")
	}

	credit, err := s.deps.Store.GetCredit(ctx, req.GetCreditId())
	if err != nil {
		return nil, mapStoreError(err)
	}
	if credit.UID != uid {
		return nil, status.Error(codes.PermissionDenied, "credit does not belong to session")
	}
	if credit.Status != model.CreditActive || credit.RemainingImageCount <= 0 {
		return nil, status.Error(codes.FailedPrecondition, "credit is not active")
	}

	tmpl, err := s.deps.Store.GetPromptTemplate(ctx, req.GetTemplateId())
	if err != nil {
		return nil, mapStoreError(err)
	}
	renderedPrompt, err := renderPrompt(tmpl, req.GetSlotValues())
	if err != nil {
		return nil, err
	}

	var refImgData []taskqueue.RefImgData
	var refKeys []string
	for _, refID := range req.GetReferenceImageIds() {
		img, err := s.deps.Store.GetRefImg(ctx, uid, refID)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "reference image %s not found", refID)
		}
		refImgData = append(refImgData, taskqueue.RefImgData{ContentType: img.ContentType, Data: img.Data})
		refKeys = append(refKeys, refID)
	}

	if err := s.deps.Store.ConsumeCreditIfActive(ctx, credit.CreditID); err != nil {
		return nil, mapStoreError(err)
	}

	taskID := idgen.NewTaskID()
	now := time.Now().Unix()
	modelTask := &model.Task{
		TaskID:              taskID,
		UID:                 uid,
		CreditID:            credit.CreditID,
		Status:              model.TaskPending,
		Prompt:              renderedPrompt,
		TemplateID:          tmpl.TemplateID,
		TemplateName:        tmpl.Name,
		TemplateVersion:     tmpl.Version,
		SlotValues:          cloneStringMap(req.GetSlotValues()),
		RenderedPrompt:      renderedPrompt,
		ReferenceImageKeys:  refKeys,
		Provider:            openAIImage2ProviderName,
		ImageCountRequested: credit.RemainingImageCount,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if err := s.deps.Store.CreateTask(ctx, modelTask, s.deps.Cfg.TaskTTL); err != nil {
		return nil, status.Errorf(codes.Internal, "create task: %v", err)
	}

	payload := &taskqueue.GeneratePayload{
		TaskID:          taskID,
		UID:             uid,
		Prompt:          renderedPrompt,
		CreditID:        credit.CreditID,
		Provider:        openAIImage2ProviderName,
		ImageCount:      credit.RemainingImageCount,
		ReferenceImages: refImgData,
	}
	if _, err := s.deps.Enqueuer.EnqueueGenerate(payload); err != nil {
		_ = s.deps.Store.UpdateTaskStatus(ctx, taskID, model.TaskFailed, err.Error())
		return nil, status.Errorf(codes.Internal, "enqueue task: %v", err)
	}
	for _, refID := range refKeys {
		_ = s.deps.Store.DeleteRefImg(ctx, uid, refID)
	}

	return &pb.CreateTaskResponse{
		TaskId:               taskID,
		Status:               pb.TaskStatus_TASK_STATUS_PENDING,
		EstimatedWaitSeconds: 30,
	}, nil
}

func (s *SlotArtService) GetTask(ctx context.Context, req *pb.GetTaskRequest) (*pb.GetTaskResponse, error) {
	uid, err := requireUID(ctx)
	if err != nil {
		return nil, err
	}
	t, err := s.deps.Store.GetTask(ctx, req.GetTaskId())
	if err != nil {
		return nil, mapStoreError(err)
	}
	if t.UID != uid {
		return nil, status.Error(codes.PermissionDenied, "task does not belong to session")
	}
	var result *model.Result
	if t.Status == model.TaskCompleted {
		result, _ = s.deps.Store.GetResult(ctx, t.TaskID)
	}
	return &pb.GetTaskResponse{Task: toProtoPublicTask(t, result)}, nil
}

func (s *SlotArtService) GetDownloadUrl(ctx context.Context, req *pb.GetDownloadUrlRequest) (*pb.GetDownloadUrlResponse, error) {
	uid, err := requireUID(ctx)
	if err != nil {
		return nil, err
	}
	t, err := s.deps.Store.GetTask(ctx, req.GetTaskId())
	if err != nil {
		return nil, mapStoreError(err)
	}
	if t.UID != uid {
		return nil, status.Error(codes.PermissionDenied, "task does not belong to session")
	}
	url, err := s.deps.Store.Client().Get(ctx, fmt.Sprintf("slot:v1:task_image:%s:%s", req.GetTaskId(), req.GetImageId())).Result()
	if err != nil {
		return nil, mapStoreError(store.ErrNotFound)
	}
	return &pb.GetDownloadUrlResponse{Url: url, ExpiresInSeconds: int32(s.deps.Cfg.ResultTTL.Seconds())}, nil
}
