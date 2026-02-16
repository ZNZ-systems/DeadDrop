package handlers

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/znz-systems/deaddrop/internal/blob"
	"github.com/znz-systems/deaddrop/internal/models"
	"github.com/znz-systems/deaddrop/internal/store"
	"github.com/znz-systems/deaddrop/internal/web/middleware"
	"github.com/znz-systems/deaddrop/internal/web/render"
)

type InboxHandler struct {
	emails        store.InboundEmailStore
	blobs         blob.Store
	render        *render.Renderer
	secureCookies bool
}

func NewInboxHandler(emails store.InboundEmailStore, blobs blob.Store, r *render.Renderer, secureCookies bool) *InboxHandler {
	return &InboxHandler{
		emails:        emails,
		blobs:         blobs,
		render:        r,
		secureCookies: secureCookies,
	}
}

func (h *InboxHandler) ShowInbox(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	query := parseInboxQuery(r)
	emails, err := h.emails.SearchInboundEmailsByUserID(r.Context(), user.ID, query)
	if err != nil {
		slog.Error("failed to list inbound emails", "user_id", user.ID, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"User":   user,
		"Emails": emails,
		"Query":  query,
	}
	if msg, msgType := consumeFlash(w, r, h.secureCookies); msg != "" {
		data["Flash"] = msg
		data["FlashType"] = msgType
	}
	h.render.Render(w, r, "inbox.html", data)
}

func (h *InboxHandler) ShowEmailDetail(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	emailID, ok := parseIDParam(w, r, "emailID")
	if !ok {
		return
	}

	email, err := h.emails.GetInboundEmailByID(r.Context(), emailID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to load inbound email", "email_id", emailID, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if email.UserID != user.ID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if !email.IsRead {
		if err := h.emails.MarkInboundEmailRead(r.Context(), email.ID); err != nil {
			slog.Warn("failed to mark inbound email read", "email_id", email.ID, "error", err)
		}
		email.IsRead = true
	}

	attachments, err := h.emails.ListInboundEmailAttachmentsByEmailID(r.Context(), email.ID)
	if err != nil {
		slog.Error("failed to load attachments", "email_id", email.ID, "error", err)
		attachments = nil
	}

	h.render.Render(w, r, "inbox_detail.html", map[string]interface{}{
		"User":        user,
		"Email":       email,
		"Attachments": attachments,
	})
}

func (h *InboxHandler) HandleMarkRead(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	emailID, ok := parseIDParam(w, r, "emailID")
	if !ok {
		return
	}
	email, err := h.emails.GetInboundEmailByID(r.Context(), emailID)
	if err != nil || email.UserID != user.ID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err := h.emails.MarkInboundEmailRead(r.Context(), emailID); err != nil {
		slog.Error("failed to mark inbound email read", "email_id", emailID, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/inbox/"+strconv.FormatInt(emailID, 10), http.StatusSeeOther)
}

func (h *InboxHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	emailID, ok := parseIDParam(w, r, "emailID")
	if !ok {
		return
	}
	email, err := h.emails.GetInboundEmailByID(r.Context(), emailID)
	if err != nil || email.UserID != user.ID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err := h.emails.DeleteInboundEmail(r.Context(), emailID); err != nil {
		slog.Error("failed to delete inbound email", "email_id", emailID, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	setFlashSuccess(w, "Email deleted.", h.secureCookies)
	http.Redirect(w, r, "/inbox", http.StatusSeeOther)
}

func (h *InboxHandler) HandleDownloadAttachment(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	emailID, ok := parseIDParam(w, r, "emailID")
	if !ok {
		return
	}
	email, err := h.emails.GetInboundEmailByID(r.Context(), emailID)
	if err != nil || email.UserID != user.ID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	attachmentID, ok := parseIDParam(w, r, "attachmentID")
	if !ok {
		return
	}
	attachment, err := h.emails.GetInboundEmailAttachmentByID(r.Context(), attachmentID)
	if err != nil || attachment.InboundEmailID != email.ID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	content := attachment.Content
	if strings.TrimSpace(attachment.BlobKey) != "" && h.blobs != nil {
		body, err := h.blobs.Get(r.Context(), attachment.BlobKey)
		if err != nil {
			slog.Warn("failed to fetch attachment blob", "attachment_id", attachment.ID, "blob_key", attachment.BlobKey, "error", err)
			if len(content) == 0 {
				http.Error(w, "attachment unavailable", http.StatusNotFound)
				return
			}
		} else {
			content = body
		}
	}
	if len(content) == 0 {
		http.Error(w, "attachment unavailable", http.StatusNotFound)
		return
	}

	contentType := attachment.ContentType
	if strings.TrimSpace(contentType) == "" {
		contentType = "application/octet-stream"
	}
	filename := filepath.Base(strings.TrimSpace(attachment.FileName))
	if filename == "" || filename == "." || filename == "/" {
		filename = "attachment.bin"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(content)), 10))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func (h *InboxHandler) HandleSearchAPI(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	query := parseInboxQuery(r)
	emails, err := h.emails.SearchInboundEmailsByUserID(r.Context(), user.ID, query)
	if err != nil {
		slog.Error("failed inbox search", "user_id", user.ID, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":     true,
		"emails": emails,
	})
}

func parseIDParam(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	raw := chi.URLParam(r, name)
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

func parseInboxQuery(r *http.Request) models.InboundEmailQuery {
	qp := r.URL.Query()
	q := models.InboundEmailQuery{
		Q:          strings.TrimSpace(qp.Get("q")),
		Domain:     strings.TrimSpace(qp.Get("domain")),
		UnreadOnly: qp.Get("unread") == "1" || strings.EqualFold(qp.Get("unread"), "true"),
		Limit:      100,
		Offset:     0,
	}
	if from := parseDateParam(qp.Get("from")); from != nil {
		q.From = from
	}
	if to := parseDateParam(qp.Get("to")); to != nil {
		toExclusive := to.Add(24 * time.Hour)
		q.To = &toExclusive
	}
	return q
}

func parseDateParam(v string) *time.Time {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", v)
	if err != nil {
		return nil
	}
	return &t
}
