package server

import (
	"context"
	"errors"
	"fmt"

	v1 "github.com/AudiusProject/audiusd/pkg/api/core/v1"
	"github.com/AudiusProject/audiusd/pkg/common"
	"github.com/AudiusProject/audiusd/pkg/core/db"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

func (s *Server) isValidFileUpload(_ context.Context, tx *v1.SignedTransaction) error {
	fu := tx.GetFileUpload()
	if fu == nil {
		return errors.New("file upload not present")
	}

	sig := fu.GetUploadSignature()
	if sig == "" {
		return errors.New("no signature in file upload")
	}

	uploader := fu.GetUploaderAddress()
	if uploader == "" {
		return errors.New("not uploader address")
	}

	cid := fu.GetCid()
	if cid == "" {
		return errors.New("no cid provided")
	}

	sigData := &v1.UploadSignature{
		Cid: cid,
	}

	sigDataBytes, err := proto.Marshal(sigData)
	if err != nil {
		return fmt.Errorf("could not marshal sig data: %v", err)
	}

	_, address, err := common.EthRecover(sig, sigDataBytes)
	if err != nil {
		return fmt.Errorf("could not recover eth key: %v", err)
	}

	if address != uploader {
		return errors.New("uploader and signer mismatch")
	}

	return nil
}

func (s *Server) finalizeFileUpload(ctx context.Context, tx *v1.SignedTransaction, txHash string, blockHeight int64) (proto.Message, error) {
	if err := s.isValidFileUpload(ctx, tx); err != nil {
		s.logger.Error("Invalid file upload:", zap.Error(err))
		return nil, err
	}

	fu := tx.GetFileUpload()
	if fu == nil {
		return nil, errors.New("finalizeFileUpload called with invalid tx")
	}

	qtx := s.getDb()

	err := qtx.InsertFileUpload(ctx, db.InsertFileUploadParams{
		UploaderAddress: fu.UploaderAddress,
		Cid:             fu.Cid,
		Upid:            fu.UploadId,
		UploadSignature: fu.UploadSignature,
		TxHash:          txHash,
		BlockHeight:     blockHeight,
	})
	if err != nil {
		return nil, fmt.Errorf("could not store file upload tx: %v", err)
	}

	return nil, nil
}
