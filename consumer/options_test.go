package consumer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnitOptions(t *testing.T) {
	wantConfig := consumeCfg{
		tag:       "test",
		autoAck:   true,
		exclusive: true,
		noWait:    true,
		args:      map[string]interface{}{"some": "arg"},
	}

	ops := []Option{
		WithConsumerTag("test"),
		WithAutoAck(),
		WithExclusive(),
		WithNoWait(),
		WithArgs(map[string]interface{}{"some": "arg"}),
	}

	cfg := consumeCfg{
		tag:       "",
		autoAck:   false,
		exclusive: false,
		noWait:    false,
		args:      map[string]interface{}{},
	}
	for _, op := range ops {
		op(&cfg)
	}

	assert.Equal(t, wantConfig, cfg, "should create expected Consumer")
}
