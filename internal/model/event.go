package model

import "time"

const SchemaVersion = "v1"

// PullEvent is the top-level event sent over SSE and emitted as a structured log line.
type PullEvent struct {
	SchemaVersion string      `json:"schemaVersion"`
	Timestamp     time.Time   `json:"timestamp"`
	Type          EventType   `json:"type"`
	NodeName      string      `json:"nodeName"`
	Pull          *PullStatus `json:"pull,omitempty"`
}

type EventType string

const (
	EventPullProgress  EventType = "pull.progress"
	EventPullCompleted EventType = "pull.completed"
)

// PullStatus describes the current state of an image pull.
type PullStatus struct {
	ID              string           `json:"id"`
	NodeName        string           `json:"nodeName,omitempty"`
	ImageRef        string           `json:"imageRef"`
	TotalBytes      int64            `json:"totalBytes"`
	DownloadedBytes int64            `json:"downloadedBytes"`
	BytesPerSec     float64          `json:"bytesPerSec"`
	ETASeconds      float64          `json:"etaSeconds,omitempty"`
	Percent         float64          `json:"percent"`
	LayerCount      int              `json:"layerCount"`
	LayersDone      int              `json:"layersDone"`
	StartedAt       time.Time        `json:"startedAt"`
	CompletedAt     *time.Time       `json:"completedAt,omitempty"`
	Error           string           `json:"error,omitempty"`
	Pods            []PodCorrelation `json:"pods,omitempty"`
	Layers          []LayerStatus    `json:"layers,omitempty"`
	TotalKnown      bool             `json:"totalKnown"`
}

// LayerStatus describes a single layer download.
type LayerStatus struct {
	PullID          string     `json:"pullId"`
	Digest          string     `json:"digest"`
	MediaType       string     `json:"mediaType,omitempty"`
	TotalBytes      int64      `json:"totalBytes"`
	DownloadedBytes int64      `json:"downloadedBytes"`
	BytesPerSec     float64    `json:"bytesPerSec"`
	Percent         float64    `json:"percent"`
	StartedAt       time.Time  `json:"startedAt"`
	CompletedAt     *time.Time `json:"completedAt,omitempty"`
	TotalKnown      bool       `json:"totalKnown"`
}

// PodCorrelation maps an image pull to a waiting pod.
type PodCorrelation struct {
	Namespace string `json:"namespace"`
	PodName   string `json:"podName"`
	Container string `json:"container"`
	Image     string `json:"image,omitempty"`
}

// AgentReport is the payload sent by an agent to the server.
type AgentReport struct {
	NodeName  string      `json:"nodeName"`
	Timestamp time.Time   `json:"timestamp"`
	Pulls     []PullState `json:"pulls"`
}

// PullState is the agent-side snapshot of a single image pull.
type PullState struct {
	ImageRef   string       `json:"imageRef"`
	Layers     []LayerState `json:"layers"`
	StartedAt  time.Time    `json:"startedAt"`
	TotalKnown bool         `json:"totalKnown"`
}

// LayerState is the agent-side snapshot of a single layer download.
type LayerState struct {
	Digest          string `json:"digest"`
	MediaType       string `json:"mediaType,omitempty"`
	TotalBytes      int64  `json:"totalBytes"`
	DownloadedBytes int64  `json:"downloadedBytes"`
	TotalKnown      bool   `json:"totalKnown"`
}

// APIResponse wraps the pulls list endpoint response.
type APIResponse struct {
	Pulls []PullStatus `json:"pulls"`
}
