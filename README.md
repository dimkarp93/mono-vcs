# mono-vcs (Go)

Go-порт инструмента `mono-vcs` — CLI для массовой синхронизации локальных
git-клонов с инстансом GitLab. Это самодостаточный **один статический бинарь**:
клиентам не нужен Python-рантайм, нужен только `git` в `PATH`.

Команды и поведение совпадают с Python-версией (см. `../vcs/README.md`, `../vcs/usage.md`,
`../vcs/arch.md`). Единственное намеренное отличие — формат конфига: **JSON** вместо
INI (см. ниже).

## Сборка

Нужен тулчейн Go (проект таргетит Go 1.26.1). Если Go установлен в локальный
префикс:

```bash
export PATH="$HOME/.local/go/bin:$PATH"
```

Затем:

```bash
make build                 # -> ./mono-vcs (статический бинарь)
make run ARGS='--help'     # собрать и запустить
make test                  # весь набор тестов
make check                 # vet + test
```

`make` без аргументов печатает список целей и примеры запуска.

### Кросс-компиляция для клиентов

```bash
make dist     # -> dist/mono-vcs-{linux,darwin}-{amd64,arm64}.tar.gz + dist/SHA256SUMS
```

Сборка идёт с `CGO_ENABLED=0 -trimpath -ldflags="-s -w -X main.version=<ver>"` →
полностью статический бинарь без зависимости от libc, с вшитой версией из
`versions.txt`. Формат архивов и `SHA256SUMS` совместим с установщиком
`github_install.sh` (см. ниже).

## Установка у клиента

Дистрибуция — **только бинарь**. Никаких дополнительных файлов не требуется:
конфиг создаётся в рантайме командой `mono-vcs init`, а `git` у клиента уже есть.

Рекомендуемый способ — универсальный установщик (репозиторий — `<owner>/mvpy-vcs`,
бинарь — `mono-vcs`):

```bash
github_install.sh <owner>/mvpy-vcs            # имя бинаря mono-vcs определится из ассета
github_install.sh <owner>/mvpy-vcs mono-vcs   # или явно
github_install.sh -u <owner>/mvpy-vcs         # обновить, только если есть новее
```

Установщик скачивает `mono-vcs-<os>-<arch>.tar.gz`, проверяет `SHA256SUMS` и
кладёт бинарь в выбранный каталог. Локально из готового архива:
`github_install.sh -F dist/mono-vcs-linux-amd64.tar.gz`.

Сборка из исходников: `make install` (ставит в `/usr/local/bin/mono-vcs`).

## Релиз

Версия живёт в `versions.txt` (голый semver). Чтобы выпустить релиз: подбампить
версию и смержить в `main`/`master` — CI (`.github/workflows/release.yml`) сам
соберёт архивы для 4 платформ, сгенерирует `SHA256SUMS` и создаст GitHub Release
с тегом `vX.Y.Z` (идемпотентно: если тег уже есть — пропускает).

```bash
make bump-patch   # 0.1.0 -> 0.1.1   (есть и bump-minor / bump-major)
git commit -am "release X.Y.Z" && git push   # мерж в main запускает релиз
```

## Конфиг (`~/.config/vcs-go`)

Управляется через `mono-vcs init`. Путь учитывает `$XDG_CONFIG_HOME`. Токен
**не** хранится. Формат — JSON:

```json
{
  "gl-url": "https://gitlab.company.com",
  "jobs": 4,
  "no-color": false,
  "main-branch": "main"
}
```

Go-версия читает **отдельный** файл `~/.config/vcs-go`, а не `~/.config/vcs`
(INI Python-версии), поэтому обе реализации спокойно сосуществуют на одной
машине во время перехода.

### Миграция со старого (Python/INI) конфига

Достаточно один раз выполнить `mono-vcs init` — он создаст `~/.config/vcs-go` в
JSON. Старый `~/.config/vcs` при этом не трогается.

## Примеры запуска

```bash
mono-vcs init                                   # настроить ~/.config/vcs-go
mono-vcs list --all                             # все репозитории с цветовой разметкой
mono-vcs clone --gl-url https://gitlab.company.com
mono-vcs pull --dry-run                         # показать план без действий
mono-vcs update-main                            # подтянуть main, ребейзнуть фичи
mono-vcs stash ; mono-vcs unstash
mono-vcs features                               # таблица фиче-веток
mono-vcs new MVPAY-290 -repo apigateway,payments  # создать ветку от main (локально)
mono-vcs switch my-feature                      # переключить все репо
mono-vcs do git status -s                       # выполнить команду в каждом репо
mono-vcs do -feat MVPAY-290 git status -s       # только репо, где есть ветка MVPAY-290
```

Те же сценарии через Makefile: `make run ARGS='list --all'` и т.д.

### Выбор репозиториев: `-repo` и `-feat`

Любая команда, принимающая `-repo`, также принимает `-feat | -f <feat-name>`:

- `-repo` — ограничить по имени/папке/пути (повторяемый, через запятую).
- `-feat | -f` — ограничить репозиториями, где **локально существует ветка** с
  таким именем (та же трактовка «фичи», что и у `mono-vcs features`), независимо
  от того, переключён ли на неё репозиторий.

`-repo` и `-feat` **взаимоисключающи** — передавать оба сразу нельзя (ошибка).
Если не указан ни один, команда работает по всем репозиториям, как и раньше.

## Тесты

```bash
make test         # go test ./...
make test-race    # с детектором гонок
make test-v       # подробно
```

Тесты не ходят в сеть: GitLab REST мокается через `httptest`, git-операции
выполняются над настоящими временными репозиториями. Набор портирован 1:1 с
Python-тестов (`../vcs/tests/`); каждый файл защищает те же инварианты.

## Структура

```
internal/
├── app/        общие типы (Args, Context)
├── colors/     ANSI-цвета
├── config/     чтение/запись конфига (JSON), применение дефолтов
├── output/     печать ошибок, таблица features
├── prompts/    интерактивные промпты (токен, choice, force/prune)
├── repos/      поиск локальных репо, --repo фильтр
├── gitlab/     REST-клиент GitLab
├── gitops/     обёртки над git + per-repo воркеры
├── jobs/       параллельный per-repo раннер
├── dryrun/     печать dry-run планов
├── cli/        парсинг флагов + диспетчер + политика токена
└── commands/   оркестрация каждой подкоманды
```

Карта пакетов повторяет Python-пакет `gl_sync/`; слоистость зависимостей
описана в `../vcs/arch.md`.
