package app

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"todo/db"
	pkglog "todo/pkg/log"
	"go.uber.org/zap"
)

func RunCLI(args []string) {
	logger := pkglog.GetLogger(nil)
	if len(args) == 0 {
		listCLI(false)
		return
	}
	cmd := args[0]
	switch cmd {
	case "list":
		inc := false
		if len(args) > 1 && args[1] == "all" {
			inc = true
		}
		listCLI(inc)
	case "add":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: todo add <content>")
			logger.Error("cli add: missing content")
			os.Exit(1)
		}
		content := strings.Join(args[1:], " ")
		threadLink := extractURL(content)
		t, err := Create(content, threadLink, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			logger.Error("cli add failed", zap.Error(err))
			os.Exit(1)
		}
		printTodoSummary(*t)
	case "detail":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: todo detail <id>")
			logger.Error("cli detail: missing id")
			os.Exit(1)
		}
		id, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Invalid id")
			logger.Error("cli detail: invalid id", zap.String("arg", args[1]), zap.Error(err))
			os.Exit(1)
		}
		tf, err := RefreshFull(id)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			logger.Error("cli detail failed", zap.Int64("id", id), zap.Error(err))
			os.Exit(1)
		}
		printCLIDetail(*tf)
	case "start", "pause", "complete", "reopen", "schedule", "later":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: todo %s <id>\n", cmd)
			logger.Error("cli: missing id", zap.String("command", cmd))
			os.Exit(1)
		}
		id, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Invalid id")
			logger.Error("cli: invalid id", zap.String("arg", args[1]), zap.Error(err))
			os.Exit(1)
		}
		var t *TodoFull
		switch cmd {
		case "start":
			t, err = Start(id)
		case "pause":
			t, err = Pause(id)
		case "complete":
			note := ""
			if len(args) > 2 {
				note = strings.Join(args[2:], " ")
			}
			t, err = Complete(id, note)
		case "reopen":
			t, err = Reopen(id)
		case "schedule":
			if len(args) < 3 {
				fmt.Fprintln(os.Stderr, "Usage: todo schedule <id> <time>")
				logger.Error("cli schedule: missing time", zap.Int64("id", id))
				os.Exit(1)
			}
			sched, parseErr := parseTimeString(args[2])
			if parseErr != nil {
				fmt.Fprintf(os.Stderr, "Invalid time: %v\n", parseErr)
				logger.Error("cli schedule: invalid time", zap.String("time", args[2]), zap.Error(parseErr))
				os.Exit(1)
			}
			t, err = SetSchedule(id, sched.Unix())
		case "later":
			if len(args) < 3 {
				fmt.Fprintln(os.Stderr, "Usage: todo later <id> <time>")
				logger.Error("cli later: missing time", zap.Int64("id", id))
				os.Exit(1)
			}
			sched, parseErr := parseTimeString(args[2])
			if parseErr != nil {
				fmt.Fprintf(os.Stderr, "Invalid time: %v\n", parseErr)
				logger.Error("cli later: invalid time", zap.String("time", args[2]), zap.Error(parseErr))
				os.Exit(1)
			}
			t, err = Later(id, sched.Unix())
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			logger.Error("cli command failed", zap.String("command", cmd), zap.Int64("id", id), zap.Error(err))
			os.Exit(1)
		}
		printTodoSummary(*t)
	case "note":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: todo note <id> <text>")
			logger.Error("cli note: missing arguments")
			os.Exit(1)
		}
		id, _ := strconv.ParseInt(args[1], 10, 64)
		note := strings.Join(args[2:], " ")
		t, err := AddNote(id, note)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			logger.Error("cli note failed", zap.Int64("id", id), zap.Error(err))
			os.Exit(1)
		}
		printTodoSummary(*t)
	case "delete":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: todo delete <id>")
			logger.Error("cli delete: missing id")
			os.Exit(1)
		}
		id, _ := strconv.ParseInt(args[1], 10, 64)
		if err := Delete(id); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			logger.Error("cli delete failed", zap.Int64("id", id), zap.Error(err))
			os.Exit(1)
		}
		fmt.Printf("Todo #%d deleted\n", id)
	default:
		// treat as content to add
		content := strings.Join(args, " ")
		threadLink := extractURL(content)
		t, err := Create(content, threadLink, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			logger.Error("cli add (implicit) failed", zap.Error(err))
			os.Exit(1)
		}
		printTodoSummary(*t)
	}
}

func listCLI(includeCompleted bool) {
	logger := pkglog.GetLogger(nil)
	todos, err := db.ListTodos(includeCompleted)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		logger.Error("cli list failed", zap.Error(err))
		os.Exit(1)
	}
	for i, t := range todos {
		printCLITodo(t, i)
	}
}

func printCLITodo(t db.Todo, index int) {
	if index > 0 {
		fmt.Println("\n========\n")
	}
	fmt.Printf("#%d [%s]\n", t.ID, t.Status)
	fmt.Printf("%s\n", time.Unix(t.CreatedAt, 0).Format("2006-01-02 15:04"))
	if t.Content != "" {
		fmt.Println(t.Content)
	}
	logs, _ := db.GetLogs(t.ID)
	reversed := make([]db.TodoLog, len(logs))
	for i, l := range logs {
		reversed[len(logs)-1-i] = l
	}
	for _, l := range reversed {
		if l.Note != "" {
			timeStr := time.Unix(l.CreatedAt, 0).Format("2006-01-02 15:04")
			fmt.Printf("[%s] %s: %s\n", timeStr, l.Action, l.Note)
		}
	}
}

func printTodoSummary(tf TodoFull) {
	fmt.Printf("#%d [%s]\n", tf.ID, tf.Status)
	fmt.Printf("%s\n", time.Unix(tf.CreatedAt, 0).Format("2006-01-02 15:04"))
	fmt.Println(tf.Content)
}

func printCLIDetail(tf TodoFull) {
	fmt.Printf("#%d [%s]\n", tf.ID, tf.Status)
	fmt.Printf("Created: %s\n", time.Unix(tf.CreatedAt, 0).Format("2006-01-02 15:04"))
	if tf.CompletedAt > 0 {
		fmt.Printf("Completed: %s\n", time.Unix(tf.CompletedAt, 0).Format("2006-01-02 15:04"))
	}
	if tf.ScheduledAt > 0 {
		fmt.Printf("Scheduled: %s\n", time.Unix(tf.ScheduledAt, 0).Format("2006-01-02 15:04"))
	}
	if tf.ThreadLink != "" {
		fmt.Printf("Main Link: %s\n", tf.ThreadLink)
	}
	fmt.Println("---")
	fmt.Println(tf.Content)
	fmt.Println("---")
	if len(tf.Tags) > 0 {
		fmt.Println("Tags:")
		for _, tg := range tf.Tags {
			fmt.Printf("  [%s: %s]\n", tg.GroupName, tg.Tag)
		}
		fmt.Println("---")
	}
	if len(tf.Logs) > 0 {
		fmt.Println("History:")
		for i := len(tf.Logs) - 1; i >= 0; i-- {
			l := tf.Logs[i]
			timeStr := time.Unix(l.CreatedAt, 0).Format("2006-01-02 15:04")
			if l.Note != "" {
				fmt.Printf("  [%s] %s: %s\n", timeStr, l.Action, l.Note)
			} else {
				fmt.Printf("  [%s] %s\n", timeStr, l.Action)
			}
		}
	}
}

func parseTimeString(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "+") {
		dur, err := time.ParseDuration(s[1:])
		if err != nil {
			return time.Time{}, err
		}
		return time.Now().Add(dur), nil
	}
	t, err := time.ParseInLocation("2006-01-02 15:04", s, time.Local)
	return t, err
}

func extractURL(s string) string {
	re := regexp.MustCompile(`https?://\S+`)
	match := re.FindString(s)
	return match
}
