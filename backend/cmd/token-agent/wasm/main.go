//go:build js && wasm

// Команда wasm — пакет подписи для браузера (FR-69, AD-14, Д-72): ядро
// cmd/token-agent/internal/agent, собранное в WebAssembly. Расширение
// «Главный — подпись» и режим «хранилище страницы» зовут функции объекта
// globalThis.glavnySigner; ответы — строки JSON (см. agent.Reply).
//
// ГОСТ Р 34.10-2012 и ML-DSA в WebCrypto нет, поэтому подпись считает тот же
// Go-код, что и сервер: отпечаток и сводка совпадают с серверными (AD-12).
//
// Слой: cmd (AD-1). Владелец: эпик 38.
package main

import (
	"syscall/js"

	"ant/cmd/token-agent/internal/agent"
)

func str(args []js.Value, i int) string {
	if i < len(args) && args[i].Type() == js.TypeString {
		return args[i].String()
	}
	return ""
}

func fn(f func(args []js.Value) agent.Reply) js.Func {
	return js.FuncOf(func(_ js.Value, args []js.Value) any {
		return f(args).JSON()
	})
}

func main() {
	api := map[string]any{
		"version":    fn(func([]js.Value) agent.Reply { return agent.APIVersion() }),
		"inspectKey": fn(func(a []js.Value) agent.Reply { return agent.APIInspectKey(str(a, 0)) }),
		"seal":       fn(func(a []js.Value) agent.Reply { return agent.APISeal(str(a, 0), str(a, 1)) }),
		"sealEach":   fn(func(a []js.Value) agent.Reply { return agent.APISealEach(str(a, 0), str(a, 1)) }),
		"unlock":     fn(func(a []js.Value) agent.Reply { return agent.APIUnlock(str(a, 0), str(a, 1)) }),
		"prepare":    fn(func(a []js.Value) agent.Reply { return agent.APIPrepare(str(a, 0), str(a, 1), str(a, 2)) }),
		"sign":       fn(func(a []js.Value) agent.Reply { return agent.APISign(str(a, 0)) }),
		"shiftReport": fn(func(a []js.Value) agent.Reply {
			return agent.APIShiftReport(str(a, 0), str(a, 1))
		}),
	}
	js.Global().Set("glavnySigner", js.ValueOf(api))
	if cb := js.Global().Get("__glavnySignerReady"); cb.Type() == js.TypeFunction {
		cb.Invoke()
	}
	select {}
}
