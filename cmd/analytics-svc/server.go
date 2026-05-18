package main

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	analyticsv1 "github.com/chris-konkol/triage/gen/analytics/v1"
)

type analyticsServer struct {
	analyticsv1.UnimplementedAnalyticsServiceServer
	pool *pgxpool.Pool
}

func (s *analyticsServer) GetDashboardStats(ctx context.Context, _ *analyticsv1.GetDashboardStatsRequest) (*analyticsv1.GetDashboardStatsResponse, error) {
	// Totals by status from tickets table directly — most accurate source
	rows, err := s.pool.Query(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE status = 1) AS open,
			COUNT(*) FILTER (WHERE status = 2) AS in_progress,
			COUNT(*) FILTER (WHERE status = 4) AS resolved,
			COUNT(*) FILTER (WHERE status = 5) AS closed,
			AVG(EXTRACT(EPOCH FROM (resolved_at - created_at)) / 3600)
				FILTER (WHERE resolved_at IS NOT NULL) AS avg_resolution_hours
		FROM tickets
	`)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query totals: %v", err)
	}
	defer rows.Close()

	resp := &analyticsv1.GetDashboardStatsResponse{}
	if rows.Next() {
		var avgHours *float64
		if err := rows.Scan(&resp.TotalOpen, &resp.TotalInProgress, &resp.TotalResolved, &resp.TotalClosed, &avgHours); err != nil {
			return nil, status.Errorf(codes.Internal, "scan totals: %v", err)
		}
		if avgHours != nil {
			resp.AvgResolutionHours = *avgHours
		}
	}
	rows.Close()

	// Tickets by category
	catRows, err := s.pool.Query(ctx, `
		SELECT category, COUNT(*) FROM tickets GROUP BY category ORDER BY category
	`)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query by category: %v", err)
	}
	defer catRows.Close()
	for catRows.Next() {
		var cat int32
		var count int32
		if err := catRows.Scan(&cat, &count); err != nil {
			continue
		}
		resp.TicketsByCategory = append(resp.TicketsByCategory, &analyticsv1.CategoryCount{
			Category: categoryName(cat),
			Count:    count,
		})
	}
	catRows.Close()

	// Tickets by priority
	priRows, err := s.pool.Query(ctx, `
		SELECT priority, COUNT(*) FROM tickets GROUP BY priority ORDER BY priority
	`)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query by priority: %v", err)
	}
	defer priRows.Close()
	for priRows.Next() {
		var pri int32
		var count int32
		if err := priRows.Scan(&pri, &count); err != nil {
			continue
		}
		resp.TicketsByPriority = append(resp.TicketsByPriority, &analyticsv1.PriorityCount{
			Priority: priorityName(pri),
			Count:    count,
		})
	}
	priRows.Close()

	// Daily counts from analytics_snapshots (last 14 days)
	snapRows, err := s.pool.Query(ctx, `
		SELECT snapshot_date, tickets_created, tickets_resolved
		FROM analytics_snapshots
		WHERE snapshot_date >= CURRENT_DATE - INTERVAL '14 days'
		ORDER BY snapshot_date
	`)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query snapshots: %v", err)
	}
	defer snapRows.Close()
	for snapRows.Next() {
		var date time.Time
		var created, resolved int32
		if err := snapRows.Scan(&date, &created, &resolved); err != nil {
			continue
		}
		resp.TicketsPerDay = append(resp.TicketsPerDay, &analyticsv1.DailyCount{
			Date:     date.Format("2006-01-02"),
			Created:  created,
			Resolved: resolved,
		})
	}

	return resp, nil
}

func categoryName(v int32) string {
	m := map[int32]string{1: "Bug", 2: "Feature Request", 3: "Support", 4: "Documentation", 5: "Infrastructure"}
	if n, ok := m[v]; ok {
		return n
	}
	return "Unknown"
}

func priorityName(v int32) string {
	m := map[int32]string{1: "Low", 2: "Medium", 3: "High", 4: "Critical"}
	if n, ok := m[v]; ok {
		return n
	}
	return "Unknown"
}

// keep json import used for future payload parsing
var _ = json.Unmarshal
