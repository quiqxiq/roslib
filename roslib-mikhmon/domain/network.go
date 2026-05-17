package domain

// QueueSimple = row /queue/simple/print.
type QueueSimple struct {
	ID         string
	Name       string
	Target     string
	MaxLimit   string // mis. "1M/2M"
	BurstLimit string
	Parent     string
	Disabled   bool
	Dynamic    bool
	Comment    string
}

// IPPool = row /ip/pool/print.
type IPPool struct {
	ID     string
	Name   string
	Ranges string
}

// ARPEntry = row /ip/arp/print.
type ARPEntry struct {
	ID         string
	Address    string
	MACAddress string
	Interface  string
	Dynamic    bool
	Disabled   bool
	Comment    string
}

// DHCPLease = row /ip/dhcp-server/lease/print.
type DHCPLease struct {
	ID           string
	Address      string
	MACAddress   string
	HostName     string
	Server       string
	Status       string
	Dynamic      bool
	Disabled     bool
	Comment      string
}

// Interface = row /interface/print.
type Interface struct {
	ID       string
	Name     string
	Type     string
	Running  bool
	Disabled bool
	Comment  string
}

// TrafficSnapshot = hasil /interface/monitor-traffic once="".
type TrafficSnapshot struct {
	Name           string
	RxBitsPerSec   int64
	TxBitsPerSec   int64
	RxPacketPerSec int64
	TxPacketPerSec int64
}

// PPPSecret = row /ppp/secret/print.
//
// Cross-ref: analisis §1.12 (inferred).
type PPPSecret struct {
	ID       string
	Name     string
	Password string
	Service  string
	Profile  string
	LocalAddr string
	RemoteAddr string
	Disabled bool
	Comment  string
}

// PPPProfile = row /ppp/profile/print.
type PPPProfile struct {
	ID         string
	Name       string
	LocalAddr  string
	RemoteAddr string
	RateLimit  string
	Comment    string
}

// PPPActive = row /ppp/active/print.
type PPPActive struct {
	ID       string
	Name     string
	Service  string
	CallerID string
	Address  string
	Uptime   string
}
