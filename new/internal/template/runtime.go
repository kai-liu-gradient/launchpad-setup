package template

import (
	"os"

	"gopkg.in/yaml.v3"
)

type RuntimeValues struct {
	GiteaAccessToken string `yaml:"gitea_access_token,omitempty"`
	GiteaUser        string `yaml:"gitea_user,omitempty"`
	IngressClusterIP string `yaml:"ingress_cluster_ip,omitempty"`
}

func LoadRuntime(path string) (*RuntimeValues, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &RuntimeValues{}, nil
		}
		return nil, err
	}
	var rv RuntimeValues
	if err := yaml.Unmarshal(data, &rv); err != nil {
		return nil, err
	}
	return &rv, nil
}

func (rv *RuntimeValues) Save(path string) error {
	data, err := yaml.Marshal(rv)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
