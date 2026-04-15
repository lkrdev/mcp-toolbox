// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package lookercreatelookmldashboard

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/embeddingmodels"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/tools/looker/lookercommon"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"

	"github.com/looker-open-source/sdk-codegen/go/rtl"
	v4 "github.com/looker-open-source/sdk-codegen/go/sdk/v4"
)

const resourceType string = "looker-create-lookml-dashboard"

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

type compatibleSource interface {
	UseClientAuthorization() bool
	GetAuthTokenHeaderName() string
	LookerApiSettings() *rtl.ApiSettings
	GetLookerSDK(string) (*v4.LookerSDK, error)
}

type Config struct {
	Name         string                 `yaml:"name" validate:"required"`
	Type         string                 `yaml:"type" validate:"required"`
	Source       string                 `yaml:"source" validate:"required"`
	Description  string                 `yaml:"description" validate:"required"`
	AuthRequired []string               `yaml:"authRequired"`
	Annotations  *tools.ToolAnnotations `yaml:"annotations,omitempty"`
}

var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigType() string {
	return resourceType
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	projectIdParameter := parameters.NewStringParameter("project_id", "The id of the project")
	dashboardNameParameter := parameters.NewStringParameter("dashboard_name", "The name of the dashboard in LookML")
	titleParameter := parameters.NewStringParameter("title", "The title of the dashboard")
	elementsParameter := parameters.NewArrayParameterWithDefault("elements", []any{}, "Dashboard elements", parameters.NewMapParameter("element", "An element in the dashboard", ""))
	filtersParameter := parameters.NewArrayParameterWithDefault("filters", []any{}, "Dashboard filters", parameters.NewMapParameter("filter", "A filter in the dashboard", ""))
	params := parameters.Parameters{projectIdParameter, dashboardNameParameter, titleParameter, elementsParameter, filtersParameter}

	annotations := cfg.Annotations
	if annotations == nil {
		readOnlyHint := false
		annotations = &tools.ToolAnnotations{
			ReadOnlyHint: &readOnlyHint,
		}
	}

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, params, annotations)

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

var _ tools.Tool = Tool{}

type Tool struct {
	Config
	Parameters  parameters.Parameters `yaml:"parameters"`
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

type LookmlDashboard struct {
	Dashboard       string `yaml:"dashboard"`
	Title           string `yaml:"title"`
	Layout          string `yaml:"layout"`
	PreferredViewer string `yaml:"preferred_viewer"`
	Filters         []any  `yaml:"filters,omitempty"`
	Elements        []any  `yaml:"elements,omitempty"`
}

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Type)
	if err != nil {
		return nil, util.NewClientServerError("source used is not compatible with the tool", http.StatusInternalServerError, err)
	}

	sdk, err := source.GetLookerSDK(string(accessToken))
	if err != nil {
		return nil, util.NewClientServerError("error getting sdk", http.StatusInternalServerError, err)
	}

	mapParams := params.AsMap()
	projectId := mapParams["project_id"].(string)
	dashboardName := mapParams["dashboard_name"].(string)
	title := mapParams["title"].(string)
	elements := mapParams["elements"].([]any)
	filters := mapParams["filters"].([]any)

	// 1. Turn on dev mode
	devModeString := "dev"
	_, err = sdk.UpdateSession(v4.WriteApiSession{WorkspaceId: &devModeString}, source.LookerApiSettings())
	if err != nil {
		return nil, util.ProcessGeneralError(err)
	}
	prodModeString := "production"
	defer sdk.UpdateSession(v4.WriteApiSession{WorkspaceId: &prodModeString}, source.LookerApiSettings())

	// Optional: create dashboards directory (ignore error if already exists)
	_ = lookercommon.CreateProjectDirectory(sdk, projectId, "dashboards", source.LookerApiSettings())

	// 2. Create LookML dashboard content
	dash := LookmlDashboard{
		Dashboard:       dashboardName,
		Title:           title,
		Layout:          "newspaper",
		PreferredViewer: "dashboards-next",
		Filters:         filters,
		Elements:        elements,
	}

	lookmlBytes, err := yaml.Marshal([]LookmlDashboard{dash})
	if err != nil {
		return nil, util.NewClientServerError("error marshaling yaml", http.StatusInternalServerError, err)
	}

	// 3. Write it to dashboards/:new-relevant-unique-name.lookml
	filePath := fmt.Sprintf("dashboards/%s.lookml", dashboardName)
	fileReq := lookercommon.FileContent{
		Path:    filePath,
		Content: string(lookmlBytes),
	}

	err = lookercommon.CreateProjectFile(sdk, projectId, fileReq, source.LookerApiSettings())
	if err != nil {
		return nil, util.ProcessGeneralError(err)
	}

	// 4. Find personal space ID / folder ID using me API call
	mresp, err := sdk.Me("personal_folder_id", source.LookerApiSettings())
	if err != nil {
		return nil, util.ProcessGeneralError(err)
	}
	if mresp.PersonalFolderId == nil || *mresp.PersonalFolderId == "" {
		return nil, util.NewAgentError("user does not have a personal folder", nil)
	}
	folderId := *mresp.PersonalFolderId

	// 5. Copy the LookML dashboard to a UDD
	lookmlDashboardId := fmt.Sprintf("%s::%s", projectId, dashboardName)
	
	var udd v4.Dashboard
	path := fmt.Sprintf("/lookml_dashboards/%s/import/%s", url.PathEscape(lookmlDashboardId), url.PathEscape(folderId))
	err = sdk.AuthSession.Do(&udd, "POST", "/4.0", path, nil, nil, source.LookerApiSettings())
	if err != nil {
		return nil, util.ProcessGeneralError(err)
	}

	data := make(map[string]any)
	data["type"] = "text"
	data["text"] = fmt.Sprintf("Successfully created LookML dashboard and copied to UDD with ID %s", *udd.Id)
	if udd.Id != nil {
		data["udd_id"] = *udd.Id
	}
	data["personal_folder_id"] = folderId

	return data, nil
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
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Type)
	if err != nil {
		return false, err
	}
	return source.UseClientAuthorization(), nil
}

func (t Tool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}

func (t Tool) GetAuthTokenHeaderName(resourceMgr tools.SourceProvider) (string, error) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Type)
	if err != nil {
		return "", err
	}
	return source.GetAuthTokenHeaderName(), nil
}

func (t Tool) GetParameters() parameters.Parameters {
	return t.Parameters
}
