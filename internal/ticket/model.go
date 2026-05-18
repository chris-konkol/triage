package ticket

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("ticket not found")

type Priority int32
type Status int32
type Category int32

const (
	PriorityUnspecified Priority = 0
	PriorityLow         Priority = 1
	PriorityMedium      Priority = 2
	PriorityHigh        Priority = 3
	PriorityCritical    Priority = 4
)

const (
	StatusUnspecified Status = 0
	StatusOpen        Status = 1
	StatusInProgress  Status = 2
	StatusWaiting     Status = 3
	StatusResolved    Status = 4
	StatusClosed      Status = 5
)

const (
	CategoryUnspecified    Category = 0
	CategoryBug            Category = 1
	CategoryFeatureRequest Category = 2
	CategorySupport        Category = 3
	CategoryDocumentation  Category = 4
	CategoryInfrastructure Category = 5
)

type Ticket struct {
	ID          string
	Title       string
	Description string
	Priority    Priority
	Status      Status
	Category    Category
	CreatedBy   string
	AssignedTo  string
	Tags        []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ResolvedAt  *time.Time
}

type Comment struct {
	ID        string
	TicketID  string
	Author    string
	Body      string
	CreatedAt time.Time
}

type ListFilter struct {
	Status     Status
	Priority   Priority
	Category   Category
	AssignedTo string
	Search     string
	Page       int32
	PageSize   int32
}
