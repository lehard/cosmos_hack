package main

import (
	"context"
	"fmt"
)

// pendingRole — роль процесса, зарегистрированная заранее (волна 1, AD-25):
// долгоживущая ждёт остановки, разовая завершается ошибкой «не реализовано».
// Эпик-владелец заменяет тело роли, не меняя реестр.
func pendingRole(name, epic string, oneShot bool) role {
	return role{oneShot: oneShot, pending: true, run: func(ctx context.Context, env *environment) error {
		if oneShot {
			return fmt.Errorf("роль %s ещё не реализована (%s)", name, epic)
		}
		env.log.Warn("роль ещё не реализована — ожидаю остановки", "role", name, "epic", epic)
		<-ctx.Done()
		return nil
	}}
}
