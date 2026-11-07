package desktop

import (
	"github.com/shirou/gopsutil/v3/mem"
	"strconv"
)

func sceneBudgetFor(available uint64, bits int) uint64 {
	budget := uint64(512 << 20)
	if available != 0 {
		budget = max(128<<20, min(1<<30, available/8))
	}
	if bits == 32 {
		budget = min(budget, 256<<20)
	}
	return budget
}
func availableSceneBudget() uint64 {
	available := uint64(0)
	if memory, err := mem.VirtualMemory(); err == nil {
		available = memory.Available
	}
	return sceneBudgetFor(available, strconv.IntSize)
}
