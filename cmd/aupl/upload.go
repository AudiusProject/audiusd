package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"io/ioutil"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"

	"github.com/AudiusProject/audiusd/pkg/common"
	"github.com/AudiusProject/audiusd/pkg/core/gen/core_proto"
	adx "github.com/AudiusProject/audiusd/pkg/core/gen/core_proto/audiusddex/v1beta1"
	core_sdk "github.com/AudiusProject/audiusd/pkg/core/sdk"
	"github.com/AudiusProject/audiusd/pkg/sdk"
)

var (
	privKeyPath string
	filePath    string
	title       string
	genre       string
	serverAddr  string
)

var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload a file with metadata and signature",
	RunE: func(cmd *cobra.Command, args []string) error {
		privateKey, err := loadPrivateKey(privKeyPath)
		if err != nil {
			return err
		}

		// upload audio file
		storageSdk := sdk.NewStorageSDK(serverAddr)
		uploadRes, err := storageSdk.UploadAudio(filePath)
		if err != nil || len(uploadRes) == 0 {
			return fmt.Errorf("failed to upload file: %w", err)
		}
		upload := uploadRes[0]

		coreSdk, err := core_sdk.NewSdk(core_sdk.WithGrpcendpoint(serverAddr))
		if err != nil {
			return fmt.Errorf("failed to connect to gRPC server: %w", err)
		}

		imgCid := "asdf"

		ern := &adx.NewReleaseMessage{
			ReleaseHeader: &adx.ReleaseHeader{},
			ResourceList: []*adx.Resource{
				&adx.Resource{
					ResourceReference: "AI1",
					Resource: &adx.Resource_Image{
						Image: &adx.Image{
							Cid: imgCid,
							Id: &adx.ImageId{
								ProprietaryId: imgCid,
							},
						},
					},
				},
				&adx.Resource{
					ResourceReference: "AT1",
					&adx.Resource_SoundRecording{
						SoundRecording: &adx.SoundRecording{
							Cid: upload.OrigFileCID,
							Id: &adx.SoundRecordingId{
								Isrc: upload.ID,
							},
						},
					},
				},
			},
			ReleaseList: []*adx.Release{
				&adx.Release_TrackRelease{
					TrackRelease: &adx.TrackRelease{
						ReleaseId: &adx.ReleaseId{
							Isrc: uuid.NewString(),
						},
						ReleaseResourceReference:       "AT1",
						LinkedReleaseResourceReference: "AI1",
						Title:                          title,
						Genre:                          genre,
					},
				},
			},
		}

		ernBytes, err := proto.Marshal(verificationTx)
		if err != nil {
			return fmt.Errorf("failure to marshal ern: %v", err)
		}

		sig, err := common.EthSign(privateKey, ernBytes)
		if err != nil {
			return fmt.Errorf("failed to sign message: %w", err)
		}

		tx := &core_proto.SignedTransaction{
			Signature: sig,
			RequestId: uuid.NewString(),
			Transaction: &core_proto.SignedTransaction_Release{
				Release: ern,
			},
		}
		txhash, err := coreSdk.SendTransaction(context.Background(), &core_proto.SendTransactionRequest{Transaction: tx})
		if err != nil {
			return fmt.Errorf("ern failed: %w", err)
		}

		fmt.Printf("Upload successful: %v\n", resp.GetMessage())
		return nil
	},
}

func loadPrivateKey(path string) (*ecdsa.PrivateKey, error) {
	keyBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read private key file: %w", err)
	}
	key, err := crypto.HexToECDSA(string(keyBytes))
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	return key, nil
}

func init() {
	uploadCmd.Flags().StringVar(&privKeyPath, "key", "", "Path to Ethereum private key file")
	uploadCmd.Flags().StringVar(&filePath, "file", "", "Path to file to upload")
	uploadCmd.Flags().StringVar(&title, "title", "", "Title of the upload")
	uploadCmd.Flags().StringVar(&genre, "genre", "", "Genre of the upload")
	uploadCmd.Flags().StringVar(&serverAddr, "server", "", "gRPC server address")
	uploadCmd.MarkFlagRequired("key")
	uploadCmd.MarkFlagRequired("file")

	rootCmd.AddCommand(uploadCmd)
}

var rootCmd = &cobra.Command{Use: "mycli"}

func Execute() error {
	return rootCmd.Execute()
}
