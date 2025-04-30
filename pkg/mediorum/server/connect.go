package server

import (
	"context"

	"connectrpc.com/connect"
	v1 "github.com/AudiusProject/audiusd/pkg/api/storage/v1"
	"github.com/AudiusProject/audiusd/pkg/api/storage/v1/v1connect"
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

// GetUpload implements v1connect.StorageServiceHandler.
func (s *StorageService) GetUpload(ctx context.Context, req *connect.Request[v1.GetUploadRequest]) (*connect.Response[v1.GetUploadResponse], error) {
	dbUpload, err := s.mediorum.serveUpload(req.Msg.Id, req.Msg.Fix, req.Msg.Analyze)
	if err != nil {
		return nil, err
	}

	// Convert FFProbeResult to proto FFProbeResult
	var probe *v1.FFProbeResult
	if dbUpload.FFProbe != nil {
		probe = &v1.FFProbeResult{
			Format: &v1.FFProbeResult_Format{
				Filename:       dbUpload.FFProbe.Format.Filename,
				FormatName:     dbUpload.FFProbe.Format.FormatName,
				FormatLongName: dbUpload.FFProbe.Format.FormatLongName,
				Duration:       dbUpload.FFProbe.Format.Duration,
				Size:           dbUpload.FFProbe.Format.Size,
				BitRate:        dbUpload.FFProbe.Format.BitRate,
			},
		}
	}

	// Convert AudioAnalysisResult to proto AudioAnalysisResult
	var audioAnalysisResults *v1.AudioAnalysisResult
	if dbUpload.AudioAnalysisResults != nil {
		audioAnalysisResults = &v1.AudioAnalysisResult{
			Bpm: dbUpload.AudioAnalysisResults.BPM,
			Key: dbUpload.AudioAnalysisResults.Key,
		}
	}

	upload := &v1.Upload{
		Id:                      dbUpload.ID,
		UserWallet:              dbUpload.UserWallet.String,
		Template:                string(dbUpload.Template),
		OrigFilename:            dbUpload.OrigFileName,
		OrigFileCid:             dbUpload.OrigFileCID,
		SelectedPreview:         dbUpload.SelectedPreview.String,
		Probe:                   probe,
		Error:                   dbUpload.Error,
		ErrorCount:              int32(dbUpload.ErrorCount),
		Mirrors:                 dbUpload.Mirrors,
		TranscodedMirrors:       dbUpload.TranscodedMirrors,
		Status:                  dbUpload.Status,
		PlacementHosts:          dbUpload.PlacementHosts,
		CreatedBy:               dbUpload.CreatedBy,
		CreatedAt:               timestamppb.New(dbUpload.CreatedAt),
		UpdatedAt:               timestamppb.New(dbUpload.UpdatedAt),
		TranscodedBy:            dbUpload.TranscodedBy,
		TranscodeProgress:       dbUpload.TranscodeProgress,
		TranscodedAt:            timestamppb.New(dbUpload.TranscodedAt),
		TranscodeResults:        dbUpload.TranscodeResults,
		AudioAnalysisStatus:     dbUpload.AudioAnalysisStatus,
		AudioAnalysisError:      dbUpload.AudioAnalysisError,
		AudioAnalysisErrorCount: int32(dbUpload.AudioAnalysisErrorCount),
		AudioAnalyzedBy:         dbUpload.AudioAnalyzedBy,
		AudioAnalyzedAt:         timestamppb.New(dbUpload.AudioAnalyzedAt),
		AudioAnalysisResults:    audioAnalysisResults,
	}

	return connect.NewResponse(&v1.GetUploadResponse{
		Upload: upload,
	}), nil
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

// UploadImage implements v1connect.StorageServiceHandler.
func (s *StorageService) UploadImage(context.Context, *connect.Request[v1.UploadImageRequest]) (*connect.Response[v1.UploadImageResponse], error) {
	panic("unimplemented")
}

// UploadTrack implements v1connect.StorageServiceHandler.
func (s *StorageService) UploadTrack(context.Context, *connect.Request[v1.UploadTrackRequest]) (*connect.Response[v1.UploadTrackResponse], error) {
	panic("unimplemented")
}
