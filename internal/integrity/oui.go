package integrity

import "strings"

var ouiVendors = map[string]string{
	"DC:62:79": "TP-Link",
	"BC:07:1D": "TP-Link",
	"00:45:E2": "Intel",
	"E4:5F:01": "Realtek",
	"00:E0:4C": "Realtek",
	"3C:58:C2": "Intel",
	"EE:91:12": "Thinkrider",
}

func vendorFromMAC(addr string) string {
	if len(addr) < 8 {
		return "Unknown"
	}
	prefix := strings.ToUpper(addr[:8])
	if v, ok := ouiVendors[prefix]; ok {
		return v
	}
	return "Unknown"
}
