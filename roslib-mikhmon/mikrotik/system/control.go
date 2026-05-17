package system

import "context"

// Reboot → /system/reboot (analisis §1.1).
//
// PERINGATAN: men-restart router. Caller harus konfirmasi dengan user.
func (c *Client) Reboot(ctx context.Context) error {
	_, err := c.dev.Path("/system").Run(ctx, "reboot")
	return err
}

// Shutdown → /system/shutdown (analisis §1.1).
//
// PERINGATAN: mematikan router (perlu power cycle untuk on lagi).
func (c *Client) Shutdown(ctx context.Context) error {
	_, err := c.dev.Path("/system").Run(ctx, "shutdown")
	return err
}
