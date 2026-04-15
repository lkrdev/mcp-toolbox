package lookergetlookmldashboardexamples



import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"github.com/googleapis/mcp-toolbox/internal/embeddingmodels"
	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

const resourceType string = "looker-get-lookml-dashboard-examples"

//go:embed lookml_dashboard_examples.lookml
var examplesFile string

func init() {
	if !tools.Register(resourceType, newConfig) {
		panic(fmt.Sprintf("tool type %q already registered", resourceType))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (tools.ToolConfig, error) {
	actual := Config{Name: name}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type Config struct {
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description"`
	Source       string   `yaml:"source"`
	Type         string   `yaml:"type"`
	AuthRequired []string `yaml:"authRequired"`
}

func (cfg Config) ToolConfigType() string {
	return resourceType
}

type compatibleSource interface {
	UseClientAuthorization() bool
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	exampleParameter := parameters.NewStringParameterWithDefault("example_name", "all", "Specific example to fetch: business_pulse, brand_lookup, customer_lookup, web_analytics_overview, all")
	params := parameters.Parameters{exampleParameter}

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, params, nil)
	
	return Tool{
		Config:     cfg,
		Parameters: params,
		manifest: tools.Manifest{
			Description:  cfg.Description,
			Parameters:   params.Manifest(),
			AuthRequired: cfg.AuthRequired,
		},
		mcpManifest: mcpManifest,
	}, nil
}

type Tool struct {
	Config
	Parameters  parameters.Parameters `yaml:"parameters"`
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) EmbedParams(ctx context.Context, paramValues parameters.ParamValues, embeddingModelsMap map[string]embeddingmodels.EmbeddingModel) (parameters.ParamValues, error) {
	return parameters.EmbedParams(ctx, t.Parameters, paramValues, embeddingModelsMap, nil)
}

func (t Tool) Manifest() tools.Manifest {
	return t.manifest
}

func (t Tool) McpManifest() tools.McpManifest {
	return t.mcpManifest
}

func (t Tool) RequiresClientAuthorization(resourceMgr tools.SourceProvider) (bool, error) {
	return false, nil
}

func (t Tool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}

func (t Tool) GetAuthTokenHeaderName(resourceMgr tools.SourceProvider) (string, error) {
	return "", nil
}

func (t Tool) GetParameters() parameters.Parameters {
	return t.Parameters
}


func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	mapParams := params.AsMap()
	exampleName := mapParams["example_name"].(string)

	if exampleName == "all" {
		data := make(map[string]any)
		data["examples"] = examplesFile
		return data, nil
	}

	documents := strings.Split(examplesFile, "\n---")
	for _, doc := range documents {
		if strings.Contains(doc, fmt.Sprintf("dashboard: %s", exampleName)) {
			data := make(map[string]any)
			data["examples"] = strings.TrimSpace(doc)
			return data, nil
		}
	}

	return nil, util.NewAgentError(fmt.Sprintf("example %s not found", exampleName), nil)
}

