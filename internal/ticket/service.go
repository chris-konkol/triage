package ticket

import (
	"context"
	"errors"
	"time"

	ticketv1 "github.com/chris-konkol/triage/gen/ticket/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Service implements the gRPC TicketServiceServer interface.
type Service struct {
	ticketv1.UnimplementedTicketServiceServer
	repo     TicketRepository
	producer EventPublisher
}

func NewService(repo TicketRepository, producer EventPublisher) *Service {
	return &Service{repo: repo, producer: producer}
}

func (s *Service) CreateTicket(ctx context.Context, req *ticketv1.CreateTicketRequest) (*ticketv1.CreateTicketResponse, error) {
	createdBy := "system"
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if users := md.Get("x-user"); len(users) > 0 {
			createdBy = users[0]
		}
	}

	t := &Ticket{
		Title:       req.Title,
		Description: req.Description,
		Priority:    Priority(req.Priority),
		Category:    Category(req.Category),
		AssignedTo:  req.AssignedTo,
		Tags:        req.Tags,
		Status:      StatusOpen,
		CreatedBy:   createdBy,
	}
	if err := s.repo.Create(ctx, t); err != nil {
		return nil, status.Errorf(codes.Internal, "create ticket: %v", err)
	}
	s.producer.Publish(ctx, TopicCreated, t) //nolint:errcheck
	return &ticketv1.CreateTicketResponse{Ticket: toProto(t)}, nil
}

func (s *Service) GetTicket(ctx context.Context, req *ticketv1.GetTicketRequest) (*ticketv1.GetTicketResponse, error) {
	t, err := s.repo.GetByID(ctx, req.Id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "ticket %s not found", req.Id)
		}
		return nil, status.Errorf(codes.Internal, "get ticket: %v", err)
	}

	comments, err := s.repo.GetComments(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get comments: %v", err)
	}

	protoComments := make([]*ticketv1.Comment, len(comments))
	for i, c := range comments {
		protoComments[i] = commentToProto(c)
	}

	return &ticketv1.GetTicketResponse{Ticket: toProto(t), Comments: protoComments}, nil
}

func (s *Service) ListTickets(ctx context.Context, req *ticketv1.ListTicketsRequest) (*ticketv1.ListTicketsResponse, error) {
	tickets, total, err := s.repo.List(ctx, ListFilter{
		Status:     Status(req.StatusFilter),
		Priority:   Priority(req.PriorityFilter),
		Category:   Category(req.CategoryFilter),
		AssignedTo: req.AssignedToFilter,
		Search:     req.SearchQuery,
		Page:       req.Page,
		PageSize:   req.PageSize,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list tickets: %v", err)
	}

	protoTickets := make([]*ticketv1.Ticket, len(tickets))
	for i, t := range tickets {
		protoTickets[i] = toProto(t)
	}
	return &ticketv1.ListTicketsResponse{Tickets: protoTickets, TotalCount: total}, nil
}

func (s *Service) UpdateTicket(ctx context.Context, req *ticketv1.UpdateTicketRequest) (*ticketv1.UpdateTicketResponse, error) {
	t, err := s.repo.GetByID(ctx, req.Id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "ticket %s not found", req.Id)
		}
		return nil, status.Errorf(codes.Internal, "get ticket: %v", err)
	}

	if req.Title != nil {
		t.Title = req.GetTitle()
	}
	if req.Description != nil {
		t.Description = req.GetDescription()
	}
	if req.Priority != nil {
		t.Priority = Priority(req.GetPriority())
	}
	if req.Status != nil {
		newStatus := Status(req.GetStatus())
		if newStatus == StatusResolved && t.Status != StatusResolved {
			now := time.Now()
			t.ResolvedAt = &now
		}
		t.Status = newStatus
	}
	if req.Category != nil {
		t.Category = Category(req.GetCategory())
	}
	if req.AssignedTo != nil {
		t.AssignedTo = req.GetAssignedTo()
	}
	if len(req.Tags) > 0 {
		t.Tags = req.Tags
	}

	if err := s.repo.Update(ctx, t); err != nil {
		return nil, status.Errorf(codes.Internal, "update ticket: %v", err)
	}
	s.producer.Publish(ctx, TopicUpdated, t) //nolint:errcheck
	return &ticketv1.UpdateTicketResponse{Ticket: toProto(t)}, nil
}

func (s *Service) DeleteTicket(ctx context.Context, req *ticketv1.DeleteTicketRequest) (*ticketv1.DeleteTicketResponse, error) {
	if err := s.repo.Delete(ctx, req.Id); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "ticket %s not found", req.Id)
		}
		return nil, status.Errorf(codes.Internal, "delete ticket: %v", err)
	}
	return &ticketv1.DeleteTicketResponse{}, nil
}

func (s *Service) AddComment(ctx context.Context, req *ticketv1.AddCommentRequest) (*ticketv1.AddCommentResponse, error) {
	c := &Comment{
		TicketID: req.TicketId,
		Author:   "system",
		Body:     req.Body,
	}
	if err := s.repo.AddComment(ctx, c); err != nil {
		return nil, status.Errorf(codes.Internal, "add comment: %v", err)
	}
	s.producer.Publish(ctx, TopicCommented, c) //nolint:errcheck
	return &ticketv1.AddCommentResponse{Comment: commentToProto(c)}, nil
}

func toProto(t *Ticket) *ticketv1.Ticket {
	pb := &ticketv1.Ticket{
		Id:          t.ID,
		Title:       t.Title,
		Description: t.Description,
		Priority:    ticketv1.Priority(t.Priority),
		Status:      ticketv1.Status(t.Status),
		Category:    ticketv1.Category(t.Category),
		CreatedBy:   t.CreatedBy,
		AssignedTo:  t.AssignedTo,
		Tags:        t.Tags,
		CreatedAt:   timestamppb.New(t.CreatedAt),
		UpdatedAt:   timestamppb.New(t.UpdatedAt),
	}
	if t.ResolvedAt != nil {
		pb.ResolvedAt = timestamppb.New(*t.ResolvedAt)
	}
	return pb
}

func commentToProto(c *Comment) *ticketv1.Comment {
	return &ticketv1.Comment{
		Id:        c.ID,
		TicketId:  c.TicketID,
		Author:    c.Author,
		Body:      c.Body,
		CreatedAt: timestamppb.New(c.CreatedAt),
	}
}
