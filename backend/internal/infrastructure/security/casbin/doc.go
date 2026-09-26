// Пакет casbin — адаптер порта application/access.AccessControl (ключ
// access_control = casbin; AD-15): Casbin — единственный вычислитель решений о
// доступе; модель «роль в области + наследование + атрибуты»; политика — из
// журнала через проекцию; время — атрибут запроса (доменное время записи).
// Встроенные timeMatch, link-условия, eval и Explain запрещены (AD-15).
//
// Слой: infrastructure/security. Владелец: эпик 08 (вход и права — ядро).
package casbin
