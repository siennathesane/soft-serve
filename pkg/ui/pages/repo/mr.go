package repo

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/spinner"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/ui/common"
	"github.com/charmbracelet/soft-serve/pkg/ui/components/code"
	"github.com/charmbracelet/soft-serve/pkg/ui/components/selector"
)

type mrView int

const (
	mrViewLoading mrView = iota
	mrViewList
	mrViewDetail
)

// MergeRequests is the merge requests component.
type MergeRequests struct {
	common      common.Common
	selector    *selector.Selector
	code        *code.Code
	activeView  mrView
	repo        proto.Repository
	ref         *git.Reference
	spinner     spinner.Model
	items       []MRItem
	selectedMR  *models.MergeRequest
	mrDetails   string
	stateFilter string
}

// MRItemsMsg is a message for merge request items.
type MRItemsMsg []MRItem

// MRDetailMsg is a message for merge request details.
type MRDetailMsg struct {
	MR      models.MergeRequest
	Details string
}

// MRActionMsg is a message for MR actions.
type MRActionMsg struct {
	Action string
	MRID   int64
}

// NewMergeRequests creates a new merge requests component.
func NewMergeRequests(c common.Common) *MergeRequests {
	mr := &MergeRequests{
		common:      c,
		activeView:  mrViewLoading,
		stateFilter: "open",
	}

	s := selector.New(c, []selector.IdentifiableItem{}, MRItemDelegate{&c})
	s.SetShowFilter(true)
	s.SetShowHelp(false)
	s.SetShowPagination(true)
	s.SetShowStatusBar(false)
	s.SetShowTitle(false)
	s.DisableQuitKeybindings()
	mr.selector = s

	codeViewer := code.New(c, "", "")
	codeViewer.NoContentStyle = codeViewer.NoContentStyle.SetString("No merge request selected")
	mr.code = codeViewer

	sp := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(c.Styles.Spinner))
	mr.spinner = sp

	return mr
}

// SetSize implements common.Component.
func (mr *MergeRequests) SetSize(width, height int) {
	mr.common.SetSize(width, height)
	mr.selector.SetSize(width, height)
	mr.code.SetSize(width, height)
}

// ShortHelp implements help.KeyMap.
func (mr *MergeRequests) ShortHelp() []key.Binding {
	k := mr.common.KeyMap
	switch mr.activeView {
	case mrViewList:
		return []key.Binding{
			k.UpDown,
			k.Select,
		}
	case mrViewDetail:
		return []key.Binding{
			k.UpDown,
			k.Back,
		}
	}
	return []key.Binding{}
}

// FullHelp implements help.KeyMap.
func (mr *MergeRequests) FullHelp() [][]key.Binding {
	k := mr.common.KeyMap
	switch mr.activeView {
	case mrViewList:
		return [][]key.Binding{
			{k.UpDown, k.Select},
			{k.Back},
		}
	case mrViewDetail:
		return [][]key.Binding{
			{k.UpDown, k.Back},
		}
	}
	return [][]key.Binding{}
}

// Init implements tea.Model.
func (mr *MergeRequests) Init() tea.Cmd {
	mr.activeView = mrViewLoading
	return tea.Batch(
		mr.spinner.Tick,
		mr.fetchMRsCmd,
	)
}

// Update implements tea.Model.
func (mr *MergeRequests) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)

	switch msg := msg.(type) {
	case RepoMsg:
		mr.repo = msg
		return mr, mr.Init()

	case RefMsg:
		mr.ref = msg
		return mr, mr.Init()

	case MRItemsMsg:
		mr.activeView = mrViewList
		mr.items = msg
		items := make([]selector.IdentifiableItem, len(msg))
		for i, item := range msg {
			items[i] = item
		}
		cmds = append(cmds, mr.selector.SetItems(items))

	case MRDetailMsg:
		mr.activeView = mrViewDetail
		mr.selectedMR = &msg.MR
		mr.mrDetails = msg.Details
		cmds = append(cmds, mr.code.SetContent(msg.Details, ""))

	case selector.SelectMsg:
		switch item := msg.IdentifiableItem.(type) {
		case MRItem:
			mr.selectedMR = &item.MR
			cmds = append(cmds, mr.fetchMRDetailCmd(item.MR.ID))
		}

	case tea.KeyPressMsg:
		switch mr.activeView {
		case mrViewList:
			switch {
			case key.Matches(msg, mr.common.KeyMap.SelectItem):
				cmds = append(cmds, mr.selector.SelectItemCmd)
			}
		case mrViewDetail:
			switch {
			case key.Matches(msg, mr.common.KeyMap.Back):
				mr.activeView = mrViewList
				mr.selectedMR = nil
				return mr, nil
			}
		}

	case spinner.TickMsg:
		if mr.activeView == mrViewLoading && mr.spinner.ID() == msg.ID {
			s, cmd := mr.spinner.Update(msg)
			mr.spinner = s
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case common.ErrorMsg:
		mr.activeView = mrViewList
	}

	switch mr.activeView {
	case mrViewList:
		s, cmd := mr.selector.Update(msg)
		mr.selector = s.(*selector.Selector)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case mrViewDetail:
		c, cmd := mr.code.Update(msg)
		mr.code = c.(*code.Code)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return mr, tea.Batch(cmds...)
}

// View implements tea.Model.
func (mr *MergeRequests) View() string {
	switch mr.activeView {
	case mrViewLoading:
		return renderLoading(mr.common, mr.spinner)
	case mrViewList:
		return mr.selector.View()
	case mrViewDetail:
		return mr.code.View()
	}
	return ""
}

// StatusBarValue implements statusbar.StatusBar.
func (mr *MergeRequests) StatusBarValue() string {
	switch mr.activeView {
	case mrViewList:
		return fmt.Sprintf("Merge Requests (%d)", len(mr.items))
	case mrViewDetail:
		if mr.selectedMR != nil {
			return fmt.Sprintf("MR #%d", mr.selectedMR.ID)
		}
		return "Merge Request"
	}
	return ""
}

// StatusBarInfo implements statusbar.StatusBar.
func (mr *MergeRequests) StatusBarInfo() string {
	switch mr.activeView {
	case mrViewList:
		return fmt.Sprintf("Filter: %s", mr.stateFilter)
	case mrViewDetail:
		if mr.selectedMR != nil {
			return fmt.Sprintf("%s → %s • %s",
				mr.selectedMR.SourceBranch,
				mr.selectedMR.TargetBranch,
				mr.selectedMR.State.String())
		}
	}
	return ""
}

// SpinnerID implements common.TabComponent.
func (mr *MergeRequests) SpinnerID() int {
	return mr.spinner.ID()
}

// TabName implements common.TabComponent.
func (mr *MergeRequests) TabName() string {
	return "Merge Requests"
}

// Path implements common.TabComponent.
func (mr *MergeRequests) Path() string {
	if mr.selectedMR != nil {
		return fmt.Sprintf("#%d", mr.selectedMR.ID)
	}
	return ""
}

// fetchMRsCmd fetches merge requests for the repository.
func (mr *MergeRequests) fetchMRsCmd() tea.Msg {
	if mr.repo == nil {
		return common.ErrorMsg(common.ErrMissingRepo)
	}

	ctx := mr.common.Context()
	be := backend.FromContext(ctx)

	// Parse state filter
	var state *models.MergeRequestState
	switch mr.stateFilter {
	case "open":
		s := models.MergeRequestStateOpen
		state = &s
	case "merged":
		s := models.MergeRequestStateMerged
		state = &s
	case "closed":
		s := models.MergeRequestStateClosed
		state = &s
	}

	mrs, err := be.ListMergeRequests(ctx, mr.repo.Name(), state)
	if err != nil {
		return common.ErrorMsg(err)
	}

	items := make([]MRItem, 0, len(mrs))
	for _, m := range mrs {
		// Get author name
		authorName := "unknown"
		if m.AuthorID > 0 {
			author, err := be.UserByID(ctx, m.AuthorID)
			if err == nil && author != nil {
				authorName = author.Username()
			}
		}

		items = append(items, MRItem{
			MR:         m,
			AuthorName: authorName,
		})
	}

	// Sort by update time (most recent first)
	sort.Sort(MRItems(items))

	return MRItemsMsg(items)
}

// fetchMRDetailCmd fetches details for a specific merge request.
func (mr *MergeRequests) fetchMRDetailCmd(mrID int64) tea.Cmd {
	return func() tea.Msg {
		if mr.repo == nil {
			return common.ErrorMsg(common.ErrMissingRepo)
		}

		ctx := mr.common.Context()
		be := backend.FromContext(ctx)

		m, err := be.GetMergeRequest(ctx, mr.repo.Name(), mrID)
		if err != nil {
			return common.ErrorMsg(err)
		}

		// Build detailed view
		details := mr.buildMRDetails(ctx, m)

		return MRDetailMsg{
			MR:      m,
			Details: details,
		}
	}
}

// buildMRDetails builds a detailed text view of the merge request.
func (mr *MergeRequests) buildMRDetails(ctx context.Context, m models.MergeRequest) string {
	var sb strings.Builder
	be := backend.FromContext(ctx)

	st := mr.common.Styles.MR

	// Header
	sb.WriteString(st.DetailTitle.Render(fmt.Sprintf("Merge Request #%d", m.ID)))
	sb.WriteString("\n\n")

	// Title
	sb.WriteString(st.DetailLabel.Render("Title: "))
	sb.WriteString(m.Title)
	sb.WriteString("\n\n")

	// Description
	if m.Description != "" {
		sb.WriteString(st.DetailLabel.Render("Description:"))
		sb.WriteString("\n")
		sb.WriteString(m.Description)
		sb.WriteString("\n\n")
	}

	// Branches
	sb.WriteString(st.DetailLabel.Render("Branches:"))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  %s → %s\n\n", m.SourceBranch, m.TargetBranch))

	// State
	sb.WriteString(st.DetailLabel.Render("State: "))
	sb.WriteString(m.State.String())
	sb.WriteString("\n\n")

	// Author
	if m.AuthorID > 0 {
		author, err := be.UserByID(ctx, m.AuthorID)
		if err == nil && author != nil {
			sb.WriteString(st.DetailLabel.Render("Author: "))
			sb.WriteString(author.Username())
			sb.WriteString("\n\n")
		}
	}

	// Timestamps
	sb.WriteString(st.DetailLabel.Render("Created: "))
	sb.WriteString(m.CreatedAt.Format("2006-01-02 15:04:05"))
	sb.WriteString("\n")

	sb.WriteString(st.DetailLabel.Render("Updated: "))
	sb.WriteString(m.UpdatedAt.Format("2006-01-02 15:04:05"))
	sb.WriteString("\n")

	if m.MergedAt.Valid {
		sb.WriteString(st.DetailLabel.Render("Merged: "))
		sb.WriteString(m.MergedAt.Time.Format("2006-01-02 15:04:05"))
		if m.MergedBy.Valid {
			mergedBy, err := be.UserByID(ctx, m.MergedBy.Int64)
			if err == nil && mergedBy != nil {
				sb.WriteString(fmt.Sprintf(" by %s", mergedBy.Username()))
			}
		}
		sb.WriteString("\n")
	}

	if m.ClosedAt.Valid {
		sb.WriteString(st.DetailLabel.Render("Closed: "))
		sb.WriteString(m.ClosedAt.Time.Format("2006-01-02 15:04:05"))
		if m.ClosedBy.Valid {
			closedBy, err := be.UserByID(ctx, m.ClosedBy.Int64)
			if err == nil && closedBy != nil {
				sb.WriteString(fmt.Sprintf(" by %s", closedBy.Username()))
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(st.DetailSeparator.Render(strings.Repeat("─", 80)))
	sb.WriteString("\n\n")

	// Try to show diff
	sb.WriteString(st.DetailLabel.Render("Changes:"))
	sb.WriteString("\n\n")

	r, err := mr.repo.Open()
	if err == nil {
		diff, err := mr.getDiff(r, m.SourceBranch, m.TargetBranch)
		if err == nil && diff != "" {
			sb.WriteString(diff)
		} else {
			sb.WriteString("Unable to generate diff\n")
		}
	}

	return sb.String()
}

// getDiff gets the diff between two branches.
func (mr *MergeRequests) getDiff(repo *git.Repository, source, target string) (string, error) {
	// Get commit for source branch
	commit, err := repo.CatFileCommit(fmt.Sprintf("refs/heads/%s", source))
	if err != nil {
		return "", fmt.Errorf("failed to get source commit: %w", err)
	}

	// Get diff for the commit
	diff, err := repo.Diff(commit)
	if err != nil {
		return "", fmt.Errorf("failed to get diff: %w", err)
	}

	return diff.Patch(), nil
}
