package kernel

import "hash/fnv"

// PartitionOf — партиция изделия: hash(item_id) mod P (AD-6, AD-41); hash —
// FNV-1a 32 бита над байтами внутреннего ID. Одна функция для приёма (выбор
// партиции факта), движка и стадии (адресованные записи изделию) и
// верификатора: иначе события одной детали разошлись бы по партициям.
func PartitionOf(itemID string, partitions int) int {
	if partitions <= 1 {
		return 0
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(itemID))
	return int(h.Sum32() % uint32(partitions))
}
