//go:build wasip1

package main

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	gameap "github.com/gameap/gameap/pkg/proto"
	pluginproto "github.com/gameap/gameap/pkg/plugin/proto"
	hosthttp "github.com/gameap/gameap/pkg/plugin/sdk/http"
	"github.com/gameap/gameap/pkg/plugin/sdk/storage"
)

func main() {}

//go:embed frontend/dist/plugin.js
var frontendBundle []byte

//go:embed frontend/dist/gameap-palworld-plugin-frontend.css
var frontendStyles []byte

var (
	httpClient = hosthttp.NewHTTPService()
	store      = storage.NewStorageService()
)

type Plugin struct{}

type Config struct {
	BaseURL  string `json:"base_url"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func init() { pluginproto.RegisterPluginService(&Plugin{}) }

func (p *Plugin) GetInfo(context.Context, *pluginproto.GetInfoRequest) (*pluginproto.PluginInfo, error) {
	return &pluginproto.PluginInfo{Id: "palworldrest1", Name: "Palworld REST Control / 幻兽帕鲁管理", Version: "0.1.0", Description: "Native GameAP server tab for the official Palworld REST API", Author: "RANSUROTTO", ApiVersion: "1", RequiredPermissions: []string{"manage_servers"}}, nil
}

func (p *Plugin) Initialize(context.Context, *pluginproto.InitializeRequest) (*pluginproto.InitializeResponse, error) {
	return &pluginproto.InitializeResponse{Result: &pluginproto.Result{Success: true}}, nil
}

func (p *Plugin) Shutdown(context.Context, *pluginproto.ShutdownRequest) (*pluginproto.ShutdownResponse, error) {
	return &pluginproto.ShutdownResponse{Result: &pluginproto.Result{Success: true}}, nil
}

func (p *Plugin) GetSubscribedEvents(context.Context, *pluginproto.GetSubscribedEventsRequest) (*pluginproto.GetSubscribedEventsResponse, error) {
	return &pluginproto.GetSubscribedEventsResponse{}, nil
}

func (p *Plugin) HandleEvent(context.Context, *pluginproto.Event) (*pluginproto.EventResult, error) {
	return &pluginproto.EventResult{Handled: false}, nil
}

func (p *Plugin) GetServerAbilities(context.Context, *pluginproto.GetServerAbilitiesRequest) (*pluginproto.GetServerAbilitiesResponse, error) {
	return &pluginproto.GetServerAbilitiesResponse{Abilities: []*pluginproto.ServerAbility{
		{Name: "palworld-view", Title: "plugins.palworldrest1.abilities.view"},
		{Name: "palworld-announce", Title: "plugins.palworldrest1.abilities.announce"},
		{Name: "palworld-manage", Title: "plugins.palworldrest1.abilities.manage"},
	}}, nil
}

func (p *Plugin) GetFrontendBundle(context.Context, *pluginproto.GetFrontendBundleRequest) (*pluginproto.GetFrontendBundleResponse, error) {
	return &pluginproto.GetFrontendBundleResponse{Bundle: frontendBundle, HasBundle: true, Styles: frontendStyles, HasStyles: true}, nil
}

func (p *Plugin) GetHTTPRoutes(context.Context, *pluginproto.GetHTTPRoutesRequest) (*pluginproto.GetHTTPRoutesResponse, error) {
	routes := []*pluginproto.HTTPRoute{
		{Path: "/servers/{id}/config", Methods: []string{"GET"}, RequiresAuth: true, AdminOnly: false},
		{Path: "/servers/{id}/config", Methods: []string{"POST"}, RequiresAuth: true, AdminOnly: true},
	}
	for _, path := range []string{"info", "players", "settings", "metrics", "game-data"} {
		routes = append(routes, &pluginproto.HTTPRoute{Path: "/servers/{id}/" + path, Methods: []string{"GET"}, RequiresAuth: true})
	}
	for _, path := range []string{"announce", "kick", "ban", "unban", "save", "shutdown", "stop"} {
		routes = append(routes, &pluginproto.HTTPRoute{Path: "/servers/{id}/" + path, Methods: []string{"POST"}, RequiresAuth: true, AdminOnly: true})
	}
	return &pluginproto.GetHTTPRoutesResponse{Routes: routes}, nil
}

func (p *Plugin) HandleHTTPRequest(ctx context.Context, req *pluginproto.HTTPRequest) (*pluginproto.HTTPResponse, error) {
	serverID, err := strconv.ParseUint(req.PathParams["id"], 10, 64)
	if err != nil || serverID == 0 { return jsonError(http.StatusBadRequest, "invalid server id"), nil }
	action := strings.TrimPrefix(req.Path, "/servers/"+req.PathParams["id"]+"/")
	if action == "config" {
		if req.Method == http.MethodPost { return saveConfig(ctx, serverID, req.Body) }
		cfg, found, err := loadConfig(ctx, serverID)
		if err != nil { return jsonError(500, err.Error()), nil }
		return jsonResponse(200, map[string]any{"configured": found, "base_url": cfg.BaseURL, "username": cfg.Username}), nil
	}
	allowed := map[string]string{"info":"GET", "players":"GET", "settings":"GET", "metrics":"GET", "game-data":"GET", "announce":"POST", "kick":"POST", "ban":"POST", "unban":"POST", "save":"POST", "shutdown":"POST", "stop":"POST"}
	method, ok := allowed[action]
	if !ok || method != req.Method { return jsonError(404, "route not found"), nil }
	cfg, found, err := loadConfig(ctx, serverID)
	if err != nil { return jsonError(500, err.Error()), nil }
	if !found { return jsonError(409, "Palworld REST API is not configured for this server"), nil }
	return proxy(ctx, cfg, method, action, req.Body)
}

func saveConfig(ctx context.Context, serverID uint64, body []byte) (*pluginproto.HTTPResponse, error) {
	var cfg Config
	if err := json.Unmarshal(body, &cfg); err != nil { return jsonError(400, "invalid JSON"), nil }
	cfg.BaseURL = strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	cfg.Username = strings.TrimSpace(cfg.Username)
	u, err := url.Parse(cfg.BaseURL)
	if err != nil || u.Scheme != "http" || u.Hostname() != "host.docker.internal" || u.Port() != "8212" || u.Path != "/v1/api" { return jsonError(400, "base_url must be http://host.docker.internal:8212/v1/api"), nil }
	if cfg.Username == "" || cfg.Password == "" || len(cfg.Password) > 256 { return jsonError(400, "username and password are required"), nil }
	payload, _ := json.Marshal(cfg)
	entity := gameap.EntityType_ENTITY_TYPE_SERVER
	resp, err := store.Set(ctx, &storage.StorageSetRequest{Key: "palworld-rest-config", EntityType: &entity, EntityId: &serverID, Payload: payload})
	if err != nil || !resp.Success { return jsonError(500, "failed to store configuration"), nil }
	return jsonResponse(200, map[string]any{"success": true}), nil
}

func loadConfig(ctx context.Context, serverID uint64) (Config, bool, error) {
	entity := gameap.EntityType_ENTITY_TYPE_SERVER
	resp, err := store.Get(ctx, &storage.StorageGetRequest{Key: "palworld-rest-config", EntityType: &entity, EntityId: &serverID})
	if err != nil { return Config{}, false, err }
	if !resp.Found { return Config{BaseURL:"http://host.docker.internal:8212/v1/api", Username:"admin"}, false, nil }
	var cfg Config
	if err := json.Unmarshal(resp.Payload, &cfg); err != nil { return Config{}, false, err }
	return cfg, true, nil
}

func proxy(ctx context.Context, cfg Config, method, action string, body []byte) (*pluginproto.HTTPResponse, error) {
	headers := map[string]string{"Accept":"application/json", "Authorization":"Basic "+base64.StdEncoding.EncodeToString([]byte(cfg.Username+":"+cfg.Password))}
	if len(body) > 0 { headers["Content-Type"] = "application/json" }
	resp, err := httpClient.Fetch(ctx, &hosthttp.HTTPFetchRequest{Method: method, Url: cfg.BaseURL+"/"+action, Headers: headers, Body: body, TimeoutSeconds: 15})
	if err != nil { return jsonError(502, "Palworld REST request failed"), nil }
	if resp.Error != nil { return jsonError(502, *resp.Error), nil }
	// Never forward Palworld's 401/403 to the GameAP browser. GameAP's global
	// Axios interceptor interprets those codes as an expired panel session and
	// logs the administrator out. Authentication here belongs to the upstream
	// game server, so expose it as a plugin gateway error instead.
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return jsonError(http.StatusBadGateway, "Palworld REST authentication failed; check the server-side AdminPassword"), nil
	}
	if resp.StatusCode == http.StatusNotFound && action == "game-data" {
		return jsonError(http.StatusNotImplemented, "PalGameDataBridge GameData API is not enabled by this Palworld server build"), nil
	}
	if (resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusNotFound) && (action == "kick" || action == "ban" || action == "unban") {
		return jsonError(http.StatusUnprocessableEntity, "player or ban record was not found; use the exact userId from the player list"), nil
	}
	return &pluginproto.HTTPResponse{StatusCode: resp.StatusCode, Headers: map[string]string{"Content-Type":"application/json", "Cache-Control":"no-store"}, Body: resp.Body}, nil
}

func jsonResponse(status int, v any) *pluginproto.HTTPResponse { b, _ := json.Marshal(v); return &pluginproto.HTTPResponse{StatusCode:int32(status), Headers:map[string]string{"Content-Type":"application/json", "Cache-Control":"no-store"}, Body:b} }
func jsonError(status int, message string) *pluginproto.HTTPResponse { return jsonResponse(status, map[string]string{"error":message}) }
