package types

import "encoding/json"

type ServiceCredentials struct {
	RootUsername string `json:"ROOT_USERNAME"`
	RootPassword string `json:"ROOT_PASSWORD"`

	DBName     string `json:"DB_NAME"`
	AuthSource string `json:"AUTH_SOURCE"`

	Port string `json:"PORT"`

	// Host is an alias for .KLOUDLITE_HOST
	Host string `json:"HOST"`

	// Addr is an alias for .KLOUDLITE_ADDR
	Addr string `json:"ADDR"`

	// URI is an alias for .KLOUDLITE_URI
	URI string `json:"URI"`

	ClusterLocalHost string `json:".CLUSTER_LOCAL_HOST"`
	ClusterLocalAddr string `json:".CLUSTER_LOCAL_ADDR"`
	ClusterLocalURI  string `json:".CLUSTER_LOCAL_URI"`

	MultiClusterHost string `json:".MUTLI_CLUSTER_HOST,omitempty"`
	MultiClusterAddr string `json:".MUTLI_CLUSTER_ADDR,omitempty"`
	MultiClusterURI  string `json:".MUTLI_CLUSTER_URI,omitempty"`

	KloudliteHost string `json:".KLOUDLITE_HOST,omitempty"`
	KloudliteAddr string `json:".KLOUDLITE_ADDR,omitempty"`
	KloudliteURI  string `json:".KLOUDLITE_URI,omitempty"`
}

func (c *ServiceCredentials) ToMap() (map[string]string, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	var m map[string]string
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

type DatabaseOutput struct {
	Username string `json:"USERNAME"`
	Password string `json:"PASSWORD"`

	DbName string `json:"DB_NAME"`

	Port string `json:"PORT"`

	// Host is an alias for .KLOUDLITE_HOST
	Host string `json:"HOST"`

	// Host is an alias for .KLOUDLITE_URI
	URI string `json:"URI"`

	ClusterLocalHost string `json:".CLUSTER_LOCAL_HOST,omitempty"`
	ClusterLocalURI  string `json:".CLUSTER_LOCAL_URI,omitempty"`

	MultiClusterHost string `json:".MUTLI_CLUSTER_HOST,omitempty"`
	MultiClusterURI  string `json:".MUTLI_CLUSTER_URI,omitempty"`

	KloudliteHost string `json:".KLOUDLITE_HOST,omitempty"`
	KloudliteURI  string `json:".KLOUDLITE_URI,omitempty"`
}

func (c DatabaseOutput) ToMap() (map[string]string, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	var m map[string]string
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}
