package dryrun

import (
	"fmt"
	"io"
	"strings"

	"github.com/dimkarp93/mono-vcs/internal/app"
	"github.com/dimkarp93/mono-vcs/internal/output"
)

func PrintTable(w io.Writer, repos []string, maxCols int) {
	if len(repos) == 0 {
		return
	}
	if maxCols < 1 {
		maxCols = 1
	}
	width := output.Width(w)
	cell := 0
	for _, p := range repos {
		if l := output.DisplayWidth(p); l > cell {
			cell = l
		}
	}
	if limit := width - 4; cell > limit {
		cell = limit
	}
	if cell < 1 {
		cell = 1
	}
	cols := (width - 1) / (cell + 3)
	if cols > maxCols {
		cols = maxCols
	}
	if cols < 1 {
		cols = 1
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
			cells[j] = output.Cell(c, cell)
		}
		fmt.Fprintf(w, "│ %s │\n", strings.Join(cells, " │ "))
		if i < len(rows)-1 {
			fmt.Fprintln(w, mid)
		}
	}
	fmt.Fprintln(w, bot)
}

func header(w io.Writer, label string, repos []string) {
	fmt.Fprintf(w, "DRY-RUN: %s — будет применено к %d репозитори(й/ев):\n\n", label, len(repos))
	PrintTable(w, repos, 6)
	fmt.Fprint(w, "\nПлан действий (для каждого репозитория, <repo-name> — путь к нему):\n\n")
}

func Pull(w io.Writer, a *app.Args, repos []string) {
	header(w, "pull", repos)
	fmt.Fprintln(w, "  <default-branch> — дефолтная ветка репозитория (из state db, наполняется из GitLab)")
	fmt.Fprintln(w, "  Рабочее дерево и текущая ветка не меняются.")
	fmt.Fprintln(w, "  Условие: если текущая ветка = <default-branch>")
	fmt.Fprintln(w, "    git -C <repo-name> pull --ff-only --quiet")
	fmt.Fprintln(w, "  Иначе:")
	fmt.Fprintln(w, "    git -C <repo-name> fetch origin \\")
	fmt.Fprintln(w, "      refs/heads/<default-branch>:refs/heads/<default-branch> \\")
	fmt.Fprintln(w, "      +refs/heads/<default-branch>:refs/remotes/origin/<default-branch>")
	fmt.Fprintln(w, "  При расхождении локальной <default-branch> с origin — пропустить с ошибкой (diverged).")
}

func Update(w io.Writer, a *app.Args, repos []string) {
	header(w, "update", repos)
	fmt.Fprintln(w, "  <default-branch> — дефолтная ветка репозитория (из state db, наполняется из GitLab)")
	fmt.Fprintln(w, "  Условие: если в репозитории есть незакоммиченные изменения "+
		"(staged / unstaged / untracked) — пропустить с ошибкой.")
	fmt.Fprintln(w, "  Иначе:")
	fmt.Fprintln(w, "    Условие: если текущая ветка ≠ <default-branch>")
	fmt.Fprintln(w, "      git -C <repo-name> checkout <default-branch>")
	fmt.Fprintln(w, "    git -C <repo-name> pull --ff-only --quiet")
	fmt.Fprintln(w, "    Условие: если на входе была другая ветка <orig>")
	fmt.Fprintln(w, "      git -C <repo-name> checkout <orig>")
	fmt.Fprintln(w, "      git -C <repo-name> rebase <default-branch>   # при конфликте остановится для ручного разрешения")
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
	header(w, "new "+branch, repos)
	fmt.Fprintln(w, "  <default-branch> — дефолтная ветка репозитория (из state db, наполняется из GitLab)")
	fmt.Fprintf(w, "  Условие: если `%s` — дефолтная ветка репозитория — пропустить (default).\n", branch)
	fmt.Fprintf(w, "  Условие: если репозиторий уже на `%s` — пропустить (already), изменения и так на месте.\n", branch)
	fmt.Fprintf(w, "  Условие: если ветки `%s` нет и нет локальной <default-branch> — пропустить (absent).\n", branch)
	fmt.Fprintln(w, "  Иначе:")
	fmt.Fprintln(w, "    Условие: если есть незакоммиченные или неотслеживаемые файлы")
	fmt.Fprintf(w, "      git -C <repo-name> stash push --include-untracked -m \"mono-vcs new %s\"\n", branch)
	fmt.Fprintf(w, "    Условие: если ветка `%s` уже существует локально\n", branch)
	fmt.Fprintf(w, "      git -C <repo-name> checkout %s\n", branch)
	fmt.Fprintln(w, "    Иначе:")
	fmt.Fprintf(w, "      git -C <repo-name> checkout -b %s <default-branch>   # создаётся только локально, без push\n", branch)
	fmt.Fprintln(w, "    Условие: если изменения стешились")
	fmt.Fprintln(w, "      git -C <repo-name> stash pop   # при конфликте стеш остаётся, репо попадёт в conflicts")
}

func Switch(w io.Writer, a *app.Args, repos []string) {
	branch := a.Branch
	label := "<default-branch>"
	if branch != "" {
		label = branch
	}
	header(w, strings.TrimSpace("switch "+branch), repos)
	fmt.Fprintln(w, "  <default-branch> — дефолтная ветка репозитория (из state db, наполняется из GitLab)")
	fmt.Fprintln(w, "  Условие: если в репозитории есть незакоммиченные изменения — пропустить (попадёт в dirty-список).")
	fmt.Fprintln(w, "  Иначе:")
	fmt.Fprintf(w, "    Условие: если ветка %s существует (локально или origin/%s)\n", label, label)
	fmt.Fprintf(w, "      git -C <repo-name> checkout %s   # если ещё не на ней\n", label)
	fmt.Fprintln(w, "    Иначе (фоллбэк — освежить <default-branch>, чтобы воркспейс был синхронен по фиче):")
	fmt.Fprintln(w, "      git -C <repo-name> checkout <default-branch>   # если текущая ветка ≠ <default-branch>")
	fmt.Fprintln(w, "      git -C <repo-name> pull --ff-only --quiet")
}

func Finish(w io.Writer, a *app.Args, repos []string) {
	branch := a.Branch
	header(w, "finish "+branch, repos)
	fmt.Fprintln(w, "  <default-branch> — дефолтная ветка репозитория (из state db, наполняется из GitLab)")
	fmt.Fprintf(w, "  Условие: если `%s` — дефолтная ветка репозитория — пропустить (default).\n", branch)
	fmt.Fprintf(w, "  Условие: если локальной ветки `%s` нет — пропустить молча.\n", branch)
	fmt.Fprintln(w, "  Иначе (ветка удаляется независимо от незакоммиченных и неотслеживаемых файлов):")
	fmt.Fprintln(w, "    Условие: если есть незакоммиченные или неотслеживаемые файлы")
	fmt.Fprintln(w, "      git -C <repo-name> reset --hard HEAD")
	fmt.Fprintln(w, "      git -C <repo-name> clean -fd        # игнорируемые файлы сохраняются")
	fmt.Fprintln(w, "    Условие: если текущая ветка ≠ <default-branch>")
	fmt.Fprintln(w, "      git -C <repo-name> checkout <default-branch>")
	fmt.Fprintln(w, "    git -C <repo-name> pull --ff-only --quiet")
	fmt.Fprintf(w, "    git -C <repo-name> branch -D %s\n", branch)
}

func Done(w io.Writer, a *app.Args, repos []string) {
	header(w, "done", repos)
	fmt.Fprintln(w, "  <default-branch> — дефолтная ветка репозитория (из state db, наполняется из GitLab)")
	fmt.Fprintln(w, "  Команда применяется ко всем репозиториям: -repo и -feat она не принимает.")
	fmt.Fprintln(w, "  1. git -C <repo-name> fetch --prune --quiet origin")
	fmt.Fprintln(w, "     При ошибке fetch репозиторий исключается целиком (ветки по устаревшим рефам не удаляются).")
	fmt.Fprintln(w, "  2. Точка сравнения <target> — origin/<default-branch>, а если такого рефа нет — <default-branch>.")
	fmt.Fprintln(w, "  3. Кандидаты — все локальные ветки, кроме <default-branch>.")
	fmt.Fprintln(w, "  4. Ветка <B> считается завершённой, если выполнены оба условия:")
	fmt.Fprintln(w, "     а. если <B> — текущая ветка, рабочее дерево чистое (нет незакоммиченных и неотслеживаемых файлов);")
	fmt.Fprintln(w, "     б. git -C <repo-name> merge-base --is-ancestor <B> <target> вернул 0 —")
	fmt.Fprintln(w, "        все коммиты <B> уже лежат в истории дефолтной ветки.")
	if a.Yes {
		fmt.Fprintln(w, "  5. -y передан — удалить без подтверждения")
	} else {
		fmt.Fprintln(w, "  5. по каждому репозиторию спросить подтверждение ([y]es / [a] yes-all / [s]kip / [sa] skip-all)")
	}
	fmt.Fprintln(w, "  6. При согласии для каждой такой ветки:")
	fmt.Fprintln(w, "     Условие: если <B> — текущая ветка")
	fmt.Fprintln(w, "       git -C <repo-name> checkout <default-branch>")
	fmt.Fprintln(w, "       git -C <repo-name> pull --ff-only --quiet")
	fmt.Fprintln(w, "     git -C <repo-name> branch -D <B>")
}

func Do(w io.Writer, a *app.Args, repos []string) {
	action := a.ShellAction()
	header(w, "do "+action, repos)
	fmt.Fprintf(w, "  cd <repo-name> && %s\n", action)
}

func MR(w io.Writer, a *app.Args, branch string, repos []string) {
	header(w, "mr "+branch, repos)
	fmt.Fprintf(w, "  Фича — `%s`; ниже перечислены все репозитории, в которых она есть.\n", branch)
	fmt.Fprintln(w, "  Условие: если хотя бы в одном из них есть незакоммиченные изменения — не пушить ничего.")
	fmt.Fprintf(w, "  Условие: если локальная `%s` совпадает с origin/%s — пропустить (up-to-date, push не вызывается).\n", branch, branch)
	fmt.Fprintln(w, "  Иначе:")
	fmt.Fprintf(w, "    git -C <repo-name> push -u origin refs/heads/%s:refs/heads/%s \\\n", branch, branch)
	fmt.Fprintln(w, "      -o merge_request.create \\")
	if a.Title != "" {
		fmt.Fprintln(w, "      -o merge_request.remove_source_branch \\")
		fmt.Fprintf(w, "      -o merge_request.title=%s\n", a.Title)
	} else {
		fmt.Fprintln(w, "      -o merge_request.remove_source_branch")
	}
}
