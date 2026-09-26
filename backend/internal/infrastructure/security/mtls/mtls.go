// Пакет mtls — взаимный TLS между ant, хранителем и верификатором (AD-8,
// AD-23): TLS 1.3 (в Go по умолчанию — гибридный обмен X25519MLKEM768),
// сертификаты от своего демо-УЦ; клиент без сертификата этого УЦ хранителю
// ничего не передаст. В промышленной эксплуатации — ГОСТ-TLS через СКЗИ
// (в crypto/x509 нет ГОСТ; X.509 с ГОСТ и stand УЦ — описание, AD-11).
//
// Слой: infrastructure/security — технический механизм (AD-1).
// Владелец: эпик 29 (доверие). До эпика 05 (`ant init`) сертификаты
// выпускает `keeper -init` в том pki.
package mtls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
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

// Files — файлы участника в каталоге pki: ‹name›.crt, ‹name›.key и ca.crt.
type Files struct {
	Dir  string
	Name string
}

func (f Files) cert() string { return filepath.Join(f.Dir, f.Name+".crt") }
func (f Files) key() string  { return filepath.Join(f.Dir, f.Name+".key") }
func (f Files) ca() string   { return filepath.Join(f.Dir, "ca.crt") }

// Init выпускает демо-УЦ и сертификаты участников в dir (если их нет):
// сервер — keeper (имена hosts), клиенты — clients. Закрытый ключ УЦ
// остаётся только в dir (том хранителя).
func Init(dir string, hosts []string, clients []string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	caCert := filepath.Join(dir, "ca.crt")
	caKey := filepath.Join(dir, "ca.key")
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
		if err := writePEM(caCert, "CERTIFICATE", der, 0o444); err != nil {
			return err
		}
		kb, _ := x509.MarshalECPrivateKey(k)
		if err := writePEM(caKey, "EC PRIVATE KEY", kb, 0o400); err != nil {
			return err
		}
	}
	ca, caPrv, err := loadCA(caCert, caKey)
	if err != nil {
		return err
	}
	if err := issue(dir, "keeper", hosts, true, ca, caPrv); err != nil {
		return err
	}
	for _, c := range clients {
		if err := issue(dir, c, nil, false, ca, caPrv); err != nil {
			return err
		}
	}
	return nil
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
		return nil, nil, errors.New("mtls: УЦ не в PEM")
	}
	c, err := x509.ParseCertificate(cp.Bytes)
	if err != nil {
		return nil, nil, err
	}
	k, err := x509.ParseECPrivateKey(kp.Bytes)
	return c, k, err
}

func issue(dir, name string, hosts []string, server bool, ca *x509.Certificate, caPrv *ecdsa.PrivateKey) error {
	f := Files{Dir: dir, Name: name}
	if _, err := os.Stat(f.cert()); err == nil {
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
	if err := writePEM(f.key(), "EC PRIVATE KEY", kb, 0o400); err != nil {
		return err
	}
	return writePEM(f.cert(), "CERTIFICATE", der, 0o444)
}

func pool(caPath string) (*x509.CertPool, error) {
	b, err := os.ReadFile(caPath)
	if err != nil {
		return nil, err
	}
	p := x509.NewCertPool()
	if !p.AppendCertsFromPEM(b) {
		return nil, fmt.Errorf("mtls: %s — не PEM", caPath)
	}
	return p, nil
}

// Server — tls.Config хранителя: TLS 1.3, клиентский сертификат своего УЦ обязателен.
func Server(f Files) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(f.cert(), f.key())
	if err != nil {
		return nil, err
	}
	p, err := pool(f.ca())
	if err != nil {
		return nil, err
	}
	return &tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{cert}, ClientCAs: p,
		ClientAuth: tls.RequireAndVerifyClientCert}, nil
}

// Client — tls.Config участника (ant, verifier): TLS 1.3, сервер — сертификат своего УЦ.
func Client(f Files) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(f.cert(), f.key())
	if err != nil {
		return nil, err
	}
	p, err := pool(f.ca())
	if err != nil {
		return nil, err
	}
	return &tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{cert}, RootCAs: p}, nil
}

// ClientName — имя участника из проверенного клиентского сертификата.
func ClientName(cs *tls.ConnectionState) string {
	if cs == nil || len(cs.PeerCertificates) == 0 {
		return ""
	}
	return cs.PeerCertificates[0].Subject.CommonName
}
