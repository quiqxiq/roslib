package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/quiqxiq/roslib-mikhmon/api/dto"
	"github.com/quiqxiq/roslib-mikhmon/service/devmgr"
	"github.com/quiqxiq/roslib-mikhmon/store"
)

type Devices struct {
	Store  store.DeviceStore
	DevMgr *devmgr.Manager
}

func NewDevices(s store.DeviceStore, mgr *devmgr.Manager) *Devices {
	return &Devices{Store: s, DevMgr: mgr}
}

func (h *Devices) Register(g *gin.RouterGroup) {
	d := g.Group("/devices")
	d.GET("", h.List)
	d.POST("", h.Create)
	d.GET("/:device_id", h.Get)
	d.PUT("/:device_id", h.Update)
	d.DELETE("/:device_id", h.Delete)
}

func (h *Devices) List(c *gin.Context) {
	devs, err := h.Store.List(c.Request.Context())
	if err != nil {
		WriteErr(c, err)
		return
	}
	out := make([]dto.DeviceResponse, len(devs))
	for i, d := range devs {
		out[i] = dto.FromModelDevice(d)
	}
	WriteList(c, out, len(out))
}

func (h *Devices) Get(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Err("INVALID_ID", "invalid device id", c.Param("device_id")))
		return
	}
	d, err := h.Store.Get(c.Request.Context(), id)
	if err != nil {
		WriteErr(c, err)
		return
	}
	WriteOK(c, dto.FromModelDevice(d))
}

func (h *Devices) Create(c *gin.Context) {
	var req dto.DeviceCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteValidationErr(c, err)
		return
	}
	d := req.ToModel()
	if err := h.Store.Create(c.Request.Context(), &d); err != nil {
		WriteErr(c, err)
		return
	}
	// Terhubung ke device setelah dibuat
	if err := h.DevMgr.Add(c.Request.Context(), d); err != nil {
		// Koneksi gagal tapi record sudah tersimpan — kembalikan data + warning
		c.JSON(http.StatusCreated, dto.OK(gin.H{
			"device":  dto.FromModelDevice(d),
			"warning": "device saved but connection failed: " + err.Error(),
		}))
		return
	}
	WriteCreated(c, dto.FromModelDevice(d))
}

func (h *Devices) Update(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Err("INVALID_ID", "invalid device id", c.Param("device_id")))
		return
	}
	existing, err := h.Store.Get(c.Request.Context(), id)
	if err != nil {
		WriteErr(c, err)
		return
	}
	var req dto.DeviceUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteValidationErr(c, err)
		return
	}
	req.Apply(&existing)
	if err := h.Store.Update(c.Request.Context(), &existing); err != nil {
		WriteErr(c, err)
		return
	}
	// Re-connect dengan config baru
	h.DevMgr.Remove(existing.Slug)
	_ = h.DevMgr.Add(context.Background(), existing)
	WriteOK(c, dto.FromModelDevice(existing))
}

func (h *Devices) Delete(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Err("INVALID_ID", "invalid device id", c.Param("device_id")))
		return
	}
	d, err := h.Store.Get(c.Request.Context(), id)
	if err != nil {
		WriteErr(c, err)
		return
	}
	h.DevMgr.Remove(d.Slug)
	if err := h.Store.Delete(c.Request.Context(), id); err != nil {
		WriteErr(c, err)
		return
	}
	WriteNoContent(c)
}

func parseID(c *gin.Context) (uint, error) {
	n, err := strconv.ParseUint(c.Param("device_id"), 10, 64)
	return uint(n), err
}
