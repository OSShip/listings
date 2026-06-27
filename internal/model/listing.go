package model

import "time"

type Listing struct {
	ID                   string    `json:"id"`
	MentorID             string    `json:"mentor_id"`
	MentorDisplayName    string    `json:"mentor_display_name,omitempty"`
	MentorGithubUsername string    `json:"mentor_github_username,omitempty"`
	OSSProjectName       string    `json:"oss_project_name"`
	OSSRepoURL           string    `json:"oss_repo_url"`
	Description          string    `json:"description"`
	PriceCents           int       `json:"price_cents"`
	DurationWeeks        int       `json:"duration_weeks"`
	TotalSlots           int       `json:"total_slots"`
	FilledSlots          int       `json:"filled_slots"`
	Status               string    `json:"status"`
	CreatedAt            time.Time `json:"created_at"`
}

const ListingSelect = `SELECT l.id, l.mentor_id, COALESCE(u.display_name,''), COALESCE(u.github_username,''),
	l.oss_project_name, l.oss_repo_url, l.description, l.price_cents, l.duration_weeks,
	l.total_slots, l.filled_slots, l.status, l.created_at
	FROM listings l
	JOIN users u ON u.id = l.mentor_id`
