package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func reflectType[T any]() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}

type Repo struct {
	DSN string
}

type Service struct {
	Repo *Repo
}

type Handler struct {
	Service *Service
}

func newRepo() *Repo {
	return &Repo{DSN: "postgres://localhost/test"}
}

func newService(repo *Repo) *Service {
	return &Service{Repo: repo}
}

func newHandler(svc *Service) *Handler {
	return &Handler{Service: svc}
}

func TestProvideAndResolve(t *testing.T) {
	app := New("test")
	if err := Provide(app, newRepo); err != nil {
		t.Fatalf("provide repo: %v", err)
	}

	repo, err := Resolve[*Repo](app)
	if err != nil {
		t.Fatalf("resolve repo: %v", err)
	}
	if repo.DSN != "postgres://localhost/test" {
		t.Fatalf("repo.DSN = %q, want %q", repo.DSN, "postgres://localhost/test")
	}
}

func TestResolveWithDependencies(t *testing.T) {
	app := New("test")
	if err := Provide(app, newRepo); err != nil {
		t.Fatalf("provide repo: %v", err)
	}
	if err := Provide(app, newService); err != nil {
		t.Fatalf("provide service: %v", err)
	}

	svc, err := Resolve[*Service](app)
	if err != nil {
		t.Fatalf("resolve service: %v", err)
	}
	if svc.Repo == nil {
		t.Fatal("service.Repo is nil")
	}
	if svc.Repo.DSN != "postgres://localhost/test" {
		t.Fatalf("service.Repo.DSN = %q, want %q", svc.Repo.DSN, "postgres://localhost/test")
	}
}

func TestResolveDeepDependencyChain(t *testing.T) {
	app := New("test")
	if err := Provide(app, newRepo); err != nil {
		t.Fatalf("provide repo: %v", err)
	}
	if err := Provide(app, newService); err != nil {
		t.Fatalf("provide service: %v", err)
	}
	if err := Provide(app, newHandler); err != nil {
		t.Fatalf("provide handler: %v", err)
	}

	handler, err := Resolve[*Handler](app)
	if err != nil {
		t.Fatalf("resolve handler: %v", err)
	}
	if handler.Service == nil {
		t.Fatal("handler.Service is nil")
	}
	if handler.Service.Repo == nil {
		t.Fatal("handler.Service.Repo is nil")
	}
	if handler.Service.Repo.DSN != "postgres://localhost/test" {
		t.Fatalf("DSN = %q, want %q", handler.Service.Repo.DSN, "postgres://localhost/test")
	}
}

func TestResolveSingleton(t *testing.T) {
	app := New("test")
	if err := Provide(app, newRepo); err != nil {
		t.Fatalf("provide repo: %v", err)
	}

	repo1, err := Resolve[*Repo](app)
	if err != nil {
		t.Fatalf("resolve repo1: %v", err)
	}
	repo2, err := Resolve[*Repo](app)
	if err != nil {
		t.Fatalf("resolve repo2: %v", err)
	}

	if repo1 != repo2 {
		t.Fatal("expected same instance (singleton), got different instances")
	}
}

func TestDuplicateProviderFails(t *testing.T) {
	app := New("test")
	if err := Provide(app, newRepo); err != nil {
		t.Fatalf("first provide: %v", err)
	}
	err := Provide(app, newRepo)
	if err == nil {
		t.Fatal("expected duplicate provider error")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("error = %q, want duplicate mention", err.Error())
	}
}

func TestMissingDependencyError(t *testing.T) {
	app := New("test")
	if err := Provide(app, newService); err != nil {
		t.Fatalf("provide service: %v", err)
	}

	_, err := Resolve[*Service](app)
	if err == nil {
		t.Fatal("expected missing dependency error")
	}
	if !strings.Contains(err.Error(), "no provider") {
		t.Fatalf("error = %q, want 'no provider' mention", err.Error())
	}
}

type CycleA struct {
	B *CycleB
}

type CycleB struct {
	A *CycleA
}

func newCycleA(b *CycleB) *CycleA {
	return &CycleA{B: b}
}

func newCycleB(a *CycleA) *CycleB {
	return &CycleB{A: a}
}

func TestCycleDetection(t *testing.T) {
	app := New("test")
	if err := Provide(app, newCycleA); err != nil {
		t.Fatalf("provide CycleA: %v", err)
	}
	if err := Provide(app, newCycleB); err != nil {
		t.Fatalf("provide CycleB: %v", err)
	}

	_, err := Resolve[*CycleA](app)
	if err == nil {
		t.Fatal("expected cycle detection error")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("error = %q, want 'cycle' mention", err.Error())
	}
}

type FieldInjected struct {
	Repo *Repo `inject:""`
}

func newFieldInjected() *FieldInjected {
	return &FieldInjected{}
}

func TestFieldInjection(t *testing.T) {
	app := New("test")
	if err := Provide(app, newRepo); err != nil {
		t.Fatalf("provide repo: %v", err)
	}
	if err := Provide(app, newFieldInjected); err != nil {
		t.Fatalf("provide FieldInjected: %v", err)
	}

	fi, err := Resolve[*FieldInjected](app)
	if err != nil {
		t.Fatalf("resolve FieldInjected: %v", err)
	}
	if fi.Repo == nil {
		t.Fatal("fi.Repo is nil, expected injection")
	}
	if fi.Repo.DSN != "postgres://localhost/test" {
		t.Fatalf("fi.Repo.DSN = %q, want %q", fi.Repo.DSN, "postgres://localhost/test")
	}
}

type unexportedInject struct {
	repo *Repo `inject:""`
}

func newUnexportedInject() *unexportedInject {
	return &unexportedInject{}
}

func TestUnexportedFieldInjectionFails(t *testing.T) {
	app := New("test")
	if err := Provide(app, newRepo); err != nil {
		t.Fatalf("provide repo: %v", err)
	}
	if err := Provide(app, newUnexportedInject); err != nil {
		t.Fatalf("provide: %v", err)
	}

	_, err := Resolve[*unexportedInject](app)
	if err == nil {
		t.Fatal("expected error for unexported field injection")
	}
	if !strings.Contains(err.Error(), "unexported") {
		t.Fatalf("error = %q, want 'unexported' mention", err.Error())
	}
}

func TestProviderWithErrorReturn(t *testing.T) {
	app := New("test")
	constructor := func() (*Repo, error) {
		return &Repo{DSN: "from-error-provider"}, nil
	}
	if err := Provide(app, constructor); err != nil {
		t.Fatalf("provide: %v", err)
	}

	repo, err := Resolve[*Repo](app)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if repo.DSN != "from-error-provider" {
		t.Fatalf("DSN = %q, want %q", repo.DSN, "from-error-provider")
	}
}

func TestProviderReturnsError(t *testing.T) {
	app := New("test")
	constructor := func() (*Repo, error) {
		return nil, errors.New("connection refused")
	}
	if err := Provide(app, constructor); err != nil {
		t.Fatalf("provide: %v", err)
	}

	_, err := Resolve[*Repo](app)
	if err == nil {
		t.Fatal("expected error from provider")
	}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Fatalf("error = %q, want 'connection refused'", err.Error())
	}
}

func TestProvideNonFunctionFails(t *testing.T) {
	app := New("test")
	err := Provide(app, "not a function")
	if err == nil {
		t.Fatal("expected error for non-function provider")
	}
	if !strings.Contains(err.Error(), "must be a function") {
		t.Fatalf("error = %q, want 'must be a function'", err.Error())
	}
}

func TestProvideNilFails(t *testing.T) {
	app := New("test")
	err := Provide(app, nil)
	if err == nil {
		t.Fatal("expected error for nil provider")
	}
	if !strings.Contains(err.Error(), "must be a function") {
		t.Fatalf("error = %q, want 'must be a function'", err.Error())
	}
}

func TestProvideZeroReturnFails(t *testing.T) {
	app := New("test")
	constructor := func() {}
	err := Provide(app, constructor)
	if err == nil {
		t.Fatal("expected error for zero-return provider")
	}
	if !strings.Contains(err.Error(), "must return 1 or 2 values") {
		t.Fatalf("error = %q, want return value count mention", err.Error())
	}
}

func TestProvideThreeReturnFails(t *testing.T) {
	app := New("test")
	constructor := func() (int, string, error) { return 0, "", nil }
	err := Provide(app, constructor)
	if err == nil {
		t.Fatal("expected error for three-return provider")
	}
	if !strings.Contains(err.Error(), "must return 1 or 2 values") {
		t.Fatalf("error = %q, want return value count mention", err.Error())
	}
}

func TestProvideSecondReturnNotError(t *testing.T) {
	app := New("test")
	constructor := func() (int, string) { return 0, "" }
	err := Provide(app, constructor)
	if err == nil {
		t.Fatal("expected error for non-error second return")
	}
	if !strings.Contains(err.Error(), "must be error") {
		t.Fatalf("error = %q, want 'must be error'", err.Error())
	}
}

func TestWireLazyResolution(t *testing.T) {
	app := New("test")
	if err := Provide(app, newRepo); err != nil {
		t.Fatalf("provide repo: %v", err)
	}

	wire := Lazy[*Repo](app)

	repo, err := wire.Get()
	if err != nil {
		t.Fatalf("wire.Get: %v", err)
	}
	if repo.DSN != "postgres://localhost/test" {
		t.Fatalf("DSN = %q, want %q", repo.DSN, "postgres://localhost/test")
	}
}

func TestWireLazyDeferredRegistration(t *testing.T) {
	app := New("test")

	wire := Lazy[*Repo](app)

	if err := Provide(app, newRepo); err != nil {
		t.Fatalf("provide repo: %v", err)
	}

	repo, err := wire.Get()
	if err != nil {
		t.Fatalf("wire.Get: %v", err)
	}
	if repo.DSN != "postgres://localhost/test" {
		t.Fatalf("DSN = %q, want %q", repo.DSN, "postgres://localhost/test")
	}
}

func TestResolveFromNamedService(t *testing.T) {
	app := New("test")
	repo := &Repo{DSN: "named-service"}
	if err := app.Register("myrepo", repo); err != nil {
		t.Fatalf("register: %v", err)
	}

	resolved, err := Resolve[*Repo](app)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved.DSN != "named-service" {
		t.Fatalf("DSN = %q, want %q", resolved.DSN, "named-service")
	}
}

func TestProviderPreferredOverNamedService(t *testing.T) {
	app := New("test")
	namedRepo := &Repo{DSN: "named"}
	if err := app.Register("myrepo", namedRepo); err != nil {
		t.Fatalf("register: %v", err)
	}

	providerRepo := func() *Repo { return &Repo{DSN: "provider"} }
	if err := Provide(app, providerRepo); err != nil {
		t.Fatalf("provide: %v", err)
	}

	resolved, err := Resolve[*Repo](app)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved.DSN != "provider" {
		t.Fatalf("DSN = %q, want %q (provider should take precedence)", resolved.DSN, "provider")
	}
}

func TestProvideNamed(t *testing.T) {
	app := New("test")
	constructor := func() *Repo { return &Repo{DSN: "custom"} }
	if err := ProvideNamed(app, "customRepo", constructor); err != nil {
		t.Fatalf("provide named: %v", err)
	}

	repo, err := Resolve[*Repo](app)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if repo.DSN != "custom" {
		t.Fatalf("DSN = %q, want %q", repo.DSN, "custom")
	}
}

func TestGraph(t *testing.T) {
	app := New("test")
	if err := Provide(app, newRepo); err != nil {
		t.Fatalf("provide repo: %v", err)
	}
	if err := Provide(app, newService); err != nil {
		t.Fatalf("provide service: %v", err)
	}
	if err := Provide(app, newHandler); err != nil {
		t.Fatalf("provide handler: %v", err)
	}

	graph := app.Container().Graph()
	if len(graph) != 3 {
		t.Fatalf("graph has %d nodes, want 3", len(graph))
	}

	nodeMap := make(map[string]Node)
	for _, node := range graph {
		nodeMap[node.Name] = node
	}

	if node, ok := nodeMap["newRepo"]; !ok {
		t.Error("missing newRepo node")
	} else if len(node.Dependencies) != 0 {
		t.Errorf("newRepo has %d deps, want 0", len(node.Dependencies))
	}

	if node, ok := nodeMap["newService"]; !ok {
		t.Error("missing newService node")
	} else if len(node.Dependencies) != 1 {
		t.Errorf("newService has %d deps, want 1", len(node.Dependencies))
	}

	if node, ok := nodeMap["newHandler"]; !ok {
		t.Error("missing newHandler node")
	} else if len(node.Dependencies) != 1 {
		t.Errorf("newHandler has %d deps, want 1", len(node.Dependencies))
	}
}

func TestResolveUnregisteredTypeFails(t *testing.T) {
	app := New("test")
	_, err := Resolve[*Repo](app)
	if err == nil {
		t.Fatal("expected error for unregistered type")
	}
	if !strings.Contains(err.Error(), "no provider") {
		t.Fatalf("error = %q, want 'no provider'", err.Error())
	}
}

type MultiDepService struct {
	Repo    *Repo
	Handler *Handler
}

func newMultiDepService(repo *Repo, handler *Handler) *MultiDepService {
	return &MultiDepService{Repo: repo, Handler: handler}
}

func TestMultipleDependencies(t *testing.T) {
	app := New("test")
	if err := Provide(app, newRepo); err != nil {
		t.Fatalf("provide repo: %v", err)
	}
	if err := Provide(app, newService); err != nil {
		t.Fatalf("provide service: %v", err)
	}
	if err := Provide(app, newHandler); err != nil {
		t.Fatalf("provide handler: %v", err)
	}
	if err := Provide(app, newMultiDepService); err != nil {
		t.Fatalf("provide multi-dep service: %v", err)
	}

	svc, err := Resolve[*MultiDepService](app)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if svc.Repo == nil {
		t.Fatal("svc.Repo is nil")
	}
	if svc.Handler == nil {
		t.Fatal("svc.Handler is nil")
	}
}

type InterfaceService interface {
	DoWork() string
}

type ConcreteService struct{}

func (c *ConcreteService) DoWork() string { return "done" }

func newConcreteService() InterfaceService {
	return &ConcreteService{}
}

func TestInterfaceProvider(t *testing.T) {
	app := New("test")
	if err := Provide(app, newConcreteService); err != nil {
		t.Fatalf("provide: %v", err)
	}

	svc, err := Resolve[InterfaceService](app)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if svc.DoWork() != "done" {
		t.Fatalf("DoWork() = %q, want %q", svc.DoWork(), "done")
	}
}

func TestHasProvider(t *testing.T) {
	app := New("test")
	if err := Provide(app, newRepo); err != nil {
		t.Fatalf("provide: %v", err)
	}

	container := app.Container()
	if !container.HasProvider(reflectType[*Repo]()) {
		t.Fatal("expected HasProvider to return true for *Repo")
	}
	if container.HasProvider(reflectType[*Service]()) {
		t.Fatal("expected HasProvider to return false for *Service")
	}
}

func TestFieldInjectionWithConstructorAndField(t *testing.T) {
	type Hybrid struct {
		Repo    *Repo
		Service *Service `inject:""`
	}

	app := New("test")
	if err := Provide(app, newRepo); err != nil {
		t.Fatalf("provide repo: %v", err)
	}
	if err := Provide(app, newService); err != nil {
		t.Fatalf("provide service: %v", err)
	}

	constructor := func(repo *Repo) *Hybrid {
		return &Hybrid{Repo: repo}
	}
	if err := Provide(app, constructor); err != nil {
		t.Fatalf("provide hybrid: %v", err)
	}

	hybrid, err := Resolve[*Hybrid](app)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if hybrid.Repo == nil {
		t.Fatal("hybrid.Repo is nil (constructor injection)")
	}
	if hybrid.Service == nil {
		t.Fatal("hybrid.Service is nil (field injection)")
	}
	if hybrid.Service.Repo != hybrid.Repo {
		t.Fatal("expected same Repo instance via singleton")
	}
}

func TestCycleDetectionThreeNodes(t *testing.T) {
	type A struct{}
	type B struct{}
	type C struct{}

	app := New("test")

	newA := func(c *C) *A { return &A{} }
	newB := func(a *A) *B { return &B{} }
	newC := func(b *B) *C { return &C{} }

	if err := Provide(app, newA); err != nil {
		t.Fatalf("provide A: %v", err)
	}
	if err := Provide(app, newB); err != nil {
		t.Fatalf("provide B: %v", err)
	}
	if err := Provide(app, newC); err != nil {
		t.Fatalf("provide C: %v", err)
	}

	_, err := Resolve[*A](app)
	if err == nil {
		t.Fatal("expected cycle detection error")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("error = %q, want 'cycle' mention", err.Error())
	}
}

func TestNoDependencyConstructor(t *testing.T) {
	app := New("test")
	constructor := func() *Repo {
		return &Repo{DSN: "no-deps"}
	}
	if err := Provide(app, constructor); err != nil {
		t.Fatalf("provide: %v", err)
	}

	repo, err := Resolve[*Repo](app)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if repo.DSN != "no-deps" {
		t.Fatalf("DSN = %q, want %q", repo.DSN, "no-deps")
	}
}

func TestFieldInjectionMissingDependency(t *testing.T) {
	type NeedsMissing struct {
		Service *Service `inject:""`
	}

	app := New("test")
	constructor := func() *NeedsMissing { return &NeedsMissing{} }
	if err := Provide(app, constructor); err != nil {
		t.Fatalf("provide: %v", err)
	}

	_, err := Resolve[*NeedsMissing](app)
	if err == nil {
		t.Fatal("expected error for missing injected dependency")
	}
	if !strings.Contains(err.Error(), "inject field") {
		t.Fatalf("error = %q, want 'inject field' mention", err.Error())
	}
}

func TestProviderNames(t *testing.T) {
	app := New("test")
	if err := Provide(app, newRepo); err != nil {
		t.Fatalf("provide repo: %v", err)
	}
	if err := Provide(app, newService); err != nil {
		t.Fatalf("provide service: %v", err)
	}

	names := app.Container().ProviderNames()
	if len(names) != 2 {
		t.Fatalf("got %d provider names, want 2", len(names))
	}

	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}
	if !nameSet["newRepo"] {
		t.Error("missing newRepo in provider names")
	}
	if !nameSet["newService"] {
		t.Error("missing newService in provider names")
	}
}

func TestGraphJSON(t *testing.T) {
	app := New("test")
	if err := Provide(app, newRepo); err != nil {
		t.Fatalf("provide repo: %v", err)
	}
	if err := Provide(app, newService); err != nil {
		t.Fatalf("provide service: %v", err)
	}

	data, err := app.Container().GraphJSON()
	if err != nil {
		t.Fatalf("GraphJSON: %v", err)
	}

	var nodes []GraphNode
	if err := json.Unmarshal(data, &nodes); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("got %d nodes, want 2", len(nodes))
	}

	nodeMap := make(map[string]GraphNode)
	for _, n := range nodes {
		nodeMap[n.Name] = n
	}

	if node, ok := nodeMap["newRepo"]; !ok {
		t.Error("missing newRepo")
	} else if len(node.Dependencies) != 0 {
		t.Errorf("newRepo has %d deps, want 0", len(node.Dependencies))
	}

	if node, ok := nodeMap["newService"]; !ok {
		t.Error("missing newService")
	} else if len(node.Dependencies) != 1 {
		t.Errorf("newService has %d deps, want 1", len(node.Dependencies))
	} else if node.Dependencies[0] != "newRepo" {
		t.Errorf("newService dep = %q, want %q", node.Dependencies[0], "newRepo")
	}
}

func TestFormatGraph(t *testing.T) {
	app := New("test")
	if err := Provide(app, newRepo); err != nil {
		t.Fatalf("provide repo: %v", err)
	}
	if err := Provide(app, newService); err != nil {
		t.Fatalf("provide service: %v", err)
	}

	var buf bytes.Buffer
	app.Container().FormatGraph(&buf)
	output := buf.String()

	if !strings.Contains(output, "Dependency Graph:") {
		t.Fatalf("missing header: %s", output)
	}
	if !strings.Contains(output, "newRepo") {
		t.Fatalf("missing newRepo: %s", output)
	}
	if !strings.Contains(output, "newService") {
		t.Fatalf("missing newService: %s", output)
	}
	if !strings.Contains(output, "<-") {
		t.Fatalf("missing dependency arrow: %s", output)
	}
}

func TestFormatGraphEmpty(t *testing.T) {
	app := New("test")
	var buf bytes.Buffer
	app.Container().FormatGraph(&buf)
	if !strings.Contains(buf.String(), "no providers registered") {
		t.Fatalf("expected empty message, got: %s", buf.String())
	}
}

func TestWriteGraph(t *testing.T) {
	app := New("test")
	if err := Provide(app, newRepo); err != nil {
		t.Fatalf("provide: %v", err)
	}

	var buf bytes.Buffer
	WriteGraph(app, &buf)
	if !strings.Contains(buf.String(), "newRepo") {
		t.Fatalf("WriteGraph missing newRepo: %s", buf.String())
	}
}
