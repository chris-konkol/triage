package ticket

import "context"

// TicketRepository is the data access interface used by Service.
type TicketRepository interface {
	Create(ctx context.Context, t *Ticket) error
	GetByID(ctx context.Context, id string) (*Ticket, error)
	List(ctx context.Context, f ListFilter) ([]*Ticket, int32, error)
	Update(ctx context.Context, t *Ticket) error
	Delete(ctx context.Context, id string) error
	AddComment(ctx context.Context, c *Comment) error
	GetComments(ctx context.Context, ticketID string) ([]*Comment, error)
}

// EventPublisher publishes domain events to the message bus.
type EventPublisher interface {
	Publish(ctx context.Context, eventType string, payload any) error
}
