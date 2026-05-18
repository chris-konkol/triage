package ticket_test

import (
	"context"
	"testing"

	ticketv1 "github.com/chris-konkol/triage/gen/ticket/v1"
	"github.com/chris-konkol/triage/internal/ticket"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// ---- mock repo ----

type mockRepo struct {
	tickets  map[string]*ticket.Ticket
	comments []*ticket.Comment
	err      error
}

func newMockRepo() *mockRepo {
	return &mockRepo{tickets: make(map[string]*ticket.Ticket)}
}

func (m *mockRepo) Create(_ context.Context, t *ticket.Ticket) error {
	if m.err != nil {
		return m.err
	}
	t.ID = "mock-id"
	m.tickets[t.ID] = t
	return nil
}

func (m *mockRepo) GetByID(_ context.Context, id string) (*ticket.Ticket, error) {
	if m.err != nil {
		return nil, m.err
	}
	t, ok := m.tickets[id]
	if !ok {
		return nil, ticket.ErrNotFound
	}
	return t, nil
}

func (m *mockRepo) List(_ context.Context, _ ticket.ListFilter) ([]*ticket.Ticket, int32, error) {
	if m.err != nil {
		return nil, 0, m.err
	}
	out := make([]*ticket.Ticket, 0, len(m.tickets))
	for _, t := range m.tickets {
		out = append(out, t)
	}
	return out, int32(len(out)), nil
}

func (m *mockRepo) Update(_ context.Context, t *ticket.Ticket) error {
	if m.err != nil {
		return m.err
	}
	m.tickets[t.ID] = t
	return nil
}

func (m *mockRepo) Delete(_ context.Context, id string) error {
	if m.err != nil {
		return m.err
	}
	if _, ok := m.tickets[id]; !ok {
		return ticket.ErrNotFound
	}
	delete(m.tickets, id)
	return nil
}

func (m *mockRepo) AddComment(_ context.Context, c *ticket.Comment) error {
	if m.err != nil {
		return m.err
	}
	c.ID = "comment-id"
	m.comments = append(m.comments, c)
	return nil
}

func (m *mockRepo) GetComments(_ context.Context, _ string) ([]*ticket.Comment, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.comments, nil
}

// ---- mock publisher ----

type mockPublisher struct {
	events []string
}

func (m *mockPublisher) Publish(_ context.Context, eventType string, _ any) error {
	m.events = append(m.events, eventType)
	return nil
}

// ---- tests ----

func TestService_CreateTicket_HappyPath(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	svc := ticket.NewService(repo, pub)

	resp, err := svc.CreateTicket(context.Background(), &ticketv1.CreateTicketRequest{
		Title:    "Fix the login bug",
		Priority: ticketv1.Priority_PRIORITY_HIGH,
	})
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}
	if resp.Ticket.Title != "Fix the login bug" {
		t.Errorf("Title = %q, want %q", resp.Ticket.Title, "Fix the login bug")
	}
	if resp.Ticket.Status != ticketv1.Status_STATUS_OPEN {
		t.Errorf("Status = %v, want STATUS_OPEN", resp.Ticket.Status)
	}
	if len(pub.events) != 1 || pub.events[0] != ticket.TopicCreated {
		t.Errorf("published events = %v, want [%q]", pub.events, ticket.TopicCreated)
	}
}

func TestService_CreateTicket_UsesXUserHeader(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	svc := ticket.NewService(repo, pub)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-user", "bob"))
	resp, err := svc.CreateTicket(ctx, &ticketv1.CreateTicketRequest{Title: "T"})
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}
	if resp.Ticket.CreatedBy != "bob" {
		t.Errorf("CreatedBy = %q, want %q", resp.Ticket.CreatedBy, "bob")
	}
}

func TestService_CreateTicket_DefaultsCreatedByToSystem(t *testing.T) {
	svc := ticket.NewService(newMockRepo(), &mockPublisher{})
	resp, err := svc.CreateTicket(context.Background(), &ticketv1.CreateTicketRequest{Title: "T"})
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}
	if resp.Ticket.CreatedBy != "system" {
		t.Errorf("CreatedBy = %q, want %q", resp.Ticket.CreatedBy, "system")
	}
}

func TestService_GetTicket_NotFound(t *testing.T) {
	svc := ticket.NewService(newMockRepo(), &mockPublisher{})
	_, err := svc.GetTicket(context.Background(), &ticketv1.GetTicketRequest{Id: "nonexistent"})
	if err == nil {
		t.Fatal("expected NotFound error, got nil")
	}
	if status.Code(err) != codes.NotFound {
		t.Errorf("code = %v, want NotFound", status.Code(err))
	}
}

func TestService_UpdateTicket_SetsResolvedAt(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	svc := ticket.NewService(repo, pub)

	createResp, _ := svc.CreateTicket(context.Background(), &ticketv1.CreateTicketRequest{Title: "T"})
	id := createResp.Ticket.Id

	resolved := ticketv1.Status_STATUS_RESOLVED
	resp, err := svc.UpdateTicket(context.Background(), &ticketv1.UpdateTicketRequest{
		Id:     id,
		Status: &resolved,
	})
	if err != nil {
		t.Fatalf("UpdateTicket: %v", err)
	}
	if resp.Ticket.Status != ticketv1.Status_STATUS_RESOLVED {
		t.Errorf("Status = %v, want STATUS_RESOLVED", resp.Ticket.Status)
	}
	if resp.Ticket.ResolvedAt == nil {
		t.Error("expected ResolvedAt to be set when transitioning to RESOLVED")
	}
}

func TestService_UpdateTicket_NotFound(t *testing.T) {
	svc := ticket.NewService(newMockRepo(), &mockPublisher{})
	resolved := ticketv1.Status_STATUS_RESOLVED
	_, err := svc.UpdateTicket(context.Background(), &ticketv1.UpdateTicketRequest{
		Id:     "ghost",
		Status: &resolved,
	})
	if status.Code(err) != codes.NotFound {
		t.Errorf("code = %v, want NotFound", status.Code(err))
	}
}

func TestService_DeleteTicket_RemovesTicket(t *testing.T) {
	repo := newMockRepo()
	svc := ticket.NewService(repo, &mockPublisher{})

	createResp, _ := svc.CreateTicket(context.Background(), &ticketv1.CreateTicketRequest{Title: "T"})
	id := createResp.Ticket.Id

	if _, err := svc.DeleteTicket(context.Background(), &ticketv1.DeleteTicketRequest{Id: id}); err != nil {
		t.Fatalf("DeleteTicket: %v", err)
	}
	if _, err := svc.GetTicket(context.Background(), &ticketv1.GetTicketRequest{Id: id}); status.Code(err) != codes.NotFound {
		t.Errorf("expected NotFound after delete, got code %v", status.Code(err))
	}
}

func TestService_DeleteTicket_NotFound(t *testing.T) {
	svc := ticket.NewService(newMockRepo(), &mockPublisher{})
	_, err := svc.DeleteTicket(context.Background(), &ticketv1.DeleteTicketRequest{Id: "ghost"})
	if status.Code(err) != codes.NotFound {
		t.Errorf("code = %v, want NotFound", status.Code(err))
	}
}

func TestService_AddComment_PublishesEvent(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	svc := ticket.NewService(repo, pub)

	createResp, _ := svc.CreateTicket(context.Background(), &ticketv1.CreateTicketRequest{Title: "T"})
	pub.events = nil // reset after create

	resp, err := svc.AddComment(context.Background(), &ticketv1.AddCommentRequest{
		TicketId: createResp.Ticket.Id,
		Body:     "looks good",
	})
	if err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	if resp.Comment.Body != "looks good" {
		t.Errorf("Body = %q, want %q", resp.Comment.Body, "looks good")
	}
	if len(pub.events) != 1 || pub.events[0] != ticket.TopicCommented {
		t.Errorf("published events = %v, want [%q]", pub.events, ticket.TopicCommented)
	}
}

func TestService_ListTickets_ReturnsAll(t *testing.T) {
	repo := newMockRepo()
	svc := ticket.NewService(repo, &mockPublisher{})

	svc.CreateTicket(context.Background(), &ticketv1.CreateTicketRequest{Title: "A"}) //nolint:errcheck

	resp, err := svc.ListTickets(context.Background(), &ticketv1.ListTicketsRequest{PageSize: 50})
	if err != nil {
		t.Fatalf("ListTickets: %v", err)
	}
	if resp.TotalCount != 1 {
		t.Errorf("TotalCount = %d, want 1", resp.TotalCount)
	}
}
