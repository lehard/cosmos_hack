package documents

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Строки проверки QR: короткая, отпечаток документа (версия 6–9) и длинная
// (версия ≥ 10, длина — 16 бит).
var qrSamples = []string{
	"ant:doc:DOC-0001:streebog256:03e6a60374c5881703bb53ae6d68ae2eb4775d97efb981d7b615e05e7cbf96c8",
	"ant:doc:TRV-ENT01:FL-0007:streebog256:8e27d3109db2cc32ac35639fad767845c1547a7f972611cd3304a80fc8e408f9",
	"HELLO",
	"ant:doc:NCS-0192f000-0000-7000-8000-000000000123:streebog256:" + strings.Repeat("ab", 32) + "/" + strings.Repeat("x", 90),
}

// TestQRShape — размер матрицы по версии, узоры поиска на месте. Если задан
// ANT_QR_OUT, матрицы пишутся в PBM для проверки внешним декодером
// (zbarimg): так кодировщик сверяется с независимой реализацией.
func TestQRShape(t *testing.T) {
	dir := os.Getenv("ANT_QR_OUT")
	for i, s := range qrSamples {
		m, err := EncodeQR(s)
		if err != nil {
			t.Fatal(err)
		}
		n := len(m)
		if (n-17)%4 != 0 || n < 21 {
			t.Fatalf("размер %d", n)
		}
		for _, c := range [][2]int{{0, 0}, {n - 7, 0}, {0, n - 7}} {
			if !m[c[1]][c[0]] || !m[c[1]+6][c[0]+6] || m[c[1]+1][c[0]+1] {
				t.Fatalf("узор поиска в %v", c)
			}
		}
		if dir == "" {
			continue
		}
		var b strings.Builder
		scale, quiet := 4, 4
		dim := (n + 2*quiet) * scale
		fmt.Fprintf(&b, "P1\n%d %d\n", dim, dim)
		for y := 0; y < dim; y++ {
			for x := 0; x < dim; x++ {
				mx, my := x/scale-quiet, y/scale-quiet
				v := "0"
				if mx >= 0 && my >= 0 && mx < n && my < n && m[my][mx] {
					v = "1"
				}
				b.WriteString(v)
				b.WriteString(" ")
			}
			b.WriteString("\n")
		}
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("qr%d.pbm", i)), []byte(b.String()), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("qr%d.txt", i)), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
