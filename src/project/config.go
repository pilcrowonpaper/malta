package project

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/pilcrowonpaper/malta/build"
)

func ParseConfigFile() (ProjectConfig, error) {
	var unmarshalledConfig struct {
		Name          string `json:"name"`
		Description   string `json:"description"`
		Domain        string `json:"domain"`
		TwitterHandle string `json:"twitter"`
		Sidebar       []struct {
			Title string     `json:"title"`
			Pages [][]string `json:"pages"`
		} `json:"sidebar"`
		AssetHashing bool `json:"asset_hashing"`
		Alert        *struct {
			Message  string `json:"message"`
			LinkText string `json:"link_text"`
			LinkURL  string `json:"link_url"`
		} `json:"alert"`
	}
	var config ProjectConfig

	configJson, err := os.ReadFile("malta.config.json")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return config, &MissingConfigFileError{}
		}
		panic(err)
	}

	err = json.Unmarshal(configJson, &unmarshalledConfig)
	if err != nil {
		panic(err)
	}

	if unmarshalledConfig.Name == "" {
		return config, &InvalidConfigError{Field: "name"}
	}
	config.Name = unmarshalledConfig.Name

	if unmarshalledConfig.Domain == "" {
		return config, &InvalidConfigError{Field: "domain"}
	}
	config.Domain = unmarshalledConfig.Domain

	if unmarshalledConfig.Description == "" {
		return config, &InvalidConfigError{Field: "description"}
	}
	config.Description = unmarshalledConfig.Description

	config.TwitterHandle = unmarshalledConfig.TwitterHandle

	config.AssetHashing = unmarshalledConfig.AssetHashing

	for _, sidebarSection := range unmarshalledConfig.Sidebar {
		navSection := build.NavSection{Title: sidebarSection.Title, Pages: []build.NavPage{}}
		for _, sidebarSectionPage := range sidebarSection.Pages {
			navPage := build.NavPage{Title: sidebarSectionPage[0], Href: sidebarSectionPage[1]}
			navSection.Pages = append(navSection.Pages, navPage)
		}
		config.NavSections = append(config.NavSections, navSection)
	}

	if unmarshalledConfig.Alert != nil {
		config.Alert = &build.Alert{
			Message:  unmarshalledConfig.Alert.Message,
			LinkText: unmarshalledConfig.Alert.LinkText,
			LinkURL:  unmarshalledConfig.Alert.LinkURL,
		}
	}
	return config, nil
}

type ProjectConfig struct {
	Name          string
	Description   string
	Domain        string
	TwitterHandle string
	NavSections   []build.NavSection
	AssetHashing  bool
	Alert         *build.Alert
}

type MissingConfigFileError struct {
}

func (e *MissingConfigFileError) Error() string {
	return "missing config file"
}

type InvalidConfigError struct {
	Field string
}

func (e *InvalidConfigError) Error() string {
	return fmt.Sprintf("missing config: %s", e.Field)
}
