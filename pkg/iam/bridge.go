// Package iam provides the bridge between pkg/iam and management/middleware.
// This file registers the legacy-to-new permission mapping at package init time,
// so that RequirePermission can match both old and new format JWT claims.
package iam

import (
	"github.com/Veritas-Calculus/vc-stack/internal/management/middleware"
)

func init() {
	middleware.RegisterLegacyToNewMapping(LegacyToNew())
	middleware.RegisterNewToLegacyMapping(NewToLegacy())
}
