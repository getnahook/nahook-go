package management

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	nahook "github.com/getnahook/nahook-go"
)

func TestNew_InvalidToken(t *testing.T) {
	_, err := New("bad_token")
	if err == nil {
		t.Fatal("expected error for invalid management token")
	}
	if !strings.Contains(err.Error(), "must start with 'nhm_'") {
		t.Fatalf("unexpected error message: %s", err.Error())
	}
}

func TestNew_ValidToken(t *testing.T) {
	mgmt, err := New("nhm_test123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mgmt.Endpoints == nil {
		t.Error("expected Endpoints to be initialized")
	}
	if mgmt.EventTypes == nil {
		t.Error("expected EventTypes to be initialized")
	}
	if mgmt.Applications == nil {
		t.Error("expected Applications to be initialized")
	}
	if mgmt.Subscriptions == nil {
		t.Error("expected Subscriptions to be initialized")
	}
	if mgmt.PortalSessions == nil {
		t.Error("expected PortalSessions to be initialized")
	}
	if mgmt.Environments == nil {
		t.Error("expected Environments to be initialized")
	}
	if mgmt.Deliveries == nil {
		t.Error("expected Deliveries to be initialized")
	}
}

// ── Endpoints ───────────────────────────────────────────────────────────────

func TestEndpoints_List(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		assertPath(t, r, "/management/v1/workspaces/ws_123/endpoints")
		assertAuth(t, r, "nhm_test123")

		json.NewEncoder(w).Encode([]map[string]interface{}{
			{"id": "ep_1", "url": "https://example.com", "isActive": true, "type": "webhook"},
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	result, err := mgmt.Endpoints.List(context.Background(), "ws_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(result.Data))
	}
	if result.Data[0].ID != "ep_1" {
		t.Errorf("expected id ep_1, got %s", result.Data[0].ID)
	}
}

func TestEndpoints_Create(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "POST")
		assertPath(t, r, "/management/v1/workspaces/ws_123/endpoints")
		assertContentType(t, r)

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["url"] != "https://example.com/webhook" {
			t.Errorf("unexpected url: %v", body["url"])
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "ep_new", "url": "https://example.com/webhook", "isActive": true, "type": "webhook",
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	result, err := mgmt.Endpoints.Create(context.Background(), "ws_123", nahook.CreateEndpointOptions{
		URL: "https://example.com/webhook",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "ep_new" {
		t.Errorf("expected id ep_new, got %s", result.ID)
	}
}

func TestEndpoints_Get(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		assertPath(t, r, "/management/v1/workspaces/ws_123/endpoints/ep_1")

		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "ep_1", "url": "https://example.com", "isActive": true, "type": "webhook",
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	result, err := mgmt.Endpoints.Get(context.Background(), "ws_123", "ep_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "ep_1" {
		t.Errorf("expected id ep_1, got %s", result.ID)
	}
}

func TestEndpoints_Update(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "PATCH")
		assertPath(t, r, "/management/v1/workspaces/ws_123/endpoints/ep_1")
		assertContentType(t, r)

		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "ep_1", "url": "https://updated.com", "isActive": true, "type": "webhook",
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	url := "https://updated.com"
	result, err := mgmt.Endpoints.Update(context.Background(), "ws_123", "ep_1", nahook.UpdateEndpointOptions{
		URL: &url,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.URL != "https://updated.com" {
		t.Errorf("expected updated URL, got %s", result.URL)
	}
}

func TestEndpoints_Delete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "DELETE")
		assertPath(t, r, "/management/v1/workspaces/ws_123/endpoints/ep_1")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	err := mgmt.Endpoints.Delete(context.Background(), "ws_123", "ep_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ── Event Types ─────────────────────────────────────────────────────────────

func TestEventTypes_List(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		assertPath(t, r, "/management/v1/workspaces/ws_123/event-types")

		json.NewEncoder(w).Encode([]map[string]interface{}{
			{"id": "et_1", "name": "order.created"},
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	result, err := mgmt.EventTypes.List(context.Background(), "ws_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 1 {
		t.Fatalf("expected 1 event type, got %d", len(result.Data))
	}
	if result.Data[0].Name != "order.created" {
		t.Errorf("expected name order.created, got %s", result.Data[0].Name)
	}
}

func TestEventTypes_Create(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "POST")
		assertPath(t, r, "/management/v1/workspaces/ws_123/event-types")

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "order.created" {
			t.Errorf("unexpected name: %v", body["name"])
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "et_new", "name": "order.created",
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	result, err := mgmt.EventTypes.Create(context.Background(), "ws_123", nahook.CreateEventTypeOptions{
		Name: "order.created",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "et_new" {
		t.Errorf("expected id et_new, got %s", result.ID)
	}
}

func TestEventTypes_Get(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		assertPath(t, r, "/management/v1/workspaces/ws_123/event-types/et_1")

		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "et_1", "name": "order.created",
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	result, err := mgmt.EventTypes.Get(context.Background(), "ws_123", "et_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "et_1" {
		t.Errorf("expected id et_1, got %s", result.ID)
	}
}

func TestEventTypes_Update(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "PATCH")
		assertPath(t, r, "/management/v1/workspaces/ws_123/event-types/et_1")

		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "et_1", "name": "order.created", "description": "Updated",
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	desc := "Updated"
	result, err := mgmt.EventTypes.Update(context.Background(), "ws_123", "et_1", nahook.UpdateEventTypeOptions{
		Description: &desc,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "et_1" {
		t.Errorf("expected id et_1, got %s", result.ID)
	}
}

func TestEventTypes_Delete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "DELETE")
		assertPath(t, r, "/management/v1/workspaces/ws_123/event-types/et_1")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	err := mgmt.EventTypes.Delete(context.Background(), "ws_123", "et_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ── Applications ────────────────────────────────────────────────────────────

func TestApplications_List(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		assertPath(t, r, "/management/v1/workspaces/ws_123/applications")

		json.NewEncoder(w).Encode([]map[string]interface{}{
			{"id": "app_1", "name": "My App", "metadata": map[string]string{}},
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	result, err := mgmt.Applications.List(context.Background(), "ws_123", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 1 {
		t.Fatalf("expected 1 application, got %d", len(result.Data))
	}
}

func TestApplications_ListWithPagination(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")

		if r.URL.Query().Get("limit") != "10" {
			t.Errorf("expected limit=10, got %s", r.URL.Query().Get("limit"))
		}
		if r.URL.Query().Get("offset") != "20" {
			t.Errorf("expected offset=20, got %s", r.URL.Query().Get("offset"))
		}

		json.NewEncoder(w).Encode([]map[string]interface{}{})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	limit, offset := 10, 20
	_, err := mgmt.Applications.List(context.Background(), "ws_123", &nahook.ListOptions{
		Limit:  &limit,
		Offset: &offset,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplications_Create(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "POST")
		assertPath(t, r, "/management/v1/workspaces/ws_123/applications")

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "My App" {
			t.Errorf("unexpected name: %v", body["name"])
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "app_new", "name": "My App", "metadata": map[string]string{},
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	result, err := mgmt.Applications.Create(context.Background(), "ws_123", nahook.CreateApplicationOptions{
		Name: "My App",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "app_new" {
		t.Errorf("expected id app_new, got %s", result.ID)
	}
}

func TestApplications_Get(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		assertPath(t, r, "/management/v1/workspaces/ws_123/applications/app_1")

		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "app_1", "name": "My App", "metadata": map[string]string{},
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	result, err := mgmt.Applications.Get(context.Background(), "ws_123", "app_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "app_1" {
		t.Errorf("expected id app_1, got %s", result.ID)
	}
}

func TestApplications_Update(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "PATCH")
		assertPath(t, r, "/management/v1/workspaces/ws_123/applications/app_1")

		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "app_1", "name": "Updated App", "metadata": map[string]string{},
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	name := "Updated App"
	result, err := mgmt.Applications.Update(context.Background(), "ws_123", "app_1", nahook.UpdateApplicationOptions{
		Name: &name,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "Updated App" {
		t.Errorf("expected Updated App, got %s", result.Name)
	}
}

func TestApplications_Create_MaxEndpoints(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["maxEndpoints"] != float64(2) {
			t.Errorf("expected maxEndpoints 2, got %v", body["maxEndpoints"])
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "app_new", "name": "My App", "metadata": map[string]string{},
			"maxEndpoints": 2, "showEventTypes": true,
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	max := 2
	result, err := mgmt.Applications.Create(context.Background(), "ws_123", nahook.CreateApplicationOptions{
		Name:         "My App",
		MaxEndpoints: &max,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MaxEndpoints == nil || *result.MaxEndpoints != 2 {
		t.Errorf("expected MaxEndpoints 2, got %v", result.MaxEndpoints)
	}
}

func TestApplications_Create_ShowEventTypesFalse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["showEventTypes"] != false {
			t.Errorf("expected showEventTypes false, got %v", body["showEventTypes"])
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "app_new", "name": "My App", "metadata": map[string]string{},
			"maxEndpoints": nil, "showEventTypes": false,
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	show := false
	result, err := mgmt.Applications.Create(context.Background(), "ws_123", nahook.CreateApplicationOptions{
		Name:           "My App",
		ShowEventTypes: &show,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ShowEventTypes != false {
		t.Errorf("expected ShowEventTypes false, got %v", result.ShowEventTypes)
	}
}

func TestApplications_Create_OmitsUnsetCapFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if _, present := body["maxEndpoints"]; present {
			t.Errorf("expected maxEndpoints omitted, got %v", body["maxEndpoints"])
		}
		if _, present := body["showEventTypes"]; present {
			t.Errorf("expected showEventTypes omitted, got %v", body["showEventTypes"])
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "app_new", "name": "My App", "metadata": map[string]string{},
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	_, err := mgmt.Applications.Create(context.Background(), "ws_123", nahook.CreateApplicationOptions{Name: "My App"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplications_Update_MaxEndpointsExplicitNull(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		v, present := body["maxEndpoints"]
		if !present {
			t.Error("expected maxEndpoints present as explicit null, got omitted")
		}
		if v != nil {
			t.Errorf("expected maxEndpoints null, got %v", v)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "app_1", "name": "My App", "metadata": map[string]string{},
			"maxEndpoints": nil, "showEventTypes": true,
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	result, err := mgmt.Applications.Update(context.Background(), "ws_123", "app_1", nahook.UpdateApplicationOptions{
		MaxEndpoints: nahook.IntNull(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MaxEndpoints != nil {
		t.Errorf("expected MaxEndpoints nil, got %v", *result.MaxEndpoints)
	}
}

func TestApplications_Update_MaxEndpointsOmittedWhenNil(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if _, present := body["maxEndpoints"]; present {
			t.Errorf("expected maxEndpoints omitted, got %v", body["maxEndpoints"])
		}
		if _, present := body["showEventTypes"]; present {
			t.Errorf("expected showEventTypes omitted, got %v", body["showEventTypes"])
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "app_1", "name": "Renamed", "metadata": map[string]string{},
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	name := "Renamed"
	_, err := mgmt.Applications.Update(context.Background(), "ws_123", "app_1", nahook.UpdateApplicationOptions{
		Name: &name,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplications_Get_DefaultsShowEventTypesWhenAbsent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Response without the cap fields — older/cached payload shape.
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "app_1", "name": "My App", "metadata": map[string]string{},
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	result, err := mgmt.Applications.Get(context.Background(), "ws_123", "app_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MaxEndpoints != nil {
		t.Errorf("expected MaxEndpoints nil, got %v", *result.MaxEndpoints)
	}
	if !result.ShowEventTypes {
		t.Error("expected ShowEventTypes to default to true when absent (server default)")
	}
}

func TestApplications_Update_MaxEndpointsSetValue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["maxEndpoints"] != float64(5) {
			t.Errorf("expected maxEndpoints 5, got %v", body["maxEndpoints"])
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "app_1", "name": "My App", "metadata": map[string]string{},
			"maxEndpoints": 5, "showEventTypes": false,
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	result, err := mgmt.Applications.Update(context.Background(), "ws_123", "app_1", nahook.UpdateApplicationOptions{
		MaxEndpoints: nahook.IntValue(5),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MaxEndpoints == nil || *result.MaxEndpoints != 5 {
		t.Errorf("expected MaxEndpoints 5, got %v", result.MaxEndpoints)
	}
	if result.ShowEventTypes != false {
		t.Errorf("expected ShowEventTypes false, got %v", result.ShowEventTypes)
	}
}

func TestApplications_Delete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "DELETE")
		assertPath(t, r, "/management/v1/workspaces/ws_123/applications/app_1")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	err := mgmt.Applications.Delete(context.Background(), "ws_123", "app_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplications_ListEndpoints(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		assertPath(t, r, "/management/v1/workspaces/ws_123/applications/app_1/endpoints")

		json.NewEncoder(w).Encode([]map[string]interface{}{
			{"id": "ep_1", "url": "https://example.com", "isActive": true, "type": "webhook"},
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	result, err := mgmt.Applications.ListEndpoints(context.Background(), "ws_123", "app_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(result.Data))
	}
}

func TestApplications_CreateEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "POST")
		assertPath(t, r, "/management/v1/workspaces/ws_123/applications/app_1/endpoints")

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "ep_new", "url": "https://example.com/hook", "isActive": true, "type": "webhook",
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	result, err := mgmt.Applications.CreateEndpoint(context.Background(), "ws_123", "app_1", nahook.CreateEndpointOptions{
		URL: "https://example.com/hook",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "ep_new" {
		t.Errorf("expected id ep_new, got %s", result.ID)
	}
}

// ── Subscriptions ───────────────────────────────────────────────────────────

func TestSubscriptions_List(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		assertPath(t, r, "/management/v1/workspaces/ws_123/endpoints/ep_1/subscriptions")

		json.NewEncoder(w).Encode([]map[string]interface{}{
			{"id": "sub_1", "eventTypeId": "et_1", "eventTypeName": "order.created", "createdAt": "2026-04-10T12:00:00Z"},
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	result, err := mgmt.Subscriptions.List(context.Background(), "ws_123", "ep_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 1 {
		t.Fatalf("expected 1 subscription, got %d", len(result.Data))
	}
	if result.Data[0].EventTypeName != "order.created" {
		t.Errorf("expected eventTypeName order.created, got %s", result.Data[0].EventTypeName)
	}
}

func TestSubscriptions_Create(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "POST")
		assertPath(t, r, "/management/v1/workspaces/ws_123/endpoints/ep_1/subscriptions")

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		ids, ok := body["eventTypeIds"].([]interface{})
		if !ok || len(ids) != 1 || ids[0] != "et_1" {
			t.Errorf("unexpected eventTypeIds: %v", body["eventTypeIds"])
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"subscribed": 1,
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	result, err := mgmt.Subscriptions.Create(context.Background(), "ws_123", "ep_1", nahook.CreateSubscriptionOptions{
		EventTypeIDs: []string{"et_1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Subscribed != 1 {
		t.Errorf("expected subscribed 1, got %d", result.Subscribed)
	}
}

func TestSubscriptions_Delete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "DELETE")
		assertPath(t, r, "/management/v1/workspaces/ws_123/endpoints/ep_1/subscriptions/et_1")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	err := mgmt.Subscriptions.Delete(context.Background(), "ws_123", "ep_1", "et_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ── Portal Sessions ─────────────────────────────────────────────────────────

func TestPortalSessions_Create(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "POST")
		assertPath(t, r, "/management/v1/workspaces/ws_123/applications/app_1/portal")
		assertContentType(t, r)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"url":       "https://portal.nahook.com/session/abc",
			"code":      "abc123",
			"expiresAt": "2026-04-10T12:00:00Z",
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	result, err := mgmt.PortalSessions.Create(context.Background(), "ws_123", "app_1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Code != "abc123" {
		t.Errorf("expected code abc123, got %s", result.Code)
	}
	if result.URL != "https://portal.nahook.com/session/abc" {
		t.Errorf("unexpected URL: %s", result.URL)
	}
}

func TestPortalSessions_CreateWithOptions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		meta, ok := body["metadata"].(map[string]interface{})
		if !ok {
			t.Fatal("expected metadata in body")
		}
		if meta["userId"] != "user_123" {
			t.Errorf("unexpected metadata: %v", meta)
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"url":       "https://portal.nahook.com/session/abc",
			"code":      "abc123",
			"expiresAt": "2026-04-10T12:00:00Z",
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	result, err := mgmt.PortalSessions.Create(context.Background(), "ws_123", "app_1", &nahook.CreatePortalSessionOptions{
		Metadata: map[string]string{"userId": "user_123"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Code != "abc123" {
		t.Errorf("expected code abc123, got %s", result.Code)
	}
}

// ── Environments ───────────────────────────────────────────────────────────

func TestEnvironments_List(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		assertPath(t, r, "/management/v1/workspaces/ws_123/environments")
		assertAuth(t, r, "nhm_test123")

		json.NewEncoder(w).Encode([]map[string]interface{}{
			{"id": "env_1", "name": "Production", "slug": "production", "isDefault": true},
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	result, err := mgmt.Environments.List(context.Background(), "ws_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 1 {
		t.Fatalf("expected 1 environment, got %d", len(result.Data))
	}
	if result.Data[0].ID != "env_1" {
		t.Errorf("expected id env_1, got %s", result.Data[0].ID)
	}
	if result.Data[0].Name != "Production" {
		t.Errorf("expected name Production, got %s", result.Data[0].Name)
	}
	if result.Data[0].Slug != "production" {
		t.Errorf("expected slug production, got %s", result.Data[0].Slug)
	}
	if !result.Data[0].IsDefault {
		t.Error("expected isDefault to be true")
	}
}

func TestEnvironments_Create(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "POST")
		assertPath(t, r, "/management/v1/workspaces/ws_123/environments")
		assertContentType(t, r)

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "Staging" {
			t.Errorf("unexpected name: %v", body["name"])
		}
		if body["slug"] != "staging" {
			t.Errorf("unexpected slug: %v", body["slug"])
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "env_new", "name": "Staging", "slug": "staging", "isDefault": false,
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	result, err := mgmt.Environments.Create(context.Background(), "ws_123", nahook.CreateEnvironmentOptions{
		Name: "Staging",
		Slug: "staging",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "env_new" {
		t.Errorf("expected id env_new, got %s", result.ID)
	}
}

func TestEnvironments_Get(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		assertPath(t, r, "/management/v1/workspaces/ws_123/environments/env_1")

		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "env_1", "name": "Production", "slug": "production", "isDefault": true,
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	result, err := mgmt.Environments.Get(context.Background(), "ws_123", "env_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "env_1" {
		t.Errorf("expected id env_1, got %s", result.ID)
	}
}

func TestEnvironments_Update(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "PATCH")
		assertPath(t, r, "/management/v1/workspaces/ws_123/environments/env_1")
		assertContentType(t, r)

		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "env_1", "name": "Updated Env", "slug": "production", "isDefault": true,
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	name := "Updated Env"
	result, err := mgmt.Environments.Update(context.Background(), "ws_123", "env_1", nahook.UpdateEnvironmentOptions{
		Name: &name,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "Updated Env" {
		t.Errorf("expected Updated Env, got %s", result.Name)
	}
}

func TestEnvironments_Delete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "DELETE")
		assertPath(t, r, "/management/v1/workspaces/ws_123/environments/env_1")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	err := mgmt.Environments.Delete(context.Background(), "ws_123", "env_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnvironments_ListEventTypeVisibility(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		assertPath(t, r, "/management/v1/workspaces/ws_123/environments/env_1/event-types")

		json.NewEncoder(w).Encode([]map[string]interface{}{
			{"eventTypeName": "order.created", "published": true},
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	result, err := mgmt.Environments.ListEventTypeVisibility(context.Background(), "ws_123", "env_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 1 {
		t.Fatalf("expected 1 visibility entry, got %d", len(result.Data))
	}
	if result.Data[0].EventTypeName != "order.created" {
		t.Errorf("expected eventTypeName order.created, got %s", result.Data[0].EventTypeName)
	}
}

func TestEnvironments_SetEventTypeVisibility(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "PUT")
		assertPath(t, r, "/management/v1/workspaces/ws_123/environments/env_1/event-types/evt_1/visibility")
		assertContentType(t, r)

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["published"] != true {
			t.Errorf("unexpected published: %v", body["published"])
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"eventTypeName": "order.created", "published": true,
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	result, err := mgmt.Environments.SetEventTypeVisibility(context.Background(), "ws_123", "env_1", "evt_1", nahook.SetVisibilityOptions{
		Published: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.EventTypeName != "order.created" {
		t.Errorf("expected eventTypeName order.created, got %s", result.EventTypeName)
	}
	if !result.Published {
		t.Error("expected published to be true")
	}
}

// ── Error handling ──────────────────────────────────────────────────────────

func TestAPIError_Properties(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "rate_limited",
				"message": "Too many requests",
			},
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	_, err := mgmt.Endpoints.List(context.Background(), "ws_123")
	if err == nil {
		t.Fatal("expected error")
	}

	apiErr, ok := err.(*nahook.APIError)
	if !ok {
		t.Fatalf("expected *nahook.APIError, got %T", err)
	}
	if apiErr.Status != 429 {
		t.Errorf("expected status 429, got %d", apiErr.Status)
	}
	if !apiErr.IsRateLimited() {
		t.Error("expected IsRateLimited() to be true")
	}
	if !apiErr.IsRetryable() {
		t.Error("expected IsRetryable() to be true")
	}
	if apiErr.RetryAfter == nil || *apiErr.RetryAfter != 30 {
		t.Errorf("expected RetryAfter 30, got %v", apiErr.RetryAfter)
	}
}

func TestAPIError_AuthError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "unauthorized",
				"message": "Invalid token",
			},
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	_, err := mgmt.Endpoints.List(context.Background(), "ws_123")
	apiErr := err.(*nahook.APIError)
	if !apiErr.IsAuthError() {
		t.Error("expected IsAuthError() to be true")
	}
}

func TestAPIError_ValidationError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "validation_error",
				"message": "url is required",
			},
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	_, err := mgmt.Endpoints.Create(context.Background(), "ws_123", nahook.CreateEndpointOptions{})
	apiErr := err.(*nahook.APIError)
	if !apiErr.IsValidationError() {
		t.Error("expected IsValidationError() to be true")
	}
}

func TestAPIError_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "internal_error",
				"message": "Something went wrong",
			},
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	_, err := mgmt.Endpoints.List(context.Background(), "ws_123")
	apiErr := err.(*nahook.APIError)
	if !apiErr.IsRetryable() {
		t.Error("expected IsRetryable() to be true for 500")
	}
}

// ── URL encoding ────────────────────────────────────────────────────────────

func TestURLEncoding(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The path should have the workspace ID properly encoded
		expectedPath := "/management/v1/workspaces/ws%20with%20spaces/endpoints"
		if r.URL.RawPath != "" && r.URL.RawPath != expectedPath {
			// RawPath is only set when encoding differs from Path
			t.Logf("raw path: %s", r.URL.RawPath)
		}
		json.NewEncoder(w).Encode([]map[string]interface{}{})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	_, err := mgmt.Endpoints.List(context.Background(), "ws with spaces")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ── Deliveries ──────────────────────────────────────────────────────────────

func TestDeliveries_ListReturnsPaginatedDataAndNextCursor(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		assertPath(t, r, "/management/v1/workspaces/ws_abc/endpoints/ep_1/deliveries")

		json.NewEncoder(w).Encode(map[string]interface{}{
			"deliveries": []map[string]interface{}{
				{
					"id": "del_a", "idempotencyKey": "k1", "endpointId": "ep_1",
					"status": "delivered", "totalAttempts": 1,
					"firstAttemptAt": "2026-05-28T14:30:59Z",
					"deliveredAt":    "2026-05-28T14:30:59Z",
					"nextRetryAt":    nil,
					"hasPayload":     true,
					"createdAt":      "2026-05-28T14:30:59Z",
					"updatedAt":      "2026-05-28T14:30:59Z",
				},
				{
					"id": "del_b", "idempotencyKey": "k2", "endpointId": "ep_1",
					"status": "failed", "totalAttempts": 3,
					"firstAttemptAt": "2026-05-28T14:31:00Z",
					"deliveredAt":    nil,
					"nextRetryAt":    nil,
					"hasPayload":     false,
					"createdAt":      "2026-05-28T14:31:00Z",
					"updatedAt":      "2026-05-28T14:31:00Z",
				},
			},
			"nextCursor": "opaque-token-aaa",
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	result, err := mgmt.Deliveries.List(context.Background(), "ws_abc", "ep_1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 2 {
		t.Fatalf("expected 2 deliveries, got %d", len(result.Data))
	}
	if result.Data[0].ID != "del_a" {
		t.Errorf("expected first id del_a, got %s", result.Data[0].ID)
	}
	if result.Data[0].HasPayload != true {
		t.Errorf("expected hasPayload true, got %v", result.Data[0].HasPayload)
	}
	if result.NextCursor == nil {
		t.Fatal("expected non-nil NextCursor")
	}
	if *result.NextCursor != "opaque-token-aaa" {
		t.Errorf("expected nextCursor opaque-token-aaa, got %s", *result.NextCursor)
	}
}

func TestDeliveries_ListReturnsNullCursorWhenLastPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"deliveries": []map[string]interface{}{},
			"nextCursor": nil,
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	result, err := mgmt.Deliveries.List(context.Background(), "ws_abc", "ep_1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 0 {
		t.Errorf("expected empty data, got %d items", len(result.Data))
	}
	if result.NextCursor != nil {
		t.Errorf("expected NextCursor to be nil, got %v", *result.NextCursor)
	}
}

func TestDeliveries_ListForwardsQueryParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("limit"); got != "25" {
			t.Errorf("expected limit=25, got %s", got)
		}
		if got := r.URL.Query().Get("cursor"); got != "opaque-token-xyz" {
			t.Errorf("expected cursor=opaque-token-xyz, got %s", got)
		}
		if got := r.URL.Query().Get("status"); got != "failed" {
			t.Errorf("expected status=failed, got %s", got)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"deliveries": []interface{}{}, "nextCursor": nil})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	limit := 25
	_, err := mgmt.Deliveries.List(context.Background(), "ws_abc", "ep_1", &nahook.ListDeliveriesOptions{
		Limit:  &limit,
		Cursor: "opaque-token-xyz",
		Status: "failed",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeliveries_ListOmitsUnsetQueryParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Has("limit") {
			t.Error("expected limit to be omitted")
		}
		if q.Has("cursor") {
			t.Error("expected cursor to be omitted")
		}
		if q.Has("status") {
			t.Error("expected status to be omitted")
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"deliveries": []interface{}{}, "nextCursor": nil})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	_, err := mgmt.Deliveries.List(context.Background(), "ws_abc", "ep_1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeliveries_GetReturnsMetadataWithoutEnvelopeByDefault(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		assertPath(t, r, "/management/v1/workspaces/ws_abc/deliveries/del_a")
		if r.URL.Query().Has("include") {
			t.Error("expected include query param to be omitted")
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "del_a", "idempotencyKey": "k1", "endpointId": "ep_1",
			"status": "delivered", "totalAttempts": 1,
			"firstAttemptAt": "2026-05-28T14:30:59Z",
			"deliveredAt":    "2026-05-28T14:30:59Z",
			"nextRetryAt":    nil,
			"hasPayload":     true,
			"createdAt":      "2026-05-28T14:30:59Z",
			"updatedAt":      "2026-05-28T14:30:59Z",
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	delivery, err := mgmt.Deliveries.Get(context.Background(), "ws_abc", "del_a", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if delivery.ID != "del_a" {
		t.Errorf("expected id del_a, got %s", delivery.ID)
	}
	if !delivery.HasPayload {
		t.Errorf("expected hasPayload true, got false")
	}
	if delivery.Payload != nil {
		t.Errorf("expected Payload to be nil, got %+v", delivery.Payload)
	}
}

func TestDeliveries_GetWithIncludePayloadReturnsEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		assertPath(t, r, "/management/v1/workspaces/ws_abc/deliveries/del_a")
		if got := r.URL.Query().Get("include"); got != "payload" {
			t.Errorf("expected include=payload, got %s", got)
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "del_a", "idempotencyKey": "k1", "endpointId": "ep_1",
			"status": "delivered", "totalAttempts": 1,
			"firstAttemptAt": "2026-05-28T14:30:59Z",
			"deliveredAt":    "2026-05-28T14:30:59Z",
			"nextRetryAt":    nil,
			"hasPayload":     true,
			"createdAt":      "2026-05-28T14:30:59Z",
			"updatedAt":      "2026-05-28T14:30:59Z",
			"payload": map[string]interface{}{
				"status":      "available",
				"data":        map[string]interface{}{"orderId": "ord_123"},
				"contentType": "application/json",
			},
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	delivery, err := mgmt.Deliveries.Get(context.Background(), "ws_abc", "del_a", &nahook.GetDeliveryOptions{
		IncludePayload: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if delivery.Payload == nil {
		t.Fatal("expected non-nil Payload envelope")
	}
	if delivery.Payload.Status != "available" {
		t.Errorf("expected payload status available, got %s", delivery.Payload.Status)
	}
	if delivery.Payload.ContentType != "application/json" {
		t.Errorf("expected contentType application/json, got %s", delivery.Payload.ContentType)
	}
	// Decode the opaque Data payload to verify it round-trips.
	var decoded struct {
		OrderID string `json:"orderId"`
	}
	if err := json.Unmarshal(delivery.Payload.Data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal payload data: %v", err)
	}
	if decoded.OrderID != "ord_123" {
		t.Errorf("expected orderId ord_123, got %s", decoded.OrderID)
	}
}

func TestDeliveries_GetReturnsForbiddenEnvelopeForPlanGatedWorkspace(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "del_a", "idempotencyKey": "k1", "endpointId": "ep_1",
			"status": "delivered", "totalAttempts": 1,
			"firstAttemptAt": nil,
			"deliveredAt":    "2026-05-28T14:30:59Z",
			"nextRetryAt":    nil,
			"hasPayload":     true,
			"createdAt":      "2026-05-28T14:30:59Z",
			"updatedAt":      "2026-05-28T14:30:59Z",
			"payload": map[string]interface{}{
				"status": "forbidden",
			},
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	delivery, err := mgmt.Deliveries.Get(context.Background(), "ws_abc", "del_a", &nahook.GetDeliveryOptions{
		IncludePayload: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if delivery.Payload == nil {
		t.Fatal("expected non-nil Payload envelope")
	}
	if delivery.Payload.Status != "forbidden" {
		t.Errorf("expected payload status forbidden, got %s", delivery.Payload.Status)
	}
	if len(delivery.Payload.Data) != 0 {
		t.Errorf("expected Data to be empty for forbidden envelope, got %s", string(delivery.Payload.Data))
	}
	if delivery.Payload.ContentType != "" {
		t.Errorf("expected ContentType to be empty for forbidden envelope, got %s", delivery.Payload.ContentType)
	}
}

func TestDeliveries_GetAttemptsReturnsArray(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		assertPath(t, r, "/management/v1/workspaces/ws_abc/deliveries/del_a/attempts")

		json.NewEncoder(w).Encode([]map[string]interface{}{
			{
				"id": "att_1", "attemptNumber": 1, "status": "failed",
				"responseStatusCode": 502, "responseTimeMs": 142,
				"errorMessage": "Bad gateway", "createdAt": "2026-05-28T14:31:00Z",
			},
			{
				"id": "att_2", "attemptNumber": 2, "status": "success",
				"responseStatusCode": 200, "responseTimeMs": 88,
				"errorMessage": nil, "createdAt": "2026-05-28T14:31:30Z",
			},
		})
	}))
	defer srv.Close()

	mgmt := newTestClient(t, srv.URL)
	attempts, err := mgmt.Deliveries.GetAttempts(context.Background(), "ws_abc", "del_a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(attempts) != 2 {
		t.Fatalf("expected 2 attempts, got %d", len(attempts))
	}
	if attempts[0].AttemptNumber != 1 {
		t.Errorf("expected first attemptNumber 1, got %d", attempts[0].AttemptNumber)
	}
	if attempts[1].Status != "success" {
		t.Errorf("expected second status success, got %s", attempts[1].Status)
	}
	if attempts[0].ResponseStatusCode == nil || *attempts[0].ResponseStatusCode != 502 {
		t.Errorf("expected first responseStatusCode 502, got %v", attempts[0].ResponseStatusCode)
	}
	if attempts[1].ErrorMessage != nil {
		t.Errorf("expected second errorMessage nil, got %v", *attempts[1].ErrorMessage)
	}
}

// ── Helpers ─────────────────────────────────────────────────────────────────

func newTestClient(t *testing.T, baseURL string) *Management {
	t.Helper()
	mgmt, err := New("nhm_test123", WithBaseURL(baseURL))
	if err != nil {
		t.Fatalf("failed to create management client: %v", err)
	}
	return mgmt
}

func assertMethod(t *testing.T, r *http.Request, expected string) {
	t.Helper()
	if r.Method != expected {
		t.Errorf("expected method %s, got %s", expected, r.Method)
	}
}

func assertPath(t *testing.T, r *http.Request, expected string) {
	t.Helper()
	if r.URL.Path != expected {
		t.Errorf("expected path %s, got %s", expected, r.URL.Path)
	}
}

func assertAuth(t *testing.T, r *http.Request, token string) {
	t.Helper()
	expected := "Bearer " + token
	if r.Header.Get("Authorization") != expected {
		t.Errorf("expected auth %s, got %s", expected, r.Header.Get("Authorization"))
	}
}

func assertContentType(t *testing.T, r *http.Request) {
	t.Helper()
	if r.Header.Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
	}
}
