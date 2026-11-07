package main

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Sanskarzz/tsunami/internal/connector"
	"github.com/Sanskarzz/tsunami/internal/inference"
	"github.com/Sanskarzz/tsunami/internal/kube"
	"github.com/Sanskarzz/tsunami/internal/platform"
)

// Optional configuration failures deny that route, not authenticated revocation.
// Errors exposed in status are closed codes; never configuration/secret bytes.
func configureRoutes(ctx context.Context, s *platform.Server) {
	s.SetRouteHealth("onboarding", "not_configured", "")
	validSetupPolicy := true
	for _, raw := range strings.Split(os.Getenv("TSUNAMI_ONBOARDING_DENIED_CIDRS"), ",") {
		if raw == "" {
			continue
		}
		p, err := netip.ParsePrefix(raw)
		if err != nil {
			validSetupPolicy = false
			break
		}
		s.SetupDenied = append(s.SetupDenied, p)
	}
	if file := os.Getenv("TSUNAMI_ONBOARDING_KEY_FILE"); file != "" {
		key, err := os.ReadFile(file)
		if err == nil && len(key) == 32 && validSetupPolicy {
			s.SetupKey = key
			s.SetRouteHealth("onboarding", "ready", "")
		} else {
			s.SetRouteHealth("onboarding", "unavailable", "setup_encryption_key_unavailable")
		}
	}
	s.Targets = map[string]platform.Target{}
	publicReady := true
	for _, name := range []string{"vulnerable", "fixed"} {
		raw := env("TSUNAMI_VULNERABLE_URL", "http://tsunami-vulnerable:8080")
		if name == "fixed" {
			raw = env("TSUNAMI_FIXED_URL", "http://tsunami-fixed:8080")
		}
		resolve, done := context.WithTimeout(ctx, 2*time.Second)
		target, err := platform.LabTarget(resolve, raw)
		done()
		if err != nil {
			publicReady = false
			continue
		}
		s.Targets[name] = target
	}
	if publicReady {
		s.SetRouteHealth("public", "ready", "")
	} else {
		s.Targets = nil
		s.SetRouteHealth("public", "unavailable", "fixture_configuration_unavailable")
	}
	s.SetRouteHealth("applications", "not_configured", "")
	if file := os.Getenv("TSUNAMI_APPLICATION_CONFIG_FILE"); file != "" {
		if err := s.LoadApplications(file); err != nil {
			s.Applications = nil
			s.SetRouteHealth("applications", "unavailable", "application_configuration_unavailable")
		} else {
			s.SetRouteHealth("applications", "ready", "")
		}
	}
	s.EnableCurlAdapter = os.Getenv("TSUNAMI_ENABLE_CURL_ADAPTER") == "true"
	s.SetRouteHealth("curl", "not_configured", "")
	if s.EnableCurlAdapter {
		if os.Getenv("TSUNAMI_RUNTIME_CLASS") != "gvisor" || os.Getenv("TSUNAMI_TOOL_IMAGE") == "" {
			s.EnableCurlAdapter = false
			s.SetRouteHealth("curl", "unavailable", "sandbox_or_tool_image_unavailable")
		} else {
			s.SetRouteHealth("curl", "ready", "")
		}
	}
	s.SetRouteHealth("private", "not_configured", "")
	if file := os.Getenv("TSUNAMI_PRIVATE_CONFIG_FILE"); file != "" {
		err := configurePrivate(s, file)
		if err != nil {
			s.PrivateTargets = nil
			s.SetRouteHealth("private", "unavailable", "private_configuration_unavailable")
		} else {
			s.SetRouteHealth("private", "ready", "")
		}
	}
	s.SetRouteHealth("model", "not_configured", "")
	if provider := os.Getenv("TSUNAMI_MODEL_PROVIDER"); provider != "" {
		cfg, err := configureInference(provider)
		if err != nil {
			s.Inference = nil
			s.SetRouteHealth("model", "unavailable", "model_configuration_unavailable")
		} else {
			s.Inference = cfg
			s.SetRouteHealth("model", "ready", "")
		}
	}
}

func configurePrivate(s *platform.Server, file string) error {
	if os.Getenv("TSUNAMI_ALLOW_PRIVATE_LAB") != "true" {
		return errors.New("private lab acknowledgement required")
	}
	cfg, targets, _, err := connector.LoadConfig(file)
	if err != nil {
		return err
	}
	return connector.ConfigureBroker(s, cfg, targets, env("TSUNAMI_SIGNING_KEY_FILE", "/run/connector/signing-key"))
}

func configureInference(provider string) (*inference.Config, error) {
	if os.Getenv("TSUNAMI_ALLOW_MODEL_CALLS") != "true" {
		return nil, errors.New("model acknowledgement required")
	}
	key := ""
	if provider == "responses" || provider == "openrouter-free" {
		var err error
		key, err = secretFile(env("TSUNAMI_MODEL_KEY_FILE", "/run/model/api-key"))
		if err != nil {
			return nil, err
		}
	}
	return inference.NewConfig(provider, os.Getenv("TSUNAMI_MODEL_ID"), env("TSUNAMI_MODEL_BASE_URL", "https://api.openai.com/v1/"), key)
}

func configureScheduler(ctx context.Context, s *platform.Server) {
	s.SetRouteHealth("scheduler", "unavailable", "scheduler_configuration_unavailable")
	broker, err := url.Parse(env("TSUNAMI_BROKER_URL", "http://tsunami-control:8080"))
	if err != nil || broker.Scheme != "http" || broker.Hostname() == "" || broker.User != nil || broker.RawQuery != "" || broker.Fragment != "" || broker.Path != "" {
		return
	}
	resolve, done := context.WithTimeout(ctx, 2*time.Second)
	ips, err := net.DefaultResolver.LookupNetIP(resolve, "ip", broker.Hostname())
	done()
	if err != nil || len(ips) == 0 {
		return
	}
	port := broker.Port()
	if port == "" {
		port = "8080"
	}
	broker.Host = net.JoinHostPort(ips[0].String(), port)
	scheduler, err := kube.InCluster(env("TSUNAMI_NAMESPACE", "tsunami"), env("TSUNAMI_IMAGE", "tsunami:dev"), broker.String(), os.Getenv("TSUNAMI_RUNTIME_CLASS"))
	if err != nil {
		return
	}
	s.Scheduler = scheduler
	s.SetRouteHealth("scheduler", "ready", "")
}
