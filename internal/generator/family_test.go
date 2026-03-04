package generator

import (
	"math"
	"math/rand/v2"
	"testing"
)

func TestNewFamilyRegistry(t *testing.T) {
	rng := rand.New(rand.NewPCG(42, 0))
	reg := NewFamilyRegistry(1000, rng)

	if reg.Count() != 1000 {
		t.Errorf("Count() = %d, want 1000", reg.Count())
	}

	// All families should have 2-10 members
	for i := 0; i < reg.Count(); i++ {
		f := &reg.families[i]
		if len(f.Members) < 2 || len(f.Members) > 10 {
			t.Errorf("family %d has %d members, want 2-10", f.ID, len(f.Members))
		}
		if f.ID != int64(i+1) {
			t.Errorf("family ID = %d, want %d", f.ID, i+1)
		}
	}
}

func TestFamilyDistribution(t *testing.T) {
	rng := rand.New(rand.NewPCG(42, 0))
	count := 100000
	reg := NewFamilyRegistry(count, rng)

	sizeCounts := make(map[int]int)
	for i := range reg.families {
		sizeCounts[len(reg.families[i].Members)]++
	}

	expected := map[int]float64{
		2: 0.15,
		3: 0.25,
		4: 0.40,
		5: 0.15,
	}

	for size, expectedRatio := range expected {
		actual := float64(sizeCounts[size]) / float64(count)
		if math.Abs(actual-expectedRatio) > 0.02 {
			t.Errorf("size %d: ratio = %.3f, want %.3f (±0.02)", size, actual, expectedRatio)
		}
	}

	// 6-10 person families should be ~5%
	largeCount := 0
	for size := 6; size <= 10; size++ {
		largeCount += sizeCounts[size]
	}
	largeRatio := float64(largeCount) / float64(count)
	if math.Abs(largeRatio-0.05) > 0.02 {
		t.Errorf("size 6-10: ratio = %.3f, want 0.05 (±0.02)", largeRatio)
	}
}

func TestRandomFamily(t *testing.T) {
	rng := rand.New(rand.NewPCG(42, 0))
	reg := NewFamilyRegistry(100, rng)

	f := reg.RandomFamily()
	if f == nil {
		t.Fatal("RandomFamily() returned nil")
	}
	if f.ID < 1 || f.ID > 100 {
		t.Errorf("family ID %d out of range [1, 100]", f.ID)
	}
}

func TestRandomMember(t *testing.T) {
	rng := rand.New(rand.NewPCG(42, 0))
	reg := NewFamilyRegistry(10, rng)

	f := &reg.families[0]
	for i := 0; i < 100; i++ {
		m := reg.RandomMember(f)
		found := false
		for _, id := range f.Members {
			if id == m {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("RandomMember() returned %d, not in family members %v", m, f.Members)
		}
	}
}

func TestCustomerIDsUnique(t *testing.T) {
	rng := rand.New(rand.NewPCG(42, 0))
	reg := NewFamilyRegistry(100, rng)

	seen := make(map[int64]bool)
	for i := range reg.families {
		for _, m := range reg.families[i].Members {
			if seen[m] {
				t.Errorf("duplicate customerID %d", m)
			}
			seen[m] = true
		}
	}
}
