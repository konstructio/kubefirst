package generate

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed scaffold
var scaffoldFS embed.FS

type ScaffoldData struct {
	AppName        string
	DeploymentName string
	Description    string
	Environment    string
	Namespace      string
}

func AppScaffold(appName string, environments []string, outputPath string) error {
	for _, env := range environments {
		files, err := generateAppScaffoldEnvironmentFiles(appName, env)
		if err != nil {
			return err
		}

		if err := files.Save(filepath.Join(outputPath, env)); err != nil {
			return err
		}
	}
	return nil
}

func generateAppScaffoldEnvironmentFiles(appName, environment string) (*Files, error) {
	rootDir := "scaffold/"

	tpl, err := template.New("tpl").ParseFS(scaffoldFS, rootDir+"*.yaml", rootDir+"**/*.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	data := ScaffoldData{
		AppName:        appName,
		DeploymentName: fmt.Sprintf("%s-environment-%s", environment, appName),
		Description:    fmt.Sprintf("%s example application", appName),
		Environment:    environment,
		Namespace:      environment,
	}

	files := &Files{}
	err = fs.WalkDir(scaffoldFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("error walking directory: %w", err)
		}

		if d.IsDir() {
			return nil
		}

		// Get the file name without the root path
		file, _ := strings.CutPrefix(path, rootDir)

		// Parse any template variables in the file name
		fileTpl, err := tpl.Parse(file)
		if err != nil {
			return fmt.Errorf("error parsing file name: %w", err)
		}

		var fileNameOutput bytes.Buffer
		if err := fileTpl.Execute(&fileNameOutput, data); err != nil {
			return fmt.Errorf("error executing file name: %w", err)
		}

		// Parse the contents of the file
		var fileContent bytes.Buffer
		if err := tpl.ExecuteTemplate(&fileContent, d.Name(), data); err != nil {
			return fmt.Errorf("error executing template: %w", err)
		}

		// Now store everything for output
		files.Add(fileNameOutput.String(), fileContent)

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking directory: %w", err)
	}

	return files, nil
}
