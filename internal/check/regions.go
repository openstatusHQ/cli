package check

// regionDisplayNames is a snapshot of:
// https://github.com/openstatusHQ/skills/blob/main/skills/global-speed-checker/references/regions-detailed.md
// Update by hand when the upstream skill repo gains a region.
var regionDisplayNames = map[string]string{
	"ams":                            "Amsterdam (Fly)",
	"arn":                            "Stockholm (Fly)",
	"bom":                            "Mumbai (Fly)",
	"cdg":                            "Paris (Fly)",
	"dfw":                            "Dallas (Fly)",
	"ewr":                            "Secaucus (Fly)",
	"fra":                            "Frankfurt (Fly)",
	"gru":                            "São Paulo (Fly)",
	"iad":                            "Ashburn (Fly)",
	"jnb":                            "Johannesburg (Fly)",
	"lax":                            "Los Angeles (Fly)",
	"lhr":                            "London (Fly)",
	"nrt":                            "Tokyo (Fly)",
	"ord":                            "Chicago (Fly)",
	"sjc":                            "San Jose (Fly)",
	"sin":                            "Singapore (Fly)",
	"syd":                            "Sydney (Fly)",
	"yyz":                            "Toronto (Fly)",
	"koyeb_fra":                      "Frankfurt (Koyeb)",
	"koyeb_par":                      "Paris (Koyeb)",
	"koyeb_sfo":                      "San Francisco (Koyeb)",
	"koyeb_sin":                      "Singapore (Koyeb)",
	"koyeb_tyo":                      "Tokyo (Koyeb)",
	"koyeb_was":                      "Washington (Koyeb)",
	"railway_us-west2":               "California (Railway)",
	"railway_us-east4-eqdc4a":        "Virginia (Railway)",
	"railway_europe-west4-drams3a":   "Amsterdam (Railway)",
	"railway_asia-southeast1-eqsg3a": "Singapore (Railway)",
}

func DisplayName(code string) string {
	if n, ok := regionDisplayNames[code]; ok {
		return n
	}
	return code
}
