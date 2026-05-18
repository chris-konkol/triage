package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc/metadata"

	ticketv1 "github.com/chris-konkol/triage/gen/ticket/v1"
	"github.com/chris-konkol/triage/internal/auth"
)

type ticketHandlers struct {
	client ticketv1.TicketServiceClient
}

func (h *ticketHandlers) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	req := &ticketv1.ListTicketsRequest{
		Page:        1,
		PageSize:    50,
		SearchQuery: q.Get("search"),
	}
	if v := q.Get("status"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			req.StatusFilter = ticketv1.Status(n)
		}
	}
	if v := q.Get("priority"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			req.PriorityFilter = ticketv1.Priority(n)
		}
	}
	if v := q.Get("category"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			req.CategoryFilter = ticketv1.Category(n)
		}
	}
	if v := q.Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			req.Page = int32(n)
		}
	}
	if v := q.Get("page_size"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			req.PageSize = int32(n)
		}
	}
	resp, err := h.client.ListTickets(r.Context(), req)
	if err != nil {
		writeError(w, grpcCodeToHTTP(err), grpcMessage(err))
		return
	}
	writeProto(w, http.StatusOK, resp)
}

func (h *ticketHandlers) create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Priority    int32    `json:"priority"`
		Category    int32    `json:"category"`
		AssignedTo  string   `json:"assignedTo"`
		Tags        []string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Propagate the authenticated user's identity to the ticket service via gRPC metadata.
	claims := auth.GetClaims(r)
	ctx := metadata.AppendToOutgoingContext(r.Context(), "x-user", claims.Username)

	resp, err := h.client.CreateTicket(ctx, &ticketv1.CreateTicketRequest{
		Title:       req.Title,
		Description: req.Description,
		Priority:    ticketv1.Priority(req.Priority),
		Category:    ticketv1.Category(req.Category),
		AssignedTo:  req.AssignedTo,
		Tags:        req.Tags,
	})
	if err != nil {
		writeError(w, grpcCodeToHTTP(err), grpcMessage(err))
		return
	}
	writeProto(w, http.StatusCreated, resp)
}

func (h *ticketHandlers) get(w http.ResponseWriter, r *http.Request) {
	resp, err := h.client.GetTicket(r.Context(), &ticketv1.GetTicketRequest{
		Id: chi.URLParam(r, "id"),
	})
	if err != nil {
		writeError(w, grpcCodeToHTTP(err), grpcMessage(err))
		return
	}
	writeProto(w, http.StatusOK, resp)
}

func (h *ticketHandlers) update(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title       *string  `json:"title"`
		Description *string  `json:"description"`
		Priority    *int32   `json:"priority"`
		Status      *int32   `json:"status"`
		Category    *int32   `json:"category"`
		AssignedTo  *string  `json:"assignedTo"`
		Tags        []string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &ticketv1.UpdateTicketRequest{Id: chi.URLParam(r, "id")}
	if req.Title != nil {
		grpcReq.Title = req.Title
	}
	if req.Description != nil {
		grpcReq.Description = req.Description
	}
	if req.Priority != nil {
		p := ticketv1.Priority(*req.Priority)
		grpcReq.Priority = &p
	}
	if req.Status != nil {
		s := ticketv1.Status(*req.Status)
		grpcReq.Status = &s
	}
	if req.Category != nil {
		c := ticketv1.Category(*req.Category)
		grpcReq.Category = &c
	}
	if req.AssignedTo != nil {
		grpcReq.AssignedTo = req.AssignedTo
	}
	grpcReq.Tags = req.Tags

	resp, err := h.client.UpdateTicket(r.Context(), grpcReq)
	if err != nil {
		writeError(w, grpcCodeToHTTP(err), grpcMessage(err))
		return
	}
	writeProto(w, http.StatusOK, resp)
}

func (h *ticketHandlers) delete(w http.ResponseWriter, r *http.Request) {
	_, err := h.client.DeleteTicket(r.Context(), &ticketv1.DeleteTicketRequest{
		Id: chi.URLParam(r, "id"),
	})
	if err != nil {
		writeError(w, grpcCodeToHTTP(err), grpcMessage(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ticketHandlers) addComment(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	resp, err := h.client.AddComment(r.Context(), &ticketv1.AddCommentRequest{
		TicketId: chi.URLParam(r, "id"),
		Body:     req.Body,
	})
	if err != nil {
		writeError(w, grpcCodeToHTTP(err), grpcMessage(err))
		return
	}
	writeProto(w, http.StatusCreated, resp)
}

