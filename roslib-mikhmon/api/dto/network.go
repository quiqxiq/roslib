package dto

import "github.com/quiqxiq/roslib-mikhmon/domain"

// ── Interface ──────────────────────────────────────────────────────────

type InterfaceResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type,omitempty"`
	Running  bool   `json:"running"`
	Disabled bool   `json:"disabled"`
	Comment  string `json:"comment,omitempty"`
}

func FromDomainInterface(i domain.Interface) InterfaceResponse {
	return InterfaceResponse{
		ID: i.ID, Name: i.Name, Type: i.Type,
		Running: i.Running, Disabled: i.Disabled, Comment: i.Comment,
	}
}

func FromDomainInterfaces(is []domain.Interface) []InterfaceResponse {
	out := make([]InterfaceResponse, len(is))
	for i, x := range is {
		out[i] = FromDomainInterface(x)
	}
	return out
}

// ── Queue Simple ───────────────────────────────────────────────────────

type QueueSimpleResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Target     string `json:"target,omitempty"`
	MaxLimit   string `json:"max_limit,omitempty"`
	BurstLimit string `json:"burst_limit,omitempty"`
	Parent     string `json:"parent,omitempty"`
	Disabled   bool   `json:"disabled"`
	Dynamic    bool   `json:"dynamic"`
	Comment    string `json:"comment,omitempty"`
}

func FromDomainQueue(q domain.QueueSimple) QueueSimpleResponse {
	return QueueSimpleResponse{
		ID: q.ID, Name: q.Name, Target: q.Target,
		MaxLimit: q.MaxLimit, BurstLimit: q.BurstLimit, Parent: q.Parent,
		Disabled: q.Disabled, Dynamic: q.Dynamic, Comment: q.Comment,
	}
}

func FromDomainQueues(qs []domain.QueueSimple) []QueueSimpleResponse {
	out := make([]QueueSimpleResponse, len(qs))
	for i, q := range qs {
		out[i] = FromDomainQueue(q)
	}
	return out
}

// ── IP Pool ────────────────────────────────────────────────────────────

type IPPoolResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Ranges string `json:"ranges,omitempty"`
}

func FromDomainPool(p domain.IPPool) IPPoolResponse {
	return IPPoolResponse{ID: p.ID, Name: p.Name, Ranges: p.Ranges}
}

func FromDomainPools(ps []domain.IPPool) []IPPoolResponse {
	out := make([]IPPoolResponse, len(ps))
	for i, p := range ps {
		out[i] = FromDomainPool(p)
	}
	return out
}

// ── ARP ────────────────────────────────────────────────────────────────

type ARPEntryResponse struct {
	ID         string `json:"id"`
	Address    string `json:"address,omitempty"`
	MACAddress string `json:"mac_address,omitempty"`
	Interface  string `json:"interface,omitempty"`
	Dynamic    bool   `json:"dynamic"`
	Disabled   bool   `json:"disabled"`
	Comment    string `json:"comment,omitempty"`
}

func FromDomainARP(a domain.ARPEntry) ARPEntryResponse {
	return ARPEntryResponse{
		ID: a.ID, Address: a.Address, MACAddress: a.MACAddress,
		Interface: a.Interface, Dynamic: a.Dynamic, Disabled: a.Disabled, Comment: a.Comment,
	}
}

func FromDomainARPs(as []domain.ARPEntry) []ARPEntryResponse {
	out := make([]ARPEntryResponse, len(as))
	for i, a := range as {
		out[i] = FromDomainARP(a)
	}
	return out
}

// ── DHCP Lease ─────────────────────────────────────────────────────────

type DHCPLeaseResponse struct {
	ID         string `json:"id"`
	Address    string `json:"address,omitempty"`
	MACAddress string `json:"mac_address,omitempty"`
	HostName   string `json:"host_name,omitempty"`
	Server     string `json:"server,omitempty"`
	Status     string `json:"status,omitempty"`
	Dynamic    bool   `json:"dynamic"`
	Disabled   bool   `json:"disabled"`
	Comment    string `json:"comment,omitempty"`
}

func FromDomainLease(l domain.DHCPLease) DHCPLeaseResponse {
	return DHCPLeaseResponse{
		ID: l.ID, Address: l.Address, MACAddress: l.MACAddress,
		HostName: l.HostName, Server: l.Server, Status: l.Status,
		Dynamic: l.Dynamic, Disabled: l.Disabled, Comment: l.Comment,
	}
}

func FromDomainLeases(ls []domain.DHCPLease) []DHCPLeaseResponse {
	out := make([]DHCPLeaseResponse, len(ls))
	for i, l := range ls {
		out[i] = FromDomainLease(l)
	}
	return out
}

// ── TrafficSnapshot (untuk SSE stream payload) ─────────────────────────

type TrafficResponse struct {
	Name             string `json:"name,omitempty"`
	RxBitsPerSec     int64  `json:"rx_bits_per_sec"`
	TxBitsPerSec     int64  `json:"tx_bits_per_sec"`
	RxPacketsPerSec  int64  `json:"rx_packets_per_sec"`
	TxPacketsPerSec  int64  `json:"tx_packets_per_sec"`
}
