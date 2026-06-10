package core

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"
)

type provider struct {
	name       string
	fn         reflect.Value
	fnType     reflect.Type
	outType    reflect.Type
	paramTypes []reflect.Type
}

type Container struct {
	mu        sync.Mutex
	providers map[reflect.Type]*provider
	resolved  map[reflect.Type]any
	resolving map[reflect.Type]bool
}

func newContainer() *Container {
	return &Container{
		providers: make(map[reflect.Type]*provider),
		resolved:  make(map[reflect.Type]any),
		resolving: make(map[reflect.Type]bool),
	}
}

func (c *Container) addProvider(name string, fn any) error {
	fnVal := reflect.ValueOf(fn)
	fnType := fnVal.Type()

	if fnType.Kind() != reflect.Func {
		return fmt.Errorf("provider %q must be a function, got %T", name, fn)
	}

	numOut := fnType.NumOut()
	if numOut == 0 || numOut > 2 {
		return fmt.Errorf("provider %q must return 1 or 2 values, got %d", name, numOut)
	}

	outType := fnType.Out(0)

	if numOut == 2 {
		errType := reflect.TypeOf((*error)(nil)).Elem()
		if !fnType.Out(1).Implements(errType) {
			return fmt.Errorf("provider %q second return value must be error, got %s", name, fnType.Out(1))
		}
	}

	paramTypes := make([]reflect.Type, fnType.NumIn())
	for i := 0; i < fnType.NumIn(); i++ {
		paramTypes[i] = fnType.In(i)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.providers[outType]; exists {
		return fmt.Errorf("duplicate provider for type %s", outType)
	}

	c.providers[outType] = &provider{
		name:       name,
		fn:         fnVal,
		fnType:     fnType,
		outType:    outType,
		paramTypes: paramTypes,
	}
	return nil
}

func (c *Container) resolve(app *App, targetType reflect.Type) (any, error) {
	c.mu.Lock()
	if instance, ok := c.resolved[targetType]; ok {
		c.mu.Unlock()
		return instance, nil
	}

	if c.resolving[targetType] {
		c.mu.Unlock()
		return nil, fmt.Errorf("cycle detected: type %s depends on itself", targetType)
	}

	prov, hasProvider := c.providers[targetType]
	c.mu.Unlock()

	if !hasProvider {
		if val, ok := app.findServiceByType(targetType); ok {
			return val, nil
		}
		return nil, fmt.Errorf("no provider registered for type %s", targetType)
	}

	c.mu.Lock()
	c.resolving[targetType] = true
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.resolving, targetType)
		c.mu.Unlock()
	}()

	args := make([]reflect.Value, len(prov.paramTypes))
	for i, paramType := range prov.paramTypes {
		dep, err := c.resolve(app, paramType)
		if err != nil {
			return nil, fmt.Errorf("resolve dependency %s for provider %q: %w", paramType, prov.name, err)
		}
		args[i] = reflect.ValueOf(dep)
	}

	results := prov.fn.Call(args)

	instance := results[0].Interface()

	if len(results) == 2 && !results[1].IsNil() {
		return nil, fmt.Errorf("provider %q: %w", prov.name, results[1].Interface().(error))
	}

	if err := c.injectFields(app, instance); err != nil {
		return nil, fmt.Errorf("inject fields for %s: %w", targetType, err)
	}

	c.mu.Lock()
	c.resolved[targetType] = instance
	c.mu.Unlock()

	return instance, nil
}

func (c *Container) injectFields(app *App, instance any) error {
	val := reflect.ValueOf(instance)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil
	}

	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if _, ok := field.Tag.Lookup("inject"); !ok {
			continue
		}
		if !field.IsExported() {
			return fmt.Errorf("cannot inject unexported field %s.%s", typ.Name(), field.Name)
		}

		fieldType := field.Type
		dep, err := c.resolve(app, fieldType)
		if err != nil {
			return fmt.Errorf("inject field %s.%s (%s): %w", typ.Name(), field.Name, fieldType, err)
		}
		val.Field(i).Set(reflect.ValueOf(dep))
	}
	return nil
}

type Node struct {
	Type         reflect.Type
	Name         string
	Dependencies []Dependency
}

type Dependency struct {
	Type reflect.Type
	Name string
}

func (c *Container) Graph() []Node {
	c.mu.Lock()
	defer c.mu.Unlock()

	nodes := make([]Node, 0, len(c.providers))
	for _, prov := range c.providers {
		deps := make([]Dependency, len(prov.paramTypes))
		for i, pt := range prov.paramTypes {
			depName := ""
			if depProv, ok := c.providers[pt]; ok {
				depName = depProv.name
			}
			deps[i] = Dependency{Type: pt, Name: depName}
		}
		nodes = append(nodes, Node{
			Type:         prov.outType,
			Name:         prov.name,
			Dependencies: deps,
		})
	}
	return nodes
}

func (c *Container) HasProvider(targetType reflect.Type) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.providers[targetType]
	return ok
}

func (c *Container) ProviderNames() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	names := make([]string, 0, len(c.providers))
	for _, prov := range c.providers {
		names = append(names, prov.name)
	}
	return names
}

func providerName(fn any) string {
	name := runtimeFuncName(fn)
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		name = name[idx+1:]
	}
	if name == "" {
		name = "anonymous"
	}
	return name
}

func runtimeFuncName(fn any) string {
	fnVal := reflect.ValueOf(fn)
	if fnVal.Kind() != reflect.Func {
		return ""
	}
	rfn := runtime.FuncForPC(fnVal.Pointer())
	if rfn == nil {
		return ""
	}
	return rfn.Name()
}

func Provide(app *App, constructor any) error {
	name := providerName(constructor)
	return app.provide(name, constructor)
}

func ProvideNamed(app *App, name string, constructor any) error {
	return app.provide(name, constructor)
}

func Resolve[T any](app *App) (T, error) {
	var zero T
	targetType := reflect.TypeOf((*T)(nil)).Elem()

	instance, err := app.container.resolve(app, targetType)
	if err != nil {
		return zero, err
	}

	typed, ok := instance.(T)
	if !ok {
		return zero, fmt.Errorf("resolved type %T does not match requested type %s", instance, targetType)
	}
	return typed, nil
}

type Wire[T any] struct {
	app *App
}

func Lazy[T any](app *App) Wire[T] {
	return Wire[T]{app: app}
}

func (w Wire[T]) Get() (T, error) {
	return Resolve[T](w.app)
}

func (a *App) provide(name string, constructor any) error {
	return a.container.addProvider(name, constructor)
}

func (a *App) findServiceByType(targetType reflect.Type) (any, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, svc := range a.services {
		if reflect.TypeOf(svc) == targetType {
			return svc, true
		}
	}
	return nil, false
}

func (a *App) Container() *Container {
	return a.container
}

type GraphNode struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Dependencies []string `json:"dependencies,omitempty"`
}

func (c *Container) GraphJSON() ([]byte, error) {
	nodes := c.Graph()
	graphNodes := make([]GraphNode, len(nodes))
	for i, node := range nodes {
		deps := make([]string, len(node.Dependencies))
		for j, dep := range node.Dependencies {
			if dep.Name != "" {
				deps[j] = dep.Name
			} else {
				deps[j] = dep.Type.String()
			}
		}
		graphNodes[i] = GraphNode{
			Name:         node.Name,
			Type:         node.Type.String(),
			Dependencies: deps,
		}
	}
	sort.Slice(graphNodes, func(i, j int) bool {
		return graphNodes[i].Name < graphNodes[j].Name
	})
	return json.MarshalIndent(graphNodes, "", "  ")
}

func (c *Container) FormatGraph(w io.Writer) {
	nodes := c.Graph()
	if len(nodes) == 0 {
		fmt.Fprintln(w, "no providers registered")
		return
	}

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Name < nodes[j].Name
	})

	fmt.Fprintln(w, "Dependency Graph:")
	fmt.Fprintln(w, strings.Repeat("-", 50))
	for _, node := range nodes {
		if len(node.Dependencies) == 0 {
			fmt.Fprintf(w, "  %s (%s)\n", node.Name, node.Type)
		} else {
			fmt.Fprintf(w, "  %s (%s)\n", node.Name, node.Type)
			for _, dep := range node.Dependencies {
				depLabel := dep.Name
				if depLabel == "" {
					depLabel = dep.Type.String()
				}
				fmt.Fprintf(w, "    <- %s (%s)\n", depLabel, dep.Type)
			}
		}
	}
}

func WriteGraph(app *App, w io.Writer) {
	app.Container().FormatGraph(w)
}
