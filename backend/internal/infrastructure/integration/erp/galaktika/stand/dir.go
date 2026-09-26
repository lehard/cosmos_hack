package stand

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	app "ant/internal/application/ingest"
	"ant/internal/infrastructure/integration/erp/galaktika"
)

// Каталог обмена (transport exchange-dir): те же подкаталоги, что у
// адаптера-клиента — out/ (Главный → Галактика), ack/ (квитанции Галактики),
// in/ (Галактика → Главный), in-ack/ (квитанции Главного), about.xml.

func (s *Stand) prepare() error {
	for _, d := range []string{"out", "ack", "in", "in-ack"} {
		if err := os.MkdirAll(filepath.Join(s.opt.Dir, d), 0o750); err != nil {
			return err
		}
	}
	return s.writeAbout()
}

// writeAbout — about.xml по текущему описанию (меняется со сбоем corrupt).
func (s *Stand) writeAbout() error {
	b, err := galaktika.MarshalXML(s.About())
	if err != nil {
		return err
	}
	p := filepath.Join(s.opt.Dir, "about.xml")
	if old, err := os.ReadFile(p); err == nil && bytes.Equal(old, b) {
		return nil
	}
	return writeAtomic(s.opt.Dir, "about.xml", b)
}

// Scan — один обход каталога: about.xml, квитанции на новые пакеты out/ со
// сбоями stand-а, квитанции Главного из in-ack/. Недоступность (offline) —
// пакеты не обрабатываются, квитанций нет: адаптер повторит тем же номером.
func (s *Stand) Scan() {
	if s.opt.Dir == "" {
		return
	}
	if err := s.prepare(); err != nil {
		s.opt.Log.Warn("stand Галактики: каталог обмена не подготовлен", "err", err)
		return
	}
	s.readInAcks()
	if _, ok := s.faults.Get(app.FaultOffline); ok {
		return
	}
	ents, err := os.ReadDir(filepath.Join(s.opt.Dir, "out"))
	if err != nil {
		s.opt.Log.Warn("stand Галактики: out/ не прочитан", "err", err)
		return
	}
	now := s.opt.Now()
	for _, en := range ents {
		name := en.Name()
		if en.IsDir() || strings.HasPrefix(name, ".") || !strings.HasSuffix(name, ".xml") {
			continue
		}
		if _, err := os.Stat(filepath.Join(s.opt.Dir, "ack", name)); err == nil {
			continue
		}
		if x, ok := s.faults.Get(app.FaultDelay); ok && x.Param > 0 {
			if fi, err := en.Info(); err == nil && now.Sub(fi.ModTime()) < time.Duration(x.Param)*time.Millisecond {
				continue
			}
		}
		s.processOut(name)
	}
}

// processOut — квитанция на пакет out/‹name›.
func (s *Stand) processOut(name string) {
	b, err := os.ReadFile(filepath.Join(s.opt.Dir, "out", name))
	if err != nil {
		return
	}
	id := strings.TrimSuffix(name, ".xml")
	e, err := galaktika.ParseExchangeXML(b)
	if err == nil {
		id = or(e.MessageID, id)
		err = galaktika.Check(e)
	}
	if err == nil && e.Contract != galaktika.ContractVersion {
		err = fmt.Errorf("версия контракта %q не поддерживается обработчиком", e.Contract)
		s.writeAck(name, galaktika.Ack{MessageID: id, Status: "error", Code: "CONTRACT_VERSION", Text: err.Error()})
		return
	}
	if err != nil {
		s.writeAck(name, galaktika.Ack{MessageID: id, Status: "error", Code: "SCHEMA_VIOLATION", Text: trim("пакет не по gal.qc.v1: "+err.Error(), 1000)})
		return
	}
	if x, ok := s.faults.Get(app.FaultError); ok && (x.Match == "" || Match(x.Match, e)) {
		code, ack := faultAck(x, e.MessageID)
		if code >= 500 {
			// Сервер приложений недоступен: квитанции нет — транспорт, повтор.
			s.mu.Lock()
			first := !s.seen[name]
			s.seen[name] = true
			s.mu.Unlock()
			if first {
				s.record(e, galaktika.TransportDir, fmt.Sprintf("%d %s (квитанции нет)", code, ack.Text))
			}
			return
		}
		s.record(e, galaktika.TransportDir, fmt.Sprintf("%d %s", code, ack.Text))
		s.writeAck(name, ack)
		return
	}
	ack, _ := s.Apply(e, galaktika.TransportDir)
	s.writeAck(name, ack)
}

func (s *Stand) writeAck(name string, a galaktika.Ack) {
	b, err := galaktika.MarshalXML(a)
	if err == nil {
		err = writeAtomic(filepath.Join(s.opt.Dir, "ack"), name, b)
	}
	if err != nil {
		s.opt.Log.Error("stand Галактики: квитанция не записана", "packet", name, "err", err)
	}
}

// readInAcks — квитанции Главного на пакеты Галактики (in-ack/).
func (s *Stand) readInAcks() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.packets {
		p := &s.packets[i]
		if p.AckStatus != "" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(s.opt.Dir, "in-ack", safeName(p.MessageID)+".xml"))
		if err != nil {
			continue
		}
		var a galaktika.Ack
		if xml.Unmarshal(b, &a) == nil {
			p.AckStatus, p.AckText = a.Status, a.Code+" "+a.Text
		}
	}
}

// Task — Галактика выдала сменное задание (кнопка страницы): пакет
// ProductionTask для Главного — в in/ каталога и в outbox фасада.
func (s *Stand) Task(quantity int, due string) (Packet, error) {
	if quantity < 1 || quantity > 1000 {
		return Packet{}, errors.New("количество — от 1 до 1000")
	}
	if _, err := time.Parse("2006-01-02", due); err != nil {
		return Packet{}, errors.New("срок — дата ГГГГ-ММ-ДД")
	}
	return s.emit(func(n int, now time.Time) (string, galaktika.Exchange) {
		num := fmt.Sprintf("СЗ-%06d", 900+n)
		return "Сменное задание " + num + " на " + fmt.Sprint(quantity) + " шт.", galaktika.Exchange{ProductionTask: &galaktika.ProductionTask{
			NRec: nrec(4611686018427399000, n), Table: "MnPlan", Number: num, Date: now.Format("2006-01-02"), DueDate: due,
			Quantity: quantity, RouteSheet: fmt.Sprintf("МЛ-%06d", 900+n), KdRevision: "Б",
			Item: galaktika.Item{Designation: "ФЛ-100.00.000 СБ", Name: "Фланец люка гермокорпуса в сборе", NRec: "4611686018427388123", Table: "KatMC"}}}
	})
}

// Lot — поступила партия (кнопка страницы): пакет LotReceived для Главного.
func (s *Stand) Lot(number string, quantity int) (Packet, error) {
	number = strings.TrimSpace(number)
	if number == "" || len(number) > 64 {
		return Packet{}, errors.New("номер партии — от 1 до 64 знаков")
	}
	if quantity < 1 || quantity > 100000 {
		return Packet{}, errors.New("количество — от 1 до 100000")
	}
	return s.emit(func(n int, now time.Time) (string, galaktika.Exchange) {
		inv := fmt.Sprintf("ПН-%06d", 500+n)
		return "Приходная накладная " + inv + ": партия " + number + ", " + fmt.Sprint(quantity) + " шт.", galaktika.Exchange{LotReceived: &galaktika.LotReceived{
			NRec: nrec(4611686018427402000, n), Table: "KatSopr", Number: inv, Date: now.Format("2006-01-02"),
			Supplier: galaktika.Supplier{NRec: "4611686018427377001", Code: "ПОСТ-003", Name: "АО «Уралкольцо»"},
			Item:     galaktika.Item{Designation: "ФЛ-100.01.002", Name: "Патрубок (кольцо)", NRec: "4611686018427388130", Table: "KatMC"},
			Lot:      galaktika.Lot{Number: number, Quantity: quantity}}}
	})
}

// emit — пакет Галактики: номер, проверка схемой (FR-111 — и на стороне
// stand-а), в outbox фасада и в in/ каталога обмена.
func (s *Stand) emit(build func(n int, now time.Time) (string, galaktika.Exchange)) (Packet, error) {
	s.mu.Lock()
	n := len(s.packets) + 1
	now := s.opt.Now().UTC()
	title, e := build(n, now.In(time.FixedZone("MSK", 3*3600)))
	e.MessageID = fmt.Sprintf("GAL-STAND-%s-%04d", now.Format("20060102T150405"), n)
	e.CreatedAt, e.From, e.To, e.Contract = now.Format("2006-01-02T15:04:05.000Z"), Node, "ant", galaktika.ContractVersion
	if err := galaktika.Check(e); err != nil {
		s.mu.Unlock()
		return Packet{}, fmt.Errorf("пакет stand-а не по gal.qc.v1: %w", err)
	}
	p := Packet{MessageID: e.MessageID, Title: title, Exchange: e, CreatedAt: now}
	s.packets = append(s.packets, p)
	s.mu.Unlock()
	if s.opt.Dir != "" {
		b, err := galaktika.MarshalXML(e)
		if err == nil {
			err = writeAtomic(filepath.Join(s.opt.Dir, "in"), safeName(e.MessageID)+".xml", b)
		}
		if err != nil {
			return p, fmt.Errorf("пакет не записан в каталог обмена: %w", err)
		}
	}
	return p, nil
}

func nrec(base int64, n int) string { return fmt.Sprint(base + int64(n)) }

// safeName — имя файла пакета по номеру (то же правило, что у адаптера).
func safeName(id string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.' {
			return r
		}
		return '_'
	}, id)
}

// writeAtomic — файл целиком или никак (временный файл и переименование).
func writeAtomic(dir, name string, b []byte) error {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+name+".*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), filepath.Join(dir, name))
}
