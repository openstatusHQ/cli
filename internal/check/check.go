package check

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/urfave/cli/v3"

	output "github.com/openstatusHQ/cli/internal/cli"
)

func CheckCmd() *cli.Command {
	return &cli.Command{
		Name:    "check",
		Aliases: []string{"c"},
		Usage:   "Run an HTTP check against a URL from 28 global regions",
		UsageText: `openstatus check <URL>
  openstatus check https://openstat.us
  openstatus check https://openstat.us -X POST -H 'Authorization: Bearer …' -d '{"ping":true}'
  openstatus check https://openstat.us -d @payload.json
  openstatus check https://openstat.us --timing
  openstatus check https://openstat.us --json | jq '.summary'`,
		Description: `Run a one-shot HTTP check against a URL from 28 global regions.

The check is executed by the public OpenStatus speed checker. No API token is
required. Results stream to the terminal as they arrive from each region.

Output is sorted in the order regions report back (roughly fastest first).
Pass --timing to see DNS/Connection/TLS/TTFB/Transfer phase breakdowns.
Pass --json for a machine-readable single object including all phase data.

Rate limit: 3 requests per 60 seconds.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "method",
				Aliases: []string{"X"},
				Usage:   "HTTP method",
				Value:   http.MethodGet,
			},
			&cli.StringSliceFlag{
				Name:    "header",
				Aliases: []string{"H"},
				Usage:   "Header in \"Key: Value\" form (repeatable)",
			},
			&cli.StringFlag{
				Name:    "body",
				Aliases: []string{"d"},
				Usage:   "Request body. Use @filename to read a file, @- for stdin.",
			},
			&cli.BoolFlag{
				Name:  "timing",
				Usage: "Show DNS/Connection/TLS/TTFB/Transfer phases",
			},
		},
		Action: runCheck,
	}
}

func runCheck(ctx context.Context, cmd *cli.Command) error {
	rawURL := cmd.Args().Get(0)
	if rawURL == "" {
		return cli.Exit("URL is required.\n\nUsage: openstatus check <URL>\nExample: openstatus check https://openstat.us", 1)
	}
	if err := validateURL(rawURL); err != nil {
		return cli.Exit(err.Error(), 1)
	}

	headers, err := parseHeaders(cmd.StringSlice("header"))
	if err != nil {
		return cli.Exit(err.Error(), 1)
	}

	body, err := resolveBody(cmd.String("body"))
	if err != nil {
		return cli.Exit(err.Error(), 1)
	}

	payload := Payload{
		URL:     rawURL,
		Method:  strings.ToUpper(cmd.String("method")),
		Headers: headers,
		Body:    body,
	}

	timing := cmd.Bool("timing")

	spinner := output.StartSpinner(fmt.Sprintf("Checking %s…", rawURL))
	var renderer *Renderer
	if !output.IsJSONOutput() && !output.IsQuiet() {
		renderer = NewRenderer(os.Stdout, timing)
	}

	onRow := func(r RegionResult) {
		if renderer == nil {
			return
		}
		output.StopSpinner(spinner)
		spinner = nil
		renderer.Row(r)
	}

	results, checkID, runErr := Run(ctx, nil, payload, onRow)
	output.StopSpinner(spinner)

	if runErr != nil {
		return formatRunError(runErr)
	}

	if output.IsJSONOutput() {
		return output.PrintJSON(buildJSONOutput(payload.URL, checkID, results))
	}

	if renderer != nil {
		renderer.Footer(payload.URL, results, checkID)
	}
	return nil
}

func validateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid URL %q: %w", raw, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("invalid URL %q: missing scheme or host (did you mean https://%s?)", raw, raw)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("invalid URL %q: only http and https schemes are supported", raw)
	}
	return nil
}

func parseHeaders(raw []string) (map[string]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	out := make(map[string]string, len(raw))
	for _, h := range raw {
		key, value, ok := strings.Cut(h, ":")
		if !ok {
			return nil, fmt.Errorf("invalid header %q: expected \"Key: Value\"", h)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			return nil, fmt.Errorf("invalid header %q: key is empty", h)
		}
		out[key] = value
	}
	return out, nil
}

func resolveBody(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}
	if raw == "@-" {
		if output.IsStdinTerminal() {
			return "", errors.New("body \"@-\" requires piped stdin (e.g. echo … | openstatus check …)")
		}
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("read stdin: %w", err)
		}
		return string(data), nil
	}
	if strings.HasPrefix(raw, "@") {
		path := raw[1:]
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read body file %q: %w", path, err)
		}
		return string(data), nil
	}
	return raw, nil
}

func shareURL(checkID string) string {
	if checkID == "" {
		return ""
	}
	return "https://www.openstatus.dev/play/checker/" + checkID
}

func formatRunError(err error) error {
	var rl *RateLimitError
	if errors.As(err, &rl) {
		if rl.RetryAfter > 0 {
			return cli.Exit(fmt.Sprintf("Rate limited. Retry after %s. (3 requests per 60s allowed.)", rl.RetryAfter), 1)
		}
		return cli.Exit("Rate limited. Try again in a moment. (3 requests per 60s allowed.)", 1)
	}
	var br *BadRequestError
	if errors.As(err, &br) {
		msg := br.Message
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", br.Status)
		}
		if strings.Contains(strings.ToLower(msg), "client ip") || strings.Contains(strings.ToLower(br.Body), "client ip") {
			return cli.Exit(msg+"\n(This often happens behind a VPN or corporate proxy.)", 1)
		}
		return cli.Exit(msg, 1)
	}
	var se *ServerError
	if errors.As(err, &se) {
		return cli.Exit(fmt.Sprintf("OpenStatus checker temporarily unavailable (HTTP %d). Try again in a moment.", se.Status), 1)
	}
	if errors.Is(err, ErrStreamTruncated) {
		fmt.Fprintln(os.Stderr, "Warning: stream ended before completion; results may be incomplete.")
		return cli.Exit("Stream ended before all regions reported.", 1)
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return cli.Exit("Check cancelled.", 130)
	}
	return cli.Exit(fmt.Sprintf("Could not reach OpenStatus: %s", err.Error()), 1)
}
