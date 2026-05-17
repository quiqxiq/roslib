//go:build integration

package integration

import (
	"testing"

	"github.com/quiqxiq/roslib-mikhmon/internal/testutil"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/network"
	"github.com/stretchr/testify/require"
)

func TestIntegration_InterfaceList(t *testing.T) {
	c := testutil.NewClient(t)
	net := network.New(c)

	ifs, err := net.InterfaceList(testutil.Context(t))
	require.NoError(t, err)
	require.NotEmpty(t, ifs, "router minimal punya 1 interface")
	t.Logf("interfaces: %d", len(ifs))
	for _, i := range ifs {
		t.Logf("  %s [%s] running=%v disabled=%v", i.Name, i.Type, i.Running, i.Disabled)
	}
}

func TestIntegration_DHCPLeaseCount(t *testing.T) {
	c := testutil.NewClient(t)
	net := network.New(c)
	n, err := net.DHCPLeaseCount(testutil.Context(t))
	require.NoError(t, err)
	t.Logf("dhcp leases: %d", n)
}

func TestIntegration_QueueSimpleStatic(t *testing.T) {
	c := testutil.NewClient(t)
	net := network.New(c)
	qs, err := net.QueueSimpleStatic(testutil.Context(t))
	require.NoError(t, err)
	t.Logf("static queues: %d", len(qs))
}
