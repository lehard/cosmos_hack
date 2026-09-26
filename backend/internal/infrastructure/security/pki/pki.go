// Пакет pki — выпуск сертификатов mTLS при установке (ant init, AD-33, AD-8):
// демо-УЦ в томе хранителя, серверный сертификат хранителя и клиентские —
// участников (ant, verifier), каждому — в его том. TLS 1.3 (в Go — гибридный
// обмен X25519MLKEM768); в промышленной эксплуатации — ГОСТ-TLS через СКЗИ
// (в crypto/x509 нет ГОСТ, AD-11).
//
// Раскладка файлов совпадает с `keeper -init` эпика 29 (пакет
// infrastructure/security/mtls): ‹keeper›/pki/{ca.crt, ca.key, keeper.crt,
// keeper.key, ‹участник›.crt}, ‹том участника›/pki/{ca.crt, ‹участник›.crt,
// ‹участник›.key}. Существующие файлы не перезаписываются, поэтому порядок
// «ant init / keeper -init» не важен; закрытый ключ участника в томе
// хранителя не остаётся.
//
// Слой: infrastructure/security — технический механизм (AD-1). Владелец: эпик 05.
package pki

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

// Server — имя серверного сертификата (хранитель).
const Server = "keeper"

// Participant — клиент mTLS и каталог его тома (сертификат ляжет в ‹Dir›/pki).
type Participant struct {
	Name string
	Dir  string
}

// Init выпускает демо-УЦ и сертификаты в keeperDir/pki (если их нет) и
// раскладывает клиентские сертификаты участникам.
func Init(keeperDir string, hosts []string, participants []Participant) error {
	dir := filepath.Join(keeperDir, "pki")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	caCert, caKey := filepath.Join(dir, "ca.crt"), filepath.Join(dir, "ca.key")
	if _, err := os.Stat(caCert); errors.Is(err, os.ErrNotExist) {
		k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return err
		}
		tpl := &x509.Certificate{SerialNumber: serial(), Subject: pkix.Name{CommonName: "ant demo CA (keeper, verifier)"},
			NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().AddDate(5, 0, 0), IsCA: true, BasicConstraintsValid: true,
			KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign}
		der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &k.PublicKey, k)
		if err != nil {
			return err
		}
		kb, _ := x509.MarshalECPrivateKey(k)
		if err := writePEM(caKey, "EC PRIVATE KEY", kb, 0o400); err != nil {
			return err
		}
		if err := writePEM(caCert, "CERTIFICATE", der, 0o444); err != nil {
			return err
		}
	}
	ca, caPrv, err := loadCA(caCert, caKey)
	if err != nil {
		return err
	}
	if err := issue(dir, Server, hosts, true, ca, caPrv); err != nil {
		return err
	}
	for _, p := range participants {
		if err := distribute(dir, p, ca, caPrv); err != nil {
			return fmt.Errorf("mTLS %s: %w", p.Name, err)
		}
	}
	return nil
}

// distribute — сертификат участника в его том: ca.crt, ‹имя›.crt, ‹имя›.key.
// Выпуск — только если у участника ещё нет ключа (повтор ничего не меняет).
func distribute(keeperPKI string, p Participant, ca *x509.Certificate, caPrv *ecdsa.PrivateKey) error {
	dst := filepath.Join(p.Dir, "pki")
	if _, err := os.Stat(filepath.Join(dst, p.Name+".key")); err == nil {
		return nil
	}
	if err := os.MkdirAll(dst, 0o700); err != nil {
		return err
	}
	// Ключа участника нет — выпуск заново (сертификат без своего ключа бесполезен).
	_ = os.Remove(filepath.Join(keeperPKI, p.Name+".crt"))
	if err := issue(keeperPKI, p.Name, nil, false, ca, caPrv); err != nil {
		return err
	}
	for _, f := range []string{"ca.crt", p.Name + ".crt", p.Name + ".key"} {
		to := filepath.Join(dst, f)
		_ = os.Remove(to)
		b, err := os.ReadFile(filepath.Join(keeperPKI, f))
		if err != nil {
			return err
		}
		mode := os.FileMode(0o444)
		if filepath.Ext(f) == ".key" {
			mode = 0o400
		}
		if err := os.WriteFile(to, b, mode); err != nil {
			return err
		}
	}
	// Закрытый ключ участника у хранителя не остаётся.
	return os.Remove(filepath.Join(keeperPKI, p.Name+".key"))
}

func serial() *big.Int {
	n, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 62))
	return n
}

func writePEM(path, typ string, der []byte, mode os.FileMode) error {
	return os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: typ, Bytes: der}), mode)
}

func loadCA(certPath, keyPath string) (*x509.Certificate, *ecdsa.PrivateKey, error) {
	cb, err := os.ReadFile(certPath)
	if err != nil {
		return nil, nil, err
	}
	kb, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, nil, err
	}
	cp, _ := pem.Decode(cb)
	kp, _ := pem.Decode(kb)
	if cp == nil || kp == nil {
		return nil, nil, errors.New("pki: УЦ не в PEM")
	}
	c, err := x509.ParseCertificate(cp.Bytes)
	if err != nil {
		return nil, nil, err
	}
	k, err := x509.ParseECPrivateKey(kp.Bytes)
	return c, k, err
}

// issue — сертификат name, если его ещё нет.
func issue(dir, name string, hosts []string, server bool, ca *x509.Certificate, caPrv *ecdsa.PrivateKey) error {
	cert := filepath.Join(dir, name+".crt")
	if _, err := os.Stat(cert); err == nil {
		return nil
	}
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	tpl := &x509.Certificate{SerialNumber: serial(), Subject: pkix.Name{CommonName: name},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().AddDate(2, 0, 0), KeyUsage: x509.KeyUsageDigitalSignature}
	if server {
		tpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
		tpl.DNSNames = hosts
	} else {
		tpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, ca, &k.PublicKey, caPrv)
	if err != nil {
		return err
	}
	kb, _ := x509.MarshalECPrivateKey(k)
	if err := writePEM(filepath.Join(dir, name+".key"), "EC PRIVATE KEY", kb, 0o400); err != nil {
		return err
	}
	return writePEM(cert, "CERTIFICATE", der, 0o444)
}
