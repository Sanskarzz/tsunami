package binding

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

// Command is local preparation only: it has no operator token, live preflight,
// target client or registration permission. All source documents are inline.
func Command(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("prepare-plan", flag.ContinueOnError)
	fs.SetOutput(stderr)
	input := fs.String("input", "", "private JSON file containing explicit facts and four inline synthetic baseline documents")
	out := fs.String("out", "", "new private package directory; no existing directory is overwritten")
	asJSON := fs.Bool("json", false, "print prepared-package metadata only")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 || *input == "" || *out == "" {
		return errors.New("prepare-plan requires --input and a new --out directory")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// Linux lab tooling: no symlink following and no blocking FIFO open. The
	// selected file must be regular before any body bytes are read.
	f, err := os.OpenFile(*input, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return errors.New("binding input must be a readable regular file, not a symlink")
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > 64<<10 {
		return errors.New("binding input must be a regular file of at most 64 KiB")
	}
	raw, err := io.ReadAll(io.LimitReader(f, (64<<10)+1))
	if err != nil || len(raw) > 64<<10 {
		return errors.New("cannot read bounded binding input")
	}
	plan, manifest, prepared, err := bytesFor(raw)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.Mkdir(*out, 0700); err != nil {
		return errors.New("output must be a new directory under an existing parent; nothing overwritten")
	}
	for _, file := range []struct {
		name string
		data []byte
	}{{"plan.json", plan}, {"manifest.json", manifest}} {
		path := filepath.Join(*out, file.name)
		if err := os.WriteFile(path, file.data, 0600); err != nil {
			return errors.New("prepared package incomplete; inspect the partial private directory before use")
		}
	}
	message := struct {
		Directory      string `json:"directory"`
		Application    string `json:"application"`
		Profile        string `json:"profile"`
		Status         string `json:"status"`
		TargetRequests int    `json:"target_requests"`
	}{*out, prepared.ID, prepared.Version, "pending_operator_registration_and_scope_approval", 0}
	if *asJSON {
		return json.NewEncoder(stdout).Encode(message)
	}
	_, err = fmt.Fprintf(stdout, "Prepared %q in %q.\nNo targets contacted. This validates supplied synthetic facts only.\nA trusted operator must register this plan with separate owner/peer credentials. Then use assess plan, review and approve its actual scope digest.\nThe empty credential fingerprint is deliberate; this package is not an approved run plan.\n", prepared.ID, *out)
	return err
}
