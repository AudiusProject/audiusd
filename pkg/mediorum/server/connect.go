package server

import (
	"context"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/AudiusProject/audiusd/pkg/api/storage/v1"
	"github.com/AudiusProject/audiusd/pkg/api/storage/v1/v1connect"
	"github.com/labstack/echo/v4"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ v1connect.StorageServiceHandler = (*StorageService)(nil)

type StorageService struct {
	mediorum *MediorumServer
}

func NewStorageService() *StorageService {
	return &StorageService{}
}

func (s *StorageService) SetMediorum(mediorum *MediorumServer) {
	s.mediorum = mediorum
}

// GetHealth implements v1connect.StorageServiceHandler.
func (s *StorageService) GetHealth(context.Context, *connect.Request[v1.GetHealthRequest]) (*connect.Response[v1.GetHealthResponse], error) {
	return connect.NewResponse(&v1.GetHealthResponse{}), nil
}

// GetUpload matches the serveUploadDetail endpoint
func (s *StorageService) GetUpload(ctx context.Context, req *connect.Request[v1.GetUploadRequest]) (*connect.Response[v1.GetUploadResponse], error) {
	id := req.Msg.Id
	var upload *Upload
	err := s.mediorum.crud.DB.First(&upload, "id = ?", id).Error
	if err != nil {
		return nil, echo.NewHTTPError(404, err.Error())
	}
	if upload.Status == JobStatusError {
		return nil, echo.NewHTTPError(422, upload)
	}

	if req.Msg.Fix && upload.Status != JobStatusDone {
		err = s.mediorum.transcode(upload)
		if err != nil {
			return nil, err
		}
	}

	if req.Msg.Analyze && upload.AudioAnalysisStatus != "done" {
		err = s.mediorum.analyzeAudio(upload, time.Minute*10)
		if err != nil {
			return nil, err
		}
	}

	ffProbeResult := &v1.FFProbeResult{
		Format: &v1.FFProbeResult_Format{
			Filename:       upload.FFProbe.Format.Filename,
			FormatName:     upload.FFProbe.Format.FormatName,
			FormatLongName: upload.FFProbe.Format.FormatLongName,
		},
	}

	audioAnalysisResult := &v1.AudioAnalysisResult{
		Bpm: upload.AudioAnalysisResults.BPM,
		Key: upload.AudioAnalysisResults.Key,
	}

	uploadResponse := &v1.Upload{
		Id:                   upload.ID,
		UserWallet:           upload.UserWallet.String,
		Template:             string(upload.Template),
		OrigFilename:         upload.OrigFileName,
		OrigFileCid:          upload.OrigFileCID,
		SelectedPreview:      upload.SelectedPreview.String,
		Probe:                ffProbeResult,
		Error:                upload.Error,
		Status:               upload.Status,
		CreatedAt:            timestamppb.New(upload.CreatedAt),
		UpdatedAt:            timestamppb.New(upload.UpdatedAt),
		AudioAnalysisResults: audioAnalysisResult,
	}

	return connect.NewResponse(&v1.GetUploadResponse{Upload: uploadResponse}), nil
}

// GetUploads implements v1connect.StorageServiceHandler.
func (s *StorageService) GetUploads(context.Context, *connect.Request[v1.GetUploadsRequest]) (*connect.Response[v1.GetUploadsResponse], error) {
	panic("unimplemented")
}

// Ping implements v1connect.StorageServiceHandler.
func (s *StorageService) Ping(context.Context, *connect.Request[v1.PingRequest]) (*connect.Response[v1.PingResponse], error) {
	return connect.NewResponse(&v1.PingResponse{Message: "pong"}), nil
}

// StreamImage implements v1connect.StorageServiceHandler.
func (s *StorageService) StreamImage(context.Context, *connect.Request[v1.StreamImageRequest], *connect.ServerStream[v1.StreamImageResponse]) error {
	panic("unimplemented")
}

// StreamTrack implements v1connect.StorageServiceHandler.
func (s *StorageService) StreamTrack(context.Context, *connect.Request[v1.StreamTrackRequest], *connect.ServerStream[v1.StreamTrackResponse]) error {
	panic("unimplemented")
}

func NewMediorumService(mediorum *MediorumServer) *StorageService {
	return &StorageService{mediorum: mediorum}
}
