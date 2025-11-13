package repo

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/list"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/ui/common"
	"github.com/dustin/go-humanize"
	"github.com/muesli/reflow/truncate"
)

// IssueItem is an issue item.
type IssueItem struct {
	Issue      models.Issue
	AuthorName string
}

// ID implements selector.IdentifiableItem.
func (i IssueItem) ID() string {
	return fmt.Sprintf("%d", i.Issue.ID)
}

// Title implements list.DefaultItem.
func (i IssueItem) Title() string {
	return i.Issue.Title
}

// Description implements list.DefaultItem.
func (i IssueItem) Description() string {
	return fmt.Sprintf("#%d • %s",
		i.Issue.ID,
		i.Issue.State.String())
}

// FilterValue implements list.Item.
func (i IssueItem) FilterValue() string {
	return fmt.Sprintf("%d %s", i.Issue.ID, i.Issue.Title)
}

// IssueItems is a list of issues.
type IssueItems []IssueItem

// Len implements sort.Interface.
func (items IssueItems) Len() int { return len(items) }

// Swap implements sort.Interface.
func (items IssueItems) Swap(i, j int) { items[i], items[j] = items[j], items[i] }

// Less implements sort.Interface (most recent first).
func (items IssueItems) Less(i, j int) bool {
	return items[i].Issue.UpdatedAt.After(items[j].Issue.UpdatedAt)
}

// IssueItemDelegate is the delegate for the issue item.
type IssueItemDelegate struct {
	common *common.Common
}

// Height implements list.ItemDelegate.
func (d IssueItemDelegate) Height() int { return 2 }

// Spacing implements list.ItemDelegate.
func (d IssueItemDelegate) Spacing() int { return 1 }

// Update implements list.ItemDelegate.
func (d IssueItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	item, ok := m.SelectedItem().(IssueItem)
	if !ok {
		return nil
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, d.common.KeyMap.Copy):
			return copyCmd(item.ID(), fmt.Sprintf("Issue #%s copied to clipboard", item.ID()))
		}
	}
	return nil
}

// Render implements list.ItemDelegate.
func (d IssueItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(IssueItem)
	if !ok {
		return
	}

	isActive := index == m.Index()
	s := d.common.Styles.MR // Reuse MR styles for now
	st := s.Normal
	selector := "  "
	if isActive {
		st = s.Active
		selector = s.ItemSelector.String()
	}

	horizontalFrameSize := st.Base.GetHorizontalFrameSize()

	// Issue number and state badge
	var stateBadge string
	var stateSt lipgloss.Style
	switch i.Issue.State {
	case models.IssueStateOpen:
		stateSt = st.ItemStateOpen
		stateBadge = "●"
	case models.IssueStateClosed:
		stateSt = st.ItemStateClosed
		stateBadge = "✕"
	}

	issueNum := st.ItemNumber.Render(fmt.Sprintf("#%d", i.Issue.ID))
	badge := stateSt.Render(stateBadge)

	// Title
	title := i.Issue.Title
	titleMargin := m.Width() -
		horizontalFrameSize -
		lipgloss.Width(selector) -
		lipgloss.Width(issueNum) -
		lipgloss.Width(badge) -
		4 // padding
	if titleMargin > 0 {
		title = common.TruncateString(title, titleMargin)
	}
	title = st.ItemTitle.Render(title)

	// First line: selector + badge + #num + title
	firstLine := lipgloss.JoinHorizontal(lipgloss.Top,
		selector,
		badge,
		" ",
		issueNum,
		" ",
		title,
	)

	// Second line: author + time
	author := ""
	if i.AuthorName != "" {
		author = "by " + i.AuthorName
	}
	authorRendered := st.ItemAuthor.Render(author)

	timeAgo := humanize.Time(i.Issue.UpdatedAt)
	timeRendered := st.ItemTime.Render(" • " + timeAgo)

	secondLineContent := authorRendered + timeRendered

	// Calculate padding for second line to align with first line
	secondLineMargin := m.Width() -
		horizontalFrameSize -
		lipgloss.Width(secondLineContent) -
		2 // for selector width
	padding := ""
	if secondLineMargin > 0 {
		padding = "  " // align with content after selector
	}

	secondLine := padding + truncate.String(secondLineContent,
		uint(m.Width()-horizontalFrameSize-2)) //nolint:gosec

	// Combine lines
	content := lipgloss.JoinVertical(lipgloss.Left,
		firstLine,
		secondLine,
	)

	fmt.Fprint(w, //nolint:errcheck
		d.common.Zone.Mark(
			i.ID(),
			st.Base.Render(content),
		),
	)
}
