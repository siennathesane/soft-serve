package repo

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/spinner"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/ui/common"
	"github.com/charmbracelet/soft-serve/pkg/ui/components/code"
	"github.com/charmbracelet/soft-serve/pkg/ui/components/selector"
)

type issueView int

const (
	issueViewLoading issueView = iota
	issueViewList
	issueViewDetail
)

// Issues is the issues component.
type Issues struct {
	common        common.Common
	selector      *selector.Selector
	code          *code.Code
	activeView    issueView
	repo          proto.Repository
	spinner       spinner.Model
	items         []IssueItem
	selectedIssue *models.Issue
	issueDetails  string
	stateFilter   string
}

// IssueItemsMsg is a message for issue items.
type IssueItemsMsg []IssueItem

// IssueDetailMsg is a message for issue details.
type IssueDetailMsg struct {
	Issue   models.Issue
	Details string
}

// NewIssues creates a new issues component.
func NewIssues(c common.Common) *Issues {
	issue := &Issues{
		common:      c,
		activeView:  issueViewLoading,
		stateFilter: "open",
	}

	s := selector.New(c, []selector.IdentifiableItem{}, IssueItemDelegate{&c})
	s.SetShowFilter(true)
	s.SetShowHelp(false)
	s.SetShowPagination(true)
	s.SetShowStatusBar(false)
	s.SetShowTitle(false)
	s.DisableQuitKeybindings()
	issue.selector = s

	codeViewer := code.New(c, "", "")
	codeViewer.NoContentStyle = codeViewer.NoContentStyle.SetString("No issue selected")
	issue.code = codeViewer

	sp := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(c.Styles.Spinner))
	issue.spinner = sp

	return issue
}

// SetSize implements common.Component.
func (i *Issues) SetSize(width, height int) {
	i.common.SetSize(width, height)
	i.selector.SetSize(width, height)
	i.code.SetSize(width, height)
}

// ShortHelp implements help.KeyMap.
func (i *Issues) ShortHelp() []key.Binding {
	k := i.common.KeyMap
	switch i.activeView {
	case issueViewList:
		return []key.Binding{
			k.UpDown,
			k.Select,
		}
	case issueViewDetail:
		return []key.Binding{
			k.UpDown,
			k.Back,
		}
	}
	return []key.Binding{}
}

// FullHelp implements help.KeyMap.
func (i *Issues) FullHelp() [][]key.Binding {
	k := i.common.KeyMap
	switch i.activeView {
	case issueViewList:
		return [][]key.Binding{
			{k.UpDown, k.Select},
			{k.Back},
		}
	case issueViewDetail:
		return [][]key.Binding{
			{k.UpDown, k.Back},
		}
	}
	return [][]key.Binding{}
}

// Init implements tea.Model.
func (i *Issues) Init() tea.Cmd {
	i.activeView = issueViewLoading
	return tea.Batch(
		i.spinner.Tick,
		i.fetchIssuesCmd,
	)
}

// Update implements tea.Model.
func (i *Issues) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)

	switch msg := msg.(type) {
	case RepoMsg:
		i.repo = msg
		return i, i.Init()

	case IssueItemsMsg:
		i.activeView = issueViewList
		i.items = msg
		items := make([]selector.IdentifiableItem, len(msg))
		for idx, item := range msg {
			items[idx] = item
		}
		cmds = append(cmds, i.selector.SetItems(items))

	case IssueDetailMsg:
		i.activeView = issueViewDetail
		i.selectedIssue = &msg.Issue
		i.issueDetails = msg.Details
		cmds = append(cmds, i.code.SetContent(msg.Details, ""))

	case selector.SelectMsg:
		switch item := msg.IdentifiableItem.(type) {
		case IssueItem:
			i.selectedIssue = &item.Issue
			cmds = append(cmds, i.fetchIssueDetailCmd(item.Issue.ID))
		}

	case tea.KeyPressMsg:
		switch i.activeView {
		case issueViewList:
			switch {
			case key.Matches(msg, i.common.KeyMap.SelectItem):
				cmds = append(cmds, i.selector.SelectItemCmd)
			}
		case issueViewDetail:
			switch {
			case key.Matches(msg, i.common.KeyMap.Back):
				i.activeView = issueViewList
				i.selectedIssue = nil
				return i, nil
			}
		}

	case spinner.TickMsg:
		if i.activeView == issueViewLoading && i.spinner.ID() == msg.ID {
			s, cmd := i.spinner.Update(msg)
			i.spinner = s
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case common.ErrorMsg:
		i.activeView = issueViewList
	}

	switch i.activeView {
	case issueViewList:
		s, cmd := i.selector.Update(msg)
		i.selector = s.(*selector.Selector)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case issueViewDetail:
		c, cmd := i.code.Update(msg)
		i.code = c.(*code.Code)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return i, tea.Batch(cmds...)
}

// View implements tea.Model.
func (i *Issues) View() string {
	switch i.activeView {
	case issueViewLoading:
		return renderLoading(i.common, i.spinner)
	case issueViewList:
		return i.selector.View()
	case issueViewDetail:
		return i.code.View()
	}
	return ""
}

// StatusBarValue implements statusbar.StatusBar.
func (i *Issues) StatusBarValue() string {
	switch i.activeView {
	case issueViewList:
		return fmt.Sprintf("Issues (%d)", len(i.items))
	case issueViewDetail:
		if i.selectedIssue != nil {
			return fmt.Sprintf("Issue #%d", i.selectedIssue.ID)
		}
		return "Issue"
	}
	return ""
}

// StatusBarInfo implements statusbar.StatusBar.
func (i *Issues) StatusBarInfo() string {
	switch i.activeView {
	case issueViewList:
		return fmt.Sprintf("Filter: %s", i.stateFilter)
	case issueViewDetail:
		if i.selectedIssue != nil {
			return i.selectedIssue.State.String()
		}
	}
	return ""
}

// SpinnerID implements common.TabComponent.
func (i *Issues) SpinnerID() int {
	return i.spinner.ID()
}

// TabName implements common.TabComponent.
func (i *Issues) TabName() string {
	return "Issues"
}

// Path implements common.TabComponent.
func (i *Issues) Path() string {
	if i.selectedIssue != nil {
		return fmt.Sprintf("#%d", i.selectedIssue.ID)
	}
	return ""
}

// fetchIssuesCmd fetches issues for the repository.
func (i *Issues) fetchIssuesCmd() tea.Msg {
	if i.repo == nil {
		return common.ErrorMsg(common.ErrMissingRepo)
	}

	ctx := i.common.Context()
	be := backend.FromContext(ctx)

	// Parse state filter
	var state *models.IssueState
	switch i.stateFilter {
	case "open":
		s := models.IssueStateOpen
		state = &s
	case "closed":
		s := models.IssueStateClosed
		state = &s
	}

	issues, err := be.ListIssues(ctx, i.repo.Name(), state)
	if err != nil {
		return common.ErrorMsg(err)
	}

	items := make([]IssueItem, 0, len(issues))
	for _, issue := range issues {
		// Get author name
		authorName := "unknown"
		if issue.AuthorID > 0 {
			author, err := be.UserByID(ctx, issue.AuthorID)
			if err == nil && author != nil {
				authorName = author.Username()
			}
		}

		items = append(items, IssueItem{
			Issue:      issue,
			AuthorName: authorName,
		})
	}

	// Sort by update time (most recent first)
	sort.Sort(IssueItems(items))

	return IssueItemsMsg(items)
}

// fetchIssueDetailCmd fetches details for a specific issue.
func (i *Issues) fetchIssueDetailCmd(issueID int64) tea.Cmd {
	return func() tea.Msg {
		if i.repo == nil {
			return common.ErrorMsg(common.ErrMissingRepo)
		}

		ctx := i.common.Context()
		be := backend.FromContext(ctx)

		issue, err := be.GetIssue(ctx, i.repo.Name(), issueID)
		if err != nil {
			return common.ErrorMsg(err)
		}

		// Build detailed view
		details := i.buildIssueDetails(ctx, issue)

		return IssueDetailMsg{
			Issue:   issue,
			Details: details,
		}
	}
}

// buildIssueDetails builds a detailed text view of the issue.
func (i *Issues) buildIssueDetails(ctx context.Context, issue models.Issue) string {
	var sb strings.Builder
	be := backend.FromContext(ctx)

	st := i.common.Styles.MR // Reuse MR styles for now

	// Header
	sb.WriteString(st.DetailTitle.Render(fmt.Sprintf("Issue #%d", issue.ID)))
	sb.WriteString("\n\n")

	// Title
	sb.WriteString(st.DetailLabel.Render("Title: "))
	sb.WriteString(issue.Title)
	sb.WriteString("\n\n")

	// Description
	if issue.Description != "" {
		sb.WriteString(st.DetailLabel.Render("Description:"))
		sb.WriteString("\n")
		sb.WriteString(issue.Description)
		sb.WriteString("\n\n")
	}

	// State
	sb.WriteString(st.DetailLabel.Render("State: "))
	sb.WriteString(issue.State.String())
	sb.WriteString("\n\n")

	// Author
	if issue.AuthorID > 0 {
		author, err := be.UserByID(ctx, issue.AuthorID)
		if err == nil && author != nil {
			sb.WriteString(st.DetailLabel.Render("Author: "))
			sb.WriteString(author.Username())
			sb.WriteString("\n\n")
		}
	}

	// Timestamps
	sb.WriteString(st.DetailLabel.Render("Created: "))
	sb.WriteString(issue.CreatedAt.Format("2006-01-02 15:04:05"))
	sb.WriteString("\n")

	sb.WriteString(st.DetailLabel.Render("Updated: "))
	sb.WriteString(issue.UpdatedAt.Format("2006-01-02 15:04:05"))
	sb.WriteString("\n")

	if issue.ClosedAt.Valid {
		sb.WriteString(st.DetailLabel.Render("Closed: "))
		sb.WriteString(issue.ClosedAt.Time.Format("2006-01-02 15:04:05"))
		if issue.ClosedBy.Valid {
			closedBy, err := be.UserByID(ctx, issue.ClosedBy.Int64)
			if err == nil && closedBy != nil {
				sb.WriteString(fmt.Sprintf(" by %s", closedBy.Username()))
			}
		}
		sb.WriteString("\n")
	}

	// Dependencies
	dependencies, err := be.GetIssueDependencies(ctx, i.repo.Name(), issue.ID)
	if err == nil && len(dependencies) > 0 {
		sb.WriteString("\n")
		sb.WriteString(st.DetailLabel.Render("Depends on:"))
		sb.WriteString("\n")
		for _, dep := range dependencies {
			sb.WriteString(fmt.Sprintf("  #%d - %s\n", dep.ID, dep.Title))
		}
	}

	// Dependents (issues that depend on this one)
	dependents, err := be.GetIssueDependents(ctx, i.repo.Name(), issue.ID)
	if err == nil && len(dependents) > 0 {
		sb.WriteString("\n")
		sb.WriteString(st.DetailLabel.Render("Blocked by:"))
		sb.WriteString("\n")
		for _, dep := range dependents {
			sb.WriteString(fmt.Sprintf("  #%d - %s\n", dep.ID, dep.Title))
		}
	}

	return sb.String()
}
