package documents

import (
	"errors"
	"html"
	"strconv"
	"strings"
)

// QR печатной рамки (AD-12, FR-139, Д-30: «QR рисует сервер»): кодировщик
// QR Code (ISO/IEC 18004) — байтовый режим, уровень коррекции M, версии
// 1–15, маска по штрафу. Внешних зависимостей нет (офлайн-сборка, NFR-SEC-1);
// отпечаток `ant:doc:‹id›:streebog256:‹64 hex›` помещается в версию 6–9.

// qrECM — блоки коррекции уровня M по версиям 1–15: число кодовых слов
// коррекции на блок и группы блоков (число блоков, данных в блоке).
var qrECM = [16]struct {
	ec             int
	g1, d1, g2, d2 int
}{
	{}, {10, 1, 16, 0, 0}, {16, 1, 28, 0, 0}, {26, 1, 44, 0, 0}, {18, 2, 32, 0, 0}, {24, 2, 43, 0, 0},
	{16, 4, 27, 0, 0}, {18, 4, 31, 0, 0}, {22, 2, 38, 2, 39}, {22, 3, 36, 2, 37}, {26, 4, 43, 1, 44},
	{30, 1, 50, 4, 51}, {22, 6, 36, 2, 37}, {22, 8, 37, 1, 38}, {24, 4, 40, 5, 41}, {24, 5, 41, 5, 42},
}

// qrAlign — центры выравнивающих узоров по версиям.
var qrAlign = [16][]int{
	nil, nil, {6, 18}, {6, 22}, {6, 26}, {6, 30}, {6, 34}, {6, 22, 38}, {6, 24, 42}, {6, 26, 46}, {6, 28, 50},
	{6, 30, 54}, {6, 32, 58}, {6, 34, 62}, {6, 26, 46, 66}, {6, 26, 48, 70},
}

// QRMatrix — модули QR (true — тёмный), [строка][столбец].
type QRMatrix [][]bool

type qrGrid struct {
	size int
	mod  [][]bool
	fn   [][]bool
}

func newGrid(size int) *qrGrid {
	g := &qrGrid{size: size, mod: make([][]bool, size), fn: make([][]bool, size)}
	for i := range size {
		g.mod[i], g.fn[i] = make([]bool, size), make([]bool, size)
	}
	return g
}

func (g *qrGrid) set(x, y int, dark bool) { g.mod[y][x], g.fn[y][x] = dark, true }

// EncodeQR — матрица QR для текста (байтовый режим, уровень M).
func EncodeQR(text string) (QRMatrix, error) {
	data := []byte(text)
	ver := 0
	for v := 1; v <= 15; v++ {
		e := qrECM[v]
		capBits := (e.g1*e.d1 + e.g2*e.d2) * 8
		cc := 8
		if v >= 10 {
			cc = 16
		}
		if 4+cc+len(data)*8 <= capBits {
			ver = v
			break
		}
	}
	if ver == 0 {
		return nil, errors.New("qr: текст длиннее ёмкости версии 15-M")
	}
	e := qrECM[ver]
	nData := e.g1*e.d1 + e.g2*e.d2
	// Поток бит: режим 0100, длина, байты, терминатор, выравнивание, заполнение.
	var bits []bool
	put := func(v, n int) {
		for i := n - 1; i >= 0; i-- {
			bits = append(bits, (v>>i)&1 == 1)
		}
	}
	put(4, 4)
	if ver >= 10 {
		put(len(data), 16)
	} else {
		put(len(data), 8)
	}
	for _, b := range data {
		put(int(b), 8)
	}
	put(0, min(4, nData*8-len(bits)))
	for len(bits)%8 != 0 {
		bits = append(bits, false)
	}
	cw := make([]byte, 0, nData)
	for i := 0; i < len(bits); i += 8 {
		var b byte
		for j := range 8 {
			if bits[i+j] {
				b |= 1 << (7 - j)
			}
		}
		cw = append(cw, b)
	}
	for pad := byte(0xEC); len(cw) < nData; {
		cw = append(cw, pad)
		pad ^= 0xEC ^ 0x11
	}
	// Блоки и коррекция Рида — Соломона, перемежение.
	div := rsDivisor(e.ec)
	var blocks, ecs [][]byte
	off := 0
	for i := range e.g1 + e.g2 {
		n := e.d1
		if i >= e.g1 {
			n = e.d2
		}
		b := cw[off : off+n]
		off += n
		blocks = append(blocks, b)
		ecs = append(ecs, rsRemainder(b, div))
	}
	var final []byte
	for i := range max(e.d1, e.d2) {
		for _, b := range blocks {
			if i < len(b) {
				final = append(final, b[i])
			}
		}
	}
	for i := range e.ec {
		for _, b := range ecs {
			final = append(final, b[i])
		}
	}
	size := ver*4 + 17
	best, bestPenalty := QRMatrix(nil), -1
	for mask := range 8 {
		g := newGrid(size)
		g.drawFunction(ver)
		g.drawCodewords(final)
		g.applyMask(mask)
		g.drawFormat(mask)
		if p := g.penalty(); bestPenalty < 0 || p < bestPenalty {
			best, bestPenalty = g.mod, p
		}
	}
	return best, nil
}

func (g *qrGrid) drawFunction(ver int) {
	n := g.size
	for i := range n {
		g.set(6, i, i%2 == 0)
		g.set(i, 6, i%2 == 0)
	}
	for _, c := range [][2]int{{3, 3}, {n - 4, 3}, {3, n - 4}} {
		for dy := -4; dy <= 4; dy++ {
			for dx := -4; dx <= 4; dx++ {
				x, y := c[0]+dx, c[1]+dy
				if x < 0 || x >= n || y < 0 || y >= n {
					continue
				}
				d := max(abs(dx), abs(dy))
				g.set(x, y, d != 2 && d != 4)
			}
		}
	}
	al := qrAlign[ver]
	for i, ax := range al {
		for j, ay := range al {
			if (i == 0 && j == 0) || (i == 0 && j == len(al)-1) || (i == len(al)-1 && j == 0) {
				continue
			}
			for dy := -2; dy <= 2; dy++ {
				for dx := -2; dx <= 2; dx++ {
					g.set(ax+dx, ay+dy, max(abs(dx), abs(dy)) != 1)
				}
			}
		}
	}
	g.drawFormat(0) // резервирует место; перерисовывается после маски
	if ver >= 7 {
		rem := ver
		for range 12 {
			rem = (rem << 1) ^ ((rem >> 11) * 0x1F25)
		}
		v := ver<<12 | rem
		for i := range 18 {
			bit := (v>>i)&1 == 1
			a, b := n-11+i%3, i/3
			g.set(a, b, bit)
			g.set(b, a, bit)
		}
	}
}

func (g *qrGrid) drawFormat(mask int) {
	data := 0<<3 | mask // уровень M = 00
	rem := data
	for range 10 {
		rem = (rem << 1) ^ ((rem >> 9) * 0x537)
	}
	bits := (data<<10 | rem) ^ 0x5412
	bit := func(i int) bool { return (bits>>i)&1 == 1 }
	n := g.size
	for i := 0; i <= 5; i++ {
		g.set(8, i, bit(i))
	}
	g.set(8, 7, bit(6))
	g.set(8, 8, bit(7))
	g.set(7, 8, bit(8))
	for i := 9; i < 15; i++ {
		g.set(14-i, 8, bit(i))
	}
	for i := range 8 {
		g.set(n-1-i, 8, bit(i))
	}
	for i := 8; i < 15; i++ {
		g.set(8, n-15+i, bit(i))
	}
	g.set(8, n-8, true)
}

func (g *qrGrid) drawCodewords(data []byte) {
	n, i := g.size, 0
	for right := n - 1; right >= 1; right -= 2 {
		if right == 6 {
			right = 5
		}
		for vert := range n {
			for j := range 2 {
				x := right - j
				y := vert
				if (right+1)&2 == 0 {
					y = n - 1 - vert
				}
				if !g.fn[y][x] && i < len(data)*8 {
					g.mod[y][x] = (data[i>>3]>>(7-(i&7)))&1 == 1
					i++
				}
			}
		}
	}
}

func (g *qrGrid) applyMask(mask int) {
	for y := range g.size {
		for x := range g.size {
			var inv bool
			switch mask {
			case 0:
				inv = (x+y)%2 == 0
			case 1:
				inv = y%2 == 0
			case 2:
				inv = x%3 == 0
			case 3:
				inv = (x+y)%3 == 0
			case 4:
				inv = (x/3+y/2)%2 == 0
			case 5:
				inv = x*y%2+x*y%3 == 0
			case 6:
				inv = (x*y%2+x*y%3)%2 == 0
			case 7:
				inv = ((x+y)%2+x*y%3)%2 == 0
			}
			if inv && !g.fn[y][x] {
				g.mod[y][x] = !g.mod[y][x]
			}
		}
	}
}

// penalty — штраф маски (ISO/IEC 18004, правила N1–N4).
func (g *qrGrid) penalty() int {
	n, res := g.size, 0
	line := func(get func(i int) bool) {
		color, run := false, 0
		hist := make([]int, 7)
		add := func(r int) {
			if hist[0] == 0 {
				r += n
			}
			copy(hist[1:], hist[:6])
			hist[0] = r
		}
		count := func() int {
			m := hist[1]
			core := m > 0 && hist[2] == m && hist[3] == m*3 && hist[4] == m && hist[5] == m
			c := 0
			if core && hist[0] >= m*4 && hist[6] >= m {
				c++
			}
			if core && hist[6] >= m*4 && hist[0] >= m {
				c++
			}
			return c
		}
		for i := range n {
			if get(i) == color {
				run++
				if run == 5 {
					res += 3
				} else if run > 5 {
					res++
				}
			} else {
				add(run)
				if !color {
					res += count() * 40
				}
				color, run = get(i), 1
			}
		}
		if color {
			add(run)
			run = 0
		}
		run += n
		add(run)
		res += count() * 40
	}
	for y := range n {
		line(func(i int) bool { return g.mod[y][i] })
	}
	for x := range n {
		line(func(i int) bool { return g.mod[i][x] })
	}
	dark := 0
	for y := range n {
		for x := range n {
			if g.mod[y][x] {
				dark++
			}
			if y < n-1 && x < n-1 {
				c := g.mod[y][x]
				if c == g.mod[y][x+1] && c == g.mod[y+1][x] && c == g.mod[y+1][x+1] {
					res += 3
				}
			}
		}
	}
	total := n * n
	k := (abs(dark*20-total*10)+total-1)/total - 1
	return res + k*10
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func gfMul(x, y byte) byte {
	var z int
	for i := 7; i >= 0; i-- {
		z = (z << 1) ^ ((z >> 7) * 0x11D)
		z ^= int((y>>i)&1) * int(x)
	}
	return byte(z)
}

func rsDivisor(degree int) []byte {
	res := make([]byte, degree)
	res[degree-1] = 1
	root := byte(1)
	for range degree {
		for j := range degree {
			res[j] = gfMul(res[j], root)
			if j+1 < degree {
				res[j] ^= res[j+1]
			}
		}
		root = gfMul(root, 0x02)
	}
	return res
}

func rsRemainder(data, div []byte) []byte {
	res := make([]byte, len(div))
	for _, b := range data {
		f := b ^ res[0]
		copy(res, res[1:])
		res[len(res)-1] = 0
		for i := range res {
			res[i] ^= gfMul(div[i], f)
		}
	}
	return res
}

// QRSVG — QR текста как SVG (поле 4 модуля, модуль — 1 единица).
func QRSVG(text string) (string, error) {
	m, err := EncodeQR(text)
	if err != nil {
		return "", err
	}
	n := len(m)
	var b strings.Builder
	dim := strconv.Itoa(n + 8)
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ` + dim + " " + dim + `" shape-rendering="crispEdges">`)
	b.WriteString(`<rect width="100%" height="100%" fill="#fff"/><path fill="#000" d="`)
	for y := range n {
		for x := range n {
			if m[y][x] {
				b.WriteString("M" + strconv.Itoa(x+4) + " " + strconv.Itoa(y+4) + "h1v1h-1z")
			}
		}
	}
	b.WriteString(`"/></svg>`)
	return b.String(), nil
}

// PrintFrame — страница для печати: каноническая отрисовка в рамке с QR
// `ant:doc:‹id›:‹отпечаток›`, датой печати и колонтитулом «получено из
// системы «Главный»» (Д-65: имя платформы для людей; техническое имя в QR —
// `ant`). Рамка в отрисовку и отпечаток не входит (AD-12).
func PrintFrame(rendering, qr, svg, printedAt string) string {
	var b strings.Builder
	b.WriteString("<!doctype html>\n<html lang=\"ru\"><head><meta charset=\"utf-8\"><title>" + html.EscapeString(qr) + "</title>")
	b.WriteString("<style>@page{margin:12mm}body{font:11pt sans-serif}.ant-frame{border:1px solid #000;padding:8mm}" +
		".ant-frame-head{display:flex;justify-content:space-between;align-items:flex-start;gap:8mm}.ant-qr{width:32mm;height:32mm}" +
		"table{border-collapse:collapse;width:100%}td,th{border:1px solid #000;padding:2px 4px;vertical-align:top}" +
		".ant-foot{margin-top:4mm;font-size:9pt}</style></head><body>\n")
	b.WriteString("<div class=\"ant-frame\"><div class=\"ant-frame-head\"><div>")
	b.WriteString(rendering)
	b.WriteString("</div><div class=\"ant-qr\">" + svg + "<p class=\"ant-foot\">" + html.EscapeString(qr) + "</p></div></div>\n")
	b.WriteString("<p class=\"ant-foot\">Получено из системы «Главный» · напечатано " + html.EscapeString(printedAt) +
		" · подлинность — по QR: отпечаток документа сверяется при загрузке скана (FR-139)</p></div>\n</body></html>\n")
	return b.String()
}
