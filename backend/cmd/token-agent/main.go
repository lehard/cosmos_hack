// Команда token-agent — агент токена рабочего места (FR-69, AD-14, Д-72):
// адаптер «физический ключ» порта подписи. Браузерное расширение «Главный —
// подпись» говорит с ним по Native Messaging (contracts/internal/token-agent);
// PIN вводится только в окне этой программы, никогда на веб-странице.
//
// Каркас MVP: «токен» — зашифрованное под PIN хранилище ключа на съёмном
// носителе или в каталоге пользователя (класс hardware_token); PKCS#11 —
// следующий адаптер за тем же протоколом. Отпечаток, сводку, отказ уровня 1,
// частоту и локальный журнал считает то же ядро, что и WASM-пакет
// расширения (cmd/token-agent/internal/agent).
//
// Режимы:
//
//	token-agent import -key ‹файл›.key.json [-key ‹pq›.key.json] [-token ‹каталог›]
//	token-agent install -extension-id ‹id› [-browser chrome|chromium|yandex|all]
//	token-agent status
//	token-agent chrome-extension://‹id›/   (так его запускает браузер — Native Messaging)
//
// Слой: cmd (AD-1). Владелец: эпик 38.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// HostName — имя хоста Native Messaging (манифест, connectNative расширения).
const HostName = "ru.glavny.token_agent"

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) > 0 && strings.HasPrefix(args[0], "chrome-extension://") {
		return serveNative(os.Stdin, os.Stdout, args[0], defaultConfig())
	}
	if len(args) == 0 {
		usage()
		return 2
	}
	switch args[0] {
	case "import":
		return cmdImport(args[1:])
	case "install":
		return cmdInstall(args[1:])
	case "status":
		return cmdStatus(args[1:])
	case "-h", "--help", "help":
		usage()
		return 0
	}
	usage()
	return 2
}

func usage() {
	fmt.Fprintln(os.Stderr, `token-agent — агент токена «Главный» (подпись человека физическим ключом, AD-14)

  token-agent import -key ‹файл›.key.json [-key ‹pq›.key.json] [-token ‹каталог›]
      зашифровать ключ под PIN и положить на «токен» (каталог; по умолчанию — в каталоге пользователя)
  token-agent install -extension-id ‹id› [-browser all]
      записать манифест Native Messaging для Chrome, Chromium, Яндекс Браузера
  token-agent status
      что на токене и где манифест`)
}

// multi — повторяемый флаг.
type multi []string

func (m *multi) String() string     { return strings.Join(*m, ",") }
func (m *multi) Set(v string) error { *m = append(*m, v); return nil }

func newFlags(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	return fs
}
