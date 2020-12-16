package gitlab

import (
	"time"
)

type User struct {
	ID        int    `json:"id,omitempty"`
	Name      string `json:"name"`
	Username  string `json:"username"`
	State     string
	AvatarURL string
	WebURL    string
}

type MergeRequest struct {
	ID            int       `json:"id"`
	IID           int       `json:"iid"`
	ProjectID     int       `json:"project_id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	State         string    `json:"state"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	MergedAt      time.Time `json:"merged_at"`
	Labels        []string  `json:"labels"`
	WorkInProcess bool      `json:"work_in_process"`
	WebURL        string    `json:"web_url"`
	TargetBranch  string    `json:"target_branch"`
	SourceBranch  string    `json:"source_branch"`
	MergeStatus   string    `json:"merge_status"`
	Author        User      `json:"author"`
}

type Project struct {
	ID            int
	Description   string
	DefaultBranch string
	Visibility    string
	WebURL        string
	TagList       []string
	Owner         User
}

type Release struct {
	TagName     string `json:"tag_name"`
	Description string `json:"description"`
}

type Commit struct {
	ID            string    `json:"id"`
	ShortID       string    `json:"short_id"`
	CreatedAt     time.Time `json:"created_at"`
	Title         string    `json:"title"`
	Message       string    `json:"message"`
	AuthorName    string    `json:"author_name"`
	AuthorEmail   string    `json:"author_email"`
	AuthorDate    string    `json:"author_date"`
	CommitterName string    `json:"committer_name"`
	CommittedDate time.Time `json:"committed_date"`
	WebURL        string    `json:"web_url"`
}

type Tag struct {
	Name      string  `json:"name"`
	Target    string  `json:"target"`
	Message   string  `json:"message"`
	Release   Release `json:"release"`
	Protected bool    `json:"protected"`
	Commit    Commit  `json:"commit"`
}

type TagRequest struct {
	TagName            string `json:"tag_name"`
	Ref                string `json:"ref"`
	Message            string `json:"message"`
	ReleaseDescription string `json:"release_description"`
}
