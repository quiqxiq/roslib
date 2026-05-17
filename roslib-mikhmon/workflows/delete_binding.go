package workflows

import (
	"context"
	"errors"
	"fmt"

	"github.com/quiqxiq/roslib-mikhmon/mikrotik"
)

// DeleteBinding melakukan cascade hapus IP-binding mengikuti analisis §4.2.
// Total 9 step:
//
//	1. lookup binding by ID         → dapat MAC
//	2. /ip/hotspot/ip-binding/remove (.id)
//	3. /queue/simple/print           (?name=$mac)
//	4. /queue/simple/remove          (.id)         (tiap match)
//	5. /system/scheduler/print       (?name=$mac)
//	6. /system/scheduler/remove      (.id)         (tiap match)
//	7. /ip/arp/print                 (?mac-address=$mac)
//	8. /ip/arp/remove                (.id)         (tiap match)
//	9. /ip/dhcp-server/lease/print   (?mac-address=$mac)
//	   /ip/dhcp-server/lease/remove  (.id)         (tiap match)
//
// Step 1 (lookup) di-skip kalau binding sudah tidak ada (ErrNotFound)
// dan caller pass MAC manual via DeleteBindingByMAC.
func DeleteBinding(ctx context.Context, c *Clients, bindingID string) error {
	if bindingID == "" {
		return mikrotik.ErrInvalidArgument
	}
	b, err := c.Hotspot.BindingByID(ctx, bindingID)
	if err != nil {
		return fmt.Errorf("workflows.DeleteBinding: lookup: %w", err)
	}
	if rerr := c.Hotspot.BindingRemove(ctx, bindingID); rerr != nil {
		return fmt.Errorf("workflows.DeleteBinding: remove binding: %w", rerr)
	}
	return cascadeCleanupByMAC(ctx, c, b.MACAddress)
}

// DeleteBindingByMAC adalah varian yang sudah tahu MAC-nya (mis. caller
// punya UI yang pass MAC ke handler). Lookup binding-by-MAC dulu,
// hapus, lalu cascade cleanup.
func DeleteBindingByMAC(ctx context.Context, c *Clients, mac string) error {
	if mac == "" {
		return mikrotik.ErrInvalidArgument
	}
	bindings, err := c.Hotspot.BindingList(ctx)
	if err != nil {
		return fmt.Errorf("workflows.DeleteBindingByMAC: list: %w", err)
	}
	var bindingID string
	for _, b := range bindings {
		if b.MACAddress == mac {
			bindingID = b.ID
			break
		}
	}
	if bindingID == "" {
		return mikrotik.ErrNotFound
	}
	if rerr := c.Hotspot.BindingRemove(ctx, bindingID); rerr != nil {
		return fmt.Errorf("workflows.DeleteBindingByMAC: remove binding: %w", rerr)
	}
	return cascadeCleanupByMAC(ctx, c, mac)
}

// cascadeCleanupByMAC adalah step 3-9 dari analisis §4.2.
func cascadeCleanupByMAC(ctx context.Context, c *Clients, mac string) error {
	if mac == "" {
		return nil
	}

	// 3-4: queue
	queues, err := c.Network.QueueSimpleByName(ctx, mac)
	if err != nil && !errors.Is(err, mikrotik.ErrNotFound) {
		return fmt.Errorf("queue lookup: %w", err)
	}
	for _, q := range queues {
		if rerr := c.Network.QueueSimpleRemove(ctx, q.ID); rerr != nil {
			return fmt.Errorf("queue remove %q: %w", q.ID, rerr)
		}
	}

	// 5-6: scheduler
	schedulers, err := c.System.SchedulerByName(ctx, mac)
	if err != nil {
		return fmt.Errorf("scheduler lookup: %w", err)
	}
	for _, s := range schedulers {
		if rerr := c.System.SchedulerRemove(ctx, s.ID); rerr != nil {
			return fmt.Errorf("scheduler remove %q: %w", s.ID, rerr)
		}
	}

	// 7-8: ARP
	arps, err := c.Network.ARPByMAC(ctx, mac)
	if err != nil {
		return fmt.Errorf("arp lookup: %w", err)
	}
	for _, a := range arps {
		if rerr := c.Network.ARPRemove(ctx, a.ID); rerr != nil {
			return fmt.Errorf("arp remove %q: %w", a.ID, rerr)
		}
	}

	// 9: DHCP lease
	leases, err := c.Network.DHCPLeaseByMAC(ctx, mac)
	if err != nil {
		return fmt.Errorf("dhcp lookup: %w", err)
	}
	for _, l := range leases {
		if rerr := c.Network.DHCPLeaseRemove(ctx, l.ID); rerr != nil {
			return fmt.Errorf("dhcp remove %q: %w", l.ID, rerr)
		}
	}
	return nil
}
