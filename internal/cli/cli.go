// Package cli parses argv for the vscan command.
package cli

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// Options is a parsed invocation. Query may still be empty when it comes from stdin.
type Options struct {
	Help      bool
	JSON      bool
	Scanner   bool
	NoHistory bool
	History   bool
	HistoryN  int
	Every     time.Duration
	EverySet  bool
	MaxPages  int
	Listen    string
	Webhook   string
	Sources   []string
	Query     string
}

// Parse reads arguments after the program name.
func Parse(args []string) (Options, error) {
	opt := Options{MaxPages: 0, Every: 60 * time.Minute}
	var positionals []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-h" || a == "--help":
			opt.Help = true
			return opt, nil
		case a == "--json":
			opt.JSON = true
		case a == "--scanner":
			opt.Scanner = true
		case a == "--no-history":
			opt.NoHistory = true
		case a == "--history" || strings.HasPrefix(a, "--history="):
			opt.History = true
			raw := ""
			if v, ok := strings.CutPrefix(a, "--history="); ok {
				raw = v
			} else if i+1 < len(args) && isInt(args[i+1]) {
				i++
				raw = args[i]
			}
			if raw != "" {
				n, err := strconv.Atoi(raw)
				if err != nil || n < 1 {
					return Options{}, fmt.Errorf("--history: номер записи начинается с 1")
				}
				opt.HistoryN = n
			}
		case a == "--listen" || strings.HasPrefix(a, "--listen="):
			opt.Listen = "127.0.0.1:8787"
			if v, ok := strings.CutPrefix(a, "--listen="); ok {
				if v != "" {
					opt.Listen = v
				}
			} else if i+1 < len(args) && looksLikeAddr(args[i+1]) {
				i++
				opt.Listen = args[i]
			}
		case a == "--webhook" || strings.HasPrefix(a, "--webhook="):
			v, err := takeValue(args, &i, a, "--webhook")
			if err != nil {
				return Options{}, err
			}
			opt.Webhook = v
		case a == "--every" || strings.HasPrefix(a, "--every="):
			v, err := takeValue(args, &i, a, "--every")
			if err != nil {
				return Options{}, err
			}
			n, err := strconv.Atoi(v)
			if err != nil || n < 1 {
				return Options{}, fmt.Errorf("--every: нужно целое число минут, не меньше 1")
			}
			opt.Every = time.Duration(n) * time.Minute
			opt.EverySet = true
		case a == "--max-pages" || strings.HasPrefix(a, "--max-pages="):
			v, err := takeValue(args, &i, a, "--max-pages")
			if err != nil {
				return Options{}, err
			}
			n, err := strconv.Atoi(v)
			if err != nil || n < 0 {
				return Options{}, fmt.Errorf("--max-pages: нужно целое число, 0 значит до конца выдачи")
			}
			opt.MaxPages = n
		case a == "--sources" || strings.HasPrefix(a, "--sources="):
			v, err := takeValue(args, &i, a, "--sources")
			if err != nil {
				return Options{}, err
			}
			opt.Sources = splitSources(v)
			if len(opt.Sources) == 0 {
				return Options{}, fmt.Errorf("--sources: список пуст")
			}
		case a == "--":
			positionals = append(positionals, args[i+1:]...)
			i = len(args)
		case strings.HasPrefix(a, "-"):
			return Options{}, fmt.Errorf("неизвестный флаг %s (см. --help)", a)
		default:
			positionals = append(positionals, a)
		}
	}
	if len(positionals) > 1 {
		return Options{}, fmt.Errorf("запрос должен быть одним аргументом, возьмите его в кавычки")
	}
	if len(positionals) == 1 {
		opt.Query = positionals[0]
	}
	if opt.Help {
		return opt, nil
	}
	if opt.EverySet && !opt.Scanner {
		return Options{}, fmt.Errorf("--every имеет смысл только вместе с --scanner")
	}
	if opt.Listen != "" && !opt.Scanner {
		return Options{}, fmt.Errorf("--listen работает вместе с --scanner")
	}
	if opt.Webhook != "" && !opt.Scanner {
		return Options{}, fmt.Errorf("--webhook работает вместе с --scanner")
	}
	if opt.Listen != "" {
		if err := checkListen(opt.Listen); err != nil {
			return Options{}, err
		}
	}
	if opt.History && opt.HistoryN > 0 && opt.Query != "" {
		return Options{}, fmt.Errorf("нельзя одновременно указывать запрос и --history N")
	}
	if opt.History && opt.HistoryN == 0 && opt.Query != "" {
		return Options{}, fmt.Errorf("--history без номера печатает список и не запускает поиск")
	}
	return opt, nil
}

func splitSources(v string) []string {
	var out []string
	for _, part := range strings.Split(v, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func takeValue(args []string, i *int, a, name string) (string, error) {
	if v, ok := strings.CutPrefix(a, name+"="); ok {
		if v == "" {
			return "", fmt.Errorf("%s: нужно значение", name)
		}
		return v, nil
	}
	if *i+1 >= len(args) || strings.HasPrefix(args[*i+1], "-") {
		return "", fmt.Errorf("%s: нужно значение", name)
	}
	*i++
	return args[*i], nil
}

func isInt(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

func looksLikeAddr(s string) bool {
	if s == "" || strings.HasPrefix(s, "-") {
		return false
	}
	if _, err := strconv.Atoi(s); err == nil {
		return true
	}
	_, port, err := net.SplitHostPort(s)
	if err != nil {
		return false
	}
	_, err = strconv.Atoi(port)
	return err == nil
}

func checkListen(addr string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("--listen: нужен адрес host:port")
	}
	if host == "localhost" {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("--listen принимает только loopback (127.0.0.1 или localhost)")
	}
	return nil
}

// Help is the --help text.
func Help() string {
	return `vscan — поиск вакансий программистов в терминале

Использование:
  vscan [флаги] [запрос]
  echo 'Senior&|Angular' | vscan
  vscan 'Senior&Angular' | wc -l

Запрос — одна строка. Операторы без пробелов или с пробелами:
  Senior&Angular             оба слова целиком, без синонимов
  Senior||Angular            достаточно одного слова
  Senior&|Angular&|Remote    мягкое И: без регистра, ё=е, префикс
                             (Angular совпадёт с AngularJS), синонимы
                             (remote = relocate, удалённо, wfh),
                             слово может быть в заголовке или в описании
  Senior&Angular&|Remote     Senior и Angular буквально, Remote через синонимы
  (Senior||Lead)&Angular     скобки; & и &| сильнее, чем ||
  "remote work"              фраза в кавычках

Одно слово без оператора ищется буквально. Несколько слов без
оператора — ошибка: соедините их через &, &| или ||.

Флаги:
  -h, --help                 эта справка
  --json                     JSON-объект на строку вместо голого URL
  --sources hh,habr,...      площадки (по умолчанию все)
                             hh, habr, superjob, djinni, getmatch,
                             geekjob, remoteok, wwr
  --max-pages N              страниц на площадку; 0 — до конца выдачи
  --scanner                  повторять поиск
  --every N                  минуты между проходами (по умолчанию 60)
  --listen [ADDR]            SSE на 127.0.0.1:8787 (GET /events, GET /health)
  --webhook URL              POST JSON на каждый новый URL
  --history                  прошлые запросы, новые сверху
  --history N                снова выполнить запись номер N
  --no-history               не запоминать этот запрос

stdout — только ссылки (или JSON). Ошибки площадок идут в stderr,
остальные площадки при этом продолжают работу. Код выхода: 0 — есть
совпадения, 1 — пусто, 2 — ошибка аргументов. Закрытый пайп (head)
завершает процесс с кодом 0.

Сканер пишет уже виденные ссылки в кэш. Если кэш для этого запроса
ещё пуст, первый проход молчит и только запоминает выдачу. Дальше
наружу уходит ссылка, которой не было в кэше. --listen и --webhook
доступны только вместе с --scanner и могут работать одновременно
со stdout.

  vscan --scanner --every 30 --listen 127.0.0.1:8787 'Senior&|Angular'

Свой Telegram-бот подключается снаружи, в vscan его нет:

  vscan --scanner --every 30 'Senior&|Angular' | while read -r url; do
    curl -s -X POST "https://api.telegram.org/bot$TOKEN/sendMessage" \
      -d chat_id="$CHAT" --data-urlencode text="$url"
  done

Кэш: ~/.cache/vscan (или $XDG_CACHE_HOME/vscan).
История: ~/.local/share/vscan/history.
Свои синонимы: ~/.config/vscan/synonyms
  remote = relocate, удалённо, wfh

Переменные окружения:
  HH_TOKEN            необязательный ключ API hh.ru
  SUPERJOB_API_KEY    если задан, SuperJob читается через API, иначе HTML

Ленты Remote OK и We Work Remotely публичные. В stdout только ссылка
на вакансию; источник этих лент — remoteok.com и weworkremotely.com.
`
}
