package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/huh"
	"github.com/mattn/go-isatty"

	"github.com/thurstonsand/wt/internal/worktree"
)

func isInteractive() bool {
	return isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
}

type pruneSelection struct {
	stale    []worktree.StaleWorktree
	branches []repoBranches
}

type pruneItemKind int

const (
	kindStaleDir pruneItemKind = iota
	kindBranch
)

type pruneItem struct {
	kind pruneItemKind
	idx  int
}

func pickPruneItems(stale []worktree.StaleWorktree, branchResults []repoBranches, allChecked bool) (*pruneSelection, error) {
	type flatBranch struct {
		repo   string
		branch worktree.PrunableBranch
	}
	var flat []flatBranch
	for _, r := range branchResults {
		for _, b := range r.branches {
			flat = append(flat, flatBranch{repo: r.repo, branch: b})
		}
	}

	multiRepo := len(branchResults) > 1
	total := len(stale) + len(flat)
	items := make([]pruneItem, 0, total)
	opts := make([]huh.Option[int], 0, total)

	for i, s := range stale {
		label := fmt.Sprintf("[wt]     %s/%s", s.RepoName, s.Name)
		items = append(items, pruneItem{kind: kindStaleDir, idx: i})
		opts = append(opts, huh.NewOption(label, len(opts)).Selected(true))
	}

	for i, fb := range flat {
		detail := fmt.Sprintf("[branch] %s  (%s)", fb.branch.Branch, formatBranchDetail(fb.branch))
		if multiRepo {
			detail = fmt.Sprintf("[branch] [%s] %s  (%s)", filepath.Base(fb.repo), fb.branch.Branch, formatBranchDetail(fb.branch))
		}
		items = append(items, pruneItem{kind: kindBranch, idx: i})
		opts = append(opts, huh.NewOption(detail, len(opts)).Selected(allChecked || fb.branch.Landed))
	}

	var selected []int

	km := huh.NewDefaultKeyMap()
	km.Quit = key.NewBinding(key.WithKeys("ctrl+c", "esc", "q"), key.WithHelp("esc", "cancel"))
	km.MultiSelect.Toggle = key.NewBinding(key.WithKeys(" ", "x"), key.WithHelp("x", "toggle"))
	km.MultiSelect.SelectAll = key.NewBinding(key.WithKeys("ctrl+a"), key.WithHelp("ctrl+a", "all"))
	km.MultiSelect.SelectNone = key.NewBinding(key.WithKeys("ctrl+a"), key.WithHelp("ctrl+a", "none"), key.WithDisabled())
	km.MultiSelect.Next = key.NewBinding(key.WithKeys("enter", "tab"), key.WithHelp("enter", "confirm"))

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[int]().
				Title("Select items to prune").
				Options(opts...).
				Value(&selected).
				Filterable(false),
		),
	).WithKeyMap(km)

	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return nil, errAborted
		}
		return nil, err
	}

	if len(selected) == 0 {
		return nil, errAborted
	}

	kept := make(map[int]bool, len(selected))
	for _, idx := range selected {
		kept[idx] = true
	}

	result := &pruneSelection{}

	repoMap := make(map[string][]worktree.PrunableBranch)
	for i, item := range items {
		if !kept[i] {
			continue
		}
		switch item.kind {
		case kindStaleDir:
			result.stale = append(result.stale, stale[item.idx])
		case kindBranch:
			fb := flat[item.idx]
			repoMap[fb.repo] = append(repoMap[fb.repo], fb.branch)
		}
	}

	for _, r := range branchResults {
		if branches, ok := repoMap[r.repo]; ok {
			result.branches = append(result.branches, repoBranches{repo: r.repo, branches: branches})
		}
	}

	return result, nil
}
