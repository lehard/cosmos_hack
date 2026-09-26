package paperscan

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	dom "ant/internal/domain/signing"
)

// FR-139: QR печатной рамки читается со «скана» (картинка с полями и шумом
// вокруг), чистый лист без QR — ошибка.
func TestRoundTrip(t *testing.T) {
	text := dom.QRText("DOC-0001", dom.Digest([]byte("документ")))
	qr, err := PNG(text, 300)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Reader{}.ReadQR(qr)
	if err != nil || got != text {
		t.Fatalf("%q %v", got, err)
	}
	// «Скан»: QR на листе побольше, смещён.
	src, _ := png.Decode(bytes.NewReader(qr))
	sheet := image.NewGray(image.Rect(0, 0, 800, 1000))
	for i := range sheet.Pix {
		sheet.Pix[i] = 250
	}
	b := src.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			sheet.Set(420+x, 600+y, color.GrayModel.Convert(src.At(x, y)))
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, sheet)
	if got, err := (Reader{}).ReadQR(buf.Bytes()); err != nil || got != text {
		t.Fatalf("скан: %q %v", got, err)
	}
	blank := image.NewGray(image.Rect(0, 0, 200, 200))
	buf.Reset()
	_ = png.Encode(&buf, blank)
	if _, err := (Reader{}).ReadQR(buf.Bytes()); !errors.Is(err, ErrNoQR) {
		t.Fatalf("пустой лист: %v", err)
	}
	svg, err := SVG(text)
	if err != nil || !strings.HasPrefix(svg, "<svg") {
		t.Fatal(err)
	}
}
