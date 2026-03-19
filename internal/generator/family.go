package generator

import (
	"fmt"
	"math/rand/v2"
	"os"
	"time"
)

// FamilyRegistry holds pre-generated families and provides random selection.
type FamilyRegistry struct {
	families []Family
	rng      *rand.Rand
}

// familySizeDist defines the distribution of family sizes.
// 2-person: 15%, 3-person: 25%, 4-person: 40%, 5-person: 15%, 6-10: 5%
var familySizeDist = []struct {
	size   int
	weight int
}{
	{2, 15},
	{3, 25},
	{4, 40},
	{5, 15},
}
// 6-10 person families get the remaining 5%

// NewFamilyRegistryFromFamilies creates a registry from pre-built families (e.g. loaded from DB).
func NewFamilyRegistryFromFamilies(families []Family, rng *rand.Rand) *FamilyRegistry {
	return &FamilyRegistry{
		families: families,
		rng:      rng,
	}
}

// NewFamilyRegistry creates a registry with the given number of families.
func NewFamilyRegistry(count int, rng *rand.Rand) *FamilyRegistry {
	families := make([]Family, count)
	customerID := int64(1)

	for i := 0; i < count; i++ {
		size := pickFamilySize(rng)
		members := make([]int64, size)
		for j := 0; j < size; j++ {
			members[j] = customerID
			customerID++
		}
		families[i] = Family{
			ID:      int64(i + 1),
			Members: members,
		}
	}

	return &FamilyRegistry{
		families: families,
		rng:      rng,
	}
}

func pickFamilySize(rng *rand.Rand) int {
	r := rng.IntN(100)
	cumulative := 0
	for _, d := range familySizeDist {
		cumulative += d.weight
		if r < cumulative {
			return d.size
		}
	}
	// Remaining 5%: 6-10 person families
	return 6 + rng.IntN(5) // 6..10
}

// RandomFamily returns a random family from the registry.
func (fr *FamilyRegistry) RandomFamily() *Family {
	idx := fr.rng.IntN(len(fr.families))
	return &fr.families[idx]
}

// RandomMember returns a random member from the given family.
func (fr *FamilyRegistry) RandomMember(f *Family) int64 {
	idx := fr.rng.IntN(len(f.Members))
	return f.Members[idx]
}

// DumpToFile writes all family-customer mappings to a log file for verification.
func (fr *FamilyRegistry) DumpToFile(path string, source string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create dump file: %w", err)
	}
	defer f.Close()

	fmt.Fprintf(f, "# Family Registry Dump\n")
	fmt.Fprintf(f, "# Generated at: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(f, "# Source: %s\n", source)
	fmt.Fprintf(f, "# Total families: %d\n", len(fr.families))
	fmt.Fprintf(f, "# Total members: %d\n", fr.TotalMembers())
	fmt.Fprintf(f, "#\n")
	fmt.Fprintf(f, "# format: familyId -> [customerId, ...]\n\n")

	for _, fam := range fr.families {
		fmt.Fprintf(f, "familyId=%d -> members=%v\n", fam.ID, fam.Members)
	}

	return nil
}

// Count returns the number of families in the registry.
func (fr *FamilyRegistry) Count() int {
	return len(fr.families)
}

// TotalMembers returns the total number of members across all families.
func (fr *FamilyRegistry) TotalMembers() int {
	total := 0
	for i := range fr.families {
		total += len(fr.families[i].Members)
	}
	return total
}
