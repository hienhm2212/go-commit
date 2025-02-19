package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

var (
	commitType  string
	scope       string
	workItemID  string
	title       string
	description string
)

const maxWidth = 120

var (
	red    = lipgloss.AdaptiveColor{Light: "#FE5F86", Dark: "#FE5F86"}
	indigo = lipgloss.AdaptiveColor{Light: "#5A56E0", Dark: "#7571F9"}
	green  = lipgloss.AdaptiveColor{Light: "#02BA84", Dark: "#02BF87"}
)

type Styles struct {
	Base,
	HeaderText,
	Status,
	StatusHeader,
	Highlight,
	ErrorHeaderText,
	Help lipgloss.Style
}

func NewStyles(lg *lipgloss.Renderer) *Styles {
	s := Styles{}
	s.Base = lg.NewStyle().
		Padding(1, 4, 0, 1)
	s.HeaderText = lg.NewStyle().
		Foreground(indigo).
		Bold(true).
		Padding(0, 1, 0, 2)
	s.Status = lg.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(indigo).
		PaddingLeft(1).
		MarginTop(1)
	s.StatusHeader = lg.NewStyle().
		Foreground(green).
		Bold(true)
	s.Highlight = lg.NewStyle().
		Foreground(lipgloss.Color("212"))
	s.ErrorHeaderText = s.HeaderText.
		Foreground(red)
	s.Help = lg.NewStyle().
		Foreground(lipgloss.Color("240"))
	return &s
}

type state int

const (
	statusNormal state = iota
	stateDone
)

type Model struct {
	form   *huh.Form
	state  state
	lg     *lipgloss.Renderer
	styles *Styles
	width  int
}

func NewModel() Model {
	// theme := huh.ThemeDracula()
	m := Model{width: maxWidth}
	m.lg = lipgloss.DefaultRenderer()
	m.styles = NewStyles(m.lg)

	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Choose your type of commits").
				Options(
					huh.NewOption("Feature", "feat"),
					huh.NewOption("Fixbug", "fix"),
					huh.NewOption("Documentation", "docs"),
					huh.NewOption("Style", "style"),
					huh.NewOption("Refactor", "refactor"),
					huh.NewOption("Performance", "perf"),
					huh.NewOption("Tests", "test"),
					huh.NewOption("Maintenance", "chore"),
				).
				Value(&commitType),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("What's your scope?").
				Value(&scope),
			huh.NewInput().
				Title("What's your Work Item ID?").
				Value(&workItemID),
			huh.NewInput().
				Title("What's your commit title?").
				Value(&title),
			huh.NewText().
				Title("What's your commit description? (Optional)").
				CharLimit(400).
				Value(&description),
			huh.NewConfirm().
				Key("done").
				Title("All done?").
				Validate(func(v bool) error {
					if !v {
						return fmt.Errorf("Welp, finish up then")
					}
					return nil
				}).Affirmative("Yes").
				Negative("Wait, no"),
		),
	).
		WithWidth(80).
		WithShowHelp(true).
		WithShowErrors(false)

	return m
}

func (m Model) Init() tea.Cmd {
	return m.form.Init()
}

func min(x, y int) int {
	if x > y {
		return y
	}
	return x
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = min(msg.Width, maxWidth) - m.styles.Base.GetHorizontalFrameSize()
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Interrupt
		case "esc", "q":
			return m, tea.Quit
		}
	}

	var cmds []tea.Cmd

	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
		cmds = append(cmds, cmd)
	}

	if m.form.State == huh.StateCompleted {
		// Quit when form is done
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	s := m.styles
	switch m.form.State {
	case huh.StateCompleted:
		return fmt.Sprintf("%s(%s)[%s]: %s\n\n%s", commitType, scope, workItemID, title, description)
	default:
		var commitMsg string
		if commitType != "" {
			commitMsg = "Commit Type: " + commitType
		}
		v := strings.TrimSuffix(m.form.View(), "\n\n")
		form := m.lg.NewStyle().Margin(1, 0).Render(v)

		// right side
		var status string
		{
			var (
				currentScope       string
				scopeStr           string
				currentWorkItemID  string
				workItemIDStr      string
				currentTitle       string
				currentDescription string
			)
			if scope != "" {
				currentScope = "Scope: " + scope
				scopeStr = "(" + scope + ")"

			}
			if workItemID != "" {
				currentWorkItemID = "Work Item ID: " + workItemID
				workItemIDStr = "[" + workItemID + "]"
			}
			if title != "" {
				currentTitle = "Title: " + title
			}
			if description != "" {
				currentDescription = "Description(Optional): " + description
			}
			status = fmt.Sprintf("%s%s%s: %s\n\n%s", commitType, scopeStr, workItemIDStr, title, description)
			const statusWidth = 40
			statusMarginLeft := m.width - statusWidth - lipgloss.Width(form) - s.Status.GetMarginRight()
			status = s.Status.
				Height(lipgloss.Height(form)).
				Width(statusWidth).
				MarginLeft(statusMarginLeft).
				Render(s.StatusHeader.Render("Current Commit") + "\n" +
					s.Highlight.Render(commitMsg) + "\n" +
					currentScope + "\n" +
					currentWorkItemID + "\n" +
					currentTitle + "\n" +
					currentDescription + "\n\n" +
					s.StatusHeader.Render("Completed Commit") + "\n" + status)
		}
		body := lipgloss.JoinHorizontal(lipgloss.Top, form, status)
		return s.Base.Render(body)
	}

	return m.form.View()
}

func isGitRepository(dir string) bool {
	gitDir := filepath.Join(dir, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		return false
	}
	return true
}

func main() {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current directory: ", err)
		os.Exit(1)
	}
	if isGitRepository(dir) {
		fmt.Println("This is a Git repository")
	} else {
		fmt.Println("Error! Current directory is NOT a Git Repository")
		os.Exit(1)
	}
	_, err = tea.NewProgram(NewModel()).Run()
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}
	// cmd := exec.Command("git", "commit", "-m", commitMessage)
	// if err := cmd.Run(); err != nil {
	//	fmt.Println("Error executing command:", err)
	// }
}
