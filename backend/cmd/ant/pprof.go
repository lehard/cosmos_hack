package main

import (
	"net/http"
	"net/http/pprof"
	"os"
	"time"
)

// Профилирование процесса для нагрузочного прогона (эпик 35, FR-107): при
// заданной переменной ANT_PPROF_ADDR (например 127.0.0.1:6060) процесс
// открывает net/http/pprof на отдельном адресе. По умолчанию выключено: в
// demo и prod адреса нет, обработчики не регистрируются на общем HTTP API.
func init() {
	addr := os.Getenv("ANT_PPROF_ADDR")
	if addr == "" {
		return
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = srv.ListenAndServe() }()
}
