package simulation

// Rand — детерминированный генератор псевдослучайных чисел SplitMix64 (AD-4:
// в домене нет math/rand и crypto/rand). Один и тот же seed и имя потока дают
// одну и ту же последовательность на любой машине и в любом процессе — так
// прогон, заготовки и нагрузка строятся «тем же генератором и seed» (AD-36, AD-38).
type Rand struct {
	state uint64
}

// NewRand — генератор для потока stream при seed: у каждого назначения
// (шум, фон, дубли) свой поток, поэтому новое назначение не сдвигает старые.
func NewRand(seed int64, stream string) *Rand {
	return &Rand{state: uint64(seed) ^ fnv64a(stream) ^ 0x9E3779B97F4A7C15}
}

// Uint64 — следующее число.
func (r *Rand) Uint64() uint64 {
	r.state += 0x9E3779B97F4A7C15
	z := r.state
	z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
	z = (z ^ (z >> 27)) * 0x94D049BB133111EB
	return z ^ (z >> 31)
}

// Intn — число в [0, n); n ≤ 0 — 0.
func (r *Rand) Intn(n int) int {
	if n <= 0 {
		return 0
	}
	return int(r.Uint64() % uint64(n))
}

// Between — число в [lo, hi].
func (r *Rand) Between(lo, hi int) int {
	if hi <= lo {
		return lo
	}
	return lo + r.Intn(hi-lo+1)
}

// ChanceBP — событие с вероятностью bp/10000.
func (r *Rand) ChanceBP(bp int) bool {
	if bp <= 0 {
		return false
	}
	return r.Intn(10000) < bp
}

// Pick — k различных индексов из [0, n) по возрастанию (частичная перетасовка Фишера — Йетса).
func (r *Rand) Pick(n, k int) []int {
	if k > n {
		k = n
	}
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	for i := 0; i < k; i++ {
		j := i + r.Intn(n-i)
		idx[i], idx[j] = idx[j], idx[i]
	}
	out := idx[:k]
	sortInts(out)
	return out
}

func sortInts(a []int) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j] < a[j-1]; j-- {
			a[j], a[j-1] = a[j-1], a[j]
		}
	}
}

// fnv64a — FNV-1a 64 строки.
func fnv64a(s string) uint64 {
	h := uint64(14695981039346656037)
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= 1099511628211
	}
	return h
}
