package models

import "encoding/xml"

type NewReleaseMessage struct {
	XMLName               xml.Name `xml:"ernm:NewReleaseMessage"`
	AvsVersionID          string   `xml:"AvsVersionId,attr"`
	LanguageAndScriptCode string   `xml:"LanguageAndScriptCode,attr"`

	MessageHeader MessageHeader
	PartyList     []*Party          `xml:"PartyList>Party"`
	ResourceList  []*SoundRecording `xml:"ResourceList>SoundRecording"`
	ReleaseList   []*Release        `xml:"ReleaseList>Release"`
	DealList      []*Deal           `xml:"DealList>Deal"`
}

type MessageHeader struct {
	MessageId              string
	MessageSender          PartyIdBlock
	MessageRecipient       PartyIdBlock
	MessageCreatedDateTime string
}

type Party struct {
	PartyReference string
	PartyName      PartyName
	PartyId        []PartyId
}

type SoundRecording struct {
	ResourceReference string
	Type              string
	ResourceId        PartyIdBlock
	// TechnicalDetails etc.
}

type Release struct {
	ReleaseReference string
	ReleaseId        ReleaseId
	TrackReleases    []TrackRelease `xml:"Tracks>TrackRelease"`
}

type TrackRelease struct {
	ResourceReference string
}

type Deal struct {
	DealReleaseReference    string
	ApplicableTerritoryCode string
	CommercialModelType     string
	UsageType               string
	// Nested ReleaseVisibility, etc.
}
