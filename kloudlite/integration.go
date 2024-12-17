package kloudlite

import (
	"github.com/kloudlite/operator/toolkit/operator"
	v1 "github.com/kloudlite/plugin-mongodb/api/v1"
	standalone_controller "github.com/kloudlite/plugin-mongodb/internal/standalone-controller"
)

func RegisterInto(mgr operator.Operator) {
	stEnv, err := standalone_controller.LoadEnv()
	if err != nil {
		panic(err)
	}
	mgr.AddToSchemes(v1.AddToScheme)
	mgr.RegisterControllers(
		&standalone_controller.StandaloneServiceReconciler{Env: *stEnv},
	)
}
