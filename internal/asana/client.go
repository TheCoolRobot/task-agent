// Package asana wraps the asana-cli binary to fetch and manage tasks.
package asana

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Task represents a single Asana task.
type Task struct {
	ID        string   `json:"id"`
	GID       string   `json:"gid"`
	Name      string   `json:"name"`
	Completed bool     `json:"completed"`
	Priority  string   `json:"priority"`
	DueDate   string   `json:"due_date"`
	Notes     string   `json:"notes"`
	Tags      []Tag    `json:"tags"`
	Assignee  Assignee `json:"assignee"`
}

type Tag struct {
	Name string `json:"name"`
}

type Assignee struct {
	Name string `json:"name"`
}

func (t *Task) GetID() string {
	if t.GID != "" {
		return t.GID
	}
	return t.ID
}

func (t *Task) StatusIcon() string {
	if t.Completed {
		return "✅"
	}
	return "⏳"
}

func (t *Task) PriorityIcon() string {
	switch strings.ToLower(t.Priority) {
	case "high":
		return "🔴"
	case "medium":
		return "🟡"
	case "low":
		return "🟢"
	default:
		return "  "
	}
}

type apiResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   string          `json:"error"`
}

// Client wraps the asana-cli binary.
type Client struct {
	CLIPath string
}

// NewClient creates a Client, auto-detecting the asana-cli binary.
func NewClient(path string) (*Client, error) {
	if path == "" {
		// Try to find in PATH
		p, err := exec.LookPath("asana-cli")
		if err != nil {
			// Check common dev paths
			home, _ := os.UserHomeDir()
			candidates := []string{
				home + "/Developer/cmdln_dev/asana_cli-copilot/asana-cli",
				"./asana-cli",
			}
			for _, c := range candidates {
				if _, err := os.Stat(c); err == nil {
					path = c
					break
				}
			}
			if path == "" {
				return nil, fmt.Errorf("asana-cli not found in PATH or common locations")
			}
		} else {
			path = p
		}
	}
	return &Client{CLIPath: path}, nil
}

func (c *Client) run(args ...string) ([]byte, error) {
	args = append(args, "--json")
	cmd := exec.Command(c.CLIPath, args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("asana-cli error: %s", string(exitErr.Stderr))
		}
		return nil, err
	}
	return out, nil
}

// ListTasks lists tasks for a project.
func (c *Client) ListTasks(projectGID string) ([]Task, error) {
	args := []string{"list"}
	if projectGID != "" {
		args = append(args, projectGID)
	}
	data, err := c.run(args...)
	if err != nil {
		return nil, err
	}
	var resp apiResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf("asana-cli: %s", resp.Error)
	}
	var tasks []Task
	if err := json.Unmarshal(resp.Data, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// SearchTasks searches for tasks in a workspace.
func (c *Client) SearchTasks(workspaceGID, query string) ([]Task, error) {
	data, err := c.run("search", workspaceGID, query)
	if err != nil {
		return nil, err
	}
	var resp apiResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf("asana-cli: %s", resp.Error)
	}
	var tasks []Task
	if err := json.Unmarshal(resp.Data, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// ViewTask gets full details for a task.
func (c *Client) ViewTask(taskGID string) (*Task, error) {
	data, err := c.run("view", taskGID)
	if err != nil {
		return nil, err
	}
	var resp apiResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	var task Task
	if err := json.Unmarshal(resp.Data, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

// CompleteTask marks a task as done.
func (c *Client) CompleteTask(taskGID string) error {
	data, err := c.run("complete", taskGID)
	if err != nil {
		return err
	}
	var resp apiResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("asana-cli: %s", resp.Error)
	}
	return nil
}

// FormatTaskMarkdown formats a task for the AI system prompt.
func FormatTaskMarkdown(t *Task) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Task: %s\n\n", t.Name)
	fmt.Fprintf(&b, "**ID:** %s\n", t.GetID())
	fmt.Fprintf(&b, "**Status:** %s\n", func() string {
		if t.Completed {
			return "Complete"
		}
		return "Incomplete"
	}())
	if t.Priority != "" {
		fmt.Fprintf(&b, "**Priority:** %s\n", t.Priority)
	}
	if t.DueDate != "" {
		fmt.Fprintf(&b, "**Due:** %s\n", t.DueDate)
	}
	if t.Assignee.Name != "" {
		fmt.Fprintf(&b, "**Assignee:** %s\n", t.Assignee.Name)
	}
	if len(t.Tags) > 0 {
		tagNames := make([]string, len(t.Tags))
		for i, tag := range t.Tags {
			tagNames[i] = tag.Name
		}
		fmt.Fprintf(&b, "**Tags:** %s\n", strings.Join(tagNames, ", "))
	}
	if t.Notes != "" {
		fmt.Fprintf(&b, "\n## Description\n\n%s\n", t.Notes)
	}
	return b.String()
}
