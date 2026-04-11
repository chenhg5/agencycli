package store

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// CCConnectConfig holds the connection settings for a cc-connect instance.
type CCConnectConfig struct {
	APIURL string `yaml:"api_url" json:"apiUrl"`
	Token  string `yaml:"token"   json:"-"`
}

type CCConnectStore struct {
	root string
}

func NewCCConnectStore(root string) *CCConnectStore {
	return &CCConnectStore{root: root}
}

func (s *CCConnectStore) filePath() string {
	return filepath.Join(s.root, ".agencycli", "ccconnect.yaml")
}

func (s *CCConnectStore) Load() (*CCConnectConfig, error) {
	data, err := os.ReadFile(s.filePath())
	if err != nil {
		if os.IsNotExist(err) {
			return &CCConnectConfig{}, nil
		}
		return nil, err
	}
	var cfg CCConnectConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (s *CCConnectStore) Save(cfg *CCConnectConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	dir := filepath.Dir(s.filePath())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(s.filePath(), data, 0o644)
}
