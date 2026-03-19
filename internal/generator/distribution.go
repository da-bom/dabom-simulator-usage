package generator

import (
	"math/rand/v2"
)

const (
	KB = 1024
	MB = 1024 * KB
)

// AppCategory defines an app category with its selection weight and byte range.
type AppCategory struct {
	Name     string
	Apps     []string
	Weight   int
	MinBytes int64
	MaxBytes int64
}

var appCategories = []AppCategory{
	{
		Name:     "video",
		Apps:     []string{"com.youtube.app", "com.netflix.app"},
		Weight:   30,
		MinBytes: 100 * int64(MB),
		MaxBytes: 500 * int64(MB),
	},
	{
		Name:     "sns",
		Apps:     []string{"com.instagram.app", "com.tiktok.app"},
		Weight:   25,
		MinBytes: 20 * int64(MB),
		MaxBytes: 150 * int64(MB),
	},
	{
		Name:     "messenger",
		Apps:     []string{"com.kakao.talk", "com.line.app"},
		Weight:   20,
		MinBytes: 1000 * int64(KB),
		MaxBytes: 20 * int64(MB),
	},
	{
		Name:     "game",
		Apps:     []string{"com.nexon.game", "com.netmarble.game"},
		Weight:   10,
		MinBytes: 10 * int64(MB),
		MaxBytes: 100 * int64(MB),
	},
	{
		Name:     "browser",
		Apps:     []string{"com.chrome.browser", "com.samsung.browser"},
		Weight:   10,
		MinBytes: 5000 * int64(KB),
		MaxBytes: 50 * int64(MB),
	},
	{
		Name:     "other",
		Apps:     []string{"com.naver.map", "com.weather.app"},
		Weight:   5,
		MinBytes: 500 * int64(KB),
		MaxBytes: 10 * int64(MB),
	},
}

// appBytesRange maps appId to its byte range for lookup.
var appBytesRange map[string][2]int64

func init() {
	appBytesRange = make(map[string][2]int64)
	for _, cat := range appCategories {
		for _, app := range cat.Apps {
			appBytesRange[app] = [2]int64{cat.MinBytes, cat.MaxBytes}
		}
	}
}

// PickApp selects a random app based on weighted distribution.
func PickApp(rng *rand.Rand) string {
	r := rng.IntN(100)
	cumulative := 0
	for _, cat := range appCategories {
		cumulative += cat.Weight
		if r < cumulative {
			return cat.Apps[rng.IntN(len(cat.Apps))]
		}
	}
	return appCategories[0].Apps[0]
}

// GenerateBytesUsed returns a random byte count within the range for the given app.
func GenerateBytesUsed(rng *rand.Rand, appID string) int64 {
	br, ok := appBytesRange[appID]
	if !ok {
		return 100 * int64(KB) // fallback
	}
	return br[0] + rng.Int64N(br[1]-br[0]+1)
}

// networkTypes and their weights: 4G=50%, 5G=20%, WIFI=30%
var networkDist = []struct {
	name   string
	weight int
}{
	{"4G", 50},
	{"5G", 20},
	{"WIFI", 30},
}

// PickNetworkType selects a network type based on weighted distribution.
func PickNetworkType(rng *rand.Rand) string {
	r := rng.IntN(100)
	cumulative := 0
	for _, n := range networkDist {
		cumulative += n.weight
		if r < cumulative {
			return n.name
		}
	}
	return "4G"
}

// deviceModels lists realistic device model names.
var deviceModels = []string{
	"pixel_9",
	"pixel_8",
	"pixel_7",
	"galaxy_s24",
	"galaxy_s23",
	"galaxy_a54",
	"galaxy_z_flip5",
	"iphone_15",
	"iphone_14",
	"iphone_se",
	"xperia_1_v",
	"v60_thinq",
}

// GenerateDeviceID creates a device identifier using a real device model name.
func GenerateDeviceID(rng *rand.Rand) string {
	return "device_" + deviceModels[rng.IntN(len(deviceModels))]
}
