package dryrun

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"mono-vcs/internal/app"
)

func PrintTable(w io.Writer, repos []string, cols int) {
	if len(repos) == 0 {
		return
	}
	cell := 0
	for _, p := range repos {
		if l := utf8.RuneCountInString(p); l > cell {
			cell = l
		}
	}
	bar := strings.Repeat("─", cell+2)
	bars := make([]string, cols)
	for i := range bars {
		bars[i] = bar
	}
	top := "┌" + strings.Join(bars, "┬") + "┐"
	mid := "├" + strings.Join(bars, "┼") + "┤"
	bot := "└" + strings.Join(bars, "┴") + "┘"

	var rows [][]string
	for i := 0; i < len(repos); i += cols {
		end := i + cols
		if end > len(repos) {
			end = len(repos)
		}
		chunk := append([]string{}, repos[i:end]...)
		for len(chunk) < cols {
			chunk = append(chunk, "")
		}
		rows = append(rows, chunk)
	}

	fmt.Fprintln(w, top)
	for i, row := range rows {
		cells := make([]string, len(row))
		for j, c := range row {
			cells[j] = ljust(c, cell)
		}
		fmt.Fprintf(w, "│ %s │\n", strings.Join(cells, " │ "))
		if i < len(rows)-1 {
			fmt.Fprintln(w, mid)
		}
	}
	fmt.Fprintln(w, bot)
}

func ljust(s string, n int) string {
	if w := utf8.RuneCountInString(s); w < n {
		return s + strings.Repeat(" ", n-w)
	}
	return s
}

func header(w io.Writer, label string, repos []string) {
	fmt.Fprintf(w, "DRY-RUN: %s — будет применено к %d репозитори(й/ев):\n\n", label, len(repos))
	PrintTable(w, repos, 6)
	fmt.Fprint(w, "\nПлан действий (для каждого репозитория, <repo-name> — путь к нему):\n\n")
}

func Pull(w io.Writer, a *app.Args, repos []string) {
	branch := a.GetMainBranch()
	header(w, "pull", repos)
	fmt.Fprintf(w, "  Условие: если репозиторий на ветке `%s` и локальный `%s` отличается от origin/%s\n", branch, branch, branch)
	fmt.Fprintln(w, "    git -C <repo-name> pull --ff-only --quiet")
	fmt.Fprintf(w, "  Иначе: пропустить (другая ветка, уже синхронизирован, либо нет в GitLab)\n")
}

func UpdateMain(w io.Writer, a *app.Args, repos []string) {
	branch := a.GetMainBranch()
	header(w, "update", repos)
	fmt.Fprintln(w, "  Условие: если в репозитории есть незакоммиченные изменения "+
		"(staged / unstaged / untracked) — пропустить с ошибкой.")
	fmt.Fprintln(w, "  Иначе:")
	fmt.Fprintf(w, "    Условие: если текущая ветка ≠ `%s`\n", branch)
	fmt.Fprintf(w, "      git -C <repo-name> checkout %s\n", branch)
	fmt.Fprintln(w, "    git -C <repo-name> pull --ff-only --quiet")
	fmt.Fprintln(w, "    Условие: если на входе была другая ветка <orig>")
	fmt.Fprintln(w, "      git -C <repo-name> checkout <orig>")
	fmt.Fprintf(w, "      git -C <repo-name> rebase %s   # при конфликте остановится для ручного разрешения\n", branch)
}

func Stash(w io.Writer, a *app.Args, repos []string) {
	header(w, "stash", repos)
	fmt.Fprintln(w, "  Условие: если есть unstaged-изменения (git diff --quiet вернул 1)")
	fmt.Fprintln(w, `    git -C <repo-name> stash push --keep-index --include-untracked -m "mono-vcs stash"`)
	fmt.Fprintln(w, "  Иначе: пропустить")
}

func Unstash(w io.Writer, a *app.Args, repos []string) {
	header(w, "unstash", repos)
	fmt.Fprintln(w, "  Условие: если стэш не пуст")
	fmt.Fprintln(w, "    git -C <repo-name> stash pop")
	fmt.Fprintln(w, "  Иначе: пропустить")
}

func ClearStash(w io.Writer, a *app.Args, repos []string) {
	header(w, "clear-stash", repos)
	fmt.Fprintln(w, "  Условие: если стэш не пуст")
	fmt.Fprintln(w, "    git -C <repo-name> stash clear")
	fmt.Fprintln(w, "  Иначе: пропустить")
}

func Prune(w io.Writer, a *app.Args, repos []string) {
	header(w, "prune", repos)
	fmt.Fprintln(w, "  Условие: если есть незакоммиченные изменения (git status --porcelain непуст)")
	fmt.Fprintln(w, "    1. показать список того, что будет отброшено (git status --porcelain)")
	if a.Yes {
		fmt.Fprintln(w, "    2. -y передан — отбросить без подтверждения")
	} else {
		fmt.Fprintln(w, "    2. спросить подтверждение ([y]es / [a] yes-all / [s]kip / [sa] skip-all)")
	}
	fmt.Fprintln(w, "    3. при согласии: git -C <repo-name> reset --hard HEAD && git -C <repo-name> clean -fd")
	fmt.Fprintln(w, "  Иначе: пропустить (нечего отбрасывать); игнорируемые файлы сохраняются")
}

func New(w io.Writer, a *app.Args, repos []string) {
	branch := a.Branch
	main := a.GetMainBranch()
	header(w, "new "+branch, repos)
	fmt.Fprintln(w, "  Условие: если есть незакоммиченные изменения — пропустить (попадёт в dirty-список).")
	fmt.Fprintf(w, "  Условие: если ветка `%s` уже существует локально — пропустить (exists).\n", branch)
	fmt.Fprintf(w, "  Условие: если нет локальной `%s` — пропустить (absent).\n", main)
	fmt.Fprintln(w, "  Иначе:")
	fmt.Fprintf(w, "    git -C <repo-name> checkout -b %s %s   # создаётся только локально, без push\n", branch, main)
}

func Switch(w io.Writer, a *app.Args, repos []string) {
	branch := a.Branch
	main := a.GetMainBranch()
	header(w, "switch "+branch, repos)
	fmt.Fprintln(w, "  Условие: если в репозитории есть незакоммиченные изменения — пропустить (попадёт в dirty-список).")
	fmt.Fprintln(w, "  Иначе:")
	fmt.Fprintf(w, "    Условие: если ветка `%s` существует (локально или origin/%s)\n", branch, branch)
	fmt.Fprintf(w, "      git -C <repo-name> checkout %s   # если ещё не на ней\n", branch)
	fmt.Fprintf(w, "    Иначе (фоллбэк — освежить `%s`, чтобы воркспейс был синхронен по фиче):\n", main)
	fmt.Fprintf(w, "      git -C <repo-name> checkout %s   # если текущая ветка ≠ `%s`\n", main, main)
	fmt.Fprintln(w, "      git -C <repo-name> pull --ff-only --quiet")
}

func Cancel(w io.Writer, a *app.Args, repos []string) {
	branch := a.Branch
	main := a.GetMainBranch()
	header(w, "cancel "+branch, repos)
	fmt.Fprintf(w, "  Условие: если локальной ветки `%s` нет — пропустить молча.\n", branch)
	fmt.Fprintln(w, "  Иначе:")
	fmt.Fprintf(w, "    Условие: если `%s` — текущая ветка\n", branch)
	fmt.Fprintln(w, "      Условие: если есть незакоммиченные изменения — пропустить (попадёт в dirty-список).")
	fmt.Fprintln(w, "      Иначе:")
	fmt.Fprintf(w, "        git -C <repo-name> checkout %s\n", main)
	fmt.Fprintln(w, "        git -C <repo-name> pull --ff-only --quiet")
	fmt.Fprintf(w, "    git -C <repo-name> branch -D %s\n", branch)
}

func Do(w io.Writer, a *app.Args, repos []string) {
	action := strings.Join(a.Action, " ")
	header(w, "do "+action, repos)
	fmt.Fprintf(w, "  cd <repo-name> && %s\n", action)
}
