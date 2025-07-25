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
			To:          "",
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

	// *** BEGIN UPDATE TEST ***
	// Test ERN Update functionality

	// Create update ERN message with existing parties/resources/releases and add new ones
	updateERN := &ddex.ElectronicReleaseNotification{
		Address: ernAck.ErnAddress, // Set to the original ERN address
		Header: &ddex.DDEXMessageHeader{
			ControlType: ddex.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_UPDATED_MESSAGE,
			From:        "0x1234567890123456789012345678901234567890",
			To:          ernAck.ErnAddress, // Must match the address field
			Nonce:       2,                 // Increment nonce
		},
		// Keep existing parties with their addresses
		PartyList: []*ddex.Party{
			{
				Address:        ernAck.PartyAddresses[0], // Keep existing address
				PartyReference: "P_ARTIST_8888888",
				PartyName: []*ddex.Party_PartyName{
					{
						LanguageAndScriptCode: "",
						FullName:              "The Cosmic Wanderers (Updated)",
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
				Address:        ernAck.PartyAddresses[1], // Keep existing address
				PartyReference: "P_ARTIST_7777777",
				PartyName: []*ddex.Party_PartyName{
					{
						FullName: "Luna Rivers (Updated)",
					},
				},
			},
			// Add new party (will get new address)
			{
				PartyReference: "P_ARTIST_9999999",
				PartyName: []*ddex.Party_PartyName{
					{
						FullName: "Nova Phoenix",
					},
				},
			},
		},
		// Keep existing resources with their addresses
		ResourceList: []*ddex.Resource{
			{
				Address: ernAck.ResourceAddresses[0], // Keep existing address
				Resource: &ddex.Resource_SoundRecording_{
					SoundRecording: &ddex.Resource_SoundRecording{
						ResourceReference: "A1",
						Type:              "MusicalWorkSoundRecording",
						ResourceId: &ddex.Resource_ResourceId{
							Isrc: "TEST12345001",
						},
						DisplayTitleText:      "Stardust Highway (Live - Updated Mix)",
						DisplayArtistName:     "The Cosmic Wanderers, Luna Rivers, Echo Stone, Nova Black, Phoenix Wright",
						Duration:              "PT0H2M15S",
						FirstPublicationDate:  "2024-01-15",
						ParentalWarningType:   "NotExplicit",
						LanguageOfPerformance: "en",
					},
				},
			},
			// Add new resource (will get new address)
			{
				Resource: &ddex.Resource_SoundRecording_{
					SoundRecording: &ddex.Resource_SoundRecording{
						ResourceReference: "A3",
						Type:              "MusicalWorkSoundRecording",
						ResourceId: &ddex.Resource_ResourceId{
							Isrc: "TEST12345003",
						},
						DisplayTitleText:      "Interstellar Journey (Bonus Track)",
						DisplayArtistName:     "The Cosmic Wanderers",
						Duration:              "PT0H4M12S",
						FirstPublicationDate:  "2024-01-15",
						ParentalWarningType:   "NotExplicit",
						LanguageOfPerformance: "en",
					},
				},
			},
		},
		// Keep existing release with its address
		ReleaseList: []*ddex.Release{
			{
				Address: ernAck.ReleaseAddresses[0], // Keep existing address
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
						DisplayTitleText:  "Live - Cosmic Festival Sessions (Deluxe Edition)",
						DisplayArtistName: "The Cosmic Wanderers",
					},
				},
			},
		},
	}

	// Create envelope for update
	updateEnvelope := &corev1beta1.Envelope{
		Header: &corev1beta1.EnvelopeHeader{
			ChainId:    chainId,
			Expiration: recentBlock + 200,
			Nonce:      uuid.NewString(),
		},
		Messages: []*corev1beta1.Message{
			{
				Message: &corev1beta1.Message_Ern{
					Ern: updateERN,
				},
			},
		},
	}

	// Create update transaction
	updateTransaction := &corev1beta1.Transaction{
		Signature: mockSignature,
		Envelope:  updateEnvelope,
	}

	// Send the ERN update transaction
	updateReq := &corev1.SendTransactionRequest{
		Transactionv2: updateTransaction,
	}

	updateSubmitRes, err := sdk.Core.SendTransaction(ctx, connect.NewRequest(updateReq))
	assert.NoError(t, err)

	// Test the update transaction receipt
	assert.NotNil(t, updateSubmitRes.Msg.TransactionReceipt)
	updateReceipt := updateSubmitRes.Msg.TransactionReceipt

	// Verify the update ERN acknowledgment
	updateErnReceipt := updateReceipt.MessageReceipts[0]
	assert.Equal(t, int32(0), updateErnReceipt.MessageIndex)
	assert.NotNil(t, updateErnReceipt.GetErnAck())

	updateErnAck := updateErnReceipt.GetErnAck()

	// Verify the ERN address remains the same
	assert.Equal(t, ernAck.ErnAddress, updateErnAck.ErnAddress)
	assert.Equal(t, uint64(2), updateErnAck.Nonce) // Nonce should be incremented

	// Verify addresses arrays for update - should have more entries now
	assert.Len(t, updateErnAck.PartyAddresses, 3)    // Should have 3 parties (2 existing + 1 new)
	assert.Len(t, updateErnAck.ResourceAddresses, 2) // Should have 2 resources (1 existing + 1 new)
	assert.Len(t, updateErnAck.ReleaseAddresses, 1)  // Should have 1 release (same)

	// Verify that existing addresses are preserved
	assert.Equal(t, ernAck.PartyAddresses[0], updateErnAck.PartyAddresses[0])          // First party address same
	assert.Equal(t, ernAck.PartyAddresses[1], updateErnAck.PartyAddresses[1])          // Second party address same
	assert.NotEqual(t, ernAck.PartyAddresses[2], updateErnAck.PartyAddresses[2])       // Third party is new
	assert.Equal(t, ernAck.ResourceAddresses[0], updateErnAck.ResourceAddresses[0])    // First resource address same
	assert.NotEqual(t, ernAck.ResourceAddresses[1], updateErnAck.ResourceAddresses[1]) // Second resource is new
	assert.Equal(t, ernAck.ReleaseAddresses[0], updateErnAck.ReleaseAddresses[0])      // Release address same

	t.Logf("ERN Update transaction processed successfully:")
	t.Logf("- ERN address (unchanged): %s", updateErnAck.ErnAddress)
	t.Logf("- Updated party addresses: %v", updateErnAck.PartyAddresses)
	t.Logf("- Updated resource addresses: %v", updateErnAck.ResourceAddresses)
	t.Logf("- Updated release addresses: %v", updateErnAck.ReleaseAddresses)

	// Wait a moment for update transaction to be processed
	time.Sleep(time.Second * 2)

	// Test GetERN functionality for the updated ERN using the same address
	updatedErnGetRes, err := sdk.Core.GetERN(ctx, connect.NewRequest(ernGetReq))
	assert.NoError(t, err)
	assert.NotNil(t, updatedErnGetRes.Msg.Ern)

	updatedRetrievedERN := updatedErnGetRes.Msg.Ern

	// Verify the retrieved ERN shows the updated data
	assert.Equal(t, ddex.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_UPDATED_MESSAGE, updatedRetrievedERN.Header.ControlType)
	assert.Equal(t, updateERN.Header.From, updatedRetrievedERN.Header.From)
	assert.Equal(t, updateERN.Header.To, updatedRetrievedERN.Header.To)
	assert.Equal(t, uint64(2), updatedRetrievedERN.Header.Nonce) // Should be nonce 2

	// Verify updated party data
	assert.Len(t, updatedRetrievedERN.PartyList, 3)
	assert.Equal(t, "The Cosmic Wanderers (Updated)", updatedRetrievedERN.PartyList[0].PartyName[0].FullName)
	assert.Equal(t, "Luna Rivers (Updated)", updatedRetrievedERN.PartyList[1].PartyName[0].FullName)
	assert.Equal(t, "Nova Phoenix", updatedRetrievedERN.PartyList[2].PartyName[0].FullName)

	// Verify updated resource data
	assert.Len(t, updatedRetrievedERN.ResourceList, 2)
	assert.Equal(t, "Stardust Highway (Live - Updated Mix)", updatedRetrievedERN.ResourceList[0].GetSoundRecording().DisplayTitleText)
	assert.Equal(t, "Interstellar Journey (Bonus Track)", updatedRetrievedERN.ResourceList[1].GetSoundRecording().DisplayTitleText)

	// Verify updated release data
	assert.Len(t, updatedRetrievedERN.ReleaseList, 1)
	assert.Equal(t, "Live - Cosmic Festival Sessions (Deluxe Edition)", updatedRetrievedERN.ReleaseList[0].GetMainRelease().DisplayTitleText)

	t.Logf("Successfully retrieved updated ERN message for address: %s", ernAck.ErnAddress)
	t.Logf("Updated ERN contains modified data:")
	t.Logf("- Message Control Type: %v", updatedRetrievedERN.Header.ControlType)
	t.Logf("- Updated Album: %s by %s", updatedRetrievedERN.ReleaseList[0].GetMainRelease().DisplayTitleText, updatedRetrievedERN.ReleaseList[0].GetMainRelease().DisplayArtistName)
	t.Logf("- Updated parties: %s, %s, %s", updatedRetrievedERN.PartyList[0].PartyName[0].FullName, updatedRetrievedERN.PartyList[1].PartyName[0].FullName, updatedRetrievedERN.PartyList[2].PartyName[0].FullName)
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
			To:          "",
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

	// Wait for transaction to be processed
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

	// *** BEGIN UPDATE TEST ***
	// Test MEAD Update functionality

	// Create update MEAD message
	updateMEAD := &ddex.MediaEnrichmentDescription{
		Address: meadAck.MeadAddress, // Set to the original MEAD address
		Header: &ddex.DDEXMessageHeader{
			ControlType: ddex.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_UPDATED_MESSAGE,
			From:        "0x1234567890123456789012345678901234567890",
			To:          meadAck.MeadAddress, // Must match the address field
			Nonce:       2,                   // Increment nonce
		},
		Metadata: []byte(`{"genre": "electronic", "bpm": 128, "key": "A minor", "updated": true, "remix": "extended"}`),
		ResourceAddresses: []string{
			"resource_addr_1",
			"resource_addr_2",
			"resource_addr_3", // Add new resource address
		},
		ReleaseAddresses: []string{
			"release_addr_1",
			"release_addr_2", // Add new release address
		},
		Mood: &ddex.Mood{
			Mood:       "euphoric",
			Definition: "High energy, upbeat electronic track with euphoric elements",
		},
	}

	// Create envelope for update
	updateEnvelope := &corev1beta1.Envelope{
		Header: &corev1beta1.EnvelopeHeader{
			ChainId:    chainId,
			Expiration: recentBlock + 200,
			Nonce:      uuid.NewString(),
		},
		Messages: []*corev1beta1.Message{
			{
				Message: &corev1beta1.Message_Mead{
					Mead: updateMEAD,
				},
			},
		},
	}

	// Create update transaction
	updateTransaction := &corev1beta1.Transaction{
		Signature: []byte("mock_signature_for_testing"),
		Envelope:  updateEnvelope,
	}

	// Send the MEAD update transaction
	updateReq := &corev1.SendTransactionRequest{
		Transactionv2: updateTransaction,
	}

	updateSubmitRes, err := sdk.Core.SendTransaction(ctx, connect.NewRequest(updateReq))
	assert.NoError(t, err)

	// Test the update transaction receipt
	assert.NotNil(t, updateSubmitRes.Msg.TransactionReceipt)
	updateReceipt := updateSubmitRes.Msg.TransactionReceipt

	// Verify the update MEAD acknowledgment
	updateMeadReceipt := updateReceipt.MessageReceipts[0]
	assert.Equal(t, int32(0), updateMeadReceipt.MessageIndex)
	assert.NotNil(t, updateMeadReceipt.GetMeadAck())

	updateMeadAck := updateMeadReceipt.GetMeadAck()

	// Verify the MEAD address remains the same
	assert.Equal(t, meadAck.MeadAddress, updateMeadAck.MeadAddress)
	assert.Equal(t, uint64(2), updateMeadAck.Nonce) // Nonce should be incremented

	t.Logf("MEAD Update transaction processed successfully:")
	t.Logf("- MEAD address (unchanged): %s", updateMeadAck.MeadAddress)
	t.Logf("- Updated nonce: %d", updateMeadAck.Nonce)

	// Wait a moment for update transaction to be processed
	time.Sleep(time.Second * 2)

	// Test GetMEAD functionality for the updated MEAD using the same address
	updatedMeadGetRes, err := sdk.Core.GetMEAD(ctx, connect.NewRequest(meadGetReq))
	assert.NoError(t, err)
	assert.NotNil(t, updatedMeadGetRes.Msg.Mead)

	updatedRetrievedMEAD := updatedMeadGetRes.Msg.Mead

	// Verify the retrieved MEAD shows the updated data
	assert.Equal(t, ddex.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_UPDATED_MESSAGE, updatedRetrievedMEAD.Header.ControlType)
	assert.Equal(t, updateMEAD.Header.From, updatedRetrievedMEAD.Header.From)
	assert.Equal(t, updateMEAD.Header.To, updatedRetrievedMEAD.Header.To)
	assert.Equal(t, uint64(2), updatedRetrievedMEAD.Header.Nonce) // Should be nonce 2

	// Verify updated metadata
	assert.Equal(t, string(updateMEAD.Metadata), string(updatedRetrievedMEAD.Metadata))

	// Verify updated resource addresses
	assert.Len(t, updatedRetrievedMEAD.ResourceAddresses, 3)
	assert.Equal(t, updateMEAD.ResourceAddresses[0], updatedRetrievedMEAD.ResourceAddresses[0])
	assert.Equal(t, updateMEAD.ResourceAddresses[1], updatedRetrievedMEAD.ResourceAddresses[1])
	assert.Equal(t, updateMEAD.ResourceAddresses[2], updatedRetrievedMEAD.ResourceAddresses[2])

	// Verify updated release addresses
	assert.Len(t, updatedRetrievedMEAD.ReleaseAddresses, 2)
	assert.Equal(t, updateMEAD.ReleaseAddresses[0], updatedRetrievedMEAD.ReleaseAddresses[0])
	assert.Equal(t, updateMEAD.ReleaseAddresses[1], updatedRetrievedMEAD.ReleaseAddresses[1])

	// Verify updated mood
	assert.NotNil(t, updatedRetrievedMEAD.Mood)
	assert.Equal(t, updateMEAD.Mood.Mood, updatedRetrievedMEAD.Mood.Mood)
	assert.Equal(t, updateMEAD.Mood.Definition, updatedRetrievedMEAD.Mood.Definition)

	t.Logf("Successfully retrieved updated MEAD message for address: %s", meadAck.MeadAddress)
	t.Logf("Updated MEAD contains modified data:")
	t.Logf("- Message Control Type: %v", updatedRetrievedMEAD.Header.ControlType)
	t.Logf("- Updated Mood: %s - %s", updatedRetrievedMEAD.Mood.Mood, updatedRetrievedMEAD.Mood.Definition)
	t.Logf("- Updated Metadata: %s", string(updatedRetrievedMEAD.Metadata))
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
			To:          "",
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

	// Wait for transaction to be processed
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

	// *** BEGIN UPDATE TEST ***
	// Test PIE Update functionality

	// Create update PIE message
	updatePIE := &ddex.PartyIdentificationEnrichment{
		Address: pieAck.PieAddress, // Set to the original PIE address
		Header: &ddex.DDEXMessageHeader{
			ControlType: ddex.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_UPDATED_MESSAGE,
			From:        "0x1234567890123456789012345678901234567890",
			To:          pieAck.PieAddress, // Must match the address field
			Nonce:       2,                 // Increment nonce
		},
		Metadata: []byte(`{"artist_bio": "Independent electronic music producer from Berlin", "location": "Berlin, Germany", "updated": true, "genres": ["electronic", "ambient", "techno"]}`),
		PartyAddresses: []string{
			"party_addr_1",
			"party_addr_2",
			"party_addr_3", // Add new party address
		},
		HandleList: []*ddex.Handle{
			{
				HandleType:  "audius",
				HandleValue: "cosmic_wanderers_official",
			},
			{
				HandleType:  "spotify",
				HandleValue: "thecosmicwanderers",
			},
			{
				HandleType:  "soundcloud",
				HandleValue: "cosmic-wanderers-official",
			},
			{
				HandleType:  "youtube",
				HandleValue: "cosmic_wanderers_music",
			},
		},
		Verified: &ddex.Verified{
			Verified: true,
		},
	}

	// Create envelope for update
	updateEnvelope := &corev1beta1.Envelope{
		Header: &corev1beta1.EnvelopeHeader{
			ChainId:    chainId,
			Expiration: recentBlock + 200,
			Nonce:      uuid.NewString(),
		},
		Messages: []*corev1beta1.Message{
			{
				Message: &corev1beta1.Message_Pie{
					Pie: updatePIE,
				},
			},
		},
	}

	// Create update transaction
	updateTransaction := &corev1beta1.Transaction{
		Signature: []byte("mock_signature_for_testing"),
		Envelope:  updateEnvelope,
	}

	// Send the PIE update transaction
	updateReq := &corev1.SendTransactionRequest{
		Transactionv2: updateTransaction,
	}

	updateSubmitRes, err := sdk.Core.SendTransaction(ctx, connect.NewRequest(updateReq))
	assert.NoError(t, err)

	// Test the update transaction receipt
	assert.NotNil(t, updateSubmitRes.Msg.TransactionReceipt)
	updateReceipt := updateSubmitRes.Msg.TransactionReceipt

	// Verify the update PIE acknowledgment
	updatePieReceipt := updateReceipt.MessageReceipts[0]
	assert.Equal(t, int32(0), updatePieReceipt.MessageIndex)
	assert.NotNil(t, updatePieReceipt.GetPieAck())

	updatePieAck := updatePieReceipt.GetPieAck()

	// Verify the PIE address remains the same
	assert.Equal(t, pieAck.PieAddress, updatePieAck.PieAddress)
	assert.Equal(t, uint64(2), updatePieAck.Nonce) // Nonce should be incremented

	t.Logf("PIE Update transaction processed successfully:")
	t.Logf("- PIE address (unchanged): %s", updatePieAck.PieAddress)
	t.Logf("- Updated nonce: %d", updatePieAck.Nonce)

	// Wait a moment for update transaction to be processed
	time.Sleep(time.Second * 2)

	// Test GetPIE functionality for the updated PIE using the same address
	updatedPieGetRes, err := sdk.Core.GetPIE(ctx, connect.NewRequest(pieGetReq))
	assert.NoError(t, err)
	assert.NotNil(t, updatedPieGetRes.Msg.Pie)

	updatedRetrievedPIE := updatedPieGetRes.Msg.Pie

	// Verify the retrieved PIE shows the updated data
	assert.Equal(t, ddex.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_UPDATED_MESSAGE, updatedRetrievedPIE.Header.ControlType)
	assert.Equal(t, updatePIE.Header.From, updatedRetrievedPIE.Header.From)
	assert.Equal(t, updatePIE.Header.To, updatedRetrievedPIE.Header.To)
	assert.Equal(t, uint64(2), updatedRetrievedPIE.Header.Nonce) // Should be nonce 2

	// Verify updated metadata
	assert.Equal(t, string(updatePIE.Metadata), string(updatedRetrievedPIE.Metadata))

	// Verify updated party addresses
	assert.Len(t, updatedRetrievedPIE.PartyAddresses, 3)
	assert.Equal(t, updatePIE.PartyAddresses[0], updatedRetrievedPIE.PartyAddresses[0])
	assert.Equal(t, updatePIE.PartyAddresses[1], updatedRetrievedPIE.PartyAddresses[1])
	assert.Equal(t, updatePIE.PartyAddresses[2], updatedRetrievedPIE.PartyAddresses[2])

	// Verify updated handle list
	assert.Len(t, updatedRetrievedPIE.HandleList, 4)
	for i, handle := range updatePIE.HandleList {
		assert.Equal(t, handle.HandleType, updatedRetrievedPIE.HandleList[i].HandleType)
		assert.Equal(t, handle.HandleValue, updatedRetrievedPIE.HandleList[i].HandleValue)
	}

	// Verify verified status
	assert.NotNil(t, updatedRetrievedPIE.Verified)
	assert.Equal(t, updatePIE.Verified.Verified, updatedRetrievedPIE.Verified.Verified)

	t.Logf("Successfully retrieved updated PIE message for address: %s", pieAck.PieAddress)
	t.Logf("Updated PIE contains modified data:")
	t.Logf("- Message Control Type: %v", updatedRetrievedPIE.Header.ControlType)
	t.Logf("- Verified: %v", updatedRetrievedPIE.Verified.Verified)
	t.Logf("- Updated Handles: %d platforms", len(updatedRetrievedPIE.HandleList))
	t.Logf("- Updated Metadata: %s", string(updatedRetrievedPIE.Metadata))
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
			Nonce:       1,
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
			Nonce:       1,
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
			To:          "",
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
			To:          "",
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

// Validation Tests

func TestERNValidationErrors(t *testing.T) {
	ctx := context.Background()
	sdk := utils.DiscoveryOne

	nodeInfo, err := sdk.Core.GetNodeInfo(ctx, connect.NewRequest(&corev1.GetNodeInfoRequest{}))
	assert.NoError(t, err)
	chainId := nodeInfo.Msg.Chainid
	recentBlock := nodeInfo.Msg.CurrentHeight

	tests := []struct {
		name          string
		ernModifier   func(*ddex.ElectronicReleaseNotification)
		expectedError string
	}{
		{
			name: "ERN address not empty",
			ernModifier: func(ern *ddex.ElectronicReleaseNotification) {
				ern.Address = "should_be_empty"
			},
			expectedError: "ERN address is not empty",
		},
		{
			name: "ERN from address empty",
			ernModifier: func(ern *ddex.ElectronicReleaseNotification) {
				ern.Header.From = ""
			},
			expectedError: "ERN from address is empty",
		},
		{
			name: "ERN to address not empty",
			ernModifier: func(ern *ddex.ElectronicReleaseNotification) {
				ern.Header.To = "should_be_empty"
			},
			expectedError: "ERN to address is not empty",
		},
		{
			name: "ERN nonce not one",
			ernModifier: func(ern *ddex.ElectronicReleaseNotification) {
				ern.Header.Nonce = 2
			},
			expectedError: "ERN nonce is not one",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create base ERN message
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
			}

			// Apply the modification that should cause validation error
			tt.ernModifier(testERN)

			// Create envelope
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

			// Create transaction
			transaction := &corev1beta1.Transaction{
				Signature: []byte("mock_signature_for_testing"),
				Envelope:  envelope,
			}

			// Send transaction and expect error
			req := &corev1.SendTransactionRequest{
				Transactionv2: transaction,
			}

			_, err := sdk.Core.SendTransaction(ctx, connect.NewRequest(req))
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestMEADValidationErrors(t *testing.T) {
	ctx := context.Background()
	sdk := utils.DiscoveryOne

	nodeInfo, err := sdk.Core.GetNodeInfo(ctx, connect.NewRequest(&corev1.GetNodeInfoRequest{}))
	assert.NoError(t, err)
	chainId := nodeInfo.Msg.Chainid
	recentBlock := nodeInfo.Msg.CurrentHeight

	tests := []struct {
		name          string
		meadModifier  func(*ddex.MediaEnrichmentDescription)
		expectedError string
	}{
		{
			name: "MEAD address not empty",
			meadModifier: func(mead *ddex.MediaEnrichmentDescription) {
				mead.Address = "should_be_empty"
			},
			expectedError: "MEAD address is not empty",
		},
		{
			name: "MEAD from address empty",
			meadModifier: func(mead *ddex.MediaEnrichmentDescription) {
				mead.Header.From = ""
			},
			expectedError: "MEAD from address is empty",
		},
		{
			name: "MEAD to address not empty",
			meadModifier: func(mead *ddex.MediaEnrichmentDescription) {
				mead.Header.To = "should_be_empty"
			},
			expectedError: "MEAD to address is not empty",
		},
		{
			name: "MEAD nonce not one",
			meadModifier: func(mead *ddex.MediaEnrichmentDescription) {
				mead.Header.Nonce = 2
			},
			expectedError: "MEAD nonce is not one",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create base MEAD message
			testMEAD := &ddex.MediaEnrichmentDescription{
				Header: &ddex.DDEXMessageHeader{
					ControlType: ddex.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_NEW_MESSAGE,
					From:        "0x1234567890123456789012345678901234567890",
					To:          "",
					Nonce:       1,
				},
				Metadata: []byte(`{"genre": "test"}`),
			}

			// Apply the modification that should cause validation error
			tt.meadModifier(testMEAD)

			// Create envelope
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

			// Send transaction and expect error
			req := &corev1.SendTransactionRequest{
				Transactionv2: transaction,
			}

			_, err := sdk.Core.SendTransaction(ctx, connect.NewRequest(req))
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestPIEValidationErrors(t *testing.T) {
	ctx := context.Background()
	sdk := utils.DiscoveryOne

	nodeInfo, err := sdk.Core.GetNodeInfo(ctx, connect.NewRequest(&corev1.GetNodeInfoRequest{}))
	assert.NoError(t, err)
	chainId := nodeInfo.Msg.Chainid
	recentBlock := nodeInfo.Msg.CurrentHeight

	tests := []struct {
		name          string
		pieModifier   func(*ddex.PartyIdentificationEnrichment)
		expectedError string
	}{
		{
			name: "PIE address not empty",
			pieModifier: func(pie *ddex.PartyIdentificationEnrichment) {
				pie.Address = "should_be_empty"
			},
			expectedError: "PIE address is not empty",
		},
		{
			name: "PIE from address empty",
			pieModifier: func(pie *ddex.PartyIdentificationEnrichment) {
				pie.Header.From = ""
			},
			expectedError: "PIE from address is empty",
		},
		{
			name: "PIE to address not empty",
			pieModifier: func(pie *ddex.PartyIdentificationEnrichment) {
				pie.Header.To = "should_be_empty"
			},
			expectedError: "PIE to address is not empty",
		},
		{
			name: "PIE nonce not one",
			pieModifier: func(pie *ddex.PartyIdentificationEnrichment) {
				pie.Header.Nonce = 2
			},
			expectedError: "PIE nonce is not one",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create base PIE message
			testPIE := &ddex.PartyIdentificationEnrichment{
				Header: &ddex.DDEXMessageHeader{
					ControlType: ddex.DDEXMessageControlType_DDEX_MESSAGE_CONTROL_TYPE_NEW_MESSAGE,
					From:        "0x1234567890123456789012345678901234567890",
					To:          "",
					Nonce:       1,
				},
				Metadata: []byte(`{"test": "data"}`),
			}

			// Apply the modification that should cause validation error
			tt.pieModifier(testPIE)

			// Create envelope
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

			// Send transaction and expect error
			req := &corev1.SendTransactionRequest{
				Transactionv2: transaction,
			}

			_, err := sdk.Core.SendTransaction(ctx, connect.NewRequest(req))
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}
