package ticket

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, t *Ticket) error {
	t.CreatedAt = time.Now()
	t.UpdatedAt = time.Now()

	return r.db.QueryRow(ctx, `
		INSERT INTO tickets (title, description, priority, status, category, created_by, assigned_to, tags, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`, t.Title, t.Description, int32(t.Priority), int32(t.Status), int32(t.Category),
		t.CreatedBy, t.AssignedTo, t.Tags, t.CreatedAt, t.UpdatedAt,
	).Scan(&t.ID)
}

func (r *Repository) GetByID(ctx context.Context, id string) (*Ticket, error) {
	t := &Ticket{}
	var priority, status, category int32

	err := r.db.QueryRow(ctx, `
		SELECT id, title, description, priority, status, category,
		       created_by, assigned_to, tags, created_at, updated_at, resolved_at
		FROM tickets WHERE id = $1
	`, id).Scan(
		&t.ID, &t.Title, &t.Description, &priority, &status, &category,
		&t.CreatedBy, &t.AssignedTo, &t.Tags, &t.CreatedAt, &t.UpdatedAt, &t.ResolvedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	t.Priority = Priority(priority)
	t.Status = Status(status)
	t.Category = Category(category)
	return t, nil
}

func (r *Repository) List(ctx context.Context, f ListFilter) ([]*Ticket, int32, error) {
	where := "WHERE 1=1"
	args := []any{}
	n := 0

	if f.Status != StatusUnspecified {
		n++
		where += fmt.Sprintf(" AND status = $%d", n)
		args = append(args, int32(f.Status))
	}
	if f.Priority != PriorityUnspecified {
		n++
		where += fmt.Sprintf(" AND priority = $%d", n)
		args = append(args, int32(f.Priority))
	}
	if f.Category != CategoryUnspecified {
		n++
		where += fmt.Sprintf(" AND category = $%d", n)
		args = append(args, int32(f.Category))
	}
	if f.AssignedTo != "" {
		n++
		where += fmt.Sprintf(" AND assigned_to = $%d", n)
		args = append(args, f.AssignedTo)
	}
	if f.Search != "" {
		n++
		where += fmt.Sprintf(" AND (title ILIKE $%d OR description ILIKE $%d)", n, n)
		args = append(args, "%"+f.Search+"%")
	}

	var total int32
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM tickets "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	if f.PageSize == 0 {
		f.PageSize = 20
	}
	if f.Page < 1 {
		f.Page = 1
	}
	offset := (f.Page - 1) * f.PageSize

	n++
	orderLimit := fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", n)
	args = append(args, f.PageSize)
	n++
	orderLimit += fmt.Sprintf(" OFFSET $%d", n)
	args = append(args, offset)

	rows, err := r.db.Query(ctx, `
		SELECT id, title, description, priority, status, category,
		       created_by, assigned_to, tags, created_at, updated_at, resolved_at
		FROM tickets `+where+orderLimit, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tickets []*Ticket
	for rows.Next() {
		t := &Ticket{}
		var priority, status, category int32
		if err := rows.Scan(
			&t.ID, &t.Title, &t.Description, &priority, &status, &category,
			&t.CreatedBy, &t.AssignedTo, &t.Tags, &t.CreatedAt, &t.UpdatedAt, &t.ResolvedAt,
		); err != nil {
			return nil, 0, err
		}
		t.Priority = Priority(priority)
		t.Status = Status(status)
		t.Category = Category(category)
		tickets = append(tickets, t)
	}
	return tickets, total, rows.Err()
}

func (r *Repository) Update(ctx context.Context, t *Ticket) error {
	t.UpdatedAt = time.Now()
	_, err := r.db.Exec(ctx, `
		UPDATE tickets
		SET title=$2, description=$3, priority=$4, status=$5, category=$6,
		    assigned_to=$7, tags=$8, updated_at=$9, resolved_at=$10
		WHERE id=$1
	`, t.ID, t.Title, t.Description, int32(t.Priority), int32(t.Status), int32(t.Category),
		t.AssignedTo, t.Tags, t.UpdatedAt, t.ResolvedAt,
	)
	return err
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	result, err := r.db.Exec(ctx, "DELETE FROM tickets WHERE id = $1", id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) AddComment(ctx context.Context, c *Comment) error {
	c.CreatedAt = time.Now()
	return r.db.QueryRow(ctx, `
		INSERT INTO comments (ticket_id, author, body, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, c.TicketID, c.Author, c.Body, c.CreatedAt).Scan(&c.ID)
}

func (r *Repository) GetComments(ctx context.Context, ticketID string) ([]*Comment, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, ticket_id, author, body, created_at
		FROM comments WHERE ticket_id = $1 ORDER BY created_at ASC
	`, ticketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*Comment
	for rows.Next() {
		c := &Comment{}
		if err := rows.Scan(&c.ID, &c.TicketID, &c.Author, &c.Body, &c.CreatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}
