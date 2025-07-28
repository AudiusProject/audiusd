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

	// Create ERN NewReleaseMessage based on Highwaymen SME XML data
	ernMessage := createHighwaymenERNMessage()

	// Create transaction envelope
	envelope := &corev1beta1.Envelope{
		Header: &corev1beta1.EnvelopeHeader{
			ChainId:    "audiusd-1",          // Default chain ID
			From:       "PADPIDA2007040502I", // Sony Music Entertainment from XML
			To:         "PADPIDA202401120D9", // Audius from XML
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

		t.Logf("ERN transaction %s successfully processed with %d parties, %d resources, %d releases, %d deals - representing 'Live - American Outlaws' album (2h43m of live music)",
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

// createHighwaymenERNMessage creates an ERN message based on the Highwaymen SME XML data
func createHighwaymenERNMessage() *ddexv1beta1.NewReleaseMessage {
	// Create message header based on XML MessageHeader
	messageHeader := &ddexv1beta1.MessageHeader{
		MessageThreadId: stringPtr("G0100035091829_ADS"),
		MessageId:       "358280160",
		MessageSender: &ddexv1beta1.MessageSender{
			PartyId: &ddexv1beta1.Party_PartyId{
				Dpid: "PADPIDA2007040502I",
			},
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Sony Music Entertainment",
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

	// Create ALL parties from the XML PartyList (comprehensive list)
	parties := []*ddexv1beta1.Party{
		{
			PartyReference: "P_SME_SENDER",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Sony Music Entertainment",
			},
			PartyId: &ddexv1beta1.Party_PartyId{
				Dpid: "PADPIDA2007040502I",
			},
		},
		// Main Artists
		{
			PartyReference: "P_ARTIST_1199281",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "The Highwaymen",
			},
		},
		{
			PartyReference: "P_ARTIST_4729799",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Willie Nelson",
			},
		},
		{
			PartyReference: "P_ARTIST_2729598",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Johnny Cash",
			},
		},
		{
			PartyReference: "P_ARTIST_33801",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Waylon Jennings",
			},
		},
		{
			PartyReference: "P_ARTIST_753696",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Kris Kristofferson",
			},
		},
		// Composers and Songwriters
		{
			PartyReference: "P_ARTIST_5105",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Bob McDill",
			},
		},
		{
			PartyReference: "P_ARTIST_5188",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "June Carter",
			},
		},
		{
			PartyReference: "P_ARTIST_5189",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Merle Kilgore",
			},
		},
		{
			PartyReference: "P_ARTIST_7775",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Chips Moman",
			},
		},
		{
			PartyReference: "P_ARTIST_11122",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Tony Joe White",
			},
		},
		{
			PartyReference: "P_ARTIST_16163",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Herman Parker, Jr.",
			},
		},
		{
			PartyReference: "P_ARTIST_16164",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Sam C. Phillips",
			},
		},
		{
			PartyReference: "P_ARTIST_21383",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Fred Rose",
			},
		},
		{
			PartyReference: "P_ARTIST_21387",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Ed Bruce",
			},
		},
		{
			PartyReference: "P_ARTIST_21467",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Mickey Raphael",
			},
		},
		// Musicians
		{
			PartyReference: "P_ARTIST_23050",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Reggie Young",
			},
		},
		{
			PartyReference: "P_ARTIST_23052",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Gene Chrisman",
			},
		},
		{
			PartyReference: "P_ARTIST_23053",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Bobby Emmons",
			},
		},
		{
			PartyReference: "P_ARTIST_23054",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Mike Leech",
			},
		},
		{
			PartyReference: "P_ARTIST_39619",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "John R. Cash",
			},
		},
		{
			PartyReference: "P_ARTIST_41932",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Robby Turner",
			},
		},
		{
			PartyReference: "P_ARTIST_65552",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Guy Clark",
			},
		},
		{
			PartyReference: "P_ARTIST_73668",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Shel Silverstein",
			},
		},
		{
			PartyReference: "P_ARTIST_74124",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Steve Goodman",
			},
		},
		{
			PartyReference: "P_ARTIST_87653",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Patsy Bruce",
			},
		},
		{
			PartyReference: "P_ARTIST_109713",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Paul Buskirk",
			},
		},
		{
			PartyReference: "P_ARTIST_109714",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Walt Breeland",
			},
		},
		{
			PartyReference: "P_ARTIST_109768",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Lee Clayton",
			},
		},
		{
			PartyReference: "P_ARTIST_110895",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Bobby Wood",
			},
		},
		{
			PartyReference: "P_ARTIST_2729177",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Bob Dylan",
			},
		},
		{
			PartyReference: "P_ARTIST_2732982",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Stan Jones",
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
				FullName: "Mark James",
			},
		},
		{
			PartyReference: "P_ARTIST_3992112",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Wayne Carson",
			},
		},
		{
			PartyReference: "P_ARTIST_3992113",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Johnny Christopher",
			},
		},
		{
			PartyReference: "P_ARTIST_7896981",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Jimmy Webb",
			},
		},
		{
			PartyReference: "P_ARTIST_2543992",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "The Highwaymen, Willie Nelson, Johnny Cash, Waylon Jennings, Kris Kristofferson",
			},
		},
		// Label
		{
			PartyReference: "P_LABEL_COLUMBIA_NASHVILLE_LEGACY",
			PartyName: &ddexv1beta1.Party_PartyName{
				FullName: "Columbia Nashville Legacy",
			},
		},
	}

	// MASSIVE Resource collection from XML (46+ tracks + cover art!)
	// Live concert tracks (A1-A11): Mystery Train, Highwayman, Mammas Don't Let Your Babies...
	// Studio recordings (A12+): Help Me Make It Through the Night, Living Legend, etc.
	// Bob Dylan covers: One Too Many Mornings (A46)
	// Cover art: FrontCoverImage (A47) - 1400x1400 JPEG
	// Technical details: FLAC files, ISRC codes, MD5 hashes, durations, P-line info
	// TODO: Add proper Resource protobuf structure once oneof syntax is resolved
	resources := []*ddexv1beta1.Resource{}

	// Release information from XML - comprehensive album + individual track releases
	// Main Album: "Live - American Outlaws" (R0) - 2h43m total, Country genre
	// Individual track releases: R1-R44+ (each track as separate release)
	// UPC: 886445803518, Catalog: G0100035091829, Columbia Nashville Legacy
	// TODO: Add complete Release structure (currently simplified to main album only)
	releases := []*ddexv1beta1.Release{
		{
			Release: &ddexv1beta1.Release_MainRelease{
				MainRelease: &ddexv1beta1.Release_Release{
					ReleaseReference: "R0",
					ReleaseType:      "Album",
					ReleaseId: &ddexv1beta1.Release_ReleaseId{
						Grid:            "A10301A00035091829",
						Icpn:            "886445803518",
						CatalogueNumber: "G0100035091829",
					},
					DisplayTitleText: "Live - American Outlaws",
					DisplayTitle: &ddexv1beta1.Release_DisplayTitle{
						TitleText: "Live - American Outlaws",
					},
					DisplayArtistName: "The Highwaymen",
					DisplayArtist: []*ddexv1beta1.Release_DisplayArtist{
						{
							ArtistPartyReference: "P_ARTIST_1199281",
							DisplayArtistRole:    "MainArtist",
						},
					},
					ReleaseLabelReference: "P_LABEL_COLUMBIA_NASHVILLE_LEGACY",
					PLine: &ddexv1beta1.Release_Release_PLine{
						Year:      "2016",
						PLineText: "(P) 2016 Sony Music Entertainment",
					},
					Duration:            "PT2H43M8S", // 2 hours 43 minutes total!
					OriginalReleaseDate: "2016-05-20",
					ParentalWarningType: "NotExplicit",
					Genre: &ddexv1beta1.Release_Release_Genre{
						GenreText: "Country",
					},
				},
			},
		},
	}

	// Deal information from XML DealList (worldwide licensing for PermanentDownload)
	// Real XML contains extensive territory codes (US, CA, GB, DE, FR, JP, AU, etc.)
	// Commercial model: PayAsYouGoModel, UseType: PermanentDownload, Valid from 2016-05-20
	// TODO: Add proper Deal protobuf structure once schema is defined
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
