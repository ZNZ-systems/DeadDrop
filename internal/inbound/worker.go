package inbound

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/znz-systems/deaddrop/internal/store"
)

type WorkerOptions struct {
	PollInterval       time.Duration
	RetryBaseDelay     time.Duration
	MaxRetryDelay      time.Duration
	MaxAttachmentBytes int64
}

type Worker struct {
	jobs               store.InboundIngestJobStore
	ingest             *Service
	pollInterval       time.Duration
	retryBaseDelay     time.Duration
	maxRetryDelay      time.Duration
	maxAttachmentBytes int64
}

func NewWorker(jobs store.InboundIngestJobStore, ingest *Service, opts WorkerOptions) *Worker {
	poll := opts.PollInterval
	if poll <= 0 {
		poll = 500 * time.Millisecond
	}
	retryBase := opts.RetryBaseDelay
	if retryBase <= 0 {
		retryBase = 5 * time.Second
	}
	maxRetry := opts.MaxRetryDelay
	if maxRetry <= 0 {
		maxRetry = 10 * time.Minute
	}
	maxAttachmentBytes := opts.MaxAttachmentBytes
	if maxAttachmentBytes <= 0 {
		maxAttachmentBytes = defaultMaxAttachmentBytes
	}

	return &Worker{
		jobs:               jobs,
		ingest:             ingest,
		pollInterval:       poll,
		retryBaseDelay:     retryBase,
		maxRetryDelay:      maxRetry,
		maxAttachmentBytes: maxAttachmentBytes,
	}
}

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		worked, err := w.processOne(ctx)
		if err != nil {
			slog.Error("inbound job worker cycle failed", "error", err)
		}
		if worked {
			continue
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *Worker) processOne(ctx context.Context) (bool, error) {
	job, err := w.jobs.ClaimNextInboundIngestJob(ctx)
	if err != nil {
		return false, fmt.Errorf("claim inbound job: %w", err)
	}
	if job == nil {
		return false, nil
	}

	var payload IngestJobPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		markErr := w.jobs.MarkInboundIngestJobFailed(ctx, job.ID, "invalid payload: "+err.Error())
		if markErr != nil {
			return true, fmt.Errorf("mark failed after invalid payload: %w", markErr)
		}
		return true, nil
	}
	payload.Normalize()

	msg := payload.ToMessage()
	if payload.RawRFC822 != "" {
		parsed, parseErr := ParseRFC822(payload.RawRFC822, w.maxAttachmentBytes)
		if parseErr != nil {
			markErr := w.jobs.MarkInboundIngestJobFailed(ctx, job.ID, "invalid raw_rfc822: "+parseErr.Error())
			if markErr != nil {
				return true, fmt.Errorf("mark failed after parse error: %w", markErr)
			}
			return true, nil
		}
		if msg.Sender == "" {
			msg.Sender = parsed.Sender
		}
		if len(msg.Recipients) == 0 {
			msg.Recipients = parsed.Recipients
		}
		if msg.Subject == "" {
			msg.Subject = parsed.Subject
		}
		if msg.TextBody == "" {
			msg.TextBody = parsed.TextBody
		}
		if msg.HTMLBody == "" {
			msg.HTMLBody = parsed.HTMLBody
		}
		if msg.MessageID == "" {
			msg.MessageID = parsed.MessageID
		}
		msg.Attachments = parsed.Attachments
	}

	result, ingestErr := w.ingest.Ingest(ctx, msg)
	if ingestErr == nil {
		if err := w.jobs.MarkInboundIngestJobDone(ctx, job.ID, result.Accepted, result.Dropped); err != nil {
			return true, fmt.Errorf("mark inbound job done: %w", err)
		}
		return true, nil
	}

	if isPermanentIngestError(ingestErr) || job.Attempts >= job.MaxAttempts {
		if err := w.jobs.MarkInboundIngestJobFailed(ctx, job.ID, ingestErr.Error()); err != nil {
			return true, fmt.Errorf("mark inbound job failed: %w", err)
		}
		return true, nil
	}

	nextRun := time.Now().UTC().Add(w.retryDelay(job.Attempts))
	if err := w.jobs.MarkInboundIngestJobRetry(ctx, job.ID, nextRun, ingestErr.Error()); err != nil {
		return true, fmt.Errorf("mark inbound job retry: %w", err)
	}
	return true, nil
}

func (w *Worker) retryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	delay := w.retryBaseDelay
	for i := 1; i < attempt; i++ {
		delay *= 2
		if delay >= w.maxRetryDelay {
			return w.maxRetryDelay
		}
	}
	if delay > w.maxRetryDelay {
		return w.maxRetryDelay
	}
	return delay
}

func isPermanentIngestError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrSenderRequired) || errors.Is(err, ErrRecipientsRequired) {
		return true
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(message, "invalid sender address")
}
