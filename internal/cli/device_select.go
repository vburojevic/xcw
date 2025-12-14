package cli

import (
	"context"
	"strings"

	"github.com/vburojevic/xcw/internal/domain"
	"github.com/vburojevic/xcw/internal/simulator"
)

const bootedSentinel = "booted"

func simulatorArgIsBooted(s string) bool {
	return strings.EqualFold(strings.TrimSpace(s), bootedSentinel)
}

func resolveSimulatorDevice(ctx context.Context, mgr *simulator.Manager, simulatorArg string, booted bool) (*domain.Device, error) {
	if booted || simulatorArgIsBooted(simulatorArg) || strings.TrimSpace(simulatorArg) == "" {
		return mgr.FindBootedDevice(ctx)
	}
	return mgr.FindDevice(ctx, simulatorArg)
}
