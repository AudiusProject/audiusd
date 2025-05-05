package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"connectrpc.com/connect"
	v1storage "github.com/AudiusProject/audiusd/pkg/api/storage/v1"
	"github.com/AudiusProject/audiusd/pkg/common"
	"github.com/AudiusProject/audiusd/pkg/mediorum/cidutil"
	"github.com/AudiusProject/audiusd/pkg/mediorum/server/signature"
	"gocloud.dev/gcerrors"
)

const (
	// Default chunk size if client doesn't specify
	defaultChunkSize = 128 * 1024
	// Minimum chunk size to prevent too small requests
	minChunkSize = 4 * 1024
	// Maximum chunk size to prevent too large requests
	maxChunkSize = 1024 * 1024
)

func (s *MediorumServer) serveBlobGRPC(ctx context.Context, req *v1storage.StreamFileRequest, stream *connect.ServerStream[v1storage.StreamFileResponse]) error {
	if s.Config.Env != "dev" {
		return connect.NewError(connect.CodeNotFound, errors.New("not found"))
	}

	// if signature is present, it is a track stream
	isAudioFile := true
	contentType := "audio/mpeg"
	sig, err := signature.ParseFromQueryString(req.Signature)
	if err != nil {
		s.logger.Warn("unable to parse signature for request", "signature", req.Signature, "err", err)
		isAudioFile = false
		contentType = "image/jpeg"
	}

	trackId := sig.Data.TrackId

	// // check it is for this upload
	// if sig.Data.UploadID != trackId {
	// 	return connect.NewError(connect.CodePermissionDenied, errors.New("signature contains incorrect track ID"))
	// }

	var cid string
	s.crud.DB.Raw("SELECT cid FROM sound_recordings WHERE track_id = ?", trackId).Scan(&cid)
	if cid == "" {
		return connect.NewError(connect.CodeNotFound, errors.New("track not found"))
	}

	var count int
	s.crud.DB.Raw("SELECT COUNT(*) FROM management_keys WHERE track_id = ? AND address = ?", trackId, sig.SignerWallet).Scan(&count)
	if count == 0 {
		s.logger.Debug("sig no match", "signed by", sig.SignerWallet)
		return connect.NewError(connect.CodePermissionDenied, errors.New("signer not authorized to access"))
	}

	key := cidutil.ShardCID(req.Cid)

	blob, err := s.bucket.NewReader(ctx, key, nil)
	if err != nil {
		if gcerrors.Code(err) == gcerrors.NotFound {
			// If we don't have the file, find a different node
			host := s.findNodeToServeBlob(ctx, req.Cid)
			if host == "" {
				return err
			}
			// TODO: Implement redirect to other node
			return err
		}
		return err
	}
	defer func() {
		if blob != nil {
			blob.Close()
		}
	}()

	s.logger.Info("serveBlobGRPC", "contentType", contentType, "isAudioFile", isAudioFile)
	if isAudioFile {
		go func() {
			sig := sig
			// as per CN `userId: req.userId ?? delegateOwnerWallet`
			userId := s.Config.Self.Wallet
			if sig.Data.UserID != 0 {
				userId = strconv.Itoa(sig.Data.UserID)
			}

			// record play event to chain
			signatureData, err := signature.GenerateListenTimestampAndSignature(s.Config.privateKey)
			if err != nil {
				s.logger.Error("unable to build request", "err", err)
				return
			}

			parsedTime, err := time.Parse(time.RFC3339, signatureData.Timestamp)
			if err != nil {
				s.logger.Error("core error parsing time:", "err", err)
				return
			}

			ip := common.GetClientIP(ctx)
			geoData, err := s.getGeoFromIP(ip)
			if err != nil {
				s.logger.Error("core plays bad ip: %v", err)
				return
			}

			trackID := fmt.Sprint(sig.Data.TrackId)

			s.playEventQueue.pushPlayEvent(&PlayEvent{
				UserID:    userId,
				TrackID:   trackID,
				PlayTime:  parsedTime,
				Signature: signatureData.Signature,
				City:      geoData.City,
				Country:   geoData.Country,
				Region:    geoData.Region,
			})
		}()
	}

	// Determine chunk size
	chunkSize := defaultChunkSize
	reqChunkSize := req.ChunkSize
	if reqChunkSize > 0 {
		// Clamp chunk size between min and max
		if reqChunkSize < minChunkSize {
			chunkSize = minChunkSize
		} else if reqChunkSize > maxChunkSize {
			chunkSize = maxChunkSize
		} else {
			chunkSize = int(reqChunkSize)
		}
	}

	// Stream the file in chunks
	buffer := make([]byte, chunkSize)
	for {
		n, err := blob.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		if err := stream.Send(&v1storage.StreamFileResponse{
			Data: buffer[:n],
		}); err != nil {
			return err
		}
	}

	return nil
}
