package main

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gowebpki/jcs"

	"github.com/AudiusProject/audiusd/pkg/core/gen/core_openapi/protocol"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"

	"github.com/AudiusProject/audiusd/pkg/common"
	ccommon "github.com/AudiusProject/audiusd/pkg/core/common"
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
	outputPath  string
	trackId     string
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
		storageSdk := sdk.NewStorageSDK(fmt.Sprintf("https://%s", serverAddr))
		uploadRes, err := storageSdk.UploadAudio(filePath)
		if err != nil || len(uploadRes) == 0 {
			return fmt.Errorf("failed to upload file: %w", err)
		}
		upload := uploadRes[0]

		coreSdk, err := core_sdk.NewSdk(core_sdk.WithOapiendpoint(serverAddr))
		if err != nil {
			return fmt.Errorf("failed to connect to gRPC server: %w", err)
		}

		imgCid := "asdf"

		ern := &adx.NewReleaseMessage{
			ReleaseHeader: &adx.ReleaseHeader{
				Sender: &adx.Party{
					PartyId: "aupl_cli",
					PubKey:  crypto.CompressPubkey(&privateKey.PublicKey),
				},
			},
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
					Resource: &adx.Resource_SoundRecording{
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
				&adx.Release{
					Release: &adx.Release_TrackRelease{
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
			},
		}

		ernBytes, err := proto.Marshal(ern)
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
		sendParams := protocol.NewProtocolSendTransactionParams()
		sendParams.SetTransaction(ccommon.SignedTxProtoIntoSignedTxOapi(tx))
		res, err := coreSdk.ProtocolSendTransaction(sendParams)
		if err != nil {
			return fmt.Errorf("ern failed: %w", err)
		}

		fmt.Printf("Upload successful: %s\n", res.Payload.Txhash)
		return nil
	},
}

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download a file with metadata and signature",
	RunE: func(cmd *cobra.Command, args []string) error {
		privateKey, err := loadPrivateKey(privKeyPath)
		if err != nil {
			return err
		}

		dataObj := map[string]interface{}{
			"upload_id":   trackId,
			"cid":         "",
			"shouldCache": 0,
			"timestamp":   time.Now().Unix(),
			"trackId":     0,
			"userId":      0,
		}

		jsonBytes, err := json.Marshal(dataObj)
		if err != nil {
			return fmt.Errorf("failed to marshal: %w", err)
		}
		canonicalJSON, err := jcs.Transform(jsonBytes)
		if err != nil {
			return fmt.Errorf("failed to canonicalize: %w", err)
		}

		// Hash and sign
		hash := crypto.Keccak256Hash(canonicalJSON)
		sig, err := crypto.Sign(hash[:], privateKey)
		if err != nil {
			return fmt.Errorf("failed to sign: %w", err)
		}
		sigHex := "0x" + hex.EncodeToString(sig)

		// Wrap in envelope
		envelope := map[string]string{
			"Data":      string(jsonBytes),
			"Signature": sigHex,
		}
		envelopeBytes, err := json.Marshal(envelope)
		if err != nil {
			return fmt.Errorf("failed to marshal envelope: %w", err)
		}
		query := url.Values{}

		query.Set("signature", string(envelopeBytes))

		fullURL := fmt.Sprintf("https://%s/tracks/stream/%s?%s", serverAddr, trackId, query.Encode())
		fmt.Printf("Downloading from: %s\n", fullURL)

		resp, err := http.Get(fullURL)
		if err != nil {
			return fmt.Errorf("failed to make GET request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("download failed: status %d - %s", resp.StatusCode, string(body))
		}

		outFile, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer outFile.Close()

		_, err = io.Copy(outFile, resp.Body)
		if err != nil {
			return fmt.Errorf("failed to write to file: %w", err)
		}

		fmt.Printf("Download successful: saved to %s\n", outputPath)
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
	uploadCmd.Flags().StringVar(&serverAddr, "server", "", "server address")
	uploadCmd.MarkFlagRequired("key")
	uploadCmd.MarkFlagRequired("file")

	rootCmd.AddCommand(uploadCmd)

	downloadCmd.Flags().StringVar(&privKeyPath, "key", "", "Path to Ethereum private key file")
	downloadCmd.Flags().StringVar(&trackId, "id", "", "Track ID to download")
	downloadCmd.Flags().StringVar(&serverAddr, "server", "", "server address")
	downloadCmd.Flags().StringVar(&outputPath, "out", "track.mp3", "Path to save the downloaded file")
	downloadCmd.MarkFlagRequired("key")
	downloadCmd.MarkFlagRequired("id")

	rootCmd.AddCommand(downloadCmd)

}

var rootCmd = &cobra.Command{Use: "upload"}

func Execute() error {
	return rootCmd.Execute()
}
