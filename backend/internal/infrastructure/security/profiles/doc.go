// Пакет profiles — криптопрофили gost, pq, hybrid за портами
// application/signing.Signer / Verifier / Cipher (AD-10, AD-32, AD-35): GoGOST
// (ГОСТ Р 34.10-2012, Стрибог-256, «Кузнечик»-MGM) и crypto/mldsa. Криптография
// MVP — не СКЗИ; в промышленной эксплуатации — сертифицированное СКЗИ за теми же
// портами.
//
// Слой: infrastructure/security. Владелец: эпик 05 (криптоядро), 27 (подписи).
package profiles
