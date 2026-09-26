package atrest

import (
	"bytes"
	"path/filepath"
	"testing"
)

// AD-23: DEK оборачивается KEK, блок открывается только с теми же
// дополнительными данными; чужой KEK обёртку не открывает.
func TestEnvelopeScheme(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kek")
	if made, err := Generate(path); err != nil || !made {
		t.Fatal(made, err)
	}
	if made, _ := Generate(path); made {
		t.Fatal("KEK перезаписан")
	}
	k, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	id, dek, wrapped, err := k.NewDEK()
	if err != nil {
		t.Fatal(err)
	}
	got, err := k.Unwrap(id, wrapped)
	if err != nil || !bytes.Equal(got, dek) {
		t.Fatal("обёртка не открылась", err)
	}
	if _, err := k.Unwrap("dek-other", wrapped); err == nil {
		t.Fatal("обёртка подошла к другому DEK")
	}
	n, ct, err := k.Seal(dek, []byte("блок"), []byte("main|1"))
	if err != nil {
		t.Fatal(err)
	}
	if p, err := k.Open(dek, n, ct, []byte("main|1")); err != nil || string(p) != "блок" {
		t.Fatal("блок не открылся", err)
	}
	if _, err := k.Open(dek, n, ct, []byte("main|2")); err == nil {
		t.Fatal("блок открылся с чужими дополнительными данными")
	}
	other, _ := New(bytes.Repeat([]byte{7}, KeySize))
	if _, err := other.Unwrap(id, wrapped); err == nil {
		t.Fatal("чужой KEK открыл обёртку")
	}
}
