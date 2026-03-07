package teams

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/capitalrx/grafana/pkg/infra/log"
	"go.yaml.in/yaml/v3"
)

type configReader struct {
	path string
	log  log.Logger
}

func (cr *configReader) readConfig() ([]*TeamConfig, error) {
	var teams []*TeamConfig

	files, err := os.ReadDir(cr.path)
	if err != nil {
		if os.IsNotExist(err) {
			return teams, nil
		}
		cr.log.Error("Can't read team provisioning files from directory", "path", cr.path, "error", err)
		return teams, nil
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".yaml") && !strings.HasSuffix(file.Name(), ".yml") {
			continue
		}

		parsedTeams, err := cr.parseConfig(file)
		if err != nil {
			return nil, fmt.Errorf("could not parse provisioning config file: %s error: %v", file.Name(), err)
		}

		if len(parsedTeams) > 0 {
			teams = append(teams, parsedTeams...)
		}
	}

	return teams, nil
}

func (cr *configReader) parseConfig(file fs.DirEntry) ([]*TeamConfig, error) {
	filename, _ := filepath.Abs(filepath.Join(cr.path, file.Name()))
	yamlFile, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var cfg TeamsConfig
	err = yaml.Unmarshal(yamlFile, &cfg)
	if err != nil {
		return nil, err
	}

	return cfg.Teams, nil
}

