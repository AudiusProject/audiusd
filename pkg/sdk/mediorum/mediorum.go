package mediorum

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

type Mediorum struct {
	baseURL    string
	httpClient *http.Client
}

func New(baseURL string) *Mediorum {
	return &Mediorum{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Upload represents an upload response from mediorum
type Upload struct {
	ID                  string            `json:"id"`
	UserWallet          string            `json:"user_wallet"`
	Status              string            `json:"status"`
	Template            string            `json:"template"`
	OrigFileName        string            `json:"orig_file_name"`
	OrigFileCID         string            `json:"orig_file_cid"`
	TranscodeResults    map[string]string `json:"transcode_results"`
	CreatedBy           string            `json:"created_by"`
	CreatedAt           time.Time         `json:"created_at"`
	UpdatedAt           time.Time         `json:"updated_at"`
	Error               string            `json:"error,omitempty"`
	Mirrors             []string          `json:"mirrors,omitempty"`
	PlacementHosts      []string          `json:"placement_hosts,omitempty"`
	SelectedPreview     interface{}       `json:"selected_preview,omitempty"`
	FFProbe             interface{}       `json:"ffprobe,omitempty"`
	TranscodeProgress   float32           `json:"transcode_progress,omitempty"`
	AudioAnalysisStatus string            `json:"audio_analysis_status,omitempty"`
}

type UploadOptions struct {
	Template            string
	PreviewStartSeconds string
	PlacementHosts      string
}

func (m *Mediorum) UploadFile(file io.Reader, filename string, opts *UploadOptions) ([]*Upload, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("files", filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("failed to copy file: %w", err)
	}

	if opts != nil {
		if opts.Template != "" {
			if err := writer.WriteField("template", opts.Template); err != nil {
				return nil, fmt.Errorf("failed to write template field: %w", err)
			}
		}
		if opts.PreviewStartSeconds != "" {
			if err := writer.WriteField("previewStartSeconds", opts.PreviewStartSeconds); err != nil {
				return nil, fmt.Errorf("failed to write previewStartSeconds field: %w", err)
			}
		}
		if opts.PlacementHosts != "" {
			if err := writer.WriteField("placement_hosts", opts.PlacementHosts); err != nil {
				return nil, fmt.Errorf("failed to write placement_hosts field: %w", err)
			}
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/uploads", m.baseURL), body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusUnprocessableEntity {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, body)
	}

	var uploads []*Upload
	if err := json.NewDecoder(resp.Body).Decode(&uploads); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return uploads, nil
}

func (m *Mediorum) GetUpload(uploadID string) (*Upload, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/uploads/%s", m.baseURL, uploadID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, body)
	}

	var upload Upload
	if err := json.NewDecoder(resp.Body).Decode(&upload); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &upload, nil
}

func (m *Mediorum) ListUploads(after *time.Time) ([]Upload, error) {
	url := fmt.Sprintf("%s/uploads", m.baseURL)
	if after != nil {
		url = fmt.Sprintf("%s?after=%s", url, after.Format(time.RFC3339Nano))
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, body)
	}

	var uploads []Upload
	if err := json.NewDecoder(resp.Body).Decode(&uploads); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return uploads, nil
}

type StreamOptions struct {
	Signature string
	ID3       bool
	ID3Title  string
	ID3Artist string
}

func (m *Mediorum) StreamTrack(cid string, opts *StreamOptions) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s/tracks/cidstream/%s", m.baseURL, cid)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if opts != nil {
		q := req.URL.Query()
		if opts.Signature != "" {
			q.Add("signature", opts.Signature)
		}
		if opts.ID3 {
			q.Add("id3", "true")
			if opts.ID3Title != "" {
				q.Add("id3_title", opts.ID3Title)
			}
			if opts.ID3Artist != "" {
				q.Add("id3_artist", opts.ID3Artist)
			}
		}
		req.URL.RawQuery = q.Encode()
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, body)
	}

	return resp.Body, nil
}

func (m *Mediorum) GetBlob(cid string) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s/content/%s", m.baseURL, cid)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, body)
	}

	return resp.Body, nil
}
