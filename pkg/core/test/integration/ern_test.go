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

	// Create DDEX v1beta2 ERN message with correct field structure
	testERN := &ddex.ElectronicReleaseNotification{
		Header: &ddex.DDEXMessageHeader{
			ControlType: ddex.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_NEW_MESSAGE,
			From:        "0x1234567890123456789012345678901234567890",
			To:          "0x0987654321098765432109876543210987654321",
			Nonce:       1,
		},
		PartyList: []*ddex.Party{
			{
				PartyReference: "P_ARTIST_8888888",
				PartyName: []*ddex.Party_PartyName{
					{
						LanguageAndScriptCode: "",
						FullName:              "The Cosmic Wanderers",
					},
					{
						LanguageAndScriptCode: "fr",
						FullName:              "Les Vagabonds Cosmiques",
					},
				},
				PartyId: &ddex.Party_PartyId{
					Dpid: "PADPIDA2024010101T",
				},
			},
			{
				PartyReference: "P_ARTIST_7777777",
				PartyName: []*ddex.Party_PartyName{
					{
						FullName: "Luna Rivers",
					},
				},
			},
			{
				PartyReference: "P_ARTIST_6666666",
				PartyName: []*ddex.Party_PartyName{
					{
						FullName: "Echo Stone",
					},
				},
			},
		},
		ResourceList: []*ddex.Resource{
			{
				Resource: &ddex.Resource_SoundRecording_{
					SoundRecording: &ddex.Resource_SoundRecording{
						ResourceReference: "A1",
						Type:              "MusicalWorkSoundRecording",
						ResourceId: &ddex.Resource_ResourceId{
							Isrc: "TEST12345001",
						},
						DisplayTitleText:      "Stardust Highway (Live at Festival Arena, Phoenix, AZ - October 2023)",
						DisplayArtistName:     "The Cosmic Wanderers, Luna Rivers, Echo Stone, Nova Black, Phoenix Wright",
						Duration:              "PT0H2M15S",
						FirstPublicationDate:  "2024-01-15",
						ParentalWarningType:   "NotExplicit",
						LanguageOfPerformance: "en",
					},
				},
			},
			{
				Resource: &ddex.Resource_SoundRecording_{
					SoundRecording: &ddex.Resource_SoundRecording{
						ResourceReference: "A2",
						Type:              "MusicalWorkSoundRecording",
						ResourceId: &ddex.Resource_ResourceId{
							Isrc: "TEST12345002",
						},
						DisplayTitleText:      "Galactic Dreams (Live at Festival Arena, Phoenix, AZ - October 2023)",
						DisplayArtistName:     "The Cosmic Wanderers, Luna Rivers, Echo Stone, Nova Black, Phoenix Wright",
						Duration:              "PT0H3M42S",
						FirstPublicationDate:  "2024-01-15",
						ParentalWarningType:   "NotExplicit",
						LanguageOfPerformance: "en",
					},
				},
			},
		},
		ReleaseList: []*ddex.Release{
			{
				Release: &ddex.Release_MainRelease_{
					MainRelease: &ddex.Release_MainRelease{
						ReleaseReference: "R0",
						ReleaseType:      "Album",
						ReleaseId: &ddex.Release_ReleaseId{
							Grid:                   "A10301T00042156789",
							Icpn:                   "123456789012",
							CatalogNumber:          "T0100042156789",
							CatalogNumberNamespace: "DPID:PADPIDA2024010101T",
						},
						DisplayTitleText:  "Live - Cosmic Festival Sessions",
						DisplayArtistName: "The Cosmic Wanderers",
					},
				},
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

	t.Logf("Transaction receipt: %v", receipt)

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

	// Verify the ERN address and other fields using the correct structure
	assert.NotEmpty(t, ernAck.ErnAddress)
	assert.Equal(t, uint64(1), ernAck.Nonce)

	// Verify addresses arrays are present
	assert.Len(t, ernAck.PartyAddresses, 3)    // Should have 3 parties
	assert.Len(t, ernAck.ResourceAddresses, 2) // Should have 2 resources
	assert.Len(t, ernAck.ReleaseAddresses, 1)  // Should have 1 release
	assert.Len(t, ernAck.DealAddresses, 0)     // Should have 0 deals

	t.Logf("Transaction receipt verified:")
	t.Logf("- ERN address: %s", ernAck.ErnAddress)
	t.Logf("- Party addresses: %v", ernAck.PartyAddresses)
	t.Logf("- Resource addresses: %v", ernAck.ResourceAddresses)
	t.Logf("- Release addresses: %v", ernAck.ReleaseAddresses)

	// Wait a moment for transaction to be processed
	time.Sleep(time.Second * 2)

	// Test GetERN functionality using the main ERN address from the receipt
	ernGetReq := &corev1.GetERNRequest{
		Address: ernAck.ErnAddress,
	}

	ernGetRes, err := sdk.Core.GetERN(ctx, connect.NewRequest(ernGetReq))
	assert.NoError(t, err)
	assert.NotNil(t, ernGetRes.Msg.Ern)

	retrievedERN := ernGetRes.Msg.Ern

	// Verify the retrieved ERN matches our original test data
	assert.Equal(t, testERN.Header.ControlType, retrievedERN.Header.ControlType)
	assert.Equal(t, testERN.Header.From, retrievedERN.Header.From)
	assert.Equal(t, testERN.Header.To, retrievedERN.Header.To)

	// Verify resource list
	assert.Len(t, retrievedERN.ResourceList, 2)
	assert.Equal(t, testERN.ResourceList[0].GetSoundRecording().ResourceReference, retrievedERN.ResourceList[0].GetSoundRecording().ResourceReference)
	assert.Equal(t, testERN.ResourceList[0].GetSoundRecording().ResourceId.Isrc, retrievedERN.ResourceList[0].GetSoundRecording().ResourceId.Isrc)
	assert.Equal(t, testERN.ResourceList[1].GetSoundRecording().ResourceReference, retrievedERN.ResourceList[1].GetSoundRecording().ResourceReference)
	assert.Equal(t, testERN.ResourceList[1].GetSoundRecording().ResourceId.Isrc, retrievedERN.ResourceList[1].GetSoundRecording().ResourceId.Isrc)

	// Verify release list
	assert.Len(t, retrievedERN.ReleaseList, 1)
	assert.Equal(t, testERN.ReleaseList[0].GetMainRelease().ReleaseReference, retrievedERN.ReleaseList[0].GetMainRelease().ReleaseReference)
	assert.Equal(t, testERN.ReleaseList[0].GetMainRelease().DisplayTitleText, retrievedERN.ReleaseList[0].GetMainRelease().DisplayTitleText)
	assert.Equal(t, testERN.ReleaseList[0].GetMainRelease().DisplayArtistName, retrievedERN.ReleaseList[0].GetMainRelease().DisplayArtistName)
	assert.Equal(t, testERN.ReleaseList[0].GetMainRelease().ReleaseId.Grid, retrievedERN.ReleaseList[0].GetMainRelease().ReleaseId.Grid)

	// Verify party list
	assert.Len(t, retrievedERN.PartyList, 3)
	assert.Equal(t, testERN.PartyList[0].PartyReference, retrievedERN.PartyList[0].PartyReference)
	assert.Equal(t, testERN.PartyList[0].PartyName[0].FullName, retrievedERN.PartyList[0].PartyName[0].FullName)

	t.Logf("Successfully retrieved ERN message for address: %s", ernAck.ErnAddress)
	t.Logf("Retrieved ERN contains same data as original:")
	t.Logf("- Message Control Type: %v", retrievedERN.Header.ControlType)
	t.Logf("- Album: %s by %s", retrievedERN.ReleaseList[0].GetMainRelease().DisplayTitleText, retrievedERN.ReleaseList[0].GetMainRelease().DisplayArtistName)
}

func TestMEADProcessing(t *testing.T) {
	ctx := context.Background()
	sdk := utils.DiscoveryOne

	nodeInfo, err := sdk.Core.GetNodeInfo(ctx, connect.NewRequest(&corev1.GetNodeInfoRequest{}))
	assert.NoError(t, err)
	chainId := nodeInfo.Msg.Chainid
	recentBlock := nodeInfo.Msg.CurrentHeight

	// Create DDEX v1beta2 MEAD message
	testMEAD := &ddex.MediaEnrichmentDescription{
		Header: &ddex.DDEXMessageHeader{
			ControlType: ddex.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_NEW_MESSAGE,
			From:        "0x1234567890123456789012345678901234567890",
			To:          "0x0987654321098765432109876543210987654321",
			Nonce:       1,
		},
		Metadata: []byte(`{"genre": "electronic", "bpm": 128, "key": "A minor"}`),
		ResourceAddresses: []string{
			"resource_addr_1",
			"resource_addr_2",
		},
		ReleaseAddresses: []string{
			"release_addr_1",
		},
		Mood: &ddex.Mood{
			Mood:       "energetic",
			Definition: "High energy, upbeat electronic track",
		},
	}

	// Create envelope with the MEAD message
	envelope := &corev1beta1.Envelope{
		Header: &corev1beta1.EnvelopeHeader{
			ChainId:    chainId,
			Expiration: recentBlock + 100,
			Nonce:      uuid.NewString(),
		},
		Messages: []*corev1beta1.Message{
			{
				Message: &corev1beta1.Message_Mead{
					Mead: testMEAD,
				},
			},
		},
	}

	// Create transaction
	transaction := &corev1beta1.Transaction{
		Signature: []byte("mock_signature_for_testing"),
		Envelope:  envelope,
	}

	// Send the MEAD transaction
	req := &corev1.SendTransactionRequest{
		Transactionv2: transaction,
	}

	submitRes, err := sdk.Core.SendTransaction(ctx, connect.NewRequest(req))
	assert.NoError(t, err)

	// Test the transaction receipt
	assert.NotNil(t, submitRes.Msg.TransactionReceipt)
	receipt := submitRes.Msg.TransactionReceipt

	// Verify MEAD acknowledgment
	meadReceipt := receipt.MessageReceipts[0]
	assert.Equal(t, int32(0), meadReceipt.MessageIndex)
	assert.NotNil(t, meadReceipt.GetMeadAck())

	meadAck := meadReceipt.GetMeadAck()
	assert.NotEmpty(t, meadAck.MeadAddress)
	assert.Equal(t, uint64(1), meadAck.Nonce)

	t.Logf("MEAD transaction processed successfully:")
	t.Logf("- MEAD address: %s", meadAck.MeadAddress)
	t.Logf("- Nonce: %d", meadAck.Nonce)
}

func TestPIEProcessing(t *testing.T) {
	ctx := context.Background()
	sdk := utils.DiscoveryOne

	nodeInfo, err := sdk.Core.GetNodeInfo(ctx, connect.NewRequest(&corev1.GetNodeInfoRequest{}))
	assert.NoError(t, err)
	chainId := nodeInfo.Msg.Chainid
	recentBlock := nodeInfo.Msg.CurrentHeight

	// Create DDEX v1beta2 PIE message
	testPIE := &ddex.PartyIdentificationEnrichment{
		Header: &ddex.DDEXMessageHeader{
			ControlType: ddex.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_NEW_MESSAGE,
			From:        "0x1234567890123456789012345678901234567890",
			To:          "0x0987654321098765432109876543210987654321",
			Nonce:       1,
		},
		Metadata: []byte(`{"artist_bio": "Independent electronic music producer", "location": "Berlin, Germany"}`),
		PartyAddresses: []string{
			"party_addr_1",
			"party_addr_2",
		},
		HandleList: []*ddex.Handle{
			{
				HandleType:  "audius",
				HandleValue: "cosmic_wanderers",
			},
			{
				HandleType:  "spotify",
				HandleValue: "thecosmicwanderers",
			},
			{
				HandleType:  "soundcloud",
				HandleValue: "cosmic-wanderers-official",
			},
		},
		Verified: &ddex.Verified{
			Verified: true,
		},
	}

	// Create envelope with the PIE message
	envelope := &corev1beta1.Envelope{
		Header: &corev1beta1.EnvelopeHeader{
			ChainId:    chainId,
			Expiration: recentBlock + 100,
			Nonce:      uuid.NewString(),
		},
		Messages: []*corev1beta1.Message{
			{
				Message: &corev1beta1.Message_Pie{
					Pie: testPIE,
				},
			},
		},
	}

	// Create transaction
	transaction := &corev1beta1.Transaction{
		Signature: []byte("mock_signature_for_testing"),
		Envelope:  envelope,
	}

	// Send the PIE transaction
	req := &corev1.SendTransactionRequest{
		Transactionv2: transaction,
	}

	submitRes, err := sdk.Core.SendTransaction(ctx, connect.NewRequest(req))
	assert.NoError(t, err)

	// Test the transaction receipt
	assert.NotNil(t, submitRes.Msg.TransactionReceipt)
	receipt := submitRes.Msg.TransactionReceipt

	// Verify PIE acknowledgment
	pieReceipt := receipt.MessageReceipts[0]
	assert.Equal(t, int32(0), pieReceipt.MessageIndex)
	assert.NotNil(t, pieReceipt.GetPieAck())

	pieAck := pieReceipt.GetPieAck()
	assert.NotEmpty(t, pieAck.PieAddress)
	assert.Equal(t, uint64(1), pieAck.Nonce)

	t.Logf("PIE transaction processed successfully:")
	t.Logf("- PIE address: %s", pieAck.PieAddress)
	t.Logf("- Nonce: %d", pieAck.Nonce)
}

func TestMultiMessageTransaction(t *testing.T) {
	ctx := context.Background()
	sdk := utils.DiscoveryOne

	nodeInfo, err := sdk.Core.GetNodeInfo(ctx, connect.NewRequest(&corev1.GetNodeInfoRequest{}))
	assert.NoError(t, err)
	chainId := nodeInfo.Msg.Chainid
	recentBlock := nodeInfo.Msg.CurrentHeight

	// Create a transaction with ERN, MEAD, and PIE messages
	testERN := &ddex.ElectronicReleaseNotification{
		Header: &ddex.DDEXMessageHeader{
			ControlType: ddex.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_NEW_MESSAGE,
			From:        "0x1234567890123456789012345678901234567890",
			To:          "",
			Nonce:       1,
		},
		PartyList: []*ddex.Party{
			{
				PartyReference: "P_ARTIST_TEST",
				PartyName: []*ddex.Party_PartyName{
					{
						FullName: "Test Artist",
					},
				},
			},
		},
		ResourceList: []*ddex.Resource{
			{
				Resource: &ddex.Resource_SoundRecording_{
					SoundRecording: &ddex.Resource_SoundRecording{
						ResourceReference: "A1",
						Type:              "MusicalWorkSoundRecording",
						ResourceId: &ddex.Resource_ResourceId{
							Isrc: "TEST12345001",
						},
						DisplayTitleText:  "Test Track",
						DisplayArtistName: "Test Artist",
						Duration:          "PT0H3M30S",
					},
				},
			},
		},
	}

	testMEAD := &ddex.MediaEnrichmentDescription{
		Header: &ddex.DDEXMessageHeader{
			ControlType: ddex.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_NEW_MESSAGE,
			From:        "0x1234567890123456789012345678901234567890",
			To:          "",
			Nonce:       2,
		},
		Metadata: []byte(`{"genre": "test"}`),
		Mood: &ddex.Mood{
			Mood:       "test",
			Definition: "Test mood",
		},
	}

	testPIE := &ddex.PartyIdentificationEnrichment{
		Header: &ddex.DDEXMessageHeader{
			ControlType: ddex.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_NEW_MESSAGE,
			From:        "0x1234567890123456789012345678901234567890",
			To:          "",
			Nonce:       3,
		},
		Metadata: []byte(`{"test": "data"}`),
		HandleList: []*ddex.Handle{
			{
				HandleType:  "audius",
				HandleValue: "test_artist",
			},
		},
	}

	// Create envelope with all three message types
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
			{
				Message: &corev1beta1.Message_Mead{
					Mead: testMEAD,
				},
			},
			{
				Message: &corev1beta1.Message_Pie{
					Pie: testPIE,
				},
			},
		},
	}

	// Create transaction
	transaction := &corev1beta1.Transaction{
		Signature: []byte("mock_signature_for_testing"),
		Envelope:  envelope,
	}

	// Send the multi-message transaction
	req := &corev1.SendTransactionRequest{
		Transactionv2: transaction,
	}

	submitRes, err := sdk.Core.SendTransaction(ctx, connect.NewRequest(req))
	assert.NoError(t, err)

	// Test the transaction receipt
	assert.NotNil(t, submitRes.Msg.TransactionReceipt)
	receipt := submitRes.Msg.TransactionReceipt

	// Should have 3 message receipts
	assert.Len(t, receipt.MessageReceipts, 3)
	assert.Equal(t, int32(3), receipt.EnvelopeInfo.MessageCount)

	// Check each message receipt
	ernReceipt := receipt.MessageReceipts[0]
	assert.Equal(t, int32(0), ernReceipt.MessageIndex)
	assert.NotNil(t, ernReceipt.GetErnAck())

	meadReceipt := receipt.MessageReceipts[1]
	assert.Equal(t, int32(1), meadReceipt.MessageIndex)
	assert.NotNil(t, meadReceipt.GetMeadAck())

	pieReceipt := receipt.MessageReceipts[2]
	assert.Equal(t, int32(2), pieReceipt.MessageIndex)
	assert.NotNil(t, pieReceipt.GetPieAck())

	t.Logf("Multi-message transaction processed successfully:")
	t.Logf("- ERN address: %s", ernReceipt.GetErnAck().ErnAddress)
	t.Logf("- MEAD address: %s", meadReceipt.GetMeadAck().MeadAddress)
	t.Logf("- PIE address: %s", pieReceipt.GetPieAck().PieAddress)
}

func TestGetMEAD(t *testing.T) {
	ctx := context.Background()
	sdk := utils.DiscoveryOne

	nodeInfo, err := sdk.Core.GetNodeInfo(ctx, connect.NewRequest(&corev1.GetNodeInfoRequest{}))
	assert.NoError(t, err)
	chainId := nodeInfo.Msg.Chainid
	recentBlock := nodeInfo.Msg.CurrentHeight

	// Create DDEX v1beta2 MEAD message
	testMEAD := &ddex.MediaEnrichmentDescription{
		Header: &ddex.DDEXMessageHeader{
			ControlType: ddex.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_NEW_MESSAGE,
			From:        "0x1234567890123456789012345678901234567890",
			To:          "0x0987654321098765432109876543210987654321",
			Nonce:       1,
		},
		Metadata: []byte(`{"genre": "ambient", "bpm": 85, "key": "D minor", "instruments": ["synthesizer", "pad", "reverb"]}`),
		ResourceAddresses: []string{
			"resource_addr_ambient_1",
			"resource_addr_ambient_2",
		},
		ReleaseAddresses: []string{
			"release_addr_ambient_album",
		},
		Mood: &ddex.Mood{
			Mood:       "contemplative",
			Definition: "Thoughtful and introspective ambient soundscape",
		},
	}

	// Create envelope with the MEAD message
	envelope := &corev1beta1.Envelope{
		Header: &corev1beta1.EnvelopeHeader{
			ChainId:    chainId,
			Expiration: recentBlock + 100,
			Nonce:      uuid.NewString(),
		},
		Messages: []*corev1beta1.Message{
			{
				Message: &corev1beta1.Message_Mead{
					Mead: testMEAD,
				},
			},
		},
	}

	// Create transaction
	transaction := &corev1beta1.Transaction{
		Signature: []byte("mock_signature_for_testing"),
		Envelope:  envelope,
	}

	// Send the MEAD transaction
	req := &corev1.SendTransactionRequest{
		Transactionv2: transaction,
	}

	submitRes, err := sdk.Core.SendTransaction(ctx, connect.NewRequest(req))
	assert.NoError(t, err)

	// Get MEAD address from receipt
	assert.NotNil(t, submitRes.Msg.TransactionReceipt)
	receipt := submitRes.Msg.TransactionReceipt
	meadReceipt := receipt.MessageReceipts[0]
	meadAck := meadReceipt.GetMeadAck()
	assert.NotEmpty(t, meadAck.MeadAddress)

	// Wait a moment for transaction to be processed
	time.Sleep(time.Second * 2)

	// Test GetMEAD functionality using the MEAD address from the receipt
	meadGetReq := &corev1.GetMEADRequest{
		Address: meadAck.MeadAddress,
	}

	meadGetRes, err := sdk.Core.GetMEAD(ctx, connect.NewRequest(meadGetReq))
	assert.NoError(t, err)
	assert.NotNil(t, meadGetRes.Msg.Mead)

	retrievedMEAD := meadGetRes.Msg.Mead

	// Verify the retrieved MEAD matches our original test data
	assert.Equal(t, testMEAD.Header.ControlType, retrievedMEAD.Header.ControlType)
	assert.Equal(t, testMEAD.Header.From, retrievedMEAD.Header.From)
	assert.Equal(t, testMEAD.Header.To, retrievedMEAD.Header.To)
	assert.Equal(t, testMEAD.Header.Nonce, retrievedMEAD.Header.Nonce)

	// Verify metadata
	assert.Equal(t, string(testMEAD.Metadata), string(retrievedMEAD.Metadata))

	// Verify resource addresses
	assert.Len(t, retrievedMEAD.ResourceAddresses, 2)
	assert.Equal(t, testMEAD.ResourceAddresses[0], retrievedMEAD.ResourceAddresses[0])
	assert.Equal(t, testMEAD.ResourceAddresses[1], retrievedMEAD.ResourceAddresses[1])

	// Verify release addresses
	assert.Len(t, retrievedMEAD.ReleaseAddresses, 1)
	assert.Equal(t, testMEAD.ReleaseAddresses[0], retrievedMEAD.ReleaseAddresses[0])

	// Verify mood
	assert.NotNil(t, retrievedMEAD.Mood)
	assert.Equal(t, testMEAD.Mood.Mood, retrievedMEAD.Mood.Mood)
	assert.Equal(t, testMEAD.Mood.Definition, retrievedMEAD.Mood.Definition)

	t.Logf("Successfully retrieved MEAD message for address: %s", meadAck.MeadAddress)
	t.Logf("Retrieved MEAD contains same data as original:")
	t.Logf("- Message Control Type: %v", retrievedMEAD.Header.ControlType)
	t.Logf("- Mood: %s - %s", retrievedMEAD.Mood.Mood, retrievedMEAD.Mood.Definition)
	t.Logf("- Metadata: %s", string(retrievedMEAD.Metadata))
}

func TestGetPIE(t *testing.T) {
	ctx := context.Background()
	sdk := utils.DiscoveryOne

	nodeInfo, err := sdk.Core.GetNodeInfo(ctx, connect.NewRequest(&corev1.GetNodeInfoRequest{}))
	assert.NoError(t, err)
	chainId := nodeInfo.Msg.Chainid
	recentBlock := nodeInfo.Msg.CurrentHeight

	// Create DDEX v1beta2 PIE message
	testPIE := &ddex.PartyIdentificationEnrichment{
		Header: &ddex.DDEXMessageHeader{
			ControlType: ddex.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_NEW_MESSAGE,
			From:        "0x1234567890123456789012345678901234567890",
			To:          "0x0987654321098765432109876543210987654321",
			Nonce:       1,
		},
		Metadata: []byte(`{"artist_bio": "Experimental electronic music collective from Berlin", "location": "Berlin, Germany", "formed": "2019", "members": 4}`),
		PartyAddresses: []string{
			"party_addr_collective_1",
			"party_addr_collective_2",
			"party_addr_collective_3",
		},
		HandleList: []*ddex.Handle{
			{
				HandleType:  "audius",
				HandleValue: "berlin_collective",
			},
			{
				HandleType:  "spotify",
				HandleValue: "berlin-electronic-collective",
			},
			{
				HandleType:  "soundcloud",
				HandleValue: "berlin-collective-official",
			},
			{
				HandleType:  "bandcamp",
				HandleValue: "berlinelectronic",
			},
		},
		Verified: &ddex.Verified{
			Verified: true,
		},
	}

	// Create envelope with the PIE message
	envelope := &corev1beta1.Envelope{
		Header: &corev1beta1.EnvelopeHeader{
			ChainId:    chainId,
			Expiration: recentBlock + 100,
			Nonce:      uuid.NewString(),
		},
		Messages: []*corev1beta1.Message{
			{
				Message: &corev1beta1.Message_Pie{
					Pie: testPIE,
				},
			},
		},
	}

	// Create transaction
	transaction := &corev1beta1.Transaction{
		Signature: []byte("mock_signature_for_testing"),
		Envelope:  envelope,
	}

	// Send the PIE transaction
	req := &corev1.SendTransactionRequest{
		Transactionv2: transaction,
	}

	submitRes, err := sdk.Core.SendTransaction(ctx, connect.NewRequest(req))
	assert.NoError(t, err)

	// Get PIE address from receipt
	assert.NotNil(t, submitRes.Msg.TransactionReceipt)
	receipt := submitRes.Msg.TransactionReceipt
	pieReceipt := receipt.MessageReceipts[0]
	pieAck := pieReceipt.GetPieAck()
	assert.NotEmpty(t, pieAck.PieAddress)

	// Wait a moment for transaction to be processed
	time.Sleep(time.Second * 2)

	// Test GetPIE functionality using the PIE address from the receipt
	pieGetReq := &corev1.GetPIERequest{
		Address: pieAck.PieAddress,
	}

	pieGetRes, err := sdk.Core.GetPIE(ctx, connect.NewRequest(pieGetReq))
	assert.NoError(t, err)
	assert.NotNil(t, pieGetRes.Msg.Pie)

	retrievedPIE := pieGetRes.Msg.Pie

	// Verify the retrieved PIE matches our original test data
	assert.Equal(t, testPIE.Header.ControlType, retrievedPIE.Header.ControlType)
	assert.Equal(t, testPIE.Header.From, retrievedPIE.Header.From)
	assert.Equal(t, testPIE.Header.To, retrievedPIE.Header.To)
	assert.Equal(t, testPIE.Header.Nonce, retrievedPIE.Header.Nonce)

	// Verify metadata
	assert.Equal(t, string(testPIE.Metadata), string(retrievedPIE.Metadata))

	// Verify party addresses
	assert.Len(t, retrievedPIE.PartyAddresses, 3)
	assert.Equal(t, testPIE.PartyAddresses[0], retrievedPIE.PartyAddresses[0])
	assert.Equal(t, testPIE.PartyAddresses[1], retrievedPIE.PartyAddresses[1])
	assert.Equal(t, testPIE.PartyAddresses[2], retrievedPIE.PartyAddresses[2])

	// Verify handle list
	assert.Len(t, retrievedPIE.HandleList, 4)
	for i, handle := range testPIE.HandleList {
		assert.Equal(t, handle.HandleType, retrievedPIE.HandleList[i].HandleType)
		assert.Equal(t, handle.HandleValue, retrievedPIE.HandleList[i].HandleValue)
	}

	// Verify verified status
	assert.NotNil(t, retrievedPIE.Verified)
	assert.Equal(t, testPIE.Verified.Verified, retrievedPIE.Verified.Verified)

	t.Logf("Successfully retrieved PIE message for address: %s", pieAck.PieAddress)
	t.Logf("Retrieved PIE contains same data as original:")
	t.Logf("- Message Control Type: %v", retrievedPIE.Header.ControlType)
	t.Logf("- Verified: %v", retrievedPIE.Verified.Verified)
	t.Logf("- Handles: %d platforms", len(retrievedPIE.HandleList))
	t.Logf("- Metadata: %s", string(retrievedPIE.Metadata))
}
