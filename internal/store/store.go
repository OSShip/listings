package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/OSShip/listings/internal/model"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) List(ctx context.Context, status, ossProject string) ([]model.Listing, error) {
	var (
		rows interface {
			Close()
			Next() bool
			Scan(dest ...any) error
		}
		err error
	)
	if ossProject != "" {
		rows, err = s.pool.Query(ctx,
			model.ListingSelect+` WHERE l.status=$1 AND l.oss_project_name ILIKE '%' || $2 || '%' ORDER BY l.created_at DESC`,
			status, ossProject)
	} else {
		rows, err = s.pool.Query(ctx,
			model.ListingSelect+` WHERE l.status=$1 ORDER BY l.created_at DESC`, status)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanListings(rows), nil
}

func (s *Store) Get(ctx context.Context, id string) (model.Listing, error) {
	var l model.Listing
	err := s.pool.QueryRow(ctx, model.ListingSelect+` WHERE l.id=$1`, id).
		Scan(&l.ID, &l.MentorID, &l.MentorDisplayName, &l.MentorGithubUsername,
			&l.OSSProjectName, &l.OSSRepoURL, &l.Description, &l.PriceCents, &l.DurationWeeks,
			&l.TotalSlots, &l.FilledSlots, &l.Status, &l.CreatedAt)
	return l, err
}

func (s *Store) IsMentorApproved(ctx context.Context, userID string) (bool, error) {
	var approved bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM mentor_applications WHERE user_id=$1 AND status='approved')`, userID).Scan(&approved)
	return approved, err
}

func (s *Store) Create(ctx context.Context, mentorID string, req model.Listing) (model.Listing, error) {
	id := uuid.New().String()
	status := req.Status
	if status == "" {
		status = "active"
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO listings (id, mentor_id, oss_project_name, oss_repo_url, description, price_cents, duration_weeks, total_slots, status)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		id, mentorID, req.OSSProjectName, req.OSSRepoURL, req.Description, req.PriceCents, req.DurationWeeks, req.TotalSlots, status)
	if err != nil {
		return model.Listing{}, err
	}
	req.ID = id
	req.MentorID = mentorID
	req.Status = status
	return req, nil
}

func (s *Store) GetMentorID(ctx context.Context, id string) (string, error) {
	var mentorID string
	err := s.pool.QueryRow(ctx, `SELECT mentor_id FROM listings WHERE id=$1`, id).Scan(&mentorID)
	return mentorID, err
}

func (s *Store) Update(ctx context.Context, id string, description, status string, priceCents, durationWeeks int) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE listings SET description=COALESCE(NULLIF($1,''),description), status=COALESCE(NULLIF($2,''),status),
		 price_cents=CASE WHEN $3>0 THEN $3 ELSE price_cents END,
		 duration_weeks=CASE WHEN $4>0 THEN $4 ELSE duration_weeks END, updated_at=NOW() WHERE id=$5`,
		description, status, priceCents, durationWeeks, id)
	return err
}

func scanListings(rows interface {
	Next() bool
	Scan(...interface{}) error
}) []model.Listing {
	var list []model.Listing
	for rows.Next() {
		var l model.Listing
		if err := rows.Scan(&l.ID, &l.MentorID, &l.MentorDisplayName, &l.MentorGithubUsername,
			&l.OSSProjectName, &l.OSSRepoURL, &l.Description, &l.PriceCents, &l.DurationWeeks,
			&l.TotalSlots, &l.FilledSlots, &l.Status, &l.CreatedAt); err != nil {
			continue
		}
		list = append(list, l)
	}
	return list
}
