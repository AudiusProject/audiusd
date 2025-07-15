# DDEX Electronic Release Notification (ERN) Protobuf Messages

This directory contains protobuf message definitions for handling DDEX Electronic Release Notifications (ERN). These messages are designed to consume XML data from DDEX ERN messages and convert them into structured protobuf format.

## Overview

DDEX (Digital Data Exchange) is a standards organization that develops data exchange formats for the music industry. The Electronic Release Notification (ERN) is one of their key message formats used for communicating music release information between record labels, distributors, and digital service providers.

## Message Structure

The main message types correspond to the XML structure:

```
NewReleaseMessage (root)
├── MessageHeader
├── PartyList (repeated Party)
├── ResourceList (repeated SoundRecording)
├── ReleaseList (repeated Release)
└── DealList
```

## XML to Protobuf Mapping

### Root Element Mapping

XML:
```xml
<ernm:NewReleaseMessage AvsVersionId="5" LanguageAndScriptCode="en">
```

Protobuf:
```go
message := &ddex.NewReleaseMessage{
    AvsVersionId:            "5",
    LanguageAndScriptCode:   "en",
    // ... other fields
}
```

### Party Mapping

XML:
```xml
<Party>
    <PartyReference>P_ARTIST_1199281</PartyReference>
    <PartyName>
        <FullName>The Highwaymen</FullName>
    </PartyName>
    <PartyName LanguageAndScriptCode="zh-Hant">
        <FullName>公路狂徒合唱團</FullName>
    </PartyName>
    <PartyId>
        <DPID>PADPIDA2007040502I</DPID>
    </PartyId>
</Party>
```

Protobuf:
```go
party := &ddex.Party{
    PartyReference: "P_ARTIST_1199281",
    PartyName: []*ddex.PartyName{
        {
            LanguageAndScriptCode: "",
            FullName:             "The Highwaymen",
        },
        {
            LanguageAndScriptCode: "zh-Hant",
            FullName:             "公路狂徒合唱團",
        },
    },
    PartyId: &ddex.PartyId{
        Dpid: "PADPIDA2007040502I",
    },
}
```

### SoundRecording Mapping

XML:
```xml
<SoundRecording>
    <ResourceReference>A1</ResourceReference>
    <Type>MusicalWorkSoundRecording</Type>
    <SoundRecordingEdition>
        <Type>NonImmersiveEdition</Type>
        <ResourceId>
            <ISRC>USG4X1500709</ISRC>
        </ResourceId>
        <PLine>
            <Year>1990</Year>
            <PLineText>(P) 1990 Sony Music Entertainment</PLineText>
        </PLine>
        <!-- TechnicalDetails, etc. -->
    </SoundRecordingEdition>
    <DisplayTitleText LanguageAndScriptCode="en">Mystery Train (Live)</DisplayTitleText>
    <Duration>PT0H1M32S</Duration>
    <!-- DisplayArtists, Contributors, etc. -->
</SoundRecording>
```

Protobuf:
```go
soundRecording := &ddex.SoundRecording{
    ResourceReference: "A1",
    Type:             "MusicalWorkSoundRecording",
    SoundRecordingEdition: &ddex.SoundRecordingEdition{
        Type: "NonImmersiveEdition",
        ResourceId: &ddex.ResourceId{
            Isrc: "USG4X1500709",
        },
        PLine: &ddex.PLine{
            Year:      1990,
            PLineText: "(P) 1990 Sony Music Entertainment",
        },
    },
    DisplayTitleText:        "Mystery Train (Live)",
    LanguageAndScriptCode:   "en",
    Duration:               "PT0H1M32S",
    // ... other fields
}
```

### Release Mapping

XML:
```xml
<Release>
    <ReleaseReference>R0</ReleaseReference>
    <ReleaseType>Album</ReleaseType>
    <ReleaseId>
        <GRid>A10301A00035091829</GRid>
        <ICPN>886445803518</ICPN>
        <CatalogNumber Namespace="DPID:PADPIDA2007040502I">G0100035091829</CatalogNumber>
    </ReleaseId>
    <DisplayTitleText LanguageAndScriptCode="en">Live - American Outlaws</DisplayTitleText>
    <!-- ResourceGroup with track listings -->
</Release>
```

Protobuf:
```go
release := &ddex.Release{
    ReleaseReference: "R0",
    ReleaseType:     "Album",
    ReleaseId: &ddex.ReleaseId{
        Grid: "A10301A00035091829",
        Icpn: "886445803518",
        CatalogNumber: &ddex.CatalogNumber{
            Namespace: "DPID:PADPIDA2007040502I",
            Value:     "G0100035091829",
        },
    },
    DisplayTitleText:      "Live - American Outlaws",
    LanguageAndScriptCode: "en",
    // ... other fields
}
```

## Usage Example

```go
// Example of processing a DDEX ERN message
func ProcessDDEXMessage(xmlData []byte) (*ddex.NewReleaseMessage, error) {
    // 1. Parse XML (using your preferred XML parser)
    // 2. Map XML elements to protobuf messages
    // 3. Return the protobuf message
    
    message := &ddex.NewReleaseMessage{
        AvsVersionId:          "5",
        LanguageAndScriptCode: "en",
        MessageHeader:         mapMessageHeader(xmlRoot.MessageHeader),
        PartyList:            mapParties(xmlRoot.PartyList),
        ResourceList:         mapSoundRecordings(xmlRoot.ResourceList),
        ReleaseList:          mapReleases(xmlRoot.ReleaseList),
        DealList:             mapDeals(xmlRoot.DealList),
    }
    
    return message, nil
}
```

## Field Notes

### Duration Fields
Duration fields use ISO 8601 duration format (e.g., `PT0H1M32S` for 1 minute 32 seconds).

### Date Fields
Date fields use ISO 8601 date format (e.g., `2016-05-20`).

### Language Codes
Language and script codes follow ISO standards (e.g., `en`, `zh-Hant`).

### References
Many fields use reference IDs to link related entities:
- `PartyReference` links to parties in the PartyList
- `ResourceReference` links to resources in the ResourceList
- `ReleaseReference` links to releases in the ReleaseList

### Extensibility
The protobuf messages are designed to be extensible. New fields can be added without breaking existing code, following protobuf's compatibility rules.

## Service Usage

Use the `DDEXService` gRPC service to:
- Process new release messages
- Validate DDEX messages
- Search and retrieve release/recording information

```go
// Example service call
response, err := client.ProcessNewReleaseMessage(ctx, &ddex.ProcessNewReleaseMessageRequest{
    Message: ddexMessage,
    Options: &ddex.ProcessingOptions{
        ValidateOnly: false,
        Namespace:   "your-namespace",
    },
})
``` 
