.PHONY: test build local-demo model-fixture-demo image deploy verify-k8s collect-k8s isolation-check network-check sandbox-cluster configure-model-fixture model-fixture-eval live-eval disable-model free-eval free-stop-check private-deploy private-verify connector-network-check connector-tls-check catalog-local catalog-fixture-local catalog-verify catalog-private-verify inference-diagnostics engagement-local engagement-fixture-local toolkit-inventory engagement-verify engagement-private-verify application-setup application-verify tools-image tools-deploy tools-verify tools-network-check

IMAGE ?= tsunami:dev
TOOL_IMAGE ?= tsunami-tools:dev
CONTEXT ?= kind-kind
RUNTIME_CLASS ?=
TOOLKIT_PATH ?= ../vapt-toolkit

test:
	go test -race ./...

.PHONY: verify verify-toolkit
verify:
	go test -race -count=1 ./...
	go vet ./...
	python3 -B -m unittest discover -s scripts -p 'test_*.py'
	python3 -B scripts/check-examples.py

verify-toolkit:
	python3 -B scripts/check-examples.py --toolkit "$(TOOLKIT_PATH)"

.PHONY: verify-ui-metadata
verify-ui-metadata:
	node --test scripts/test_ui_metadata.mjs

.PHONY: verify-ui-activity
verify-ui-activity:
	node --test scripts/test_ui_metadata.mjs scripts/test_ui_activity.mjs

.PHONY: verify-ui-activity-browser
verify-ui-activity-browser:
	TSUNAMI_UI_BROWSER_TEST=1 go test -race -count=1 ./internal/webui -run TestLocalBrowserActivityAndStop -v

.PHONY: verify-ui-onboarding verify-ui-onboarding-browser
verify-ui-onboarding:
	node --test scripts/test_ui_metadata.mjs scripts/test_ui_activity.mjs scripts/test_ui_onboarding.mjs

.PHONY: verify-ui-findings
verify-ui-findings:
	node --test scripts/test_ui_metadata.mjs scripts/test_ui_activity.mjs scripts/test_ui_onboarding.mjs scripts/test_ui_findings.mjs

.PHONY: verify-ui-findings-browser
verify-ui-findings-browser:
	TSUNAMI_UI_FINDINGS_TEST=1 go test -race -count=1 ./internal/webui -run TestLocalBrowserFindingsAndEvidence -v

.PHONY: verify-ui-retest verify-ui-retest-browser
verify-ui-retest:
	node --test scripts/test_ui_metadata.mjs scripts/test_ui_activity.mjs scripts/test_ui_onboarding.mjs scripts/test_ui_findings.mjs scripts/test_ui_retest.mjs

verify-ui-retest-browser:
	TSUNAMI_UI_RETEST_TEST=1 go test -race -count=1 ./internal/webui -run TestLocalBrowserRetest -v

.PHONY: verify-ui-demo demo-rehearse
verify-ui-demo:
	node scripts/verify-ui.mjs --demo

demo-rehearse:
	node scripts/verify-ui.mjs --demo --rehearse

verify-ui-onboarding-browser:
	python3 -B scripts/verify-ui-onboarding.py

build:
	CGO_ENABLED=0 go build -trimpath -o bin/tsunami ./cmd/tsunami
	cp /etc/ssl/certs/ca-certificates.crt bin/ca-certificates.crt

local-demo:
	go run ./cmd/tsunami local-demo --out artifacts/local

model-fixture-demo:
	go run ./cmd/tsunami model-fixture-demo --out artifacts/model-fixture

image: build
	docker build -t $(IMAGE) .

deploy: image
	kind load docker-image $(IMAGE) --name $(patsubst kind-%,%,$(CONTEXT))
	@if [ -n "$(RUNTIME_CLASS)" ]; then kubectl --context $(CONTEXT) get runtimeclass $(RUNTIME_CLASS) >/dev/null; fi
	@if ! kubectl --context $(CONTEXT) get namespace tsunami >/dev/null 2>&1; then kubectl --context $(CONTEXT) create namespace tsunami; fi
	@if ! kubectl --context $(CONTEXT) -n tsunami get secret tsunami-operator >/dev/null 2>&1; then ./bin/tsunami bootstrap-secret | kubectl --context $(CONTEXT) -n tsunami apply -f -; fi
	kubectl --context $(CONTEXT) -n tsunami create --dry-run=client -f deploy/lab.yaml -o json | IMAGE="$(IMAGE)" RUNTIME_CLASS="$(RUNTIME_CLASS)" python3 scripts/render-lab.py | kubectl --context $(CONTEXT) -n tsunami apply -f -
	kubectl --context $(CONTEXT) -n tsunami rollout restart deployment/tsunami-control deployment/tsunami-vulnerable deployment/tsunami-fixed deployment/tsunami-tripwire
	kubectl --context $(CONTEXT) -n tsunami rollout status deployment/tsunami-vulnerable --timeout=90s
	kubectl --context $(CONTEXT) -n tsunami rollout status deployment/tsunami-fixed --timeout=90s
	kubectl --context $(CONTEXT) -n tsunami rollout status deployment/tsunami-control --timeout=90s
	kubectl --context $(CONTEXT) -n tsunami rollout status deployment/tsunami-tripwire --timeout=90s

verify-k8s:
	CONTEXT=$(CONTEXT) sh scripts/verify-k8s.sh

collect-k8s:
	go run ./cmd/tsunami collect-k8s --context $(CONTEXT) --out artifacts/kubernetes

isolation-check:
	CONTEXT=$(CONTEXT) sh scripts/isolation-check.sh

network-check:
	CONTEXT=$(CONTEXT) RUNTIME_CLASS=$(RUNTIME_CLASS) python3 scripts/network-check.py

sandbox-cluster:
	sh scripts/setup-sandbox.sh

configure-model-fixture:
	python3 scripts/configure-model.py --context $(CONTEXT) --provider fixture

model-fixture-eval:
	python3 scripts/evaluate-model.py --context $(CONTEXT) --fixture

live-eval:
	@test "$(ALLOW_MODEL_CALLS)" = "true" || { echo 'Set ALLOW_MODEL_CALLS=true explicitly; this target can incur model costs.' >&2; exit 1; }
	python3 scripts/evaluate-model.py --context $(CONTEXT) --allow-model-calls

disable-model:
	python3 scripts/configure-model.py --context $(CONTEXT) --provider disabled

free-eval:
	python3 scripts/evaluate-model.py --context $(CONTEXT) --free-only --allow-model-calls

free-stop-check:
	python3 scripts/evaluate-model.py --context $(CONTEXT) --free-only --allow-model-calls --stop-check

private-deploy:
	@test "$(CONTEXT)" = "kind-tsunami-lab" || { echo 'Use the dedicated kind-tsunami-lab context.' >&2; exit 1; }
	kubectl --context $(CONTEXT) get runtimeclass gvisor >/dev/null
	$(MAKE) image
	kind load docker-image $(IMAGE) --name tsunami-lab
	@if ! kubectl --context $(CONTEXT) -n tsunami get deployment tsunami-control >/dev/null 2>&1; then $(MAKE) deploy CONTEXT=$(CONTEXT) RUNTIME_CLASS=gvisor; fi
	kubectl --context $(CONTEXT) -n tsunami rollout restart deployment/tsunami-vulnerable deployment/tsunami-fixed deployment/tsunami-tripwire
	kubectl --context $(CONTEXT) -n tsunami rollout status deployment/tsunami-vulnerable --timeout=60s
	kubectl --context $(CONTEXT) -n tsunami rollout status deployment/tsunami-fixed --timeout=60s
	kubectl --context $(CONTEXT) -n tsunami rollout status deployment/tsunami-tripwire --timeout=60s
	CONTEXT=$(CONTEXT) IMAGE="$(IMAGE)" python3 scripts/configure-private.py

private-verify:
	CONTEXT=$(CONTEXT) python3 scripts/verify-private.py

connector-network-check:
	CONTEXT=$(CONTEXT) python3 scripts/connector-network-check.py

connector-tls-check: build
	CONTEXT=$(CONTEXT) python3 scripts/verify-connector-tls.py

catalog-local:
	go run ./cmd/tsunami catalog-local

catalog-fixture-local:
	go run ./cmd/tsunami catalog-local --fixture --out artifacts/assessment-fixture

catalog-verify:
	python3 scripts/evaluate-catalog.py --context $(CONTEXT)

catalog-private-verify:
	python3 scripts/evaluate-catalog.py --context $(CONTEXT) --private

inference-diagnostics:
	go run ./cmd/tsunami diagnose-inference --out artifacts/inference-diagnostics

engagement-local:
	go run ./cmd/tsunami engagement-local

engagement-fixture-local:
	go run ./cmd/tsunami engagement-local --fixture --out artifacts/engagements/offline

toolkit-inventory:
	python3 scripts/toolkit.py inventory --toolkit "$(TOOLKIT_PATH)" --out artifacts/engagements/toolkit-inventory.json

engagement-verify:
	python3 scripts/verify-engagement.py --context $(CONTEXT)

engagement-private-verify:
	python3 scripts/verify-engagement.py --context $(CONTEXT) --private

application-setup: build
	python3 scripts/setup-vampi.py --context $(CONTEXT)

application-verify:
	python3 scripts/verify-engagement.py --context $(CONTEXT) --application

# The scratch image remains the default. This adds only one reviewed CLI adapter.
tools-image: build
	docker build -f Dockerfile.tools -t $(TOOL_IMAGE) .

tools-deploy: image tools-image
	@test "$(CONTEXT)" = "kind-tsunami-lab" || { echo 'Use the dedicated kind-tsunami-lab context.' >&2; exit 1; }
	kind load docker-image $(IMAGE) $(TOOL_IMAGE) --name tsunami-lab
	python3 scripts/configure-tools.py --context $(CONTEXT) --tool-image "$(TOOL_IMAGE)" --native-image "$(IMAGE)"

tools-verify:
	python3 scripts/verify-engagement.py --context $(CONTEXT) --application --executor curl-http/v1

tools-network-check:
	CONTEXT=$(CONTEXT) python3 scripts/tool-network-check.py

.PHONY: stop-lab-setup stop-lab-verify
stop-lab-setup:
	python3 scripts/setup-stop-lab.py --context $(CONTEXT)

stop-lab-verify:
	python3 scripts/verify-stop-lab.py --context $(CONTEXT)

.PHONY: journey-verify
journey-verify: build
	python3 scripts/verify-journey.py --context $(CONTEXT)

.PHONY: property-lab-setup property-verify
property-lab-setup: build
	python3 scripts/setup-property-lab.py --context $(CONTEXT)

property-verify: build
	python3 scripts/verify-journey.py --context $(CONTEXT) --profile property

# Explicitly scoped to the already provisioned dedicated owned Kind lab.
.PHONY: showcase-check showcase-prepare showcase showcase-extended
showcase-check:
	python3 scripts/showcase.py check

showcase-prepare:
	python3 scripts/showcase.py prepare --include-property

showcase:
	python3 scripts/showcase.py run

showcase-extended:
	python3 scripts/showcase.py run --include-property

.PHONY: ui-deploy
ui-deploy:
	python3 scripts/configure-ui.py
