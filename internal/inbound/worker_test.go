package inbound

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/znz-systems/deaddrop/internal/models"
)

type workerTestJobStore struct {
	job       *models.InboundIngestJob
	claimed   bool
	done      bool
	retried   bool
	failed    bool
	doneRes   [2]int
	retryTime time.Time
	lastError string
}

func (m *workerTestJobStore) EnqueueInboundIngestJob(_ context.Context, _ []byte, _ int) (*models.InboundIngestJob, error) {
	return nil, errors.New("not implemented")
}

func (m *workerTestJobStore) ClaimNextInboundIngestJob(_ context.Context) (*models.InboundIngestJob, error) {
	if m.claimed {
		return nil, nil
	}
	m.claimed = true
	return m.job, nil
}

func (m *workerTestJobStore) MarkInboundIngestJobDone(_ context.Context, _ int64, accepted, dropped int) error {
	m.done = true
	m.doneRes = [2]int{accepted, dropped}
	return nil
}

func (m *workerTestJobStore) MarkInboundIngestJobRetry(_ context.Context, _ int64, nextAvailableAt time.Time, lastError string) error {
	m.retried = true
	m.retryTime = nextAvailableAt
	m.lastError = lastError
	return nil
}

func (m *workerTestJobStore) MarkInboundIngestJobFailed(_ context.Context, _ int64, lastError string) error {
	m.failed = true
	m.lastError = lastError
	return nil
}

type failingInboundEmailStore struct {
	mockInboundEmailStore
}

func (m *failingInboundEmailStore) CreateInboundEmail(_ context.Context, _ models.InboundEmailCreateParams) (*models.InboundEmail, error) {
	return nil, errors.New("temporary db error")
}

func TestWorkerProcessOne_MarksDone(t *testing.T) {
	ds := newMockDomainStore()
	ds.byName["example.com"] = &models.Domain{ID: 11, UserID: 7, Name: "example.com", Verified: true}
	es := &mockInboundEmailStore{}
	cs := newMockInboundDomainConfigStore()
	cs.byDomainID[11] = &models.InboundDomainConfig{DomainID: 11, MXTarget: "mx.example.com", MXVerified: true, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	rs := newMockInboundRuleStore()
	svc := NewService(ds, es, cs, rs, nil)

	payload, _ := json.Marshal(IngestJobPayload{
		Sender:     "sender@outside.com",
		Recipients: []string{"ideas@example.com"},
		Subject:    "Hello",
	})
	jobs := &workerTestJobStore{job: &models.InboundIngestJob{ID: 1, Payload: payload, Attempts: 1, MaxAttempts: 5}}
	w := NewWorker(jobs, svc, WorkerOptions{})

	worked, err := w.processOne(context.Background())
	if err != nil {
		t.Fatalf("processOne error: %v", err)
	}
	if !worked {
		t.Fatalf("expected worked=true")
	}
	if !jobs.done {
		t.Fatalf("expected job marked done")
	}
	if jobs.doneRes != [2]int{1, 0} {
		t.Fatalf("unexpected done result: %+v", jobs.doneRes)
	}
}

func TestWorkerProcessOne_RetriesTemporaryError(t *testing.T) {
	ds := newMockDomainStore()
	ds.byName["example.com"] = &models.Domain{ID: 11, UserID: 7, Name: "example.com", Verified: true}
	es := &failingInboundEmailStore{}
	cs := newMockInboundDomainConfigStore()
	cs.byDomainID[11] = &models.InboundDomainConfig{DomainID: 11, MXTarget: "mx.example.com", MXVerified: true, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	rs := newMockInboundRuleStore()
	svc := NewService(ds, es, cs, rs, nil)

	payload, _ := json.Marshal(IngestJobPayload{
		Sender:     "sender@outside.com",
		Recipients: []string{"ideas@example.com"},
	})
	jobs := &workerTestJobStore{job: &models.InboundIngestJob{ID: 1, Payload: payload, Attempts: 1, MaxAttempts: 3}}
	w := NewWorker(jobs, svc, WorkerOptions{RetryBaseDelay: 50 * time.Millisecond})

	worked, err := w.processOne(context.Background())
	if err != nil {
		t.Fatalf("processOne error: %v", err)
	}
	if !worked {
		t.Fatalf("expected worked=true")
	}
	if !jobs.retried {
		t.Fatalf("expected job marked retry")
	}
	if jobs.lastError == "" {
		t.Fatalf("expected retry error message")
	}
}

func TestWorkerProcessOne_FailsPermanentError(t *testing.T) {
	ds := newMockDomainStore()
	es := &mockInboundEmailStore{}
	cs := newMockInboundDomainConfigStore()
	rs := newMockInboundRuleStore()
	svc := NewService(ds, es, cs, rs, nil)

	payload, _ := json.Marshal(IngestJobPayload{
		Recipients: []string{"ideas@example.com"},
	})
	jobs := &workerTestJobStore{job: &models.InboundIngestJob{ID: 1, Payload: payload, Attempts: 1, MaxAttempts: 3}}
	w := NewWorker(jobs, svc, WorkerOptions{})

	worked, err := w.processOne(context.Background())
	if err != nil {
		t.Fatalf("processOne error: %v", err)
	}
	if !worked {
		t.Fatalf("expected worked=true")
	}
	if !jobs.failed {
		t.Fatalf("expected job marked failed")
	}
}

func TestWorkerProcessOne_FailsInvalidRawRFC822(t *testing.T) {
	ds := newMockDomainStore()
	es := &mockInboundEmailStore{}
	cs := newMockInboundDomainConfigStore()
	rs := newMockInboundRuleStore()
	svc := NewService(ds, es, cs, rs, nil)

	payload, _ := json.Marshal(IngestJobPayload{
		RawRFC822: "definitely not rfc822",
	})
	jobs := &workerTestJobStore{job: &models.InboundIngestJob{ID: 1, Payload: payload, Attempts: 1, MaxAttempts: 3}}
	w := NewWorker(jobs, svc, WorkerOptions{})

	worked, err := w.processOne(context.Background())
	if err != nil {
		t.Fatalf("processOne error: %v", err)
	}
	if !worked {
		t.Fatalf("expected worked=true")
	}
	if !jobs.failed {
		t.Fatalf("expected job marked failed")
	}
}

func TestWorkerProcessOne_UsesParsedRecipientsWhenProvidedRecipientsAreBlank(t *testing.T) {
	ds := newMockDomainStore()
	ds.byName["example.com"] = &models.Domain{ID: 11, UserID: 7, Name: "example.com", Verified: true}
	es := &mockInboundEmailStore{}
	cs := newMockInboundDomainConfigStore()
	cs.byDomainID[11] = &models.InboundDomainConfig{DomainID: 11, MXTarget: "mx.example.com", MXVerified: true, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	rs := newMockInboundRuleStore()
	svc := NewService(ds, es, cs, rs, nil)

	raw := "From: Sender <sender@outside.com>\r\nTo: ideas@example.com\r\nSubject: Test\r\nMessage-ID: <m-1@test>\r\nContent-Type: text/plain\r\n\r\nhello"
	payload, _ := json.Marshal(IngestJobPayload{
		RawRFC822:  raw,
		Recipients: []string{""},
	})
	jobs := &workerTestJobStore{job: &models.InboundIngestJob{ID: 1, Payload: payload, Attempts: 1, MaxAttempts: 5}}
	w := NewWorker(jobs, svc, WorkerOptions{})

	worked, err := w.processOne(context.Background())
	if err != nil {
		t.Fatalf("processOne error: %v", err)
	}
	if !worked {
		t.Fatalf("expected worked=true")
	}
	if !jobs.done {
		t.Fatalf("expected job marked done")
	}
	if len(es.items) != 1 || es.items[0].Recipient != "ideas@example.com" {
		t.Fatalf("expected parsed recipient, got %+v", es.items)
	}
}
