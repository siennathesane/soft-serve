package repo

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/list"
	"github.com/charmbracelet/bubbles/v2/textinput"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/ui/common"
	"github.com/charmbracelet/soft-serve/pkg/ui/components/selector"
)

type mrFormStep int

const (
	stepSelectTarget mrFormStep = iota
	stepEnterDetails
	stepSubmitting
	stepComplete
)

// MRFormMsg is a message to start the MR creation form.
type MRFormMsg struct {
	SourceBranch string
}

// MRCreatedMsg is a message sent when an MR is successfully created.
type MRCreatedMsg struct {
	MRID   int64
	RepoName string
}

// MRForm is a component for creating merge requests.
type MRForm struct {
	common       common.Common
	repo         proto.Repository
	sourceBranch string
	targetBranch string
	branches     []string

	// Form state
	step         mrFormStep
	selector     *selector.Selector
	titleInput   textinput.Model
	descInput    textinput.Model
	focusIndex   int

	// Result
	createdMRID  int64
	err          error
}

// NewMRForm creates a new merge request form.
func NewMRForm(c common.Common, sourceBranch string) *MRForm {
	form := &MRForm{
		common:       c,
		sourceBranch: sourceBranch,
		step:         stepSelectTarget,
		focusIndex:   0,
	}

	// Setup title input
	titleInput := textinput.New()
	titleInput.Placeholder = "Enter merge request title"
	titleInput.Focus()
	titleInput.CharLimit = 200
	titleInput.SetWidth(70)
	form.titleInput = titleInput

	// Setup description input
	descInput := textinput.New()
	descInput.Placeholder = "Enter description (optional)"
	descInput.CharLimit = 2000
	descInput.SetWidth(70)
	form.descInput = descInput

	return form
}

// Init implements tea.Model.
func (f *MRForm) Init() tea.Cmd {
	return f.fetchBranchesCmd()
}

// Update implements tea.Model.
func (f *MRForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case RepoMsg:
		f.repo = msg
		return f, f.fetchBranchesCmd()

	case RefItemsMsg:
		// Build branch list from refs
		f.branches = make([]string, 0)
		for _, item := range msg.items {
			if refItem, ok := item.(RefItem); ok {
				branchName := refItem.Short()
				if branchName != f.sourceBranch {
					f.branches = append(f.branches, branchName)
				}
			}
		}

		// Create selector for target branch
		items := make([]selector.IdentifiableItem, len(f.branches))
		for i, branch := range f.branches {
			items[i] = branchSelectorItem{name: branch}
		}

		sel := selector.New(f.common, items, branchSelectorDelegate{&f.common})
		sel.SetShowFilter(false)
		sel.SetShowHelp(false)
		sel.SetShowPagination(false)
		sel.SetShowStatusBar(false)
		sel.SetShowTitle(false)
		sel.DisableQuitKeybindings()
		f.selector = sel

	case selector.SelectMsg:
		if f.step == stepSelectTarget {
			if item, ok := msg.IdentifiableItem.(branchSelectorItem); ok {
				f.targetBranch = item.name
				f.step = stepEnterDetails
				f.titleInput.Focus()
				return f, textinput.Blink
			}
		}

	case tea.KeyPressMsg:
		switch f.step {
		case stepSelectTarget:
			if f.selector != nil {
				sel, cmd := f.selector.Update(msg)
				f.selector = sel.(*selector.Selector)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}

		case stepEnterDetails:
			switch {
			case key.Matches(msg, f.common.KeyMap.Back):
				f.step = stepSelectTarget
				return f, nil

			case msg.String() == "ctrl+s":
				// Submit the form
				f.step = stepSubmitting
				return f, f.createMRCmd()

			case msg.String() == "tab", msg.String() == "shift+tab":
				// Switch focus between inputs
				if msg.String() == "tab" {
					f.focusIndex++
				} else {
					f.focusIndex--
				}

				if f.focusIndex > 1 {
					f.focusIndex = 0
				} else if f.focusIndex < 0 {
					f.focusIndex = 1
				}

				if f.focusIndex == 0 {
					f.titleInput.Focus()
					f.descInput.Blur()
					cmds = append(cmds, textinput.Blink)
				} else {
					f.titleInput.Blur()
					f.descInput.Focus()
					cmds = append(cmds, textinput.Blink)
				}

			default:
				// Update focused input
				if f.focusIndex == 0 {
					var cmd tea.Cmd
					f.titleInput, cmd = f.titleInput.Update(msg)
					if cmd != nil {
						cmds = append(cmds, cmd)
					}
				} else {
					var cmd tea.Cmd
					f.descInput, cmd = f.descInput.Update(msg)
					if cmd != nil {
						cmds = append(cmds, cmd)
					}
				}
			}
		}

	case MRCreatedMsg:
		f.step = stepComplete
		f.createdMRID = msg.MRID
		// Navigate to the new MR after a brief moment
		return f, tea.Sequence(
			tea.Tick(time.Millisecond*500, func(time.Time) tea.Msg {
				return SwitchTabMsg(nil) // This will be caught by parent to switch tabs
			}),
		)

	case common.ErrorMsg:
		f.err = msg
		f.step = stepEnterDetails // Go back to form
	}

	return f, tea.Batch(cmds...)
}

// View implements tea.Model.
func (f *MRForm) View() string {
	s := f.common.Styles

	switch f.step {
	case stepSelectTarget:
		return f.viewSelectTarget()

	case stepEnterDetails:
		return f.viewEnterDetails()

	case stepSubmitting:
		return s.Spinner.Render("Creating merge request...")

	case stepComplete:
		return s.NoContent.Render(fmt.Sprintf("✓ Created merge request #%d", f.createdMRID))
	}

	return ""
}

func (f *MRForm) viewSelectTarget() string {
	s := f.common.Styles

	var b strings.Builder

	title := s.MR.DetailTitle.Render("Create Merge Request")
	b.WriteString(title)
	b.WriteString("\n\n")

	b.WriteString(s.MR.DetailLabel.Render("Source Branch: "))
	b.WriteString(f.sourceBranch)
	b.WriteString("\n\n")

	b.WriteString(s.MR.DetailLabel.Render("Select Target Branch:"))
	b.WriteString("\n\n")

	if f.selector != nil {
		b.WriteString(f.selector.View())
	} else {
		b.WriteString("Loading branches...")
	}

	b.WriteString("\n\n")
	b.WriteString(s.HelpValue.Render("↑/↓: select • enter: continue • esc: cancel"))

	return b.String()
}

func (f *MRForm) viewEnterDetails() string {
	s := f.common.Styles

	var b strings.Builder

	title := s.MR.DetailTitle.Render("Create Merge Request")
	b.WriteString(title)
	b.WriteString("\n\n")

	// Show branches
	branches := fmt.Sprintf("%s → %s", f.sourceBranch, f.targetBranch)
	b.WriteString(s.MR.DetailLabel.Render(branches))
	b.WriteString("\n\n")

	// Title input
	b.WriteString(s.MR.DetailLabel.Render("Title:"))
	b.WriteString("\n")
	b.WriteString(f.titleInput.View())
	b.WriteString("\n\n")

	// Description input
	b.WriteString(s.MR.DetailLabel.Render("Description:"))
	b.WriteString("\n")
	b.WriteString(f.descInput.View())
	b.WriteString("\n\n")

	if f.err != nil {
		b.WriteString(s.ErrorBody.Render(fmt.Sprintf("Error: %v", f.err)))
		b.WriteString("\n\n")
	}

	// Buttons
	createBtn := "[Create]"
	cancelBtn := "[Cancel]"

	if f.focusIndex == 0 || f.focusIndex == 1 {
		createBtn = s.MR.DetailLabel.Render(createBtn)
	}

	b.WriteString(createBtn)
	b.WriteString("  ")
	b.WriteString(cancelBtn)
	b.WriteString("\n\n")

	b.WriteString(s.HelpValue.Render("tab: next field • ctrl+s: create • esc: back"))

	return b.String()
}

// fetchBranchesCmd fetches branches for the repository.
func (f *MRForm) fetchBranchesCmd() tea.Cmd {
	return func() tea.Msg {
		if f.repo == nil {
			return common.ErrorMsg(common.ErrMissingRepo)
		}

		r, err := f.repo.Open()
		if err != nil {
			return common.ErrorMsg(err)
		}

		refs, err := r.References()
		if err != nil {
			return common.ErrorMsg(err)
		}

		refItems := make([]RefItem, 0)
		for _, ref := range refs {
			if ref.IsBranch() {
				commit, _ := r.CatFileCommit(ref.ID)
				refItems = append(refItems, RefItem{
					Reference: ref,
					Commit:    commit,
				})
			}
		}

		// Sort by commit date
		sort.Sort(RefItems(refItems))

		items := make([]selector.IdentifiableItem, len(refItems))
		for i, item := range refItems {
			items[i] = item
		}

		return RefItemsMsg{
			prefix: git.RefsHeads,
			items:  items,
		}
	}
}

// createMRCmd creates the merge request via backend.
func (f *MRForm) createMRCmd() tea.Cmd {
	return func() tea.Msg {
		if f.repo == nil {
			return common.ErrorMsg(common.ErrMissingRepo)
		}

		ctx := f.common.Context()
		be := backend.FromContext(ctx)

		title := strings.TrimSpace(f.titleInput.Value())
		if title == "" {
			return common.ErrorMsg(fmt.Errorf("title is required"))
		}

		description := strings.TrimSpace(f.descInput.Value())

		mrID, err := be.CreateMergeRequest(ctx, f.repo.Name(), title, description, f.sourceBranch, f.targetBranch)
		if err != nil {
			return common.ErrorMsg(err)
		}

		return MRCreatedMsg{
			MRID:     mrID,
			RepoName: f.repo.Name(),
		}
	}
}

// branchSelectorItem is a simple item for branch selection.
type branchSelectorItem struct {
	name string
}

// ID implements selector.IdentifiableItem.
func (b branchSelectorItem) ID() string {
	return b.name
}

// FilterValue implements list.Item.
func (b branchSelectorItem) FilterValue() string {
	return b.name
}

// Description implements list.DefaultItem.
func (b branchSelectorItem) Description() string {
	return ""
}

// Title implements list.DefaultItem.
func (b branchSelectorItem) Title() string {
	return b.name
}

// branchSelectorDelegate renders branch items in the selector.
type branchSelectorDelegate struct {
	common *common.Common
}

// Height implements list.ItemDelegate.
func (d branchSelectorDelegate) Height() int { return 1 }

// Spacing implements list.ItemDelegate.
func (d branchSelectorDelegate) Spacing() int { return 0 }

// Update implements list.ItemDelegate.
func (d branchSelectorDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

// Render implements list.ItemDelegate.
func (d branchSelectorDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(branchSelectorItem)
	if !ok {
		return
	}

	s := d.common.Styles
	selector := "  "
	style := s.Ref.Normal.Item

	if index == m.Index() {
		selector = s.Ref.ItemSelector.String()
		style = s.Ref.Active.Item
	}

	_, _ = fmt.Fprint(w, d.common.Zone.Mark(
		item.ID(),
		selector+style.Render(item.name),
	))
}
