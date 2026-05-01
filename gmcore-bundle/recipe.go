package gmcorebundle

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	gmcoreinstaller "gmcore-installer"
	"gopkg.in/yaml.v3"
)

type Recipe struct {
	Install   []RecipeStep `yaml:"install"`
	Uninstall []RecipeStep `yaml:"uninstall"`
}

type RecipeStep struct {
	Action               string            `yaml:"action"`
	From                 string            `yaml:"from"`
	To                   string            `yaml:"to"`
	Path                 string            `yaml:"path"`
	Content              string            `yaml:"content"`
	URL                  string            `yaml:"url"`
	Command              string            `yaml:"command"`
	Args                 []string          `yaml:"args"`
	Name                 string            `yaml:"name"`
	Manager              string            `yaml:"manager"`
	Optional             bool              `yaml:"optional"`
	Overwrite            bool              `yaml:"overwrite"`
	RequiresConfirmation bool              `yaml:"requires_confirmation"`
	Env                  map[string]string `yaml:"env"`
}

func LoadRecipe(manifest Manifest) (Recipe, error) {
	path := strings.TrimSpace(manifest.Recipe.File)
	if path == "" {
		path = "recipe.yaml"
	}
	path = filepath.Join(manifest.Root, path)
	data, err := os.ReadFile(path)
	if err != nil {
		return Recipe{}, err
	}
	var recipe Recipe
	if err := yaml.Unmarshal(data, &recipe); err != nil {
		return Recipe{}, err
	}
	return recipe, nil
}

func InstallRecipeSteps(manifest Manifest) ([]RecipeStep, error) {
	recipe, err := LoadRecipe(manifest)
	if err == nil {
		return recipe.Install, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return legacyInstallSteps(manifest), nil
}

func UninstallRecipeSteps(manifest Manifest) ([]RecipeStep, error) {
	recipe, err := LoadRecipe(manifest)
	if err == nil {
		return recipe.Uninstall, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return legacyUninstallSteps(manifest), nil
}

func ExecuteInstallRecipe(manifest Manifest, appRoot string) error {
	steps, err := InstallRecipeSteps(manifest)
	if err != nil {
		return err
	}
	return executeRecipeSteps(manifest.Root, appRoot, steps)
}

func ExecuteUninstallRecipe(manifest Manifest, appRoot string) error {
	steps, err := UninstallRecipeSteps(manifest)
	if err != nil {
		return err
	}
	return executeRecipeSteps(manifest.Root, appRoot, steps)
}

func legacyInstallSteps(manifest Manifest) []RecipeStep {
	steps := []RecipeStep{}
	if path := strings.TrimSpace(manifest.Install.Entities); path != "" {
		steps = append(steps, RecipeStep{Action: "copy_tree", From: path, To: filepath.ToSlash(filepath.Join("src", "Entities"))})
	}
	if path := strings.TrimSpace(manifest.Install.Examples); path != "" {
		steps = append(steps, RecipeStep{Action: "copy_tree", From: path, To: filepath.ToSlash(filepath.Join("src", "Examples", manifest.Package))})
	}
	if path := strings.TrimSpace(manifest.Install.Config); path != "" {
		steps = append(steps, RecipeStep{Action: "copy_tree", From: path, To: filepath.ToSlash(filepath.Join("config", "bundles", manifest.Package))})
	}
	return steps
}

func legacyUninstallSteps(manifest Manifest) []RecipeStep {
	steps := []RecipeStep{}
	if strings.TrimSpace(manifest.Install.Entities) != "" {
		steps = append(steps, RecipeStep{Action: "remove_tree", Path: filepath.ToSlash(filepath.Join("src", "Entities"))})
	}
	if strings.TrimSpace(manifest.Install.Examples) != "" {
		steps = append(steps, RecipeStep{Action: "remove_tree", Path: filepath.ToSlash(filepath.Join("src", "Examples", manifest.Package))})
	}
	if strings.TrimSpace(manifest.Install.Config) != "" {
		steps = append(steps, RecipeStep{Action: "remove_tree", Path: filepath.ToSlash(filepath.Join("config", "bundles", manifest.Package))})
	}
	return steps
}

func executeRecipeSteps(bundleRoot, appRoot string, steps []RecipeStep) error {
	return gmcoreinstaller.Runner{
		SourceRoot: bundleRoot,
		TargetRoot: appRoot,
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
		Stdin:      os.Stdin,
	}.RunSteps(toInstallerSteps(steps))
}

func executeRecipeStep(bundleRoot, appRoot string, step RecipeStep) error {
	return gmcoreinstaller.Runner{
		SourceRoot: bundleRoot,
		TargetRoot: appRoot,
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
		Stdin:      os.Stdin,
	}.RunStep(toInstallerStep(step))
}

func toInstallerSteps(steps []RecipeStep) []gmcoreinstaller.Step {
	out := make([]gmcoreinstaller.Step, 0, len(steps))
	for _, step := range steps {
		out = append(out, toInstallerStep(step))
	}
	return out
}

func toInstallerStep(step RecipeStep) gmcoreinstaller.Step {
	return gmcoreinstaller.Step{
		Action:               step.Action,
		From:                 step.From,
		To:                   step.To,
		Path:                 step.Path,
		Content:              step.Content,
		URL:                  step.URL,
		Command:              step.Command,
		Args:                 step.Args,
		Name:                 step.Name,
		Manager:              step.Manager,
		Optional:             step.Optional,
		Overwrite:            step.Overwrite,
		RequiresConfirmation: step.RequiresConfirmation,
		Env:                  step.Env,
	}
}
