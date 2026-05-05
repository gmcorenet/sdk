package gmcore_config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

type Recipe struct {
	Name         string
	Version      string
	ConfigFiles  []ConfigFile
	Dependencies []string
}

type ConfigFile struct {
	Path    string
	Content string
	Mode    os.FileMode
}

type RecipeProvider interface {
	Recipes() []Recipe
}

type RecipeRegistry struct {
	providers map[string]RecipeProvider
}

func NewRecipeRegistry() *RecipeRegistry {
	return &RecipeRegistry{
		providers: make(map[string]RecipeProvider),
	}
}

func (r *RecipeRegistry) Register(name string, provider RecipeProvider) {
	r.providers[name] = provider
}

func (r *RecipeRegistry) GetRecipes() []Recipe {
	var recipes []Recipe
	for _, provider := range r.providers {
		recipes = append(recipes, provider.Recipes()...)
	}
	return recipes
}

func (r *RecipeRegistry) GetRecipe(name string) (Recipe, bool) {
	provider, ok := r.providers[name]
	if !ok {
		return Recipe{}, false
	}
	recipes := provider.Recipes()
	if len(recipes) == 0 {
		return Recipe{}, false
	}
	return recipes[0], true
}

func (r *RecipeRegistry) RenderRecipe(recipe Recipe, vars map[string]string) ([]ConfigFile, error) {
	var result []ConfigFile

	for _, cf := range recipe.ConfigFiles {
		tmpl, err := template.New(cf.Path).Parse(cf.Content)
		if err != nil {
			return nil, fmt.Errorf("failed to parse template for %s: %w", cf.Path, err)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, vars); err != nil {
			return nil, fmt.Errorf("failed to render %s: %w", cf.Path, err)
		}

		result = append(result, ConfigFile{
			Path:    cf.Path,
			Content: buf.String(),
			Mode:    cf.Mode,
		})
	}

	return result, nil
}

func (r *RecipeRegistry) InstallRecipes(appPath string, vars map[string]string) error {
	recipes := r.GetRecipes()

	for _, recipe := range recipes {
		files, err := r.RenderRecipe(recipe, vars)
		if err != nil {
			return fmt.Errorf("failed to render recipe %s: %w", recipe.Name, err)
		}

		for _, file := range files {
			fullPath := filepath.Join(appPath, file.Path)

			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				return fmt.Errorf("failed to create directory for %s: %w", file.Path, err)
			}

			perms := file.Mode
			if perms == 0 {
				perms = 0644
			}

			if err := os.WriteFile(fullPath, []byte(file.Content), perms); err != nil {
				return fmt.Errorf("failed to write %s: %w", file.Path, err)
			}
		}
	}

	return nil
}

func (r *RecipeRegistry) GenerateManifest() string {
	var buf bytes.Buffer
	buf.WriteString("# GMCore SDK Recipes\n\n")
	buf.WriteString("| SDK | Version | Config Files |\n")
	buf.WriteString("|-----|---------|-------------|\n")

	for _, recipe := range r.GetRecipes() {
		buf.WriteString(fmt.Sprintf("| %s | %s | ", recipe.Name, recipe.Version))
		for i, cf := range recipe.ConfigFiles {
			if i > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(cf.Path)
		}
		buf.WriteString(" |\n")
	}

	return buf.String()
}

type SDKRecipes struct{}

func (s *SDKRecipes) RegisterAll(r *RecipeRegistry) {
	r.Register("gmcore-log", &LogRecipeProvider{})
	r.Register("gmcore-router", &RouterRecipeProvider{})
	r.Register("gmcore-cache", &CacheRecipeProvider{})
	r.Register("gmcore-security", &SecurityRecipeProvider{})
	r.Register("gmcore-messenger", &MessengerRecipeProvider{})
	r.Register("gmcore-orm", &ORMRecipeProvider{})
	r.Register("gmcore-session", &SessionRecipeProvider{})
	r.Register("gmcore-mailer", &MailerRecipeProvider{})
	r.Register("gmcore-i18n", &I18NRecipeProvider{})
	r.Register("gmcore-events", &EventsRecipeProvider{})
	r.Register("gmcore-transport", &TransportRecipeProvider{})
}
