package kloudlite

import (
	"github.com/kloudlite/operator/toolkit/operator"
	v1 "github.com/kloudlite/plugin-mongodb/api/v1"
	standalone_database_controller "github.com/kloudlite/plugin-mongodb/internal/standalone/database"
	standalone_service_controller "github.com/kloudlite/plugin-mongodb/internal/standalone/service"
)

func RegisterInto(mgr operator.Operator) {
	stSvcEnv, err := standalone_service_controller.LoadEnv()
	if err != nil {
		panic(err)
	}

	stDBEnv, err := standalone_database_controller.LoadEnv()
	if err != nil {
		panic(err)
	}

	mgr.AddToSchemes(v1.AddToScheme)
	mgr.RegisterControllers(
		&standalone_service_controller.Reconciler{Env: *stSvcEnv},
		&standalone_database_controller.Reconciler{Env: stDBEnv, YAMLClient: mgr.Operator().KubeYAMLClient()},
	)
}
