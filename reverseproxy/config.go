package reverseproxy

type Host struct {
	Host string `json:"host"`
	Port int `json:"port"`
	Paths []string `json:"path"`
	DefaultPath string `json:"defaultPath"`
	WebSocket bool `json:"webSocket"`
}

type Configuration struct {
	Hosts []Host `json:"hosts"`
}

