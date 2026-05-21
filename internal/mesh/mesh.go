package mesh

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"tmux-kanban/internal/config"
)

type Role string

const (
	RoleReviewPermission Role = "review-permission"
	RoleReviewAdvice     Role = "review-advice"
	RoleDispatcher       Role = "dispatcher"
	RoleSessionLink      Role = "session-link"
	RoleMemorySummarizer Role = "memory-summarizer"
)

type ScopeKind string

const (
	ScopeGlobal  ScopeKind = "global"
	ScopeHost    ScopeKind = "host"
	ScopeSession ScopeKind = "session"
	ScopeWindow  ScopeKind = "window"
	ScopePane    ScopeKind = "pane"
)

type Scope struct {
	Host    string `json:"host,omitempty"`
	Session string `json:"session,omitempty"`
	Window  string `json:"window,omitempty"`
	Pane    string `json:"pane,omitempty"`
}

func (s Scope) Kind() ScopeKind {
	switch {
	case strings.TrimSpace(s.Pane) != "":
		return ScopePane
	case strings.TrimSpace(s.Window) != "":
		return ScopeWindow
	case strings.TrimSpace(s.Session) != "":
		return ScopeSession
	case strings.TrimSpace(s.Host) != "":
		return ScopeHost
	default:
		return ScopeGlobal
	}
}

func (s Scope) Key() string {
	parts := []string{string(s.Kind())}
	for _, part := range []string{s.Host, s.Session, s.Window, s.Pane} {
		if strings.TrimSpace(part) != "" {
			parts = append(parts, strings.TrimSpace(part))
		}
	}
	return strings.Join(parts, "/")
}

func (s Scope) Parent() (Scope, bool) {
	switch s.Kind() {
	case ScopePane:
		s.Pane = ""
	case ScopeWindow:
		s.Window = ""
	case ScopeSession:
		s.Session = ""
	case ScopeHost:
		s.Host = ""
	default:
		return Scope{}, false
	}
	return s, true
}

func (s Scope) Path() []Scope {
	path := []Scope{s}
	for {
		parent, ok := path[len(path)-1].Parent()
		if !ok {
			break
		}
		path = append(path, parent)
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}

type AgentSpec struct {
	ID        string   `json:"id"`
	Role      Role     `json:"role"`
	Scope     Scope    `json:"scope"`
	Backend   string   `json:"backend"`
	Skill     string   `json:"skill,omitempty"`
	SkillPath string   `json:"skill_path,omitempty"`
	Agent     string   `json:"agent"`
	Command   string   `json:"command"`
	Args      []string `json:"args,omitempty"`
	Shared    bool     `json:"shared"`
}

func SpecsForScope(cfg config.AgentMeshConfig, scope Scope) []AgentSpec {
	if !cfg.Enabled {
		return nil
	}
	specs := make([]AgentSpec, 0, len(cfg.Policies))
	for _, policy := range cfg.Policies {
		if !policy.Enabled {
			continue
		}
		role := NormalizeRole(policy.Role)
		if role == "" || NormalizeScopeKind(policy.Scope) != scope.Kind() {
			continue
		}
		agentName := strings.TrimSpace(policy.Agent)
		if agentName == "" {
			agentName = cfg.DefaultAgent
		}
		backend := NormalizeBackend(policy.Backend)
		if backend == "" {
			backend = NormalizeBackend(agentName)
		}
		skill := strings.TrimSpace(policy.Skill)
		if skill == "" {
			skill = string(role)
		}
		command := strings.TrimSpace(policy.Command)
		if command == "" {
			command = config.DefaultMainAgentCommand(agentName)
		}
		idName := strings.TrimSpace(policy.Name)
		if idName == "" {
			idName = string(role)
		}
		specs = append(specs, AgentSpec{
			ID:        AgentID(idName, scope, cfg.SharedShortAgent),
			Role:      role,
			Scope:     scope,
			Backend:   backend,
			Skill:     skill,
			SkillPath: MeshSkillPath(cfg.SkillRoot, skill),
			Agent:     agentName,
			Command:   command,
			Args:      append([]string(nil), policy.Args...),
			Shared:    cfg.SharedShortAgent,
		})
	}
	sort.Slice(specs, func(i, j int) bool {
		if specs[i].Role == specs[j].Role {
			return specs[i].ID < specs[j].ID
		}
		return specs[i].Role < specs[j].Role
	})
	return specs
}

func NormalizeBackend(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "codex", "":
		return "codex"
	case "claude", "claude-code":
		return "claude-code"
	case "builtin", "internal":
		return "builtin"
	case "hermes":
		return "hermes"
	case "command", "exec":
		return "command"
	default:
		return strings.TrimSpace(value)
	}
}

func MeshSkillPath(skillRoot string, skill string) string {
	skill = strings.TrimSpace(skill)
	if skill == "" {
		return ""
	}
	if strings.TrimSpace(skillRoot) == "" {
		skillRoot = "mesh-skills"
	}
	return filepath.Join(skillRoot, skill, "SKILL.md")
}

func NormalizeRole(value string) Role {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(RoleReviewPermission), "permission", "approval":
		return RoleReviewPermission
	case string(RoleReviewAdvice), "review-result", "advice":
		return RoleReviewAdvice
	case string(RoleDispatcher), "dispatch", "task-dispatcher":
		return RoleDispatcher
	case string(RoleSessionLink), "link", "linker":
		return RoleSessionLink
	case string(RoleMemorySummarizer), "summarizer", "memory":
		return RoleMemorySummarizer
	default:
		return ""
	}
}

func NormalizeScopeKind(value string) ScopeKind {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(ScopePane):
		return ScopePane
	case string(ScopeWindow):
		return ScopeWindow
	case string(ScopeSession):
		return ScopeSession
	case string(ScopeHost), "device":
		return ScopeHost
	case string(ScopeGlobal), "":
		return ScopeGlobal
	default:
		return ""
	}
}

func AgentID(name string, scope Scope, shared bool) string {
	if shared {
		return "shared/" + strings.TrimSpace(name)
	}
	return strings.TrimSpace(name) + "@" + scope.Key()
}

type MemoryNode struct {
	Scope     Scope     `json:"scope"`
	Title     string    `json:"title,omitempty"`
	Summary   string    `json:"summary,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type MemoryTree struct {
	Nodes map[string]MemoryNode `json:"nodes"`
}

func NewMemoryTree() MemoryTree {
	return MemoryTree{Nodes: map[string]MemoryNode{}}
}

func (t *MemoryTree) Upsert(scope Scope, title string, summary string, at time.Time) MemoryNode {
	if t.Nodes == nil {
		t.Nodes = map[string]MemoryNode{}
	}
	if at.IsZero() {
		at = time.Now()
	}
	node := t.Nodes[scope.Key()]
	node.Scope = scope
	if strings.TrimSpace(title) != "" {
		node.Title = title
	}
	if strings.TrimSpace(summary) != "" {
		node.Summary = summary
	}
	node.UpdatedAt = at
	t.Nodes[scope.Key()] = node
	return node
}

func (t MemoryTree) Context(scope Scope) []MemoryNode {
	out := make([]MemoryNode, 0, len(scope.Path()))
	for _, pathScope := range scope.Path() {
		if node, ok := t.Nodes[pathScope.Key()]; ok {
			out = append(out, node)
		}
	}
	return out
}

type MailStatus string

const (
	MailQueued   MailStatus = "queued"
	MailRead     MailStatus = "read"
	MailArchived MailStatus = "archived"
)

type Mail struct {
	ID        string     `json:"id"`
	From      string     `json:"from"`
	To        string     `json:"to"`
	Subject   string     `json:"subject"`
	Body      string     `json:"body"`
	Scope     Scope      `json:"scope"`
	Status    MailStatus `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
}

type Mailbox struct {
	Messages []Mail `json:"messages"`
}

func (b *Mailbox) Send(from string, to string, subject string, body string, scope Scope, at time.Time) Mail {
	if at.IsZero() {
		at = time.Now()
	}
	mail := Mail{
		ID:        fmt.Sprintf("%s/%d/%03d", scope.Key(), at.UnixNano(), len(b.Messages)+1),
		From:      strings.TrimSpace(from),
		To:        strings.TrimSpace(to),
		Subject:   strings.TrimSpace(subject),
		Body:      strings.TrimSpace(body),
		Scope:     scope,
		Status:    MailQueued,
		CreatedAt: at,
	}
	b.Messages = append(b.Messages, mail)
	return mail
}

func (b Mailbox) Inbox(agentID string) []Mail {
	out := make([]Mail, 0)
	for _, mail := range b.Messages {
		if mail.To == agentID && mail.Status != MailArchived {
			out = append(out, mail)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out
}
