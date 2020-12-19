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
	ID             int       `json:"id"`
	IID            int       `json:"iid"`
	ProjectID      int       `json:"project_id"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	State          string    `json:"state"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	MergedAt       time.Time `json:"merged_at"`
	Labels         []string  `json:"labels"`
	WorkInProcess  bool      `json:"work_in_process"`
	WebURL         string    `json:"web_url"`
	TargetBranch   string    `json:"target_branch"`
	SourceBranch   string    `json:"source_branch"`
	MergeStatus    string    `json:"merge_status"`
	Author         User      `json:"author"`
	MergeCommitSHA string    `json:"merge_commit_sha"`
}

type Project struct {
	ID            int      `json:"id"`
	Description   string   `json:"description"`
	DefaultBranch string   `json:"default_branch"`
	Visibility    string   `json:"visibility"`
	WebURL        string   `json:"web_url"`
	TagList       []string `json:"tag_list"`
	Owner         User     `json:"owner"`
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
	Name      string   `json:"name"`
	Target    string   `json:"target"`
	Message   string   `json:"message"`
	Release   *Release `json:"release"`
	Protected bool     `json:"protected"`
	Commit    Commit   `json:"commit"`
}

type TagRequest struct {
	TagName            string `json:"tag_name"`
	Ref                string `json:"ref"`
	Message            string `json:"message"`
	ReleaseDescription string `json:"release_description"`
}

type RepoFileRequest struct {
	Branch        string `json:"branch"`
	CommitMessage string `json:"commit_message"`
	Encoding      string `json:"encoding,omitempty"`
	Content       string `json:"content"`
}

type MergeRequestRequest struct {
	SourceBranch       string `json:"source_branch"`
	TargetBranch       string `json:"target_branch"`
	Title              string `json:"title"`
	AssigneeID         string `json:"assignee_id,omitempty"`
	Description        string `json:"description,omitempty"`
	RemoveSourceBranch bool   `json:"remove_source_branch"`
	Squash             bool   `json:"squash"`
}
