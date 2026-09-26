package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"ant/cmd/token-agent/internal/agent"
)

// Окно подтверждения и ввод PIN — в локальной программе (AD-14: цель — окно
// агента, а не страница): на macOS — диалог osascript, в Linux — zenity.
// Страница и расширение PIN не видят.

// Prompter — окно агента.
type Prompter interface {
	// Confirm — показать сводку; needPIN — спросить PIN. ok=false — отказ.
	Confirm(title string, fields []agent.Field, needPIN bool) (pin string, ok bool, err error)
}

// ErrNoWindow — показать окно подтверждения нечем.
var ErrNoWindow = errors.New("нет окна подтверждения (нужен osascript на macOS или zenity в Linux)")

func systemPrompter() Prompter {
	switch runtime.GOOS {
	case "darwin":
		return osaPrompter{}
	case "linux":
		if _, err := exec.LookPath("zenity"); err == nil {
			return zenityPrompter{}
		}
	}
	return noPrompter{}
}

func text(title string, fields []agent.Field) string {
	var b strings.Builder
	b.WriteString(title)
	b.WriteString("\n\n")
	for _, f := range fields {
		fmt.Fprintf(&b, "%s: %s\n", f.Label, f.Value)
	}
	return b.String()
}

// osaQuote — строка для AppleScript.
func osaQuote(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

type osaPrompter struct{}

func (osaPrompter) Confirm(title string, fields []agent.Field, needPIN bool) (string, bool, error) {
	msg := text(title, fields)
	script := "display dialog " + osaQuote(msg) + ` with title "Главный — агент токена" buttons {"Отказать", "Подписать"} default button "Подписать" cancel button "Отказать" with icon caution`
	if needPIN {
		msg += "\nВведите PIN токена:"
		script = "display dialog " + osaQuote(msg) + ` default answer "" with hidden answer with title "Главный — агент токена" buttons {"Отказать", "Подписать"} default button "Подписать" cancel button "Отказать" with icon caution`
	}
	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return "", false, nil // «Отказать» — osascript выходит с ошибкой -128
		}
		return "", false, err
	}
	s := strings.TrimSpace(string(out))
	if !strings.Contains(s, "button returned:Подписать") {
		return "", false, nil
	}
	_, pin, _ := strings.Cut(s, "text returned:")
	return pin, true, nil
}

type zenityPrompter struct{}

func (zenityPrompter) Confirm(title string, fields []agent.Field, needPIN bool) (string, bool, error) {
	if err := exec.Command("zenity", "--question", "--title=Главный — агент токена", "--ok-label=Подписать", "--cancel-label=Отказать",
		"--text="+text(title, fields)).Run(); err != nil {
		return "", false, nil
	}
	if !needPIN {
		return "", true, nil
	}
	out, err := exec.Command("zenity", "--password", "--title=PIN токена").Output()
	if err != nil {
		return "", false, nil
	}
	return strings.TrimSpace(string(out)), true, nil
}

type noPrompter struct{}

func (noPrompter) Confirm(string, []agent.Field, bool) (string, bool, error) {
	return "", false, ErrNoWindow
}

// askPINTerminal — PIN с терминала (только для import: программа запущена
// человеком у себя, страницы нет).
func askPINTerminal(prompt string) (string, error) {
	if v := os.Getenv("ANT_TOKEN_PIN"); v != "" {
		return v, nil
	}
	fmt.Fprint(os.Stderr, prompt)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return strings.TrimSpace(line), nil
}
