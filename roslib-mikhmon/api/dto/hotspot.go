package dto

import (
	"github.com/quiqxiq/roslib-mikhmon/domain"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/hotspot"
)

// ── HotspotUser ──────────────────────────────────────────────────────

// HotspotUserResponse adalah representasi response untuk /hotspot/users.
// Password sengaja TIDAK di-expose (write-only).
type HotspotUserResponse struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Profile         string `json:"profile,omitempty"`
	Server          string `json:"server,omitempty"`
	Disabled        bool   `json:"disabled"`
	Comment         string `json:"comment,omitempty"`
	MACAddress      string `json:"mac_address,omitempty"`
	LimitUptime     string `json:"limit_uptime,omitempty"`
	LimitBytesTotal int64  `json:"limit_bytes_total"`
	BytesIn         int64  `json:"bytes_in"`
	BytesOut        int64  `json:"bytes_out"`
	Uptime          string `json:"uptime,omitempty"`
}

// FromDomainUser convert domain → DTO response.
func FromDomainUser(u domain.HotspotUser) HotspotUserResponse {
	return HotspotUserResponse{
		ID:              u.ID,
		Name:            u.Name,
		Profile:         u.Profile,
		Server:          u.Server,
		Disabled:        u.Disabled,
		Comment:         u.Comment,
		MACAddress:      u.MACAddress,
		LimitUptime:     u.LimitUptime,
		LimitBytesTotal: u.LimitBytesTotal,
		BytesIn:         u.BytesIn,
		BytesOut:        u.BytesOut,
		Uptime:          u.Uptime,
	}
}

// FromDomainUsers convert slice domain → slice DTO.
func FromDomainUsers(us []domain.HotspotUser) []HotspotUserResponse {
	out := make([]HotspotUserResponse, len(us))
	for i, u := range us {
		out[i] = FromDomainUser(u)
	}
	return out
}

// HotspotUserCreateRequest body untuk POST /hotspot/users.
type HotspotUserCreateRequest struct {
	Name            string `json:"name"                       binding:"required,min=1,max=128"`
	Password        string `json:"password,omitempty"         binding:"max=128"`
	Profile         string `json:"profile,omitempty"`
	Server          string `json:"server,omitempty"`
	Disabled        *bool  `json:"disabled,omitempty"`
	LimitUptime     string `json:"limit_uptime,omitempty"`
	LimitBytesTotal int64  `json:"limit_bytes_total,omitempty" binding:"gte=0"`
	Comment         string `json:"comment,omitempty"`
}

// ToArgs map ke hotspot.UserAddArgs.
func (r HotspotUserCreateRequest) ToArgs() hotspot.UserAddArgs {
	return hotspot.UserAddArgs{
		Name:            r.Name,
		Password:        r.Password,
		Profile:         r.Profile,
		Server:          r.Server,
		Disabled:        r.Disabled,
		LimitUptime:     r.LimitUptime,
		LimitBytesTotal: r.LimitBytesTotal,
		Comment:         r.Comment,
	}
}

// HotspotUserUpdateRequest body untuk PUT /hotspot/users/:id. Field
// pointer untuk semantic "tidak di-update kalau nil".
type HotspotUserUpdateRequest struct {
	Name            string  `json:"name,omitempty"             binding:"max=128"`
	Password        string  `json:"password,omitempty"         binding:"max=128"`
	Profile         string  `json:"profile,omitempty"`
	Server          string  `json:"server,omitempty"`
	Disabled        *bool   `json:"disabled,omitempty"`
	LimitUptime     string  `json:"limit_uptime,omitempty"`
	LimitBytesTotal *int64  `json:"limit_bytes_total,omitempty"`
	Comment         *string `json:"comment,omitempty"`
	MACAddress      *string `json:"mac_address,omitempty"`
}

// ToArgs map ke hotspot.UserSetArgs (ID di-isi caller dari path param).
func (r HotspotUserUpdateRequest) ToArgs(id string) hotspot.UserSetArgs {
	return hotspot.UserSetArgs{
		ID:              id,
		Name:            r.Name,
		Password:        r.Password,
		Profile:         r.Profile,
		Server:          r.Server,
		Disabled:        r.Disabled,
		LimitUptime:     r.LimitUptime,
		LimitBytesTotal: r.LimitBytesTotal,
		Comment:         r.Comment,
		MACAddress:      r.MACAddress,
	}
}

// SetBoolRequest untuk PATCH .../disabled.
type SetBoolRequest struct {
	Value bool `json:"value"`
}

// SetStringRequest untuk PATCH .../expiry, .../mac.
type SetStringRequest struct {
	Value string `json:"value" binding:"required"`
}

// BulkIDsRequest body untuk operasi bulk (delete, dll).
type BulkIDsRequest struct {
	IDs []string `json:"ids" binding:"required,min=1,dive,required"`
}

// BulkResult response untuk bulk operation.
type BulkResult struct {
	Succeeded []string          `json:"succeeded"`
	Failed    map[string]string `json:"failed"` // id → error message
}

// ── HotspotProfile ───────────────────────────────────────────────────

type HotspotProfileResponse struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	AddressPool       string `json:"address_pool,omitempty"`
	RateLimit         string `json:"rate_limit,omitempty"`
	SharedUsers       int    `json:"shared_users"`
	StatusAutorefresh string `json:"status_autorefresh,omitempty"`
	OnLogin           string `json:"on_login,omitempty"`
	ParentQueue       string `json:"parent_queue,omitempty"`
}

func FromDomainProfile(p domain.HotspotProfile) HotspotProfileResponse {
	return HotspotProfileResponse{
		ID:                p.ID,
		Name:              p.Name,
		AddressPool:       p.AddressPool,
		RateLimit:         p.RateLimit,
		SharedUsers:       p.SharedUsers,
		StatusAutorefresh: p.StatusAutorefresh,
		OnLogin:           p.OnLogin,
		ParentQueue:       p.ParentQueue,
	}
}

func FromDomainProfiles(ps []domain.HotspotProfile) []HotspotProfileResponse {
	out := make([]HotspotProfileResponse, len(ps))
	for i, p := range ps {
		out[i] = FromDomainProfile(p)
	}
	return out
}

type HotspotProfileCreateRequest struct {
	Name              string `json:"name"                          binding:"required,min=1,max=128"`
	AddressPool       string `json:"address_pool,omitempty"`
	RateLimit         string `json:"rate_limit,omitempty"`
	SharedUsers       int    `json:"shared_users,omitempty"        binding:"gte=0"`
	StatusAutorefresh string `json:"status_autorefresh,omitempty"`
	OnLogin           string `json:"on_login,omitempty"`
	ParentQueue       string `json:"parent_queue,omitempty"`
}

func (r HotspotProfileCreateRequest) ToArgs() hotspot.ProfileAddArgs {
	return hotspot.ProfileAddArgs{
		Name:              r.Name,
		AddressPool:       r.AddressPool,
		RateLimit:         r.RateLimit,
		SharedUsers:       r.SharedUsers,
		StatusAutorefresh: r.StatusAutorefresh,
		OnLogin:           r.OnLogin,
		ParentQueue:       r.ParentQueue,
	}
}

type HotspotProfileUpdateRequest struct {
	Name              string  `json:"name,omitempty"`
	AddressPool       string  `json:"address_pool,omitempty"`
	RateLimit         string  `json:"rate_limit,omitempty"`
	SharedUsers       *int    `json:"shared_users,omitempty"`
	StatusAutorefresh string  `json:"status_autorefresh,omitempty"`
	OnLogin           *string `json:"on_login,omitempty"`
	ParentQueue       string  `json:"parent_queue,omitempty"`
}

func (r HotspotProfileUpdateRequest) ToArgs(id string) hotspot.ProfileSetArgs {
	return hotspot.ProfileSetArgs{
		ID:                id,
		Name:              r.Name,
		AddressPool:       r.AddressPool,
		RateLimit:         r.RateLimit,
		SharedUsers:       r.SharedUsers,
		StatusAutorefresh: r.StatusAutorefresh,
		OnLogin:           r.OnLogin,
		ParentQueue:       r.ParentQueue,
	}
}

// HotspotProfileDeleteRequest opsional body untuk cascade scheduler cleanup.
type HotspotProfileDeleteRequest struct {
	Name string `json:"name,omitempty"`
}

// ── HotspotActive ────────────────────────────────────────────────────

type HotspotActiveResponse struct {
	ID         string `json:"id"`
	User       string `json:"user"`
	Address    string `json:"address,omitempty"`
	MACAddress string `json:"mac_address,omitempty"`
	Server     string `json:"server,omitempty"`
	LoginBy    string `json:"login_by,omitempty"`
	Uptime     string `json:"uptime,omitempty"`
	BytesIn    int64  `json:"bytes_in"`
	BytesOut   int64  `json:"bytes_out"`
}

func FromDomainActive(a domain.HotspotActive) HotspotActiveResponse {
	return HotspotActiveResponse{
		ID: a.ID, User: a.User, Address: a.Address, MACAddress: a.MACAddress,
		Server: a.Server, LoginBy: a.LoginBy, Uptime: a.Uptime,
		BytesIn: a.BytesIn, BytesOut: a.BytesOut,
	}
}

func FromDomainActives(as []domain.HotspotActive) []HotspotActiveResponse {
	out := make([]HotspotActiveResponse, len(as))
	for i, a := range as {
		out[i] = FromDomainActive(a)
	}
	return out
}

// ── HotspotBinding ───────────────────────────────────────────────────

type HotspotBindingResponse struct {
	ID         string `json:"id"`
	MACAddress string `json:"mac_address,omitempty"`
	Address    string `json:"address,omitempty"`
	ToAddress  string `json:"to_address,omitempty"`
	Server     string `json:"server,omitempty"`
	Type       string `json:"type,omitempty"`
	Disabled   bool   `json:"disabled"`
	Comment    string `json:"comment,omitempty"`
}

func FromDomainBinding(b domain.HotspotBinding) HotspotBindingResponse {
	return HotspotBindingResponse{
		ID: b.ID, MACAddress: b.MACAddress, Address: b.Address, ToAddress: b.ToAddress,
		Server: b.Server, Type: b.Type, Disabled: b.Disabled, Comment: b.Comment,
	}
}

func FromDomainBindings(bs []domain.HotspotBinding) []HotspotBindingResponse {
	out := make([]HotspotBindingResponse, len(bs))
	for i, b := range bs {
		out[i] = FromDomainBinding(b)
	}
	return out
}

// SetBindingTypeRequest body untuk PATCH /hotspot/bindings/:id/type.
type SetBindingTypeRequest struct {
	Type string `json:"type" binding:"required,oneof=regular bypassed blocked"`
}

// ── HotspotHost ──────────────────────────────────────────────────────

type HotspotHostResponse struct {
	ID         string `json:"id"`
	MACAddress string `json:"mac_address,omitempty"`
	Address    string `json:"address,omitempty"`
	ToAddress  string `json:"to_address,omitempty"`
	Server     string `json:"server,omitempty"`
	Authorized bool   `json:"authorized"`
	Bypassed   bool   `json:"bypassed"`
	Comment    string `json:"comment,omitempty"`
}

func FromDomainHost(h domain.HotspotHost) HotspotHostResponse {
	return HotspotHostResponse{
		ID: h.ID, MACAddress: h.MACAddress, Address: h.Address, ToAddress: h.ToAddress,
		Server: h.Server, Authorized: h.Authorized, Bypassed: h.Bypassed, Comment: h.Comment,
	}
}

func FromDomainHosts(hs []domain.HotspotHost) []HotspotHostResponse {
	out := make([]HotspotHostResponse, len(hs))
	for i, h := range hs {
		out[i] = FromDomainHost(h)
	}
	return out
}

// ── HotspotCookie ────────────────────────────────────────────────────

type HotspotCookieResponse struct {
	ID      string `json:"id"`
	User    string `json:"user,omitempty"`
	Domain  string `json:"domain,omitempty"`
	Expires string `json:"expires,omitempty"`
}

func FromDomainCookie(c domain.HotspotCookie) HotspotCookieResponse {
	return HotspotCookieResponse{
		ID: c.ID, User: c.User, Domain: c.Domain, Expires: c.Expires,
	}
}

func FromDomainCookies(cs []domain.HotspotCookie) []HotspotCookieResponse {
	out := make([]HotspotCookieResponse, len(cs))
	for i, c := range cs {
		out[i] = FromDomainCookie(c)
	}
	return out
}

// ── HotspotServer ────────────────────────────────────────────────────

type HotspotServerResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Profile  string `json:"profile,omitempty"`
	Disabled bool   `json:"disabled"`
}

func FromHotspotServer(s hotspot.HotspotServer) HotspotServerResponse {
	return HotspotServerResponse{ID: s.ID, Name: s.Name, Profile: s.Profile, Disabled: s.Disabled}
}

func FromHotspotServers(ss []hotspot.HotspotServer) []HotspotServerResponse {
	out := make([]HotspotServerResponse, len(ss))
	for i, s := range ss {
		out[i] = FromHotspotServer(s)
	}
	return out
}
