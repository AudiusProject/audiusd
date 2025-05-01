package integrationtests

import (
	"bytes"
	"context"
	_ "embed"
	"io"
	"log"
	"os"
	"testing"
	"time"

	"connectrpc.com/connect"
	v1storage "github.com/AudiusProject/audiusd/pkg/api/storage/v1"
	"github.com/AudiusProject/audiusd/pkg/mediorum/server/signature"
	"github.com/AudiusProject/audiusd/pkg/sdk"
	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"
)

func TestTrackReleaseWorkflow(t *testing.T) {
	ctx := context.Background()

	serverAddr := "node3.audiusd.devnet"
	privKeyPath := "./assets/demo_key.txt"
	//privKeyPath2 := "./assets/demo_key2.txt"
	//downloadPath := fmt.Sprintf("%s/test_audio_download.mp3", os.TempDir())

	sdk := sdk.NewAudiusdSDK(serverAddr)
	if err := sdk.ReadPrivKey(privKeyPath); err != nil {
		require.Nil(t, err, "failed to read private key: %w", err)
	}

	audioFile, err := os.Open("./assets/anxiety-upgrade.mp3")
	require.Nil(t, err, "failed to open file")
	defer audioFile.Close()
	audioFileBytes, err := io.ReadAll(audioFile)
	require.Nil(t, err, "failed to read file")

	// upload the track
	uploadFileRes, err := sdk.Storage.UploadFiles(ctx, &connect.Request[v1storage.UploadFilesRequest]{
		Msg: &v1storage.UploadFilesRequest{
			UserWallet: sdk.Address(),
			Template:   "audio",
			Files: []*v1storage.File{
				{
					Filename: "anxiety-upgrade.mp3",
					Data:     audioFileBytes,
				},
			},
		},
	})
	require.Nil(t, err, "failed to upload file")
	require.EqualValues(t, 1, len(uploadFileRes.Msg.Uploads), "failed to upload file")

	upload := uploadFileRes.Msg.Uploads[0]

	// get the upload info
	uploadRes, err := sdk.Storage.GetUpload(ctx, &connect.Request[v1storage.GetUploadRequest]{
		Msg: &v1storage.GetUploadRequest{
			Id: upload.Id,
		},
	})
	require.Nil(t, err, "failed to get upload")
	require.EqualValues(t, upload.Id, uploadRes.Msg.Upload.Id, "failed to get upload")
	require.EqualValues(t, upload.UserWallet, uploadRes.Msg.Upload.UserWallet, "failed to get upload")
	require.EqualValues(t, upload.OrigFileCid, uploadRes.Msg.Upload.OrigFileCid, "failed to get upload")
	require.EqualValues(t, upload.OrigFilename, uploadRes.Msg.Upload.OrigFilename, "failed to get upload")

	// release the track
	title := "Anxiety Upgrade"
	genre := "Electronic"
	_, err = sdk.ReleaseTrack(ctx, upload.OrigFileCid, title, genre)
	require.Nil(t, err, "failed to release track")

	// TODO: get the release info

	// create stream signature
	sigData := signature.SignatureData{
		UploadID:    upload.Id,
		Cid:         upload.OrigFileCid,
		ShouldCache: 1,
		Timestamp:   time.Now().Unix(),
		TrackId:     1,
		UserID:      1,
	}

	streamSignature, err := signature.GenerateSignature(sigData, sdk.PrivKey())
	require.Nil(t, err, "failed to generate stream signature")

	// stream the file
	stream, err := sdk.Storage.StreamFile(ctx, &connect.Request[v1storage.StreamFileRequest]{
		Msg: &v1storage.StreamFileRequest{
			Cid:       upload.OrigFileCid,
			Signature: streamSignature,
		},
	})
	require.Nil(t, err, "failed to stream file")

	var fileData bytes.Buffer
	var contentType, filename, txHash string

	for stream.Receive() {
		res := stream.Msg()
		if len(res.Data) > 0 {
			fileData.Write(res.Data)
		}
		if res.ContentType != "" {
			contentType = res.ContentType
		}
		if res.Filename != "" {
			filename = res.Filename
		}
		if res.TxHash != "" {
			txHash = res.TxHash
		}
	}
	if err := stream.Err(); err != nil {
		log.Fatalf("stream error: %v", err)
	}

	spew.Dump(contentType, filename, txHash)

	// Now try to access the file
	// err = storageSdk.DownloadTrack(releaseRes.TrackID, downloadPath)
	// require.Nil(t, err, "failed to download track")

	// // Try to access the file with a different key
	// err = storageSdk.LoadPrivateKey(privKeyPath2)
	// require.Nil(t, err, "failed to set privKey2 on storage sdk")
	// err = storageSdk.DownloadTrack(releaseRes.TrackID, downloadPath)
	// require.NotNil(t, err, "expected error when downloading track with wrong key")
	// require.ErrorContains(t, err, "signer not authorized")
}
