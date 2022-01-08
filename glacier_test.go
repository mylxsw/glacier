package glacier_test

import (
	"github.com/mylxsw/glacier"
	"github.com/mylxsw/glacier/infra"
	"sort"
	"strconv"
	"strings"
	"testing"
)

type Provider struct {
	ID      int
	Counter *Counter
}

func (p Provider) Register(app infra.Binder) {
	p.Counter.Append(p.ID)
}

type ProviderPriority struct {
	priority int
	ID       int
	Counter  *Counter
}

func (p ProviderPriority) Priority() int {
	return p.priority
}

func (p ProviderPriority) Register(app infra.Binder) {
	p.Counter.Append(p.ID)
}

type Counter struct {
	IDs []int
}

func (c *Counter) Append(id int) {
	c.IDs = append(c.IDs, id)
}

func (c *Counter) String() string {
	strs := make([]string, 0)
	for _, id := range c.IDs {
		strs = append(strs, strconv.Itoa(id))
	}

	return strings.Join(strs, "")
}

func TestProvidersSort(t *testing.T) {
	providers := make([]infra.Provider, 0)

	counter := &Counter{IDs: make([]int, 0)}

	providers = append(providers, ProviderPriority{Counter: counter, priority: 10000, ID: 0})
	providers = append(providers, Provider{Counter: counter, ID: 1})
	providers = append(providers, Provider{Counter: counter, ID: 2})
	providers = append(providers, Provider{Counter: counter, ID: 3})
	providers = append(providers, ProviderPriority{Counter: counter, priority: 0, ID: 4})
	providers = append(providers, ProviderPriority{Counter: counter, priority: 10, ID: 5})
	providers = append(providers, ProviderPriority{Counter: counter, priority: 20, ID: 6})
	providers = append(providers, ProviderPriority{Counter: counter, priority: 15, ID: 7})
	providers = append(providers, Provider{Counter: counter, ID: 8})
	providers = append(providers, Provider{Counter: counter, ID: 9})

	sort.Sort(glacier.Providers(providers))

	for _, p := range providers {
		p.Register(nil)
	}

	expected := "4576"
	if !strings.HasPrefix(counter.String(), expected) || !strings.HasSuffix(counter.String(), "0") {
		t.Errorf("test failed: expect has prefix %s but got %s", expected, counter.String())
	}
}
