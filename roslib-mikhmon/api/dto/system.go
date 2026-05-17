package dto

import (
	"github.com/quiqxiq/roslib-mikhmon/domain"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/system"
)

// ── SystemIdentity ─────────────────────────────────────────────────────

type SystemIdentityResponse struct {
	Name string `json:"name"`
}

func FromDomainIdentity(i domain.SystemIdentity) SystemIdentityResponse {
	return SystemIdentityResponse{Name: i.Name}
}

// ── SystemResource ─────────────────────────────────────────────────────

type SystemResourceResponse struct {
	Uptime           string `json:"uptime,omitempty"`
	Version          string `json:"version,omitempty"`
	BoardName        string `json:"board_name,omitempty"`
	CPULoad          int    `json:"cpu_load"`
	FreeMemory       int64  `json:"free_memory"`
	TotalMemory      int64  `json:"total_memory"`
	FreeHDDSpace     int64  `json:"free_hdd_space"`
	TotalHDDSpace    int64  `json:"total_hdd_space"`
	ArchitectureName string `json:"architecture_name,omitempty"`
}

func FromDomainResource(r domain.SystemResource) SystemResourceResponse {
	return SystemResourceResponse{
		Uptime: r.Uptime, Version: r.Version, BoardName: r.BoardName,
		CPULoad: r.CPULoad, FreeMemory: r.FreeMemory, TotalMemory: r.TotalMemory,
		FreeHDDSpace: r.FreeHDDSpace, TotalHDDSpace: r.TotalHDDSpace,
		ArchitectureName: r.ArchitectureName,
	}
}

// ── SystemRouterboard ──────────────────────────────────────────────────

type SystemRouterboardResponse struct {
	Routerboard     bool   `json:"routerboard"`
	Model           string `json:"model,omitempty"`
	SerialNumber    string `json:"serial_number,omitempty"`
	FirmwareType    string `json:"firmware_type,omitempty"`
	FactoryFirmware string `json:"factory_firmware,omitempty"`
	CurrentFirmware string `json:"current_firmware,omitempty"`
	UpgradeFirmware string `json:"upgrade_firmware,omitempty"`
}

func FromDomainRouterboard(r domain.SystemRouterboard) SystemRouterboardResponse {
	return SystemRouterboardResponse{
		Routerboard: r.Routerboard, Model: r.Model, SerialNumber: r.SerialNumber,
		FirmwareType: r.FirmwareType, FactoryFirmware: r.FactoryFirmware,
		CurrentFirmware: r.CurrentFirmware, UpgradeFirmware: r.UpgradeFirmware,
	}
}

// ── SystemClock ────────────────────────────────────────────────────────

type SystemClockResponse struct {
	Time         string `json:"time,omitempty"`
	Date         string `json:"date,omitempty"`
	TimeZoneName string `json:"time_zone_name,omitempty"`
	GMTOffset    string `json:"gmt_offset,omitempty"`
	DSTActive    bool   `json:"dst_active"`
}

func FromDomainClock(c domain.SystemClock) SystemClockResponse {
	return SystemClockResponse{
		Time: c.Time, Date: c.Date, TimeZoneName: c.TimeZoneName,
		GMTOffset: c.GMTOffset, DSTActive: c.DSTActive,
	}
}

// ── Script ─────────────────────────────────────────────────────────────

type ScriptResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Owner   string `json:"owner,omitempty"`
	Source  string `json:"source,omitempty"`
	Comment string `json:"comment,omitempty"`
	Policy  string `json:"policy,omitempty"`
}

func FromDomainScript(s domain.Script) ScriptResponse {
	return ScriptResponse{
		ID: s.ID, Name: s.Name, Owner: s.Owner,
		Source: s.Source, Comment: s.Comment, Policy: s.Policy,
	}
}

func FromDomainScripts(ss []domain.Script) []ScriptResponse {
	out := make([]ScriptResponse, len(ss))
	for i, s := range ss {
		out[i] = FromDomainScript(s)
	}
	return out
}

type ScriptCreateRequest struct {
	Name    string `json:"name"             binding:"required,min=1,max=128"`
	Source  string `json:"source"           binding:"required,min=1"`
	Comment string `json:"comment,omitempty"`
	Owner   string `json:"owner,omitempty"`
	Policy  string `json:"policy,omitempty"`
}

func (r ScriptCreateRequest) ToArgs() system.ScriptAddArgs {
	return system.ScriptAddArgs{
		Name: r.Name, Source: r.Source, Comment: r.Comment, Owner: r.Owner, Policy: r.Policy,
	}
}

type ScriptUpdateRequest struct {
	Name    string `json:"name,omitempty"`
	Source  string `json:"source,omitempty"`
	Comment string `json:"comment,omitempty"`
	Owner   string `json:"owner,omitempty"`
}

func (r ScriptUpdateRequest) ToArgs(id string) system.ScriptSetArgs {
	return system.ScriptSetArgs{
		ID: id, Name: r.Name, Source: r.Source, Comment: r.Comment, Owner: r.Owner,
	}
}

// ── Scheduler ──────────────────────────────────────────────────────────

type SchedulerResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	StartDate string `json:"start_date,omitempty"`
	StartTime string `json:"start_time,omitempty"`
	Interval  string `json:"interval,omitempty"`
	OnEvent   string `json:"on_event,omitempty"`
	NextRun   string `json:"next_run,omitempty"`
	Disabled  bool   `json:"disabled"`
	Comment   string `json:"comment,omitempty"`
	RunCount  int    `json:"run_count"`
}

func FromDomainScheduler(s domain.Scheduler) SchedulerResponse {
	return SchedulerResponse{
		ID: s.ID, Name: s.Name, StartDate: s.StartDate, StartTime: s.StartTime,
		Interval: s.Interval, OnEvent: s.OnEvent, NextRun: s.NextRun,
		Disabled: s.Disabled, Comment: s.Comment, RunCount: s.RunCount,
	}
}

func FromDomainSchedulers(ss []domain.Scheduler) []SchedulerResponse {
	out := make([]SchedulerResponse, len(ss))
	for i, s := range ss {
		out[i] = FromDomainScheduler(s)
	}
	return out
}

type SchedulerCreateRequest struct {
	Name      string `json:"name"                  binding:"required,min=1,max=128"`
	OnEvent   string `json:"on_event,omitempty"`
	StartDate string `json:"start_date,omitempty"`
	StartTime string `json:"start_time,omitempty"`
	Interval  string `json:"interval,omitempty"`
	Disabled  *bool  `json:"disabled,omitempty"`
	Comment   string `json:"comment,omitempty"`
}

func (r SchedulerCreateRequest) ToArgs() system.SchedulerAddArgs {
	return system.SchedulerAddArgs{
		Name: r.Name, OnEvent: r.OnEvent, StartDate: r.StartDate,
		StartTime: r.StartTime, Interval: r.Interval,
		Disabled: r.Disabled, Comment: r.Comment,
	}
}

type SchedulerUpdateRequest struct {
	Name     string `json:"name,omitempty"`
	OnEvent  string `json:"on_event,omitempty"`
	Interval string `json:"interval,omitempty"`
	Disabled *bool  `json:"disabled,omitempty"`
	Comment  string `json:"comment,omitempty"`
}

func (r SchedulerUpdateRequest) ToArgs(id string) system.SchedulerSetArgs {
	return system.SchedulerSetArgs{
		ID: id, Name: r.Name, OnEvent: r.OnEvent, Interval: r.Interval,
		Disabled: r.Disabled, Comment: r.Comment,
	}
}

// ── LoggingAddHotspotDiskRequest ──────────────────────────────────────

type LoggingAddHotspotDiskRequest struct {
	Prefix string `json:"prefix,omitempty"` // default "->"
}
