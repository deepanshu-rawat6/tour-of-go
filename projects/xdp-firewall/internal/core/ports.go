package core

// BPFMapPort is the outbound port for writing rules into kernel eBPF maps.
type BPFMapPort interface {
	Insert(cidr string) error
	Delete(cidr string) error
	List() ([]string, error)
}

// RuleStorePort is the outbound port for persisting rules to disk.
type RuleStorePort interface {
	Save(rules []string) error
	Load() ([]string, error)
}

// CounterReader is the outbound port for reading kernel packet counters.
type CounterReader interface {
	ReadCounters() (Counters, error)
}

// Counters holds aggregated per-CPU packet statistics.
type Counters struct {
	PacketsTotal uint64
	DropsTotal   uint64
}
