package process

import (
	"maps"
	"slices"
)

// Служебные функции детерминированного обхода (AD-4: порядок обхода map —
// только через отсортированные ключи).

func contains[T comparable](xs []T, x T) bool { return slices.Contains(xs, x) }

func sortedKeys[V any](m map[string]V) []string { return slices.Sorted(maps.Keys(m)) }

// sortStableBy — устойчивая сортировка по «меньше».
func sortStableBy[T any](xs []T, less func(a, b T) bool) {
	slices.SortStableFunc(xs, func(a, b T) int {
		switch {
		case less(a, b):
			return -1
		case less(b, a):
			return 1
		}
		return 0
	})
}
