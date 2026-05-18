package main

import (
	"net/http"

	analyticsv1 "github.com/chris-konkol/triage/gen/analytics/v1"
)

type analyticsHandlers struct {
	client analyticsv1.AnalyticsServiceClient
}

func (h *analyticsHandlers) dashboard(w http.ResponseWriter, r *http.Request) {
	resp, err := h.client.GetDashboardStats(r.Context(), &analyticsv1.GetDashboardStatsRequest{})
	if err != nil {
		writeError(w, grpcCodeToHTTP(err), grpcMessage(err))
		return
	}
	writeProto(w, http.StatusOK, resp)
}
