// Команда archgen генерирует backend/.golangci.yml — правила направления
// зависимостей (depguard) и детерминизма домена (forbidigo) — из списка модулей
// и порядка импорта в layers.json (AD-1, AD-4, AD-40; NFR-ARCH-1, NFR-DEV-3).
//
// Использование (из каталога backend):
//
//	go run ./tools/archgen -layers tools/archgen/layers.json -out .golangci.yml
//	go run ./tools/archgen ... -check   # сверка без записи (make check)
//
// Правила:
//   - domain/‹m›: только stdlib (кроме запрещённых пакетов ввода-вывода,
//     случайности и часов), сгенерированные контракты, Стрибог-256 из GoGOST и
//     доменные модули раньше ‹m› в полном порядке импорта; прочие доменные
//     модули — только базовые (reference, signing, access);
//   - application: не импортирует инфраструктуру, cmd и HTTP/БД-библиотеки;
//   - зоны storage, transport, integration, fixtures не импортируют друг друга;
//     observability и security — технические библиотеки, не импортируют зоны;
//   - forbidigo в domain/**: часы, случайность, os (AD-4).
//
// Список стандартных пакетов берётся из `go list std`, поэтому результат
// зависит от версии Go — она закреплена образом golang:1.27.1.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
)

type layers struct {
	Module           string   `json:"module"`
	DomainOrder      []string `json:"domain_order"`
	DomainBase       []string `json:"domain_base"`
	DomainOther      []string `json:"domain_other"`
	DomainExtraAllow []string `json:"domain_extra_allow"`
	DomainDenyStd    []string `json:"domain_deny_std"`
	Zones            []string `json:"zones"`
	TechZones        []string `json:"tech_zones"`
	ApplicationDeny  []string `json:"application_deny"`
}

func main() {
	layersPath := flag.String("layers", "tools/archgen/layers.json", "источник правил")
	out := flag.String("out", ".golangci.yml", "куда писать конфигурацию golangci-lint")
	check := flag.Bool("check", false, "только сверить с существующим файлом")
	flag.Parse()

	raw, err := os.ReadFile(*layersPath)
	if err != nil {
		fail(err)
	}
	var l layers
	if err := json.Unmarshal(raw, &l); err != nil {
		fail(fmt.Errorf("%s: %w", *layersPath, err))
	}
	if err := l.validate(); err != nil {
		fail(fmt.Errorf("%s: %w", *layersPath, err))
	}
	std, err := stdPackages()
	if err != nil {
		fail(err)
	}
	gen := render(&l, std)

	if *check {
		cur, err := os.ReadFile(*out)
		if err != nil || !bytes.Equal(cur, gen) {
			fmt.Fprintf(os.Stderr, "archgen: %s устарел относительно %s — выполните make generate\n", *out, *layersPath)
			os.Exit(1)
		}
		return
	}
	if err := os.WriteFile(*out, gen, 0o644); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "archgen:", err)
	os.Exit(1)
}

func (l *layers) validate() error {
	seen := map[string]bool{}
	for _, m := range append(slices.Clone(l.DomainOrder), l.DomainOther...) {
		if seen[m] {
			return fmt.Errorf("модуль %q указан дважды", m)
		}
		seen[m] = true
	}
	for i, b := range l.DomainBase {
		if i >= len(l.DomainOrder) || l.DomainOrder[i] != b {
			return fmt.Errorf("domain_base должен быть началом domain_order (AD-1): %v", l.DomainBase)
		}
	}
	if l.Module == "" {
		return fmt.Errorf("не задан module")
	}
	return nil
}

func stdPackages() ([]string, error) {
	cmd := exec.Command("go", "list", "std")
	cmd.Stderr = os.Stderr
	b, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list std: %w", err)
	}
	var pkgs []string
	for p := range strings.FieldsSeq(string(b)) {
		// vendor/… и internal/… недоступны для импорта из пользовательского кода.
		if strings.HasPrefix(p, "vendor/") || strings.HasPrefix(p, "internal/") || strings.Contains(p, "/internal/") || strings.HasSuffix(p, "/internal") {
			continue
		}
		pkgs = append(pkgs, p)
	}
	slices.Sort(pkgs)
	return pkgs, nil
}

// expandStd раскрывает префиксы в точные пакеты stdlib: depguard считает более
// длинное совпадение из allow ($gostd) сильнее короткого из deny, поэтому
// запрещать нужно каждый пакет, а не только префикс.
func expandStd(prefixes, std []string) []string {
	var out []string
	for _, p := range std {
		for _, pre := range prefixes {
			if p == pre || strings.HasPrefix(p, pre+"/") {
				out = append(out, p)
				break
			}
		}
	}
	return out
}

type writer struct{ bytes.Buffer }

func (w *writer) f(format string, a ...any) { fmt.Fprintf(&w.Buffer, format+"\n", a...) }

func q(s string) string { b, _ := json.Marshal(s); return string(b) }

func render(l *layers, std []string) []byte {
	var w writer
	mod := l.Module
	dom := func(m string) string { return mod + "/internal/domain/" + m }
	infra := func(z string) string { return mod + "/internal/infrastructure/" + z }
	denyStd := expandStd(l.DomainDenyStd, std)

	w.f("# СГЕНЕРИРОВАНО backend/tools/archgen из backend/tools/archgen/layers.json — руками не править.")
	w.f("# Перегенерация: make generate. Правила: AD-1 (слои, зоны, порядок импорта), AD-4 (детерминизм домена).")
	w.f("version: \"2\"")
	w.f("run:")
	w.f("  modules-download-mode: vendor")
	w.f("  tests: true")
	w.f("linters:")
	w.f("  default: none")
	w.f("  enable: [depguard, forbidigo, govet, errcheck, ineffassign, staticcheck, unused]")
	w.f("  settings:")
	w.f("    depguard:")
	w.f("      rules:")

	domainAll := slices.Concat(l.DomainOrder, l.DomainOther)
	slices.Sort(domainAll)

	// Общее правило домена: ввод-вывод, случайность и лишние внешние пакеты.
	ruleStrict(&w, "domain", []string{"**/internal/domain/**"},
		slices.Concat([]string{"$gostd"}, l.DomainExtraAllow, mapf(domainAll, dom)),
		denyStd, "AD-1, AD-4: домен — только stdlib без ввода-вывода, часов и случайности, контракты, Стрибог-256 и доменные модули")

	// Порядок импорта доменных модулей.
	for i, m := range l.DomainOrder {
		allowed := slices.Concat(l.DomainExtraAllow, mapf(l.DomainOrder[:i], dom), []string{dom(m)})
		ruleStrict(&w, "domain-"+m, []string{"**/internal/domain/" + m + "/**"},
			slices.Concat([]string{"$gostd"}, allowed), nil,
			fmt.Sprintf("AD-1: domain/%s импортирует только доменные модули раньше себя в порядке %s", m, strings.Join(l.DomainOrder, " → ")))
	}
	for _, m := range l.DomainOther {
		allowed := slices.Concat(l.DomainExtraAllow, mapf(l.DomainBase, dom), []string{dom(m)})
		ruleStrict(&w, "domain-"+m, []string{"**/internal/domain/" + m + "/**"},
			slices.Concat([]string{"$gostd"}, allowed), nil,
			fmt.Sprintf("AD-1: domain/%s импортирует из домена только %s", m, strings.Join(l.DomainBase, ", ")))
	}
	// Модуль, которого нет в списке, — как «прочие»: только базовые.
	unknownFiles := []string{"**/internal/domain/**"}
	for _, m := range domainAll {
		unknownFiles = append(unknownFiles, "!**/internal/domain/"+m+"/**")
	}
	ruleStrict(&w, "domain-unlisted", unknownFiles,
		slices.Concat([]string{"$gostd"}, l.DomainExtraAllow, mapf(l.DomainBase, dom)), nil,
		"AD-1: доменный модуль не внесён в backend/tools/archgen/layers.json — внесите его и выполните make generate")

	// Приложение.
	ruleLax(&w, "application", []string{"**/internal/application/**"},
		slices.Concat([]string{infra(""), mod + "/cmd"}, l.ApplicationDeny),
		"AD-1: application зависит только от domain и своих портов; HTTP, БД и инфраструктура — в infrastructure")

	// Зоны инфраструктуры не импортируют друг друга.
	for _, z := range l.Zones {
		var deny []string
		for _, o := range l.Zones {
			if o != z {
				deny = append(deny, infra(o))
			}
		}
		deny = append(deny, mod+"/cmd")
		ruleLax(&w, "infrastructure-"+z, []string{"**/internal/infrastructure/" + z + "/**"}, deny,
			"AD-1: зоны storage, transport, integration, fixtures не импортируют друг друга — только порты application")
	}
	for _, z := range l.TechZones {
		deny := slices.Concat(mapf(l.Zones, infra), []string{mod + "/cmd"})
		ruleLax(&w, "infrastructure-"+z, []string{"**/internal/infrastructure/" + z + "/**"}, deny,
			"AD-1: observability и security — технические библиотеки, зоны модулей не импортируют")
	}
	// Никто, кроме cmd, не импортирует cmd.
	ruleLax(&w, "internal-no-cmd", []string{"**/internal/**", "!**/cmd/**"}, []string{mod + "/cmd"},
		"AD-1: сборка зависимостей — только в cmd/*")

	w.f("    forbidigo:")
	w.f("      analyze-types: true")
	w.f("      forbid:")
	forbid(&w, `^time\.(Now|Since|Until|After|AfterFunc|Tick|NewTimer|NewTicker|Sleep)$`, `^time$`,
		"AD-4: домен не читает часы — время приходит аргументом из записей журнала (DomainClock — в application)")
	forbid(&w, `^rand\..*$`, `^(math/rand(/v2)?|crypto/rand)$`,
		"AD-4: в домене нет случайности; идентификаторы реакций — UUIDv5 от NS_ANT")
	forbid(&w, `^os\..*$`, `^os$`, "AD-4: домен не делает ввода-вывода")
	w.f("  exclusions:")
	w.f("    generated: lax")
	w.f("    rules:")
	w.f("      - path-except: (^|/)internal/domain/")
	w.f("        linters: [forbidigo]")
	w.f("issues:")
	w.f("  # Все нарушения в строке, а не только первое: иначе unused скрывает forbidigo.")
	w.f("  uniq-by-line: false")
	w.f("  max-issues-per-linter: 0")
	w.f("  max-same-issues: 0")
	w.f("formatters:")
	w.f("  enable: [gofmt]")
	w.f("  exclusions:")
	w.f("    paths: [vendor]")
	return w.Bytes()
}

func mapf(in []string, f func(string) string) []string {
	out := make([]string, len(in))
	for i, s := range in {
		out[i] = f(s)
	}
	return out
}

func ruleStrict(w *writer, name string, files, allow, deny []string, desc string) {
	w.f("        %s:", name)
	w.f("          list-mode: strict")
	writeFiles(w, files)
	w.f("          allow:")
	for _, a := range allow {
		w.f("            - %s", q(a))
	}
	if len(deny) > 0 {
		w.f("          deny:")
		for _, d := range deny {
			w.f("            - pkg: %s", q(d))
			w.f("              desc: %s", q(desc))
		}
	}
	// В strict-режиме неразрешённый импорт получает общее сообщение depguard;
	// описание правила — в комментарии, чтобы его было видно в конфигурации.
	w.f("          # %s", desc)
}

func ruleLax(w *writer, name string, files, deny []string, desc string) {
	w.f("        %s:", name)
	w.f("          list-mode: lax")
	writeFiles(w, files)
	w.f("          deny:")
	for _, d := range deny {
		w.f("            - pkg: %s", q(d))
		w.f("              desc: %s", q(desc))
	}
}

func writeFiles(w *writer, files []string) {
	w.f("          files:")
	for _, f := range files {
		w.f("            - %s", q(f))
	}
}

func forbid(w *writer, pattern, pkg, msg string) {
	w.f("        - pattern: %s", q(pattern))
	w.f("          pkg: %s", q(pkg))
	w.f("          msg: %s", q(msg))
}
