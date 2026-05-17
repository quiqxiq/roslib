// Package integration berisi test end-to-end ke router asli.
//
// Aktifkan dengan build tag dan env var:
//
//	export ROSLIB_ROUTER_ADDRESS=192.168.88.1:8728
//	export ROSLIB_ROUTER_USERNAME=admin
//	export ROSLIB_ROUTER_PASSWORD=secret
//	go test -tags=integration -count=1 ./test/integration/...
//
// Tanpa env, semua test di-skip via testutil.RequireIntegration.
package integration
