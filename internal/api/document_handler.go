package api

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"ledger_pro/internal/service"
)

// AttachDocumentHandler processes multipart/form-data to upload a file to R2.
func (h *Handler) AttachDocumentHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	txIDStr := chi.URLParam(r, "id")
	txID, err := uuid.Parse(txIDStr)
	if err != nil {
		WriteProblem(w, r, BadRequest("Invalid transaction ID format").WithInvalidParams(InvalidParam{Name: "id", Reason: "must be a valid UUID"}))
		return
	}

	// Max 10MB upload size (protects memory before streaming)
	err = r.ParseMultipartForm(10 << 20)
	if err != nil {
		WriteProblem(w, r, BadRequest("Failed to parse multipart form").WithInvalidParams(InvalidParam{Name: "body", Reason: "must be valid multipart/form-data and under 10MB"}))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		WriteProblem(w, r, BadRequest("Missing file part").WithInvalidParams(InvalidParam{Name: "file", Reason: "multipart form must contain a 'file' field"}))
		return
	}
	defer file.Close()

	// Content Type detection
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Delegate to service to stream to R2 and save metadata
	docResp, err := h.svc.AttachDocument(ctx, txID, header.Filename, contentType, header.Size, file)
	if err != nil {
		if errors.Is(err, service.ErrTransactionNotFound) {
			WriteProblem(w, r, TransactionNotFound("Transaction not found"))
			return
		}
		h.logger.ErrorContext(ctx, "failed to attach document", slog.String("error", err.Error()))
		WriteProblem(w, r, InternalError("Failed to process document upload"))
		return
	}

	writeJSON(w, http.StatusCreated, docResp)
}

// ListTransactionDocumentsHandler returns metadata and presigned URLs for all attached documents.
func (h *Handler) ListTransactionDocumentsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	txIDStr := chi.URLParam(r, "id")
	txID, err := uuid.Parse(txIDStr)
	if err != nil {
		WriteProblem(w, r, BadRequest("Invalid transaction ID format").WithInvalidParams(InvalidParam{Name: "id", Reason: "must be a valid UUID"}))
		return
	}

	docs, err := h.svc.ListTransactionDocuments(ctx, txID)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to list documents", slog.String("error", err.Error()))
		WriteProblem(w, r, InternalError("Failed to retrieve documents"))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": docs,
	})
}
