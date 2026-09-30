# vscan

Консольный поиск вакансий программистов. Один статический бинарник для macOS и Linux, без интерфейса: ссылки печатаются в stdout по мере нахождения и нормально идут в пайп.

```
vscan 'Senior&|Angular&|Remote'
echo 'Senior&Angular' | vscan
vscan 'Senior&Angular' | wc -l
```

## Запрос

| Запись | Смысл |
| --- | --- |
| `Senior&Angular` | оба слова целиком, без синонимов. `Angular` не совпадёт с `AngularJS` |
| `Senior\|\|Angular` | достаточно одного слова |
| `Senior&\|Angular&\|Remote` | мягкое И. Слова ищутся в заголовке, описании, навыках и тегах, порядок не важен. `Angular` совпадёт с `AngularJS`, `Remote` — с `relocate`, `удалённо`, `wfh` |
| `Senior&Angular&\|Remote` | `Senior` и `Angular` буквально, `Remote` через синонимы |
| `(Senior\|\|Lead)&Angular` | скобки. `&` и `&\|` сильнее, чем `\|\|` |
| `"remote work"` | фраза |

Режим слова задаёт оператор слева от него. У первого слова — режим первого оператора.

Свои синонимы, файл `~/.config/vscan/synonyms` (`$XDG_CONFIG_HOME/vscan/synonyms`):

```
remote = relocate, удалённо, удаленно, wfh
```

Строка заменяет всю группу для этого слова.

## Площадки

`hh`, `habr`, `superjob`, `djinni`, `getmatch`, `geekjob`, `remoteok`, `wwr`.

Выбор: `--sources hh,habr`. Ошибка одной площадки пишется в stderr, остальные продолжают. Обхода логина и капчи нет: если площадка вернула пустую страницу, в stderr будет ошибка.

`HH_TOKEN` — необязательный ключ API hh.ru. `SUPERJOB_API_KEY` — если задан, SuperJob идёт через API, иначе читается публичная HTML-выдача.

Ленты Remote OK и We Work Remotely публичные. В stdout только ссылка на вакансию. Сами ленты: [remoteok.com](https://remoteok.com) и [weworkremotely.com](https://weworkremotely.com).

## Сканер

```
vscan --scanner --every 30 --listen 127.0.0.1:8787 'Senior&|Angular'
```

`--every` — минуты между проходами, по умолчанию 60. Уже виденные ссылки лежат в `~/.cache/vscan/seen`. Пока кэш этого запроса пуст, первый проход только запоминает выдачу и ничего не печатает. Со следующих проходов наружу уходит только новая ссылка.

Три выхода можно включить вместе:

- stdout — по одному URL на строку
- `--listen` — `GET /events` (SSE) и `GET /health`. Только loopback
- `--webhook URL` — `POST` JSON

Telegram в программу не встроен:

```
vscan --scanner --every 30 'Senior&|Angular' | while read -r url; do
  curl -s -X POST "https://api.telegram.org/bot$TOKEN/sendMessage" \
    -d chat_id="$CHAT" --data-urlencode text="$url"
done
```

## История

```
vscan --history
vscan --history 3
vscan --no-history 'Senior&Angular'
```

Файл `~/.local/share/vscan/history`, новые запросы сверху, повтор поднимается наверх и не копируется.

## Коды выхода

`0` — есть совпадения, сканер остановлен, или пайп закрыт (например `head`). `1` — ничего не найдено. `2` — ошибка аргументов.

`--json` печатает объект с полями `url`, `title`, `company`, `source`.

## Сборка

Нужен Go. Cgo не используется.

```
make build
make test
make install PREFIX=/usr/local
make cross
```

`make cross` собирает darwin/amd64, darwin/arm64, linux/amd64 и linux/arm64 в `dist/`.
