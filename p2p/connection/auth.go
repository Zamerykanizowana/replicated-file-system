package connection

var whitelist = make(map[string]struct{})

func Whitelist(peers ...string) {
	for _, peer := range peers {
		whitelist[peer] = struct{}{}
	}
}
