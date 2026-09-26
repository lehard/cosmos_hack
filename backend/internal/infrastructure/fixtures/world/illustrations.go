package world

import (
	"fmt"
	"slices"
	"strings"
	"time"

	materialsapp "ant/internal/application/materials"
	"ant/internal/infrastructure/fixtures/loader"
)

// Иллюстрации к наблюдениям камеры в заготовках (FR-102, PRD §11.12, NFR-UI-4):
// кадров камеры КТ-3 в мире заготовок нет, а файлы открытого набора в
// репозиторий не кладутся (Д-57). Поэтому к наблюдениям прикладывается
// схема-иллюстрация класса (SVG, строит генератор) — с пометкой
// «ИЛЛЮСТРАЦИЯ» на самой картинке и в метаданных материала (is_illustration,
// kind = illustration): это не кадр изделия и не доказательство. Адрес —
// H(байты) streebog256:… (AD-23), как у материалов live; materials.material.read
// и materials.material.content отдают метаданные и байты.

// illustrationMark — пометка иллюстрации (normative/vision/illustrations.v1.yaml, mark).
const illustrationMark = "ИЛЛЮСТРАЦИЯ"

// illustration — схема-иллюстрация к наблюдению мира.
type illustration struct {
	title string // подпись на картинке
	note  string // происхождение (provenance_note)
	svg   []byte
	addr  string
}

func newIllustration(title, note string, svg string) *illustration {
	b := []byte(svg)
	return &illustration{title: title, note: note, svg: b, addr: Digest(b)}
}

// Иллюстрации мира: прожог (два ракурса — как два наблюдения КТ-3 по НС-01) и блик.
var (
	illBurnGeneral = newIllustration("Класс «прожог» · шов W-1, участок У2 · ракурс 1 — общий вид шва",
		illustrationNote("прожог", "burn_through"), burnGeneralSVG())
	illBurnClose = newIllustration("Класс «прожог» · шов W-1, участок У2 · ракурс 2 — крупный план",
		illustrationNote("прожог", "burn_through"), burnCloseSVG())
	illGlare = newIllustration("Блик на шве · участки У6–У7 не читаются · оценка невозможна",
		illustrationNote("блик (оценка невозможна)", "glare"), glareSVG())
)

func illustrationNote(what, class string) string {
	return illustrationMark + ": схема класса «" + what + "» (" + class + "), построена миром заготовок; не кадр изделия и не доказательство. " +
		"Кадры камеры в заготовках не хранятся; на стенде образец класса берётся из открытого набора (normative/vision/illustrations.v1.yaml), если он распакован на краю."
}

// attach — приложить иллюстрацию к наблюдению (адрес — в evidence_refs).
func (e *Event) attach(il *illustration) {
	e.Materials = append(e.Materials, il)
}

// observationRefs — адреса материалов наблюдений изделия, на которых построен
// сигнал в момент at (признаки дефекта за 5 минут до сигнала), без повторов.
func (m *Model) observationRefs(it *Item, at time.Time) []string {
	out := []string{}
	for _, e := range m.Events {
		if e.Item != it || e.Type != "inspection.result.recorded" || e.Params["outcome"] != "defect_indicated" || e.Occurred.After(at) || at.Sub(e.Occurred) >= 5*time.Minute {
			continue
		}
		for _, a := range e.evidenceRefs() {
			if !slices.Contains(out, a) {
				out = append(out, a)
			}
		}
	}
	return out
}

// evidenceRefs — адреса материалов наблюдения.
func (e *Event) evidenceRefs() []string {
	out := []string{}
	for _, il := range e.Materials {
		out = append(out, il.addr)
	}
	return out
}

// renderMaterials — метаданные и содержимое иллюстраций наблюдений, известных на шаге.
func renderMaterials(c *Ctx) []loader.Response {
	var out []loader.Response
	seen := map[string]bool{}
	for _, e := range c.M.Events {
		if e.Step > c.N || len(e.Materials) == 0 {
			continue
		}
		for _, il := range e.Materials {
			if seen[il.addr] {
				continue
			}
			seen[il.addr] = true
			info := materialsapp.MaterialInfo{MaterialAddress: il.addr, MediaType: "image/svg+xml", SizeBytes: int64(len(il.svg)), Kind: "illustration",
				IsIllustration: true, ProvenanceNote: il.note}
			if e.Item != nil {
				info.ItemID = FullID(e.Item.ID)
			}
			out = append(out, resp("materials.material.read", info, "address", il.addr),
				resp("materials.material.content", il.svg, "address", il.addr)) // тело — байты (base64 в JSON), тип — в materials.material.read
		}
	}
	return out
}

// ── SVG ──

// svgFrame — общая рамка: фон, пометка «ИЛЛЮСТРАЦИЯ» сверху, подпись снизу.
func svgFrame(title, body string) string {
	var b strings.Builder
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" width="640" height="400" viewBox="0 0 640 400" font-family="Arial, Helvetica, sans-serif">` + "\n")
	b.WriteString(`<title>` + illustrationMark + `: ` + title + `</title>` + "\n")
	b.WriteString(`<defs>
<linearGradient id="plate" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#9aa1a8"/><stop offset="1" stop-color="#6d747b"/></linearGradient>
<linearGradient id="bead" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#7f868d"/><stop offset="0.45" stop-color="#d2d6da"/><stop offset="1" stop-color="#737a81"/></linearGradient>
<radialGradient id="heat"><stop offset="0" stop-color="#3b2413" stop-opacity="0.95"/><stop offset="0.5" stop-color="#7a4b22" stop-opacity="0.6"/><stop offset="0.8" stop-color="#4d5f9a" stop-opacity="0.35"/><stop offset="1" stop-color="#4d5f9a" stop-opacity="0"/></radialGradient>
<radialGradient id="hole"><stop offset="0" stop-color="#050505"/><stop offset="0.8" stop-color="#1a1410"/><stop offset="1" stop-color="#3a2a1c"/></radialGradient>
<radialGradient id="glare"><stop offset="0" stop-color="#ffffff" stop-opacity="1"/><stop offset="0.35" stop-color="#ffffff" stop-opacity="0.9"/><stop offset="1" stop-color="#ffffff" stop-opacity="0"/></radialGradient>
</defs>
`)
	b.WriteString(`<rect width="640" height="400" fill="#2b2f33"/>` + "\n")
	b.WriteString(body)
	b.WriteString(`<rect x="0" y="0" width="640" height="36" fill="#b71c1c"/>` + "\n")
	b.WriteString(`<text x="320" y="24" fill="#ffffff" font-size="17" font-weight="bold" text-anchor="middle">` + illustrationMark + ` — схема, не кадр изделия</text>` + "\n")
	b.WriteString(`<rect x="0" y="364" width="640" height="36" fill="#1e2124"/>` + "\n")
	b.WriteString(`<text x="320" y="387" fill="#e6e6e6" font-size="13" text-anchor="middle">` + title + `</text>` + "\n")
	b.WriteString("</svg>\n")
	return b.String()
}

// seam — пластины и валик шва по горизонтали: y — верх валика, h — высота, step — шаг чешуек.
func seam(y, h, step int) string {
	var b strings.Builder
	fmt.Fprintf(&b, `<rect x="0" y="36" width="640" height="%d" fill="url(#plate)"/>`+"\n", y-36)
	fmt.Fprintf(&b, `<rect x="0" y="%d" width="640" height="%d" fill="url(#plate)"/>`+"\n", y+h, 364-y-h)
	fmt.Fprintf(&b, `<rect x="0" y="%d" width="640" height="%d" rx="%d" fill="url(#bead)"/>`+"\n", y, h, h/3)
	b.WriteString(`<g fill="none" stroke="#8b9197" stroke-opacity="0.7" stroke-width="1.2">`)
	for x := step / 2; x < 640; x += step {
		fmt.Fprintf(&b, `<path d="M%d %d q%d %d 0 %d"/>`, x, y+2, step, h/2-2, h-4)
	}
	b.WriteString("</g>\n")
	return b.String()
}

// sections — разметка участков У1–У8 над валиком; hl — выделенные участки (янтарная рамка).
func sections(y, h int, hl ...int) string {
	var b strings.Builder
	const x0, w = 20, 75
	for i := 0; i < 8; i++ {
		x := x0 + i*w
		fmt.Fprintf(&b, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#e6e6e6" stroke-opacity="0.5" stroke-dasharray="3 3"/>`, x, y-18, x, y+h+8)
		fmt.Fprintf(&b, `<text x="%d" y="%d" fill="#f2f2f2" font-size="13" text-anchor="middle">У%d</text>`+"\n", x+w/2, y-8, i+1)
		for _, k := range hl {
			if k == i+1 {
				fmt.Fprintf(&b, `<rect x="%d" y="%d" width="%d" height="%d" fill="none" stroke="#ffb300" stroke-width="2.5" stroke-dasharray="7 4"/>`+"\n", x+2, y-4, w-4, h+8)
			}
		}
	}
	fmt.Fprintf(&b, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#e6e6e6" stroke-opacity="0.5" stroke-dasharray="3 3"/>`+"\n", x0+8*w, y-18, x0+8*w, y+h+8)
	return b.String()
}

// burnGeneralSVG — прожог, общий вид шва: У2 выделен, прожог 52–58 мм.
func burnGeneralSVG() string {
	y, h := 170, 48
	body := seam(y, h, 9) + sections(y, h, 2) +
		`<ellipse cx="119" cy="194" rx="22" ry="17" fill="url(#heat)"/>` + "\n" +
		`<ellipse cx="119" cy="194" rx="7" ry="6" fill="url(#hole)"/>` + "\n" +
		`<line x1="119" y1="226" x2="150" y2="290" stroke="#ffb300" stroke-width="1.5"/>` + "\n" +
		`<text x="154" y="300" fill="#ffd54f" font-size="14">прожог ≈ Ø2 мм, 52–58 мм от начала шва</text>` + "\n" +
		`<text x="20" y="80" fill="#f2f2f2" font-size="14">Шов W-1 (стыковой), вид с лицевой стороны</text>` + "\n"
	return svgFrame(illBurnTitle(1), body)
}

// burnCloseSVG — прожог, крупный план участка У2.
func burnCloseSVG() string {
	y, h := 130, 130
	body := seam(y, h, 26) +
		`<path d="M250 140 C 280 150, 360 150, 390 140 L 392 250 C 360 258, 280 258, 248 250 Z" fill="#5d646b" fill-opacity="0.35"/>` + "\n" +
		`<ellipse cx="320" cy="196" rx="70" ry="56" fill="url(#heat)"/>` + "\n" +
		`<path d="M296 178 C 306 160, 340 162, 348 180 C 356 200, 340 222, 318 220 C 296 218, 288 196, 296 178 Z" fill="url(#hole)"/>` + "\n" +
		`<path d="M300 184 C 310 172, 334 172, 340 186" fill="none" stroke="#c08a50" stroke-width="2" stroke-opacity="0.8"/>` + "\n" +
		`<line x1="296" y1="300" x2="348" y2="300" stroke="#f2f2f2" stroke-width="2"/>` + "\n" +
		`<line x1="296" y1="294" x2="296" y2="306" stroke="#f2f2f2" stroke-width="2"/><line x1="348" y1="294" x2="348" y2="306" stroke="#f2f2f2" stroke-width="2"/>` + "\n" +
		`<text x="322" y="324" fill="#f2f2f2" font-size="13" text-anchor="middle">≈ 2 мм</text>` + "\n" +
		`<text x="20" y="80" fill="#f2f2f2" font-size="14">Участок У2 (40–80 мм), крупный план: сквозное отверстие, цвета побежалости</text>` + "\n"
	return svgFrame(illBurnTitle(2), body)
}

func illBurnTitle(view int) string {
	if view == 1 {
		return "Класс «прожог» · шов W-1, участок У2 · ракурс 1 — общий вид шва"
	}
	return "Класс «прожог» · шов W-1, участок У2 · ракурс 2 — крупный план"
}

// glareSVG — блик на участках У6–У7, прижим частично закрывает зону.
func glareSVG() string {
	y, h := 170, 48
	body := seam(y, h, 9) + sections(y, h, 6, 7) +
		`<ellipse cx="470" cy="192" rx="90" ry="58" fill="url(#glare)"/>` + "\n" +
		`<rect x="505" y="120" width="54" height="150" rx="6" fill="#3c4043" stroke="#202326" stroke-width="2"/>` + "\n" +
		`<text x="532" y="112" fill="#f2f2f2" font-size="12" text-anchor="middle">прижим</text>` + "\n" +
		`<text x="20" y="80" fill="#f2f2f2" font-size="14">Шов W-1: блик и прижим — участки У6–У7 не читаются</text>` + "\n" +
		`<text x="20" y="310" fill="#ffd54f" font-size="14">Качество наблюдения 0,34 &lt; порога 0,6 → «оценка невозможна», а не «годно»</text>` + "\n"
	return svgFrame("Блик на шве · участки У6–У7 не читаются · оценка невозможна", body)
}
