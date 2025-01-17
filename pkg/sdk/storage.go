package sdk

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/AudiusProject/audiusd/pkg/mediorum/server"
)

type StorageSDK struct {
	nodeURL string
	httpClient *http.Client
}

func NewStorageSDK(nodeURL string) *StorageSDK {
	return &StorageSDK{
		nodeURL: nodeURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *StorageSDK) GetNodeURL() string {
	return s.nodeURL
}

func (s *StorageSDK) SetNodeURL(nodeURL string) {
	s.nodeURL = nodeURL
}

func (s *StorageSDK) GetHealth() (*server.HealthCheckResponse, error) {
	req, err := http.NewRequest("GET", s.nodeURL+"/health_check", nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var healthCheckResponse server.HealthCheckResponse
	err = json.Unmarshal(body, &healthCheckResponse)
	if err != nil {
		return nil, err
	}

	return &healthCheckResponse, nil
}


func (s *StorageSDK) UploadAudio() (*server.Upload, error) {
	return nil, nil
}

func (s *StorageSDK) GetAudio() (*server.Upload, error) {
	return nil, nil
}
