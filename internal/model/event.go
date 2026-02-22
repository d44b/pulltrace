// Package model defines the core data types for Pulltrace.
package model

import "time"

const SchemaVersion = "v1"

// PullEvent is the top-level event emitted as a JSON log line and sent over the API.
type PullEvent struct {
	SchemaVersion string       `json:"schemaVersion"`
	Timestamp     time.Time    `json:"timestamp"`
	Type          EventType    `json:"type"`
	NodeName      string       `json:"nodeName"`
	Pull          *PullStatus  `json:"pull,omitempty"`
	Layer         *LayerStatus `json:"layer,omitempty"`
}

// EventType distinguishes event kinds.
type EventType string

const (
	EventPullStarted    EventType = "pull.started"
	EventPullProgress   EventType = "pull.progress"
	EventPullCompleted  EventType = "pull.completed"
	EventPullFailed     EventType = "pull.failed"
	EventLayerStarted   EventType = "layer.started"
	EventLayerProgress  EventType = "layer.progress"
	EventLayerCompleted EventType = "layer.completed"
)

// PullStatus describes the overall image pull.
type PullStatus struct {
	ID              string           `json:"id"`
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
	TotalKnown      bool             `json:"totalKnown"`
}

// LayerStatus describes a single layer (content digest) download.
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

// PodCorrelation maps an image pull to waiting pods.
type PodCorrelation struct {
	Namespace string `json:"namespace"`
	PodName   string `json:"podName"`
	Container string `json:"container"`
}

// AgentReport is the payload an agent sends to the server.
type AgentReport struct {
	NodeName  string      `json:"nodeName"`
	Timestamp time.Time   `json:"timestamp"`
	Pulls     []PullState `json:"pulls"`
}

// PullState is the agent-side snapshot of a single image pull in progress.
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

// APIResponse wraps API list responses.
type APIResponse struct {
	Pulls []PullStatus `json:"pulls"`
}
