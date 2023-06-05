package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/plugins/base"
	"github.com/hashicorp/nomad/plugins/device"
	"github.com/hashicorp/nomad/plugins/shared/hclspec"
	"github.com/kr/pretty"
)

const (
	pluginName    = "rest-proxy-device"
	pluginVersion = "v0.1.0"

	fingerprintEndpoint = "/fingerprint"
	statsEndpoint       = "/stats"
	reserveEndpoint     = "/reserve"
	DefaultTimeout      = time.Second * 10
	DefaultInterval     = time.Second * 60
)

var (
	pluginInfo = &base.PluginInfoResponse{
		Type:              base.PluginTypeDevice,
		PluginApiVersions: []string{device.ApiVersion010},
		PluginVersion:     pluginVersion,
		Name:              pluginName,
	}

	configSpec = hclspec.NewObject(map[string]*hclspec.Spec{
		"address": hclspec.NewDefault(
			hclspec.NewAttr("address", "string", false),
			hclspec.NewLiteral("\"127.0.0.1:5656\""),
		),
		"fingerprint_period": hclspec.NewDefault(
			hclspec.NewAttr("fingerprint_period", "string", false),
			hclspec.NewLiteral("\"1m\""),
		),
	})
)

type Config struct {
	Address           string `codec:"address"`
	FingerprintPeriod string `codec:"fingerprint_period"`
}

type Plugin struct {
	logger            log.Logger
	address           *url.URL
	fingerprintPeriod time.Duration
}

var _ device.DevicePlugin = &Plugin{}

// NewPlugin returns a device plugin, used primarily by the main wrapper
//
// Plugin configuration isn't available yet, so there will typically be
// a limit to the initialization that can be performed at this point.
func NewPlugin(log log.Logger) *Plugin {
	return &Plugin{
		logger:            log.Named(pluginName),
		fingerprintPeriod: DefaultInterval,
	}
}

// PluginInfo returns information describing the plugin.
//
// This is called during Nomad client startup, while discovering and loading
// plugins.
func (d *Plugin) PluginInfo() (*base.PluginInfoResponse, error) {
	return pluginInfo, nil
}

// ConfigSchema returns the configuration schema for the plugin.
//
// This is called during Nomad client startup, immediately before parsing
// plugin config and calling SetConfig
func (d *Plugin) ConfigSchema() (*hclspec.Spec, error) {
	return configSpec, nil
}

// SetConfig is called by the client to pass the configuration for the plugin.
func (d *Plugin) SetConfig(c *base.Config) error {
	// decode the plugin config
	var config Config
	if err := base.MsgPackDecode(c.PluginConfig, &config); err != nil {
		return err
	}

	address, err := url.Parse(config.Address)
	if err != nil {
		return fmt.Errorf("invalid address %q: %v", config.Address, err)
	}
	period, err := time.ParseDuration(config.FingerprintPeriod)
	if err != nil {
		return fmt.Errorf("invalid period %q: %v", config.FingerprintPeriod, err)
	}
	d.address = address
	d.fingerprintPeriod = period
	d.logger.Info("config set", "config", log.Fmt("% #v", pretty.Formatter(config)))
	return nil
}

// Fingerprint streams detected devices.
// Messages should be emitted to the returned channel when there are changes
// to the devices or their health.
func (d *Plugin) Fingerprint(ctx context.Context) (<-chan *device.FingerprintResponse, error) {
	// Fingerprint returns a channel. The recommended way of organizing a plugin
	// is to pass that into a long-running goroutine and return the channel immediately.
	outCh := make(chan *device.FingerprintResponse)
	go d.doFingerprint(ctx, outCh)
	return outCh, nil
}

func (d *Plugin) doFingerprint(ctx context.Context, ch chan<- *device.FingerprintResponse) {
	defer close(ch)
	for {
		var response device.FingerprintResponse
		if err := d.jsonRequest(ctx, http.MethodGet, fingerprintEndpoint, nil, &response); err == nil {
			select {
			case ch <- &response:
			case <-ctx.Done():
				return
			}
		} else {
			d.logger.Error("failed making request", "error", err.Error())
		}
		select {
		case <-time.After(d.fingerprintPeriod):
		case <-ctx.Done():
			return
		}
	}
}

// Stats streams statistics for the detected devices.
// Messages should be emitted to the returned channel on the specified interval.
func (d *Plugin) Stats(ctx context.Context, interval time.Duration) (<-chan *device.StatsResponse, error) {
	// Similar to Fingerprint, Stats returns a channel. The recommended way of
	// organizing a plugin is to pass that into a long-running goroutine and
	// return the channel immediately.
	outCh := make(chan *device.StatsResponse)
	go d.doStats(ctx, outCh, interval)
	return outCh, nil
}

func (d *Plugin) doStats(ctx context.Context, ch chan<- *device.StatsResponse, interval time.Duration) {
	defer close(ch)
	for {
		var response device.StatsResponse
		if err := d.jsonRequest(ctx, http.MethodGet, statsEndpoint, nil, &response); err == nil {
			select {
			case ch <- &response:
			case <-ctx.Done():
				return
			}
		} else {
			d.logger.Error("failed making request", "error", err.Error())
		}
		select {
		case <-time.After(interval):
		case <-ctx.Done():
			return
		}
	}
}

func (d *Plugin) Reserve(deviceIDs []string) (*device.ContainerReservation, error) {
	var unmarshalled device.ContainerReservation
	err := d.jsonRequest(context.Background(), http.MethodPost, reserveEndpoint, deviceIDs, &unmarshalled)
	return &unmarshalled, err
}

func (d *Plugin) jsonRequest(ctx context.Context, method, endpoint string, input interface{}, output interface{}) error {
	var body io.Reader
	if input != nil {
		marshalled, err := json.Marshal(input)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %v", err)
		}
		body = bytes.NewReader(marshalled)
	}
	req, err := http.NewRequestWithContext(ctx, method, d.url(endpoint).String(), body)
	if err != nil {
		return fmt.Errorf("failed to build request: %v", err)
	}
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("request failed: got status %d", resp.StatusCode)
	}
	if output != nil {
		defer resp.Body.Close()
		// read body
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %v", err)
		}
		if err := json.Unmarshal(respBody, &output); err != nil {
			return fmt.Errorf("failed to unmarshal response: %v", err)
		}
	}
	return nil
}

func (d *Plugin) url(endpoint string) *url.URL {
	return d.address.JoinPath(endpoint)
}
