package generator

import (
	"math"
	"math/rand/v2"
	"strings"
	"testing"
)

func TestPickAppDistribution(t *testing.T) {
	rng := rand.New(rand.NewPCG(42, 0))
	n := 100000
	counts := make(map[string]int)

	for i := 0; i < n; i++ {
		app := PickApp(rng)
		// Categorize by app prefix
		switch {
		case app == "com.youtube.app" || app == "com.netflix.app":
			counts["video"]++
		case app == "com.instagram.app" || app == "com.tiktok.app":
			counts["sns"]++
		case app == "com.kakao.talk" || app == "com.line.app":
			counts["messenger"]++
		case app == "com.nexon.game" || app == "com.netmarble.game":
			counts["game"]++
		case strings.Contains(app, "browser"):
			counts["browser"]++
		default:
			counts["other"]++
		}
	}

	expected := map[string]float64{
		"video":     0.30,
		"sns":       0.25,
		"messenger": 0.20,
		"game":      0.10,
		"browser":   0.10,
		"other":     0.05,
	}

	for cat, expectedRatio := range expected {
		actual := float64(counts[cat]) / float64(n)
		if math.Abs(actual-expectedRatio) > 0.02 {
			t.Errorf("%s: ratio = %.3f, want %.3f (±0.02)", cat, actual, expectedRatio)
		}
	}
}

func TestGenerateBytesUsed_VideoRange(t *testing.T) {
	rng := rand.New(rand.NewPCG(42, 0))
	for i := 0; i < 1000; i++ {
		b := GenerateBytesUsed(rng, "com.youtube.app")
		if b < 100*int64(MB) || b > 500*int64(MB) {
			t.Errorf("bytesUsed = %d, want [%d, %d]", b, 100*int64(MB), 500*int64(MB))
		}
	}
}

func TestGenerateBytesUsed_MessengerRange(t *testing.T) {
	rng := rand.New(rand.NewPCG(42, 0))
	for i := 0; i < 1000; i++ {
		b := GenerateBytesUsed(rng, "com.kakao.talk")
		if b < 1000*int64(KB) || b > 20*int64(MB) {
			t.Errorf("bytesUsed = %d, want [%d, %d]", b, 1000*int64(KB), 20*int64(MB))
		}
	}
}

func TestGenerateBytesUsed_UnknownApp(t *testing.T) {
	rng := rand.New(rand.NewPCG(42, 0))
	b := GenerateBytesUsed(rng, "com.unknown.app")
	if b != 100*int64(KB) {
		t.Errorf("unknown app bytesUsed = %d, want %d", b, 100*int64(KB))
	}
}

func TestPickNetworkTypeDistribution(t *testing.T) {
	rng := rand.New(rand.NewPCG(42, 0))
	n := 100000
	counts := make(map[string]int)

	for i := 0; i < n; i++ {
		counts[PickNetworkType(rng)]++
	}

	expected := map[string]float64{
		"4G":   0.50,
		"5G":   0.20,
		"WIFI": 0.30,
	}

	for net, expectedRatio := range expected {
		actual := float64(counts[net]) / float64(n)
		if math.Abs(actual-expectedRatio) > 0.02 {
			t.Errorf("%s: ratio = %.3f, want %.3f (±0.02)", net, actual, expectedRatio)
		}
	}
}

func TestGenerateDeviceID(t *testing.T) {
	rng := rand.New(rand.NewPCG(42, 0))
	for i := 0; i < 100; i++ {
		id := GenerateDeviceID(rng)
		if !strings.HasPrefix(id, "device_") {
			t.Errorf("device ID %q doesn't start with 'device_'", id)
		}
		found := false
		for _, model := range deviceModels {
			if id == "device_"+model {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("device ID %q is not from known device models", id)
		}
	}
}
