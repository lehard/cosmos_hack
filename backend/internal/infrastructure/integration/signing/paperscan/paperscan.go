// Пакет paperscan — чтение и рисование QR бумажного пути подписи (FR-139,
// AD-12, AD-43): печатная рамка документа несёт QR `ant:doc:‹id›:‹отпечаток›`
// (рисует сервер, Д-30); со скана подписанной распечатки QR читается заново —
// скан с чужим QR не принимается (signing.qr_mismatch). Библиотека —
// makiuchi-d/gozxing v0.1.1 (Apache-2.0; спайн, «Stack»): и чтение, и
// кодирование, без внешних программ и сети.
//
// Слой: infrastructure/integration модуля signing (сканер как внешний
// источник, AD-18); реализует ведомый порт application/signing.QRReader.
// Владелец: эпик 27.
package paperscan

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"

	// Форматы сканов: PNG и JPEG (декодеры регистрируются импортом).
	_ "image/jpeg"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"

	app "ant/internal/application/signing"
)

// Reader — адаптер порта QRReader: QR со скана (PNG, JPEG).
type Reader struct{}

var _ app.QRReader = Reader{}

// ErrNoQR — на скане нет читаемого QR.
var ErrNoQR = errors.New("paperscan: QR на скане не найден")

// ReadQR — текст QR со скана. Сначала как есть, затем с подсказкой «чистый
// штрихкод» (ровная печать без поворота).
func (Reader) ReadQR(img []byte) (string, error) {
	m, _, err := image.Decode(bytes.NewReader(img))
	if err != nil {
		return "", fmt.Errorf("paperscan: скан не читается как изображение: %w", err)
	}
	bmp, err := gozxing.NewBinaryBitmapFromImage(m)
	if err != nil {
		return "", err
	}
	r := qrcode.NewQRCodeReader()
	res, err := r.Decode(bmp, map[gozxing.DecodeHintType]any{gozxing.DecodeHintType_TRY_HARDER: true})
	if err != nil {
		res, err = r.Decode(bmp, map[gozxing.DecodeHintType]any{gozxing.DecodeHintType_PURE_BARCODE: true})
	}
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrNoQR, err)
	}
	return res.GetText(), nil
}

// matrix — модули QR текста text (уровень коррекции M, поле 4 модуля).
func matrix(text string, size int) (*gozxing.BitMatrix, error) {
	hints := map[gozxing.EncodeHintType]any{gozxing.EncodeHintType_ERROR_CORRECTION: "M", gozxing.EncodeHintType_MARGIN: 4}
	return qrcode.NewQRCodeWriter().Encode(text, gozxing.BarcodeFormat_QR_CODE, size, size, hints)
}

// PNG — QR текста text картинкой PNG size×size точек (печатная рамка).
func PNG(text string, size int) ([]byte, error) {
	bm, err := matrix(text, size)
	if err != nil {
		return nil, err
	}
	w, h := bm.GetWidth(), bm.GetHeight()
	img := image.NewGray(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			c := color.Gray{Y: 255}
			if bm.Get(x, y) {
				c = color.Gray{Y: 0}
			}
			img.SetGray(x, y, c)
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// SVG — QR текста text векторной картинкой для печатной рамки HTML (без
// внешних ресурсов, NFR-SEC-1): один путь из квадратов-модулей.
func SVG(text string) (string, error) {
	bm, err := matrix(text, 0)
	if err != nil {
		return "", err
	}
	w, h := bm.GetWidth(), bm.GetHeight()
	var p strings.Builder
	for y := range h {
		for x := range w {
			if bm.Get(x, y) {
				fmt.Fprintf(&p, "M%d %dh1v1h-1z", x, y)
			}
		}
	}
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" shape-rendering="crispEdges">`+
		`<rect width="%d" height="%d" fill="#fff"/><path d="%s" fill="#000"/></svg>`, w, h, w, h, p.String()), nil
}

// Writer — адаптер порта QRWriter: QR печатной рамки картинкой SVG.
type Writer struct{}

var _ app.QRWriter = Writer{}

// SVG — QR текста.
func (Writer) SVG(text string) (string, error) { return SVG(text) }
