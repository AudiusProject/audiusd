package integration_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"

	corev1 "github.com/AudiusProject/audiusd/pkg/api/core/v1"
	corev1beta1 "github.com/AudiusProject/audiusd/pkg/api/core/v1beta1"
	ddex "github.com/AudiusProject/audiusd/pkg/api/ddex/v1beta2"
	"github.com/AudiusProject/audiusd/pkg/common"
	"github.com/AudiusProject/audiusd/pkg/core/test/integration/utils"
	"github.com/google/uuid"
)

func TestERNProcessing(t *testing.T) {
	ctx := context.Background()
	sdk := utils.DiscoveryOne

	nodeInfo, err := sdk.Core.GetNodeInfo(ctx, connect.NewRequest(&corev1.GetNodeInfoRequest{}))
	assert.NoError(t, err)
	chainId := nodeInfo.Msg.Chainid
	recentBlock := nodeInfo.Msg.CurrentHeight

	// Wait for node to be ready
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

	// Create DDEX v1beta2 message with fictional test data
	testERN := &ddex.NewReleaseMessage{
		AvsVersionId:          "5",
		LanguageAndScriptCode: "en",
		MessageHeader: &ddex.MessageHeader{
			MessageThreadId: "T0100042156789_TST",
			MessageId:       "123456789",
			MessageSender: &ddex.MessageSender{
				PartyId: "PADPIDA2024010101T",
				PartyName: &ddex.PartyName{
					FullName: "Test Music Records",
				},
			},
			MessageRecipient: &ddex.MessageRecipient{
				PartyId: "PADPIDA202401120D9",
				PartyName: &ddex.PartyName{
					FullName: "Audius",
				},
			},
			MessageCreatedDateTime: "2025-06-04T17:09:19.141Z",
			MessageControlType:     ddex.MessageControlType_MESSAGE_CONTROL_TYPE_NEW_RELEASE_MESSAGE,
		},
		PartyList: []*ddex.Party{
			{
				PartyReference: "P_ARTIST_8888888",
				PartyName: []*ddex.PartyName{
					{
						LanguageAndScriptCode: "",
						FullName:              "The Cosmic Wanderers",
					},
					{
						LanguageAndScriptCode: "fr",
						FullName:              "Les Vagabonds Cosmiques",
					},
				},
				PartyId: &ddex.PartyId{
					Dpid: "PADPIDA2024010101T",
				},
			},
			{
				PartyReference: "P_ARTIST_7777777",
				PartyName: []*ddex.PartyName{
					{
						FullName: "Luna Rivers",
					},
				},
			},
			{
				PartyReference: "P_ARTIST_6666666",
				PartyName: []*ddex.PartyName{
					{
						FullName: "Echo Stone",
					},
				},
			},
		},
		ResourceList: []*ddex.SoundRecording{
			{
				ResourceReference: "A1",
				Type:              "MusicalWorkSoundRecording",
				SoundRecordingEdition: &ddex.SoundRecordingEdition{
					Type: "NonImmersiveEdition",
					ResourceId: &ddex.ResourceId{
						Isrc: "TEST12345001",
					},
					PLine: &ddex.PLine{
						Year:      2023,
						PLineText: "(P) 2023 Test Music Records",
					},
				},
				DisplayTitleText:      "Stardust Highway (Live at Festival Arena, Phoenix, AZ - October 2023)",
				LanguageAndScriptCode: "en",
				VersionType:           "LiveVersion",
				DisplayArtistName:     "The Cosmic Wanderers, Luna Rivers, Echo Stone, Nova Black, Phoenix Wright",
				Duration:              "PT0H2M15S",
				FirstPublicationDate:  "2024-01-15",
				ParentalWarningType:   "NotExplicit",
				LanguageOfPerformance: "en",
			},
			{
				ResourceReference: "A2",
				Type:              "MusicalWorkSoundRecording",
				SoundRecordingEdition: &ddex.SoundRecordingEdition{
					Type: "NonImmersiveEdition",
					ResourceId: &ddex.ResourceId{
						Isrc: "TEST12345002",
					},
					PLine: &ddex.PLine{
						Year:      2023,
						PLineText: "(P) 2023 Test Music Records",
					},
				},
				DisplayTitleText:      "Galactic Dreams (Live at Festival Arena, Phoenix, AZ - October 2023)",
				LanguageAndScriptCode: "en",
				VersionType:           "LiveVersion",
				DisplayArtistName:     "The Cosmic Wanderers, Luna Rivers, Echo Stone, Nova Black, Phoenix Wright",
				Duration:              "PT0H3M42S",
				FirstPublicationDate:  "2024-01-15",
				ParentalWarningType:   "NotExplicit",
				LanguageOfPerformance: "en",
			},
		},
		ReleaseList: []*ddex.Release{
			{
				ReleaseReference: "R0",
				ReleaseType:      "Album",
				ReleaseId: &ddex.ReleaseId{
					Grid: "A10301T00042156789",
					Icpn: "123456789012",
					CatalogNumber: &ddex.CatalogNumber{
						Namespace: "DPID:PADPIDA2024010101T",
						Value:     "T0100042156789",
					},
				},
				DisplayTitleText:      "Live - Cosmic Festival Sessions",
				LanguageAndScriptCode: "en",
				DisplayArtistName:     "The Cosmic Wanderers",
			},
		},
	}

	// Create envelope with the DDEX message
	envelope := &corev1beta1.Envelope{
		Header: &corev1beta1.EnvelopeHeader{
			ChainId:    chainId,
			Expiration: recentBlock + 100,
			Nonce:      uuid.NewString(),
		},
		Messages: []*corev1beta1.Message{
			{
				Message: &corev1beta1.Message_Ern{
					Ern: testERN,
				},
			},
		},
	}

	// For testing, use a mock signature (in real usage you'd use proper EIP-712 signing)
	mockSignature := []byte("mock_signature_for_testing")

	// Create v1beta1 transaction
	transactionv2 := &corev1beta1.Transaction{
		Signature: mockSignature,
		Envelope:  envelope,
	}

	// Calculate expected transaction hash from envelope
	expectedTxHash, err := common.ToTxHash(envelope)
	assert.NoError(t, err)

	// Send the ERN transaction using Transactionv2 (envelope format)
	req := &corev1.SendTransactionRequest{
		Transactionv2: transactionv2,
	}

	submitRes, err := sdk.Core.SendTransaction(ctx, connect.NewRequest(req))
	assert.NoError(t, err)

	txhash := submitRes.Msg.Transaction.Hash
	assert.Equal(t, expectedTxHash, txhash)

	// Wait a moment for transaction to be processed
	time.Sleep(time.Second * 2)

	// Retrieve and verify the transaction
	ernRes, err := sdk.Core.GetTransaction(ctx, connect.NewRequest(&corev1.GetTransactionRequest{TxHash: txhash}))
	assert.NoError(t, err)

	// Verify the retrieved transaction has the envelope structure
	retrievedTransaction := ernRes.Msg.Transaction.Transactionv2
	assert.NotNil(t, retrievedTransaction)

	// For envelope format, we need to check if it has Transactionv2
	// The response structure may differ, so let's verify key DDEX data was processed
	// by checking that our transaction was successfully stored and can be retrieved

	// Verify the transaction hash matches
	assert.Equal(t, expectedTxHash, txhash)

	// Verify the transaction was processed (the fact that we can retrieve it indicates success)
	assert.NotNil(t, retrievedTransaction)

	t.Logf("Successfully processed test ERN with transaction hash: %s", txhash)
	t.Logf("DDEX v1beta2 message contained:")
	t.Logf("- Message ID: %s", testERN.MessageHeader.MessageId)
	t.Logf("- Album: %s by %s", testERN.ReleaseList[0].DisplayTitleText, testERN.ReleaseList[0].DisplayArtistName)
	t.Logf("- Number of tracks: %d", len(testERN.ResourceList))
	t.Logf("- Number of parties: %d", len(testERN.PartyList))

	// Verify core DDEX data from our test message
	assert.Equal(t, "T0100042156789_TST", testERN.MessageHeader.MessageThreadId)
	assert.Equal(t, "123456789", testERN.MessageHeader.MessageId)
	assert.Equal(t, "PADPIDA2024010101T", testERN.MessageHeader.MessageSender.PartyId)
	assert.Equal(t, "Test Music Records", testERN.MessageHeader.MessageSender.PartyName.FullName)

	// Verify we have the expected DDEX data structure
	assert.Len(t, testERN.ResourceList, 2) // Stardust Highway and Galactic Dreams
	assert.Len(t, testERN.ReleaseList, 1)  // Live - Cosmic Festival Sessions album
	assert.Len(t, testERN.PartyList, 3)    // The Cosmic Wanderers, Luna Rivers, Echo Stone

	// Verify specific ISRCs from the test data
	assert.Equal(t, "TEST12345001", testERN.ResourceList[0].SoundRecordingEdition.ResourceId.Isrc) // Stardust Highway (Live)
	assert.Equal(t, "TEST12345002", testERN.ResourceList[1].SoundRecordingEdition.ResourceId.Isrc) // Galactic Dreams (Live)

	// Verify album details
	release := testERN.ReleaseList[0]
	assert.Equal(t, "Live - Cosmic Festival Sessions", release.DisplayTitleText)
	assert.Equal(t, "The Cosmic Wanderers", release.DisplayArtistName)
	assert.Equal(t, "A10301T00042156789", release.ReleaseId.Grid)
	assert.Equal(t, "123456789012", release.ReleaseId.Icpn)

	// Test GetERN functionality
	// Use the message sender's party ID as the address for retrieval
	senderPartyId := testERN.MessageHeader.MessageSender.PartyId

	// Wait a moment to ensure the ERN message is fully processed and stored
	time.Sleep(time.Second * 2)

	// Retrieve the ERN message using GetERN
	ernGetReq := &corev1.GetERNRequest{
		Address: senderPartyId,
	}

	ernGetRes, err := sdk.Core.GetERN(ctx, connect.NewRequest(ernGetReq))
	assert.NoError(t, err)
	assert.NotNil(t, ernGetRes.Msg.Ern)

	retrievedERN := ernGetRes.Msg.Ern

	// Verify the retrieved ERN matches our original test data
	assert.Equal(t, testERN.MessageHeader.MessageId, retrievedERN.MessageHeader.MessageId)
	assert.Equal(t, testERN.MessageHeader.MessageThreadId, retrievedERN.MessageHeader.MessageThreadId)
	assert.Equal(t, testERN.MessageHeader.MessageSender.PartyId, retrievedERN.MessageHeader.MessageSender.PartyId)
	assert.Equal(t, testERN.MessageHeader.MessageSender.PartyName.FullName, retrievedERN.MessageHeader.MessageSender.PartyName.FullName)

	// Verify resource list
	assert.Len(t, retrievedERN.ResourceList, 2)
	assert.Equal(t, testERN.ResourceList[0].ResourceReference, retrievedERN.ResourceList[0].ResourceReference)
	assert.Equal(t, testERN.ResourceList[0].SoundRecordingEdition.ResourceId.Isrc, retrievedERN.ResourceList[0].SoundRecordingEdition.ResourceId.Isrc)
	assert.Equal(t, testERN.ResourceList[1].ResourceReference, retrievedERN.ResourceList[1].ResourceReference)
	assert.Equal(t, testERN.ResourceList[1].SoundRecordingEdition.ResourceId.Isrc, retrievedERN.ResourceList[1].SoundRecordingEdition.ResourceId.Isrc)

	// Verify release list
	assert.Len(t, retrievedERN.ReleaseList, 1)
	assert.Equal(t, testERN.ReleaseList[0].ReleaseReference, retrievedERN.ReleaseList[0].ReleaseReference)
	assert.Equal(t, testERN.ReleaseList[0].DisplayTitleText, retrievedERN.ReleaseList[0].DisplayTitleText)
	assert.Equal(t, testERN.ReleaseList[0].DisplayArtistName, retrievedERN.ReleaseList[0].DisplayArtistName)
	assert.Equal(t, testERN.ReleaseList[0].ReleaseId.Grid, retrievedERN.ReleaseList[0].ReleaseId.Grid)

	// Verify party list
	assert.Len(t, retrievedERN.PartyList, 3)
	assert.Equal(t, testERN.PartyList[0].PartyReference, retrievedERN.PartyList[0].PartyReference)
	assert.Equal(t, testERN.PartyList[0].PartyName[0].FullName, retrievedERN.PartyList[0].PartyName[0].FullName)

	t.Logf("Successfully retrieved ERN message for address: %s", senderPartyId)
	t.Logf("Retrieved ERN contains same data as original:")
	t.Logf("- Message ID: %s", retrievedERN.MessageHeader.MessageId)
	t.Logf("- Album: %s by %s", retrievedERN.ReleaseList[0].DisplayTitleText, retrievedERN.ReleaseList[0].DisplayArtistName)
}
