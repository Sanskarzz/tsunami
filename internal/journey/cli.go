// Package journey is the operator CLI for the configured-application lab.
// It uses the existing authenticated API; it never contacts a target directly.
package journey

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/netip"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/Sanskarzz/tsunami/internal/application"
	"github.com/Sanskarzz/tsunami/internal/demo"
	"github.com/Sanskarzz/tsunami/internal/platform"
	"github.com/Sanskarzz/tsunami/internal/toolcontract"
)

const help = `Usage: tsunami assess COMMAND [flags]

  applications  List applications already configured for this operator's tenant
  plan          Create a draft and show its exact scope; no target requests
  engagements   Recover engagement IDs after an interrupted command
  show          Show a saved plan, approval, progress and child run IDs
  approve       Approve a draft using the scope digest you reviewed
  start         Start its built-in procedure once, using scripted Mecatl
  status        Inspect one run without printing response bodies
  watch         Watch permission, execution, activity and cleanup (read-only)
  stop          Revoke an engagement or one run; no execution dependency needed
  export        Save an evidence bundle to a new private directory

Every command uses --url and --token-file. HTTP is allowed only on loopback;
use HTTPS for a remote API. Credentials are never command arguments.
Use COMMAND --help for flags. --json emits structured output.
This is the owned synthetic HTTP lab, not URL-only onboarding or a customer service.
`

var runID = regexp.MustCompile(`^[a-f0-9]{32}$`)
var digest = regexp.MustCompile(`^[a-f0-9]{64}$`)

type output struct {
	writer io.Writer
	json   bool
}

func (o output) write(value any, human func()) error {
	if o.json {
		encoder := json.NewEncoder(o.writer)
		encoder.SetIndent("", "  ")
		return encoder.Encode(value)
	}
	human()
	return nil
}

// Run does not retry mutations. An interrupted start may already have created
// work; show its engagement before deciding whether to create another attempt.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		_, err := io.WriteString(stdout, help)
		return err
	}
	command := args[0]
	switch command {
	case "applications", "plan", "engagements", "show", "approve", "start", "status", "watch", "stop", "export":
	default:
		return errors.New("unknown assess command; use tsunami assess --help")
	}
	fs := flag.NewFlagSet("assess "+command, flag.ContinueOnError)
	fs.SetOutput(stderr)
	origin := fs.String("url", "http://127.0.0.1:8080", "control API origin; HTTP only on loopback")
	tokenFile := fs.String("token-file", "", "private operator credential file (mode 0600)")
	asJSON := fs.Bool("json", false, "structured output; target response bodies remain in exports")
	var engagement, run, app, name, authorization, executor, reviewed, out string
	var timeout time.Duration
	if command == "plan" {
		fs.StringVar(&app, "application", "", "configured application ID; no arbitrary target URLs")
		fs.StringVar(&name, "name", "Web/API authorization assessment", "engagement name")
		fs.StringVar(&authorization, "authorization-ref", "", "reference to your permission to test this owned application")
		fs.StringVar(&executor, "executor", "native", "native or curl-http/v1; no fallback")
	}
	if command == "show" || command == "approve" || command == "start" || command == "stop" || command == "export" {
		fs.StringVar(&engagement, "engagement", "", "saved engagement ID")
	}
	if command == "status" || command == "watch" || command == "stop" {
		fs.StringVar(&run, "run", "", "run ID; stopping an engagement also prevents new children")
	}
	if command == "approve" {
		fs.StringVar(&reviewed, "scope-digest", "", "exact scope digest from the plan you reviewed")
	}
	if command == "export" {
		fs.StringVar(&out, "out", "", "new output directory; existing directories are never overwritten")
	}
	if command == "watch" {
		fs.DurationVar(&timeout, "timeout", 100*time.Second, "watch duration, from 1s to 5m; expiry does not Stop")
	}
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("unexpected positional arguments; use command flags")
	}
	if (engagement != "" && !runID.MatchString(engagement)) || (run != "" && !runID.MatchString(run)) {
		return errors.New("engagement and run IDs must be 32 lowercase hex characters")
	}
	if command == "stop" {
		if (engagement == "") == (run == "") {
			return errors.New("choose exactly one of --engagement or --run")
		}
	} else if command == "show" || command == "approve" || command == "start" || command == "export" {
		if engagement == "" {
			return errors.New("--engagement is required")
		}
	} else if command == "status" || command == "watch" {
		if run == "" {
			return errors.New("--run is required")
		}
	}
	if command == "approve" && !digest.MatchString(reviewed) {
		return errors.New("--scope-digest from the reviewed draft is required")
	}
	if command == "export" && out == "" {
		return errors.New("--out must name a new directory")
	}
	if command == "watch" && (timeout < time.Second || timeout > 5*time.Minute) {
		return errors.New("watch timeout must be between 1s and 5m")
	}
	if err := validateOrigin(*origin); err != nil {
		return err
	}
	token, err := credential(*tokenFile)
	if err != nil {
		return err
	}
	o := demo.Operator{BaseURL: *origin, Token: token}
	printer := output{stdout, *asJSON}
	base := "/v1/engagements/" + engagement
	switch command {
	case "applications":
		var plans []application.Plan
		if err := o.Call(ctx, "GET", "/v1/applications", nil, &plans); err != nil {
			return err
		}
		sort.Slice(plans, func(i, j int) bool { return plans[i].ID < plans[j].ID })
		type item struct {
			ID      string `json:"id"`
			Title   string `json:"title"`
			Origin  string `json:"origin"`
			Profile string `json:"profile"`
		}
		items := []item{}
		for _, p := range plans {
			items = append(items, item{p.ID, p.Title, p.Origin, p.Version})
		}
		return printer.write(items, func() {
			fmt.Fprintln(stdout, "Configured owned applications (credentials are not returned):")
			for _, p := range plans {
				fmt.Fprintf(stdout, "  %s  %q  %q\n", p.ID, p.Title, p.Origin)
			}
			if len(plans) == 0 {
				fmt.Fprintln(stdout, "No configured applications. A trusted operator must supply the test accounts, legitimate objects and intended access rule first.")
			}
			fmt.Fprintln(stdout, "Next: plan --application ID --authorization-ref REFERENCE. Account validity is checked during execution, not by this list.")
		})
	case "engagements":
		var all []platform.Engagement
		if err := o.Call(ctx, "GET", "/v1/engagements", nil, &all); err != nil {
			return err
		}
		sort.Slice(all, func(i, j int) bool { return all[i].CreatedAt.Before(all[j].CreatedAt) })
		type item struct {
			ID          string   `json:"id"`
			Name        string   `json:"name"`
			State       string   `json:"state"`
			Application string   `json:"application,omitempty"`
			RunIDs      []string `json:"run_ids"`
		}
		items := []item{}
		for _, e := range all {
			app := ""
			if e.Application != nil {
				app = e.Application.ID
			}
			items = append(items, item{e.ID, e.Name, e.State, app, e.RunIDs})
		}
		return printer.write(items, func() {
			for _, e := range items {
				fmt.Fprintf(stdout, "%s  state=%q application=%q name=%q runs=%q\n", e.ID, e.State, e.Application, e.Name, e.RunIDs)
			}
			fmt.Fprintln(stdout, "Next: show --engagement ID. Check saved child IDs before retrying an interrupted start.")
		})
	case "plan":
		if !shortText(name, 120) || !shortText(authorization, 256) {
			return errors.New("provide a name (1–120 bytes) and --authorization-ref (1–256 bytes), without control characters")
		}
		if executor == "native" {
			executor = ""
		} else if executor != toolcontract.CurlHTTP {
			return errors.New("executor must be native or curl-http/v1")
		}
		var plans []application.Plan
		if err := o.Call(ctx, "GET", "/v1/applications", nil, &plans); err != nil {
			return err
		}
		found := false
		for _, p := range plans {
			if p.ID == app && p.Validate() == nil {
				found = true
			}
		}
		if !found {
			return errors.New("select a valid configured application from applications; a target URL is not accepted")
		}
		var e platform.Engagement
		if err := o.Call(ctx, "POST", "/v1/engagements", platform.EngagementInput{Name: name, AuthorizationRef: authorization, Application: app, Executor: executor}, &e); err != nil {
			return fmt.Errorf("draft not confirmed; inspect engagements before retrying: %w", err)
		}
		var view platform.EngagementView
		if err := o.Call(ctx, "GET", "/v1/engagements/"+e.ID, nil, &view); err != nil {
			return fmt.Errorf("draft %s saved but preview unavailable; inspect show before creating another draft: %w", e.ID, err)
		}
		return printPlan(printer, view)
	case "show", "approve", "start":
		var view platform.EngagementView
		if err := o.Call(ctx, "GET", base, nil, &view); err != nil {
			return err
		}
		e := view.Engagement
		if e.ID != engagement || e.Application == nil || e.Application.Validate() != nil || len(e.Items) != 1 || e.Items[0].Procedure != e.Application.ProcedureID() {
			return errors.New("this CLI launch flow requires one configured HTTP application plan")
		}
		if command == "show" {
			return printPlan(printer, view)
		}
		if command == "approve" {
			if e.ScopeDigest != reviewed || e.State != platform.EngagementDraft || time.Now().After(e.ExpiresAt) {
				return errors.New("approval refused: review the current unexpired draft and its exact scope digest")
			}
			if err := o.Call(ctx, "POST", base+"/approve", struct{}{}, &view); err != nil {
				return fmt.Errorf("approval not confirmed; inspect show --engagement %s: %w", engagement, err)
			}
			return printPlan(printer, view)
		}
		if e.State != platform.EngagementActive || e.ApprovedDigest != e.ScopeDigest || time.Now().After(e.ExpiresAt) {
			return errors.New("start requires an active, unexpired engagement with the reviewed scope approved")
		}
		var r platform.Run
		if err := o.Call(ctx, "POST", base+"/runs", map[string]string{"assessment": e.Items[0].Procedure, "execution": "scripted"}, &r); err != nil {
			return fmt.Errorf("start not confirmed; no retry made; inspect show --engagement %s for child IDs before another attempt: %w", engagement, err)
		}
		type started struct {
			EngagementID string    `json:"engagement_id"`
			RunID        string    `json:"run_id"`
			Execution    string    `json:"execution"`
			Executor     string    `json:"executor"`
			Deadline     time.Time `json:"deadline"`
		}
		return printer.write(started{engagement, r.ID, r.Execution, executorName(r.Executor), r.Deadline}, func() {
			fmt.Fprintf(stdout, "Engagement: %s\nRun: %s\nExecution: %s; executor: %s\nDeadline: %s\n", engagement, r.ID, r.Execution, executorName(r.Executor), r.Deadline.Format(time.RFC3339))
			fmt.Fprintf(stdout, "Start is recorded; this does not mean the worker is running.\nNext: watch --run %s\nStop: stop --engagement %s\n", r.ID, engagement)
		})
	case "status", "watch":
		return inspect(ctx, o, printer, run, command == "watch", timeout)
	case "stop":
		path := base + "/stop"
		if run != "" {
			path = "/v1/runs/" + run + "/stop"
		}
		if err := o.Call(ctx, "POST", path, struct{}{}, nil); err != nil {
			return fmt.Errorf("Stop not acknowledged; do not assume revocation or cleanup; inspect status/show and the independent cutoff if available: %w", err)
		}
		result := struct {
			EngagementID string `json:"engagement_id,omitempty"`
			RunID        string `json:"run_id,omitempty"`
			Acknowledged bool   `json:"acknowledged"`
		}{engagement, run, true}
		return printer.write(result, func() {
			fmt.Fprintln(stdout, "Stop acknowledged. Permission is closed; request drain and worker cleanup are separate observations.")
			if run != "" {
				fmt.Fprintln(stdout, "Run Stop does not close its engagement. Use engagement Stop to prevent new attempts too.")
			}
			fmt.Fprintln(stdout, "Next: status --run ID (use show --engagement ID to find child IDs). Target-side effects cannot be undone.")
		})
	case "export":
		var b platform.EngagementBundle
		if err := o.Call(ctx, "GET", base+"/export", nil, &b); err != nil {
			return err
		}
		if b.Version != "tsunami.engagement-export/v2" || b.View.Engagement.ID != engagement || !b.SyntheticOnly {
			return errors.New("unexpected engagement export; no files written")
		}
		if err := demo.WriteEngagementBundle(out, b); err != nil {
			return fmt.Errorf("export not completed; a partial new directory may remain; existing files were not overwritten: %w", err)
		}
		result := struct {
			Directory        string `json:"directory"`
			EngagementID     string `json:"engagement_id"`
			Candidates       int    `json:"candidates"`
			ReviewedFindings int    `json:"reviewed_findings"`
			Artifacts        int    `json:"artifacts"`
		}{out, engagement, len(b.Candidates), len(b.Findings), len(b.Artifacts)}
		return printer.write(result, func() {
			fmt.Fprintf(stdout, "Saved private export to %q: %d candidates, %d reviewed findings, %d artifacts.\n", out, len(b.Candidates), len(b.Findings), len(b.Artifacts))
			fmt.Fprintln(stdout, "Candidates need human impact/severity review. No review or CVSS score was invented. Inspect progress for incomplete tests.")
		})
	}
	return nil
}

func validateOrigin(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || len(raw) > 2048 || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" || u.Path != "" {
		return errors.New("--url must be a control API origin without path, credentials, query or fragment")
	}
	if u.Scheme == "https" {
		return nil
	}
	ip, err := netip.ParseAddr(u.Hostname())
	if u.Scheme != "http" || !(u.Hostname() == "localhost" || (err == nil && ip.IsLoopback() && ip.Zone() == "")) {
		return errors.New("remote control APIs require HTTPS; HTTP is restricted to loopback/localhost port-forwarding")
	}
	return nil
}

func credential(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", errors.New("read --token-file from a private operator credential file")
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > 4096 {
		return "", errors.New("operator credential must be a regular private file (0600), at most 4096 bytes")
	}
	data, err := io.ReadAll(io.LimitReader(f, 4097))
	token := strings.TrimSpace(string(data))
	if err != nil || len(data) > 4096 || len(token) < 32 || strings.ContainsAny(token, "\r\n\t ") {
		return "", errors.New("invalid bounded operator credential file")
	}
	return token, nil
}

func shortText(s string, max int) bool {
	if strings.TrimSpace(s) == "" || len(s) > max {
		return false
	}
	for _, c := range s {
		if unicode.IsControl(c) {
			return false
		}
	}
	return true
}

func executorName(executor string) string {
	if executor == "" {
		return "native"
	}
	return executor
}

func printPlan(o output, v platform.EngagementView) error {
	e := v.Engagement
	return o.write(v, func() {
		fmt.Fprintf(o.writer, "Engagement: %s\nName: %q\nState: %q\nAuthorization reference: %q\nScope digest: %s\nApproved digest: %s\nExpires: %s\n", e.ID, e.Name, e.State, e.AuthorizationRef, e.ScopeDigest, e.ApprovedDigest, e.ExpiresAt.Format(time.RFC3339))
		if p := e.Application; p != nil {
			fmt.Fprintf(o.writer, "Application: %q\nTarget: %q\nApproved destination addresses: %q\nProfile: %q\nExecution: scripted Mecatl; executor: %q; no model calls\n", p.ID, p.Origin, p.ApprovedAddresses, p.Version, executorName(e.Executor))
			fmt.Fprintf(o.writer, "Per attempt: %d planned reads; at most %d target requests in this lab profile.\nEngagement limits: %d runs, %d concurrent. The run deadline is returned on start.\n", len(p.Reads()), p.MaxRequests(), e.MaxRuns, e.MaxConcurrent)
			fmt.Fprintf(o.writer, "Expected cross-user access: %q. The service verifies identities, legitimate reads and follow-up checks.\n", p.ExpectedAccess)
			if p.Version == application.PropertyAccessProfile {
				fields := []string{}
				for pointer := range p.Probe.Hashes {
					fields = append(fields, pointer)
				}
				sort.Strings(fields)
				fmt.Fprintf(o.writer, "The peer may read the public object. These owner-only fields must be omitted: %q. Known private values are compared by hash; redaction/unknown values are inconclusive.\n", fields)
			}
			for i, r := range p.Reads() {
				fmt.Fprintf(o.writer, "  %d. GET %q as %q\n", i+1, r.Path, r.Account)
			}
		}
		fmt.Fprintf(o.writer, "Child runs: %q\nProgress: completed=%d running=%d pending=%d failed=%d stopped=%d inconclusive=%d skipped=%d\n", e.RunIDs, v.Progress.Completed, v.Progress.Running, v.Progress.Pending, v.Progress.Failed, v.Progress.Stopped, v.Progress.Inconclusive, v.Progress.Skipped)
		fmt.Fprintln(o.writer, "Synthetic data only. No discovery, raw scanning, arbitrary shell or state-changing operations are approved. GET can still have side effects; approval depends on the application owner. Account validity is checked during execution.")
		switch e.State {
		case platform.EngagementDraft:
			fmt.Fprintf(o.writer, "Review the scope above. To approve: approve --engagement %s --scope-digest %s\nNo target requests have been made by plan creation.\n", e.ID, e.ScopeDigest)
		case platform.EngagementActive:
			fmt.Fprintf(o.writer, "Next: start --engagement %s (one new attempt; never automatically retried).\nStop: stop --engagement %s\n", e.ID, e.ID)
		case platform.EngagementStopped:
			fmt.Fprintln(o.writer, "Engagement stopped. Inspect each child's status for drain and cleanup; export remains available.")
		}
	})
}

func inspect(ctx context.Context, o demo.Operator, out output, id string, watch bool, duration time.Duration) error {
	if watch {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, duration)
		defer cancel()
	}
	path := "/v1/runs/" + id
	var previous string
	for {
		var status platform.RunStatus
		if err := o.Call(ctx, "GET", path+"/status", nil, &status); err != nil {
			if watch && ctx.Err() != nil {
				return errors.New("watch ended before permission closed; work may still be active; inspect status or use stop --engagement ID")
			}
			return fmt.Errorf("observation unavailable; work was not stopped by this read-only command: %w", err)
		}
		encoded, _ := json.Marshal(status)
		if string(encoded) != previous {
			if err := out.write(status, func() {
				fmt.Fprintf(out.writer, "Run %s: mode=%q executor=%q target_requests_reserved=%d\n  permission=%q execution=%q activity=%q cleanup=%q uncertain_operations=%d\n", id, status.ExecutionMode, executorName(status.Executor), status.TargetRequestsReserved, status.Permission, status.Execution, status.Activity, status.Cleanup, status.UncertainOperations)
				fmt.Fprintf(out.writer, "  executor_observation=%q worker_observed_at=%v exit_observed_at=%v\n", status.ObservationCode, status.WorkerObservedAt, status.ExitObservedAt)
				if status.Assessment != nil {
					fmt.Fprintf(out.writer, "  verdict=%q reason=%q\n", status.Assessment.Verdict, status.Assessment.Reason)
					fmt.Fprintln(out.writer, "  confirmed = supported candidate awaiting review; no_issue_observed covers this test only; inconclusive is not a pass.")
				}
			}); err != nil {
				return err
			}
			previous = string(encoded)
		}
		ended := status.Permission == "closed" || status.Permission == "revoked" || status.Permission == "expired"
		if !watch || ended {
			if !out.json {
				fmt.Fprintln(out.writer, "Watching is read-only. Permission closure is separate from request drain and cleanup. Use status again for later observations; use export --engagement ID --out NEW_DIRECTORY for evidence.")
			}
			return nil
		}
		timer := time.NewTimer(500 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return errors.New("watch ended before permission closed; work may still be active; inspect status or use stop --engagement ID")
		case <-timer.C:
		}
	}
}
