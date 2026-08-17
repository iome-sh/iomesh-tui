package tui

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/iome-sh/iomesh-tui/internal/agent"
	"github.com/iome-sh/iomesh-tui/internal/iomesh"
	"github.com/iome-sh/iomesh-tui/internal/router"
)

func testRuntimeWithMesh(t *testing.T, mesh *iomesh.Client) *agent.Runtime {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
			return
		}
		_ = json.NewEncoder(w).Encode(router.ChatResponse{
			Choices: []router.Choice{{
				Message:      router.Message{Role: "assistant", Content: "pong"},
				FinishReason: "stop",
			}},
			Usage: router.Usage{TotalTokens: 3},
		})
	}))
	t.Cleanup(srv.Close)
	models := []router.ModelConfig{{
		Name: "m", BaseURL: srv.URL, ModelID: "m", APIKey: "k",
		CostTier: 1, MaxContext: 10000, Capabilities: []string{"fast", "coding"}, Priority: 1,
	}}
	rtr, err := router.New(models, "m")
	if err != nil {
		t.Fatal(err)
	}
	rt, err := agent.New(agent.Config{
		Workspace: t.TempDir(), SubagentsEnabled: true, Yolo: false,
	}, rtr, mesh, nil)
	if err != nil {
		t.Fatal(err)
	}
	return rt
}

func testMeshClient(t *testing.T, h http.HandlerFunc) *iomesh.Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return iomesh.New(iomesh.Config{
		Enabled: true, Endpoint: srv.URL, Tenant: "dept.engineering", OrgID: "org_test",
	}, nil)
}

func TestProbeDashboardConsume_NoStreams(t *testing.T) {
	c := testMeshClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/streams" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode([]any{})
	})
	events, names, reason := probeDashboardConsume(context.Background(), c)
	if reason != consumeReasonNoStreams {
		t.Fatalf("reason=%q", reason)
	}
	if len(names) != 0 || len(events) != 0 {
		t.Fatalf("names=%v events=%+v", names, events)
	}
	d := newDashboardState(true)
	d.applyConsume(events, names, reason)
	out := d.Render(ThemeDefault(), 100)
	if !strings.Contains(out, "no_streams") {
		t.Fatalf("missing no_streams:\n%s", out)
	}
	if strings.Contains(out, "P2 opened") {
		t.Fatalf("must not show eval seed:\n%s", out)
	}
	if strings.Contains(out, "PULSE") {
		t.Fatalf("must not invent PULSE from empty list:\n%s", out)
	}
	if !strings.Contains(out, "or: iomesh mesh streams --create --yes") {
		t.Fatalf("missing create CTA:\n%s", out)
	}
	if !strings.Contains(out, "Mesh routing") {
		t.Fatalf("must keep console Settings path:\n%s", out)
	}
}

func TestProbeDashboardConsume_EmptyStream(t *testing.T) {
	c := testMeshClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/streams":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"name": "github", "messages": 0},
				{"name": "OPERATIONAL_EVENTS", "messages": 0},
			})
		case strings.HasSuffix(r.URL.Path, "/messages"):
			_ = json.NewEncoder(w).Encode([]any{})
		default:
			http.NotFound(w, r)
		}
	})
	events, names, reason := probeDashboardConsume(context.Background(), c)
	if reason != consumeReasonEmptyStream {
		t.Fatalf("reason=%q names=%v", reason, names)
	}
	if len(names) != 2 || len(events) != 0 {
		t.Fatalf("names=%v events=%+v", names, events)
	}
	d := newDashboardState(true)
	d.applyConsume(events, names, reason)
	out := d.Render(ThemeDefault(), 100)
	if !strings.Contains(out, "empty_stream") {
		t.Fatalf("missing empty_stream:\n%s", out)
	}
	if strings.Contains(out, "P2 opened") {
		t.Fatalf("must not show eval seed:\n%s", out)
	}
}

func TestProbeDashboardConsume_ReplayDisabled(t *testing.T) {
	c := testMeshClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/streams":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"name": "OPERATIONAL_EVENTS"},
			})
		case strings.HasSuffix(r.URL.Path, "/messages"):
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":{"code":"replay_disabled","message":"stream replay requires tenant or flag"}}`))
		default:
			http.NotFound(w, r)
		}
	})
	events, names, reason := probeDashboardConsume(context.Background(), c)
	if reason != consumeReasonReplayDisabled {
		t.Fatalf("reason=%q names=%v", reason, names)
	}
	if len(events) != 0 {
		t.Fatalf("events=%+v", events)
	}
	d := newDashboardState(true)
	d.applyConsume(events, names, reason)
	out := d.Render(ThemeDefault(), 100)
	if !strings.Contains(out, "replay_disabled") {
		t.Fatalf("missing replay_disabled:\n%s", out)
	}
	if strings.Contains(out, "P2 opened") {
		t.Fatalf("must not show eval seed:\n%s", out)
	}
}

func TestProbeDashboardConsume_OneMessageNoP2(t *testing.T) {
	payload := base64.StdEncoding.EncodeToString([]byte(`{"type":"dept.github.push","ref":"main"}`))
	c := testMeshClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/streams":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"name": "OPERATIONAL_EVENTS"},
			})
		case strings.HasSuffix(r.URL.Path, "/messages"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"messages": []map[string]any{
					{
						"stream":    "OPERATIONAL_EVENTS",
						"seq":       1,
						"subject":   "dept.engineering.events.github",
						"payload":   payload,
						"timestamp": time.Date(2026, 8, 16, 14, 2, 11, 0, time.UTC),
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	})
	events, names, reason := probeDashboardConsume(context.Background(), c)
	if reason != consumeReasonConsumed {
		t.Fatalf("reason=%q", reason)
	}
	if len(names) != 1 || len(events) != 1 {
		t.Fatalf("names=%v events=%+v", names, events)
	}
	if events[0].Title == "P2 opened — checkout p95 1.8s" || events[0].Kind != kindOps {
		t.Fatalf("invented eval title or kind: %+v", events[0])
	}
	if events[0].Dept != "engineering" {
		t.Fatalf("dept=%q want engineering", events[0].Dept)
	}
	if !strings.Contains(events[0].Title, "dept.engineering.events.github") {
		t.Fatalf("title=%q", events[0].Title)
	}
	d := newDashboardState(true)
	d.applyConsume(events, names, reason)
	out := d.Render(ThemeDefault(), 100)
	if strings.Contains(out, "P2 opened") {
		t.Fatalf("must not mix eval seed:\n%s", out)
	}
	if !strings.Contains(out, "PULSE") {
		t.Fatalf("want PULSE after ≥1 decoded message:\n%s", out)
	}
	if !strings.Contains(out, "dept.engineering.events.github") {
		t.Fatalf("missing consume row:\n%s", out)
	}
}

func TestProbeDashboardConsume_BrokerUnavailable(t *testing.T) {
	c := testMeshClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	})
	events, _, reason := probeDashboardConsume(context.Background(), c)
	if reason != consumeReasonBrokerUnavailable {
		t.Fatalf("reason=%q", reason)
	}
	if len(events) != 0 {
		t.Fatalf("events=%+v", events)
	}
	d := newDashboardState(true)
	d.applyConsume(events, nil, reason)
	out := d.Render(ThemeDefault(), 100)
	if !strings.Contains(out, "broker_unavailable") {
		t.Fatalf("missing broker_unavailable:\n%s", out)
	}
	if strings.Contains(out, "P2 opened") {
		t.Fatalf("must not seed on transport error:\n%s", out)
	}
}

func TestHandleSlash_DashboardConsumeNoStreams(t *testing.T) {
	mesh := testMeshClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/streams" {
			_ = json.NewEncoder(w).Encode([]any{})
			return
		}
		http.NotFound(w, r)
	})
	rt := testRuntimeWithMesh(t, mesh)
	var out bytes.Buffer
	if quit, err := handleSlash(&out, runtimeAdapter{rt: rt}, "/dashboard"); quit || err != nil {
		t.Fatalf("quit=%v err=%v", quit, err)
	}
	s := out.String()
	if strings.Contains(s, "P2 opened") {
		t.Fatalf("must not show eval seed:\n%s", s)
	}
	if !strings.Contains(s, "no_streams") {
		t.Fatalf("missing no_streams:\n%s", s)
	}
}

func TestHandleSlash_DashboardPreviewStillEval(t *testing.T) {
	mesh := testMeshClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/streams" {
			_ = json.NewEncoder(w).Encode([]any{})
			return
		}
		http.NotFound(w, r)
	})
	rt := testRuntimeWithMesh(t, mesh)
	var out bytes.Buffer
	_, _ = handleSlash(&out, runtimeAdapter{rt: rt}, "/dashboard preview")
	if !strings.Contains(out.String(), "P2 opened — checkout p95 1.8s") {
		t.Fatalf("preview missing eval seed:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "EVAL") {
		t.Fatalf("preview missing EVAL:\n%s", out.String())
	}
}

func TestDashboardHelp_MentionsBrokerNotV52(t *testing.T) {
	h := dashboardHelp()
	if !strings.Contains(h, "mesh streams --messages") {
		t.Fatalf("help missing probe path:\n%s", h)
	}
	if !strings.Contains(h, "/v1") {
		t.Fatalf("help missing /v1:\n%s", h)
	}
	if !strings.Contains(h, "not portal GET /v52") {
		t.Fatalf("help must say not /v52:\n%s", h)
	}
}

func TestHeartbeatFromStreamMessage_Conservative(t *testing.T) {
	ev := heartbeatFromStreamMessage(iomesh.StreamMessage{
		Stream:    "OPERATIONAL_EVENTS",
		Subject:   "dept.engineering.events.github",
		Payload:   []byte(`{"type":"push"}`),
		Timestamp: time.Date(2026, 8, 16, 9, 1, 2, 0, time.UTC),
	})
	if ev.T != "09:01:02" || ev.Dept != "engineering" || ev.Kind != kindOps {
		t.Fatalf("%+v", ev)
	}
	if ev.Title != "dept.engineering.events.github" {
		t.Fatalf("title=%q", ev.Title)
	}
	if strings.Contains(ev.Title, "P2") || strings.Contains(ev.Title, "checkout") {
		t.Fatal("must not invent P2 checkout title")
	}
}
