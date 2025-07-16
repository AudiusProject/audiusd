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

	// Test the transaction receipt functionality
	assert.NotNil(t, submitRes.Msg.TransactionReceipt)
	receipt := submitRes.Msg.TransactionReceipt

	// Verify basic receipt fields
	assert.Equal(t, expectedTxHash, receipt.TxHash)
	assert.Equal(t, chainId, receipt.EnvelopeInfo.ChainId)
	assert.Equal(t, int32(1), receipt.EnvelopeInfo.MessageCount)
	assert.Len(t, receipt.MessageReceipts, 1)

	// Verify the ERN acknowledgment
	ernReceipt := receipt.MessageReceipts[0]
	assert.Equal(t, int32(0), ernReceipt.MessageIndex)
	assert.NotNil(t, ernReceipt.GetErnAck())

	ernAck := ernReceipt.GetErnAck()

	// Verify the main ERN address is present
	assert.NotNil(t, ernAck.ReleaseAddress)
	assert.NotEmpty(t, ernAck.ReleaseAddress.Address)
	assert.Equal(t, uint32(0), ernAck.ReleaseAddress.Index)

	// Verify sound recording addresses are present (should have 2 tracks)
	assert.Len(t, ernAck.SoundRecordingAddresses, 2)
	for i, srAddr := range ernAck.SoundRecordingAddresses {
		assert.NotEmpty(t, srAddr.Address)
		assert.Equal(t, uint32(i), srAddr.Index)
	}

	// Verify release addresses are in party addresses
	assert.Len(t, ernAck.PartyAddresses, 1) // Should have 1 release
	assert.NotEmpty(t, ernAck.PartyAddresses[0].Address)

	t.Logf("Transaction receipt verified:")
	t.Logf("- ERN address: %s", ernAck.ReleaseAddress.Address)
	t.Logf("- Sound recording addresses: %v", func() []string {
		addrs := make([]string, len(ernAck.SoundRecordingAddresses))
		for i, addr := range ernAck.SoundRecordingAddresses {
			addrs[i] = addr.Address
		}
		return addrs
	}())
	t.Logf("- Release addresses: %v", func() []string {
		addrs := make([]string, len(ernAck.PartyAddresses))
		for i, addr := range ernAck.PartyAddresses {
			addrs[i] = addr.Address
		}
		return addrs
	}())

	// Wait a moment for transaction to be processed
	time.Sleep(time.Second * 2)

	// Test GetERN functionality using the main ERN address from the receipt
	ernGetReq := &corev1.GetERNRequest{
		Address: ernAck.ReleaseAddress.Address,
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

	t.Logf("Successfully retrieved ERN message for address: %s", ernAck.ReleaseAddress.Address)
	t.Logf("Retrieved ERN contains same data as original:")
	t.Logf("- Message ID: %s", retrievedERN.MessageHeader.MessageId)
	t.Logf("- Album: %s by %s", retrievedERN.ReleaseList[0].DisplayTitleText, retrievedERN.ReleaseList[0].DisplayArtistName)
}
