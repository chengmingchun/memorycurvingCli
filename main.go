package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const gap = "\n\n"

type TodoItem struct {
	content   string
	status    rune //   · * o x √
	deadline  time.Time
	completed bool
}

func (t TodoItem) String() string {
	deadlineStr := t.deadline.Format("15:04")
	dateStr := t.deadline.Format("01-02")
	statusDesc := ""
	switch t.status {
	case ' ':
		statusDesc = "   "
	case '·':
		statusDesc = "第1次"
	case '*':
		statusDesc = "第2次"
	case 'o':
		statusDesc = "第3次"
	case 'x':
		statusDesc = "第4次"
	case '√':
		statusDesc = "finish"
	}
	return fmt.Sprintf("[%c] %-30s %-8s %s %s", t.status, t.content, statusDesc, dateStr, deadlineStr)
}

func nextStatus(current rune) (rune, time.Duration) {
	switch current {
	case ' ':
		return '·', 0
	case '·':
		return '*', 2 * time.Hour
	case '*':
		return 'o', 12 * time.Hour
	case 'o':
		return 'x', 24 * time.Hour
	case 'x':
		return '√', 7 * 24 * time.Hour
	case '√':
		return '√', 0
	default:
		return ' ', 0
	}
}

type (
	errMsg error
)

type model struct {
	viewport    viewport.Model
	todos       []TodoItem
	textarea    textarea.Model
	senderStyle lipgloss.Style
	err         error
	focused     bool
	selected    int
}

func initialModel() *model {
	ta := textarea.New()
	ta.Placeholder = "todo..."
	ta.Focus()

	ta.Prompt = "➜ "
	ta.CharLimit = 280

	ta.SetWidth(50)
	ta.SetHeight(1)

	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()

	ta.ShowLineNumbers = false

	vp := viewport.New(50, 10)
	vp.SetContent("")

	// Load todos
	todos := loadTodos()

	m := &model{
		textarea:    ta,
		todos:       todos,
		viewport:    vp,
		senderStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
		err:         nil,
		focused:     false,
		selected:    0,
	}

	m.updateViewport()

	return m
}

func (m *model) updateViewport() {
	var lines []string
	for i, todo := range m.todos {
		line := todo.String()
		if m.focused && i == m.selected {
			line = "> " + line
		} else {
			line = "  " + line
		}
		lines = append(lines, line)
	}
	m.viewport.SetContent(strings.Join(lines, "\n"))
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(tea.ClearScreen, textarea.Blink, tea.SetWindowTitle("自律小助手"))
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.textarea.SetWidth(msg.Width)
		m.viewport.Height = msg.Height - m.textarea.Height() - lipgloss.Height(gap)
		m.viewport.GotoBottom()
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			cleanupOldTodos()
			return m, tea.Quit
		case tea.KeyTab:
			m.focused = !m.focused
			if m.focused {
				m.textarea.Blur()
				// Reset selection if list is empty
				if len(m.todos) == 0 {
					m.selected = 0
				} else if m.selected >= len(m.todos) {
					m.selected = len(m.todos) - 1
				}
			} else {
				m.textarea.Focus()
			}
			m.updateViewport()
		case tea.KeyEnter:
			if !m.focused {
				content := strings.TrimSpace(m.textarea.Value())
				if content != "" {
					newTodo := TodoItem{
						content:  content,
						status:   ' ',
						deadline: time.Now(),
					}
					if err := saveTodo(newTodo); err != nil {
						log.Printf("Error saving todo: %v", err)
					} else {
						m.todos = append(m.todos, newTodo)
						m.textarea.Reset()
						m.updateViewport()
					}
				}
			}
		case tea.KeyCtrlY:
			if m.focused && len(m.todos) > 0 && m.selected >= 0 && m.selected < len(m.todos) {
				nextStat, duration := nextStatus(m.todos[m.selected].status)
				m.todos[m.selected].status = nextStat
				if duration > 0 {
					m.todos[m.selected].deadline = time.Now().Add(duration)
				}

				if err := updateTodo(m.todos[m.selected]); err != nil {
					log.Printf("Error updating todo: %v", err)
				}

				if nextStat == '√' {
					// Remove completed items from the list
					m.todos = append(m.todos[:m.selected], m.todos[m.selected+1:]...)
					if len(m.todos) > 0 {
						if m.selected >= len(m.todos) {
							m.selected = len(m.todos) - 1
						}
					} else {
						m.selected = 0
					}
				}
				m.updateViewport()
			}
		case tea.KeyUp:
			if m.focused && m.selected > 0 {
				m.selected--
				m.updateViewport()
			}
		case tea.KeyDown:
			if m.focused && len(m.todos) > 0 && m.selected < len(m.todos)-1 {
				m.selected++
				m.updateViewport()
			}
		}

	case errMsg:
		m.err = msg
		return m, nil
	}

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m *model) View() string {
	return fmt.Sprintf(
		"%s\n\n%s",
		m.viewport.View(),
		m.textarea.View(),
	) + "\n\n时间表: [·]首次完成 [*]2小时 [o]12小时 [x]24小时 [√]7天" +
		"\n(Tab切换界面 | Enter添加 | Ctrl+Y更新状态 | ↑↓选择 | Ctrl+C退出)"

}

func main() {
	initDB()
	defer closeDB()

	p := tea.NewProgram(initialModel())

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
