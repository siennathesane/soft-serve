package repo

import (
	"fmt"
	"io"
	"time"

	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/list"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/ui/common"
	"github.com/dustin/go-humanize"
	"github.com/muesli/reflow/truncate"
)

// MRItem is a merge request item.
type MRItem struct {
	MR         models.MergeRequest
	AuthorName string
}

// ID implements selector.IdentifiableItem.
func (i MRItem) ID() string {
	return fmt.Sprintf("%d", i.MR.ID)
}

// Title implements list.DefaultItem.
func (i MRItem) Title() string {
	return i.MR.Title
}

// Description implements list.DefaultItem.
func (i MRItem) Description() string {
	return fmt.Sprintf("#%d • %s → %s • %s",
		i.MR.ID,
		i.MR.SourceBranch,
		i.MR.TargetBranch,
		i.MR.State.String())
}

// FilterValue implements list.Item.
func (i MRItem) FilterValue() string {
	return fmt.Sprintf("%d %s", i.MR.ID, i.MR.Title)
}

// MRItems is a list of merge requests.
type MRItems []MRItem

// Len implements sort.Interface.
func (items MRItems) Len() int { return len(items) }

// Swap implements sort.Interface.
func (items MRItems) Swap(i, j int) { items[i], items[j] = items[j], items[i] }

// Less implements sort.Interface (most recent first).
func (items MRItems) Less(i, j int) bool {
	return items[i].MR.UpdatedAt.After(items[j].MR.UpdatedAt)
}

// MRItemDelegate is the delegate for the merge request item.
type MRItemDelegate struct {
	common *common.Common
}

// Height implements list.ItemDelegate.
func (d MRItemDelegate) Height() int { return 2 }

// Spacing implements list.ItemDelegate.
func (d MRItemDelegate) Spacing() int { return 1 }

// Update implements list.ItemDelegate.
func (d MRItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	item, ok := m.SelectedItem().(MRItem)
	if !ok {
		return nil
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, d.common.KeyMap.Copy):
			return copyCmd(item.ID(), fmt.Sprintf("MR #%s copied to clipboard", item.ID()))
		}
	}
	return nil
}

// Render implements list.ItemDelegate.
func (d MRItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(MRItem)
	if !ok {
		return
	}

	isActive := index == m.Index()
	s := d.common.Styles.MR
	st := s.Normal
	selector := "  "
	if isActive {
		st = s.Active
		selector = s.ItemSelector.String()
	}

	horizontalFrameSize := st.Base.GetHorizontalFrameSize()

	// MR number and state badge
	var stateBadge string
	var stateSt lipgloss.Style
	switch i.MR.State {
	case models.MergeRequestStateOpen:
		stateSt = st.ItemStateOpen
		stateBadge = "●"
	case models.MergeRequestStateMerged:
		stateSt = st.ItemStateMerged
		stateBadge = "✓"
	case models.MergeRequestStateClosed:
		stateSt = st.ItemStateClosed
		stateBadge = "✕"
	}

	mrNum := st.ItemNumber.Render(fmt.Sprintf("#%d", i.MR.ID))
	badge := stateSt.Render(stateBadge)

	// Title
	title := i.MR.Title
	titleMargin := m.Width() -
		horizontalFrameSize -
		lipgloss.Width(selector) -
		lipgloss.Width(mrNum) -
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
		mrNum,
		" ",
		title,
	)

	// Second line: branches + author + time
	branches := fmt.Sprintf("%s → %s", i.MR.SourceBranch, i.MR.TargetBranch)
	branchesRendered := st.ItemBranches.Render(branches)

	author := ""
	if i.AuthorName != "" {
		author = " • by " + i.AuthorName
	}
	authorRendered := st.ItemAuthor.Render(author)

	timeAgo := humanize.Time(i.MR.UpdatedAt)
	timeRendered := st.ItemTime.Render(" • " + timeAgo)

	secondLineContent := branchesRendered + authorRendered + timeRendered

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
