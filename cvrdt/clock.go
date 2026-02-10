package cvrdt

const MaxValue = 1<<31 - 1
const MinValue = 0

// VectorClock tracks logical time across multiple sites
// Each site maintains a vector with entries for all known sites
type VectorClock struct {
	localId string
	clocks  map[string]int // siteId -> clock value
}

func NewVectorClock(siteId string) VectorClock {
	clocks := make(map[string]int)
	clocks[siteId] = MinValue
	return VectorClock{
		localId: siteId,
		clocks:  clocks,
	}
}

func (v *VectorClock) Increment() int {
	if v.clocks[v.localId] < MaxValue {
		v.clocks[v.localId]++
	}
	return v.clocks[v.localId]
}

func (v *VectorClock) Value() int {
	return v.clocks[v.localId]
}

// Update merges a received vector clock into this one
// For each site in the received vector, takes the maximum clock value
// Note: This does NOT increment the local clock - use Increment() for that
func (v *VectorClock) Update(receivedVector map[string]int) {
	// Merge received vector (take max of each entry)
	for site, clock := range receivedVector {
		if currentClock, exists := v.clocks[site]; exists {
			if clock > currentClock {
				v.clocks[site] = clock
			}
		} else {
			// First time seeing this site - add it
			v.clocks[site] = clock
		}
	}
}

// Copy creates a snapshot of the current vector clock
// This is used when creating operations to capture the causal context
func (v *VectorClock) Copy() map[string]int {
	copy := make(map[string]int)
	for site, clock := range v.clocks {
		copy[site] = clock
	}
	return copy
}

// Get returns the clock value for a specific site (0 if not present)
func (v *VectorClock) Get(siteId string) int {
	return v.clocks[siteId]
}
