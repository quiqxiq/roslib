package dto

import (
	"time"

	"github.com/quiqxiq/roslib-mikhmon/store/model"
)

// DeviceResponse adalah response publik untuk mikrotik device.
// Password tidak di-expose.
type DeviceResponse struct {
	ID                  uint       `json:"id"`
	Slug                string     `json:"slug"`
	DisplayName         string     `json:"display_name"`
	Address             string     `json:"address"`
	Username            string     `json:"username"`
	UseTLS              bool       `json:"use_tls"`
	Status              string     `json:"status"`
	LastSeen            *time.Time `json:"last_seen,omitempty"`
	LastError           string     `json:"last_error,omitempty"`
	ExpiryCheckInterval string     `json:"expiry_check_interval"`
	Active              bool       `json:"active"`
	CreatedAt           time.Time  `json:"created_at"`
}

func FromModelDevice(d model.MikrotikDevice) DeviceResponse {
	return DeviceResponse{
		ID:                  d.ID,
		Slug:                d.Slug,
		DisplayName:         d.DisplayName,
		Address:             d.Address,
		Username:            d.Username,
		UseTLS:              d.UseTLS,
		Status:              d.Status,
		LastSeen:            d.LastSeen,
		LastError:           d.LastError,
		ExpiryCheckInterval: d.ExpiryCheckInterval,
		Active:              d.Active,
		CreatedAt:           d.CreatedAt,
	}
}

type DeviceCreateRequest struct {
	Slug                string `json:"slug"         binding:"required,min=1,max=64"`
	DisplayName         string `json:"display_name" binding:"required,min=1,max=128"`
	Address             string `json:"address"      binding:"required"`
	Username            string `json:"username"     binding:"required"`
	Password            string `json:"password"     binding:"required"`
	UseTLS              bool   `json:"use_tls"`
	ExpiryCheckInterval string `json:"expiry_check_interval"`
}

func (r DeviceCreateRequest) ToModel() model.MikrotikDevice {
	interval := r.ExpiryCheckInterval
	if interval == "" {
		interval = "2m"
	}
	return model.MikrotikDevice{
		Slug:                r.Slug,
		DisplayName:         r.DisplayName,
		Address:             r.Address,
		Username:            r.Username,
		Password:            r.Password,
		UseTLS:              r.UseTLS,
		ExpiryCheckInterval: interval,
		Active:              true,
		Status:              "disconnected",
	}
}

type DeviceUpdateRequest struct {
	DisplayName         string `json:"display_name,omitempty"`
	Address             string `json:"address,omitempty"`
	Username            string `json:"username,omitempty"`
	Password            string `json:"password,omitempty"`
	UseTLS              *bool  `json:"use_tls,omitempty"`
	ExpiryCheckInterval string `json:"expiry_check_interval,omitempty"`
	Active              *bool  `json:"active,omitempty"`
}

func (r DeviceUpdateRequest) Apply(d *model.MikrotikDevice) {
	if r.DisplayName != "" {
		d.DisplayName = r.DisplayName
	}
	if r.Address != "" {
		d.Address = r.Address
	}
	if r.Username != "" {
		d.Username = r.Username
	}
	if r.Password != "" {
		d.Password = r.Password
	}
	if r.UseTLS != nil {
		d.UseTLS = *r.UseTLS
	}
	if r.ExpiryCheckInterval != "" {
		d.ExpiryCheckInterval = r.ExpiryCheckInterval
	}
	if r.Active != nil {
		d.Active = *r.Active
	}
}
