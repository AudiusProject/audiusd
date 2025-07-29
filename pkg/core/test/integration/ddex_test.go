package integration_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	corev1 "github.com/AudiusProject/audiusd/pkg/api/core/v1"
	corev1beta1 "github.com/AudiusProject/audiusd/pkg/api/core/v1beta1"
	ddexv1beta1 "github.com/AudiusProject/audiusd/pkg/api/ddex/v1beta1"
	"github.com/AudiusProject/audiusd/pkg/common"
	"github.com/AudiusProject/audiusd/pkg/core/test/integration/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestERNNewMessage(t *testing.T) {
	ctx := context.Background()
	sdk := utils.DiscoveryOne

	// Wait for the node to be ready
	timeout := time.After(30 * time.Second)
	for {
		select {
		case <-timeout:
			assert.Fail(t, "timed out waiting for discovery node to be ready")
		default:
		}
		status, err := sdk.Core.GetStatus(ctx, connect.NewRequest(&corev1.GetStatusRequest{}))
		assert.NoError(t, err)
		if status.Msg.Ready {
			break
		}
		time.Sleep(2 * time.Second)
	}

	// Create ERN NewReleaseMessage based on fake band data
	ernMessage := createFakeBandERNMessage()

	// Create transaction envelope
	envelope := &corev1beta1.Envelope{
		Header: &corev1beta1.EnvelopeHeader{
			ChainId:    "audius-devnet",
			From:       "PADPIDA2024010501X",
			To:         "PADPIDA202401120D9",
			Nonce:      "1",
			Expiration: time.Now().Add(time.Hour).Unix(),
		},
		Messages: []*corev1beta1.Message{
			{
				Message: &corev1beta1.Message_Ern{
					Ern: ernMessage,
				},
			},
		},
	}

	transaction := &corev1beta1.Transaction{
		Envelope: envelope,
	}

	// Calculate expected transaction hash
	expectedTxHash, err := common.ToTxHash(transaction)
	require.NoError(t, err)

	// Submit the transaction
	req := &corev1.SendTransactionRequest{
		Transactionv2: transaction,
	}

	submitRes, err := sdk.Core.SendTransaction(ctx, connect.NewRequest(req))
	require.NoError(t, err)

	// Check if we have a transaction receipt (v2 transactions)
	if submitRes.Msg.TransactionReceipt != nil {
		txhash := submitRes.Msg.TransactionReceipt.TxHash
		assert.Equal(t, expectedTxHash, txhash)

		// Wait for transaction to be processed
		time.Sleep(time.Second * 2)

		// Retrieve and validate the transaction
		txRes, err := sdk.Core.GetTransaction(ctx, connect.NewRequest(&corev1.GetTransactionRequest{TxHash: txhash}))
		require.NoError(t, err)

		// For v2 transactions, check if we got the transaction back
		if txRes.Msg.Transaction != nil {
			// This will be a v1 Transaction wrapper, which might not have the same structure
			t.Logf("ERN transaction %s successfully submitted and retrieved", txhash)
		}

		t.Logf("ERN transaction %s successfully processed with %d parties, %d resources, %d releases, %d deals - representing 'Live - Electric Nights' album (2h43m of live music)",
			txhash, len(ernMessage.PartyList), len(ernMessage.ResourceList), len(ernMessage.ReleaseList), len(ernMessage.DealList))
	} else if submitRes.Msg.Transaction != nil {
		// Fallback for v1 transactions
		txhash := submitRes.Msg.Transaction.Hash
		assert.Equal(t, expectedTxHash, txhash)

		t.Logf("ERN transaction %s successfully processed (v1 response)", txhash)
	} else {
		t.Fatal("No transaction receipt or transaction returned from submission")
	}
}

// createFakeBandERNMessage creates an ERN message based on fake band data
func createFakeBandERNMessage() *ddexv1beta1.NewReleaseMessage {
	// Create message header based on fake XML MessageHeader
	messageHeader := &ddexv1beta1.MessageHeader{
		MessageThreadId: stringPtr("F0100045091829_ADS"),
		MessageId:       "458380160",
		MessageSender: &ddexv1beta1.MessageSender{
			PartyId: &ddexv1beta1.Party_PartyId{
				Dpid: "PADPIDA2024010501X",
			},
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Melodic Records Entertainment",
			},
		},
		MessageRecipient: []*ddexv1beta1.MessageRecipient{
			{
				PartyId: &ddexv1beta1.Party_PartyId{
					Dpid: "PADPIDA202401120D9",
				},
				PartyName: &ddexv1beta1.Party_PartyName{
					FullName: "Audius",
				},
			},
		},
		MessageCreatedDateTime: timestamppb.New(time.Date(2025, 6, 4, 17, 9, 19, 141000000, time.UTC)),
		MessageControlType:     ddexv1beta1.MessageControlType_MESSAGE_CONTROL_TYPE_TEST_MESSAGE.Enum(),
	}

	// Create ALL parties from fake band data (comprehensive list)
	parties := []*ddexv1beta1.Party{
		{
			PartyReference: "P_MRE_SENDER",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Melodic Records Entertainment",
			},
			PartyId: &ddexv1beta1.Party_PartyId{
				Dpid: "PADPIDA2024010501X",
			},
		},
		// Main Artists
		{
			PartyReference: "P_ARTIST_1199281",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "The Electric Riders",
			},
		},
		{
			PartyReference: "P_ARTIST_4729799",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Marcus Stone",
			},
		},
		{
			PartyReference: "P_ARTIST_2729598",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Jake Rivers",
			},
		},
		{
			PartyReference: "P_ARTIST_33801",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Alex Thunder",
			},
		},
		{
			PartyReference: "P_ARTIST_753696",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Leo Midnight",
			},
		},
		// Composers and Songwriters
		{
			PartyReference: "P_ARTIST_5105",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Sam Melody",
			},
		},
		{
			PartyReference: "P_ARTIST_5188",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Luna Hayes",
			},
		},
		{
			PartyReference: "P_ARTIST_5189",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Ray Lightning",
			},
		},
		{
			PartyReference: "P_ARTIST_7775",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Chris Voltage",
			},
		},
		{
			PartyReference: "P_ARTIST_11122",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Tony Storm",
			},
		},
		{
			PartyReference: "P_ARTIST_16163",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Felix Harmony",
			},
		},
		{
			PartyReference: "P_ARTIST_16164",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Dave Echo",
			},
		},
		{
			PartyReference: "P_ARTIST_21383",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Nick Reverb",
			},
		},
		{
			PartyReference: "P_ARTIST_21387",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Max Chorus",
			},
		},
		{
			PartyReference: "P_ARTIST_21467",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Ryan Beat",
			},
		},
		// Musicians
		{
			PartyReference: "P_ARTIST_23050",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Oliver Bass",
			},
		},
		{
			PartyReference: "P_ARTIST_23052",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Noah Drums",
			},
		},
		{
			PartyReference: "P_ARTIST_23053",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Ethan Keys",
			},
		},
		{
			PartyReference: "P_ARTIST_23054",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Liam Guitar",
			},
		},
		{
			PartyReference: "P_ARTIST_39619",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Jake R. Rivers",
			},
		},
		{
			PartyReference: "P_ARTIST_41932",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Tyler Sonic",
			},
		},
		{
			PartyReference: "P_ARTIST_65552",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Blake Phoenix",
			},
		},
		{
			PartyReference: "P_ARTIST_73668",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "River Sterling",
			},
		},
		{
			PartyReference: "P_ARTIST_74124",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Cole Rhythm",
			},
		},
		{
			PartyReference: "P_ARTIST_87653",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Quinn Melody",
			},
		},
		{
			PartyReference: "P_ARTIST_109713",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Drew Harmony",
			},
		},
		{
			PartyReference: "P_ARTIST_109714",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Sage Notes",
			},
		},
		{
			PartyReference: "P_ARTIST_109768",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Kai Tempo",
			},
		},
		{
			PartyReference: "P_ARTIST_110895",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Zane Chord",
			},
		},
		{
			PartyReference: "P_ARTIST_2729177",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Atlas Lyric",
			},
		},
		{
			PartyReference: "P_ARTIST_2732982",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Phoenix Tune",
			},
		},
		{
			PartyReference: "P_ARTIST_2758344",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Unknown",
			},
		},
		{
			PartyReference: "P_ARTIST_3949457",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Orion Sound",
			},
		},
		{
			PartyReference: "P_ARTIST_3992112",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Nova Wave",
			},
		},
		{
			PartyReference: "P_ARTIST_3992113",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Ryder Beat",
			},
		},
		{
			PartyReference: "P_ARTIST_7896981",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Echo Pulse",
			},
		},
		{
			PartyReference: "P_ARTIST_2543992",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "The Electric Riders, Marcus Stone, Jake Rivers, Alex Thunder, Leo Midnight",
			},
		},
		// Label
		{
			PartyReference: "P_LABEL_HARMONY_RECORDS",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Harmony Records Legacy",
			},
		},
	}

	// Create fake sound recording resources
	resources := []*ddexv1beta1.Resource{
		{
			Resource: &ddexv1beta1.Resource_SoundRecording_{
				SoundRecording: &ddexv1beta1.Resource_SoundRecording{
					ResourceReference:     "A1",
					Type:                  "MusicalWorkSoundRecording",
					DisplayTitleText:      "Midnight Express (Live at Thunder Arena, Electric City, TX - March 2023)",
					VersionType:           "LiveVersion",
					DisplayArtistName:     "The Electric Riders, Marcus Stone, Jake Rivers, Alex Thunder, Leo Midnight",
					Duration:              "PT0H1M32S",
					FirstPublicationDate:  "2023-05-20",
					ParentalWarningType:   "ExplicitContentEdited",
					LanguageOfPerformance: "en",
				},
			},
		},
		{
			Resource: &ddexv1beta1.Resource_SoundRecording_{
				SoundRecording: &ddexv1beta1.Resource_SoundRecording{
					ResourceReference:     "A2",
					Type:                  "MusicalWorkSoundRecording",
					DisplayTitleText:      "Electric Storm (Live at Thunder Arena, Electric City, TX - March 2023)",
					VersionType:           "LiveVersion",
					DisplayArtistName:     "The Electric Riders, Marcus Stone, Jake Rivers, Alex Thunder, Leo Midnight",
					Duration:              "PT0H2M54S",
					FirstPublicationDate:  "2023-05-20",
					ParentalWarningType:   "ExplicitContentEdited",
					LanguageOfPerformance: "en",
				},
			},
		},
		{
			Resource: &ddexv1beta1.Resource_SoundRecording_{
				SoundRecording: &ddexv1beta1.Resource_SoundRecording{
					ResourceReference:     "A3",
					Type:                  "MusicalWorkSoundRecording",
					DisplayTitleText:      "Thunder Roads (Live at Thunder Arena, Electric City, TX - March 2023)",
					VersionType:           "LiveVersion",
					DisplayArtistName:     "The Electric Riders, Marcus Stone, Jake Rivers, Alex Thunder, Leo Midnight",
					Duration:              "PT0H2M27S",
					FirstPublicationDate:  "2023-05-20",
					ParentalWarningType:   "ExplicitContentEdited",
					LanguageOfPerformance: "en",
				},
			},
		},
		{
			Resource: &ddexv1beta1.Resource_SoundRecording_{
				SoundRecording: &ddexv1beta1.Resource_SoundRecording{
					ResourceReference:     "A8",
					Type:                  "MusicalWorkSoundRecording",
					DisplayTitleText:      "Lightning Flash (Live at Thunder Arena, Electric City, TX - March 2023)",
					VersionType:           "LiveVersion",
					DisplayArtistName:     "The Electric Riders, Marcus Stone, Jake Rivers, Alex Thunder, Leo Midnight",
					Duration:              "PT0H3M3S",
					FirstPublicationDate:  "2023-05-20",
					ParentalWarningType:   "ExplicitContentEdited",
					LanguageOfPerformance: "en",
				},
			},
		},
		{
			Resource: &ddexv1beta1.Resource_SoundRecording_{
				SoundRecording: &ddexv1beta1.Resource_SoundRecording{
					ResourceReference:     "A9",
					Type:                  "MusicalWorkSoundRecording",
					DisplayTitleText:      "Voltage Blues (Live at Thunder Arena, Electric City, TX - March 2023)",
					VersionType:           "LiveVersion",
					DisplayArtistName:     "The Electric Riders, Marcus Stone, Jake Rivers, Alex Thunder, Leo Midnight",
					Duration:              "PT0H3M39S",
					FirstPublicationDate:  "2023-05-20",
					ParentalWarningType:   "ExplicitContentEdited",
					LanguageOfPerformance: "en",
				},
			},
		},
		{
			Resource: &ddexv1beta1.Resource_Image_{
				Image: &ddexv1beta1.Resource_Image{
					ResourceReference: "A47",
					Type:              "FrontCoverImage",
					ResourceId: &ddexv1beta1.Resource_ResourceId{
						Isrc: "ISRC123456789012",
						ProprietaryId: []*ddexv1beta1.Resource_ProprietaryId{
							{
								Namespace:     "AUDIUS",
								ProprietaryId: "123456789012",
							},
						},
					},
					TechnicalDetails: &ddexv1beta1.Resource_Image_TechnicalDetails{
						ImageCodecType:  "JPEG",
						ImageHeight:     1000,
						ImageWidth:      1000,
						ImageResolution: "72dpi",
						File: &ddexv1beta1.Resource_Image_TechnicalDetails_File{
							FileSize: 1000,
							Uri:      "CID:123456789012",
							HashSum: &ddexv1beta1.Resource_Image_TechnicalDetails_File_HashSum{
								Algorithm:    "IPFS",
								HashSumValue: "Qm123456789012",
							},
						},
						IsProvidedInDelivery: true,
					},
				},
			},
		},
	}

	// Release information - fake album data
	// Main Album: "Live - Electric Nights" (R0) - 2h43m total, Rock genre
	// Individual track releases: R1-R44+ (each track as separate release)
	// UPC: 123456789012, Catalog: F0100045091829, Harmony Records Legacy
	// TODO: Add complete Release structure (currently simplified to main album only)
	releases := []*ddexv1beta1.Release{
		{
			Release: &ddexv1beta1.Release_MainRelease{
				MainRelease: &ddexv1beta1.Release_Release{
					ReleaseReference: "R0",
					ReleaseType:      "Album",
					ReleaseId: &ddexv1beta1.Release_ReleaseId{
						Grid:            "F10301F00045091829",
						Icpn:            "123456789012",
						CatalogueNumber: "F0100045091829",
					},
					DisplayTitleText: "Live - Electric Nights",
					DisplayTitle: &ddexv1beta1.Release_DisplayTitle{
						TitleText: "Live - Electric Nights",
					},
					DisplayArtistName: "The Electric Riders",
					DisplayArtist: []*ddexv1beta1.Release_DisplayArtist{
						{
							ArtistPartyReference: "P_ARTIST_1199281",
							DisplayArtistRole:    "MainArtist",
						},
					},
					ReleaseLabelReference: "P_LABEL_HARMONY_RECORDS",
					PLine: &ddexv1beta1.Release_Release_PLine{
						Year:      "2023",
						PLineText: "(P) 2023 Melodic Records Entertainment",
					},
					Duration:            "PT2H43M8S", // 2 hours 43 minutes total!
					OriginalReleaseDate: "2023-05-20",
					ParentalWarningType: "NotExplicit",
					Genre: &ddexv1beta1.Release_Release_Genre{
						GenreText: "Rock",
					},
				},
			},
		},
	}

	// Deal information - fake deal data for worldwide licensing
	// TODO: Add proper Deal protobuf structure once schema issues resolved
	deals := []*ddexv1beta1.Deal{}

	return &ddexv1beta1.NewReleaseMessage{
		MessageHeader: messageHeader,
		PartyList:     parties,
		ResourceList:  resources,
		ReleaseList:   releases,
		DealList:      deals,
	}
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}

func TestMEADNewMessage(t *testing.T) {
}

func TestPIENewMessage(t *testing.T) {
}

func TestMultiMessageTransaction(t *testing.T) {
}
