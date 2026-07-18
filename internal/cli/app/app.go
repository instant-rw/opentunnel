package app

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/opentunnel/opentunnel/internal/cli/browser"
	"github.com/opentunnel/opentunnel/internal/cli/config"
	"github.com/opentunnel/opentunnel/internal/cli/control"
	"github.com/opentunnel/opentunnel/internal/cli/credentials"
	"github.com/opentunnel/opentunnel/internal/cli/tunnel"
	"github.com/opentunnel/opentunnel/internal/gen/api"
)

type App struct {
	In          io.Reader
	Out         io.Writer
	ErrOut      io.Writer
	Version     string
	Config      *config.Store
	Credentials *credentials.Store
	OpenBrowser func(string) error
}

func New(version string) (*App, error) {
	configStore, err := config.NewStore()
	if err != nil {
		return nil, err
	}
	credentialStore, err := credentials.NewStore()
	if err != nil {
		return nil, err
	}
	return &App{
		In:          os.Stdin,
		Out:         os.Stdout,
		ErrOut:      os.Stderr,
		Version:     version,
		Config:      configStore,
		Credentials: credentialStore,
		OpenBrowser: browser.Open,
	}, nil
}

func (a *App) Run(ctx context.Context, args []string) error {
	global := flag.NewFlagSet("opentunnel", flag.ContinueOnError)
	global.SetOutput(a.ErrOut)
	apiURL := global.String("api-url", "", "control-plane API URL")
	showVersion := global.Bool("version", false, "print the OpenTunnel version")
	if err := global.Parse(args); err != nil {
		return err
	}
	if *showVersion {
		fmt.Fprintln(a.Out, a.Version)
		return nil
	}
	if global.NArg() == 0 {
		a.usage()
		return errors.New("a command is required")
	}

	cfg, err := a.Config.Load()
	if err != nil {
		return err
	}
	if *apiURL != "" {
		cfg.APIURL = strings.TrimRight(*apiURL, "/")
	}
	if cfg.APIURL == "" {
		if environmentURL := os.Getenv("OPENTUNNEL_API_URL"); environmentURL != "" {
			cfg.APIURL = strings.TrimRight(environmentURL, "/")
		} else {
			cfg.APIURL = control.DefaultAPIURL
		}
	}
	token, credentialErr := a.Credentials.Get()
	if credentialErr != nil && !errors.Is(credentialErr, credentials.ErrNotFound) {
		return credentialErr
	}
	client, err := control.New(cfg.APIURL, token)
	if err != nil {
		return err
	}

	command := global.Arg(0)
	commandArgs := global.Args()[1:]
	switch command {
	case "login":
		token, err := a.login(ctx, client)
		if err != nil {
			return err
		}
		client.SetToken(token)
		return a.Config.Save(cfg)
	case "logout":
		return a.logout(ctx, client, token)
	case "up":
		if token == "" {
			fmt.Fprintln(a.Out, "Login required.")
			token, err = a.login(ctx, client)
			if err != nil {
				return err
			}
			client.SetToken(token)
		}
		return a.up(ctx, client, token, &cfg, commandArgs)
	case "domains":
		return a.domains(ctx, client, commandArgs)
	case "requests":
		return a.requests(ctx, client, cfg, commandArgs)
	case "help", "-h", "--help":
		a.usage()
		return nil
	default:
		a.usage()
		return fmt.Errorf("unknown command %q", command)
	}
}

func (a *App) login(ctx context.Context, client *control.Client) (string, error) {
	authorization, err := client.CreateDeviceAuthorization(ctx)
	if err != nil {
		return "", err
	}
	fmt.Fprintf(a.Out, "Your code is: %s\nApprove this device at:\n%s\n", authorization.UserCode, authorization.VerificationUriComplete)
	if err := a.OpenBrowser(authorization.VerificationUriComplete); err != nil {
		fmt.Fprintf(a.ErrOut, "Could not open a browser: %v\n", err)
	}

	interval := control.PollInterval(authorization)
	deadline := time.NewTimer(time.Duration(authorization.ExpiresIn) * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-deadline.C:
			return "", errors.New("device authorization expired; run `opentunnel login` again")
		case <-ticker.C:
			token, err := client.ExchangeDeviceCode(ctx, authorization.DeviceCode)
			if errors.Is(err, control.ErrAuthorizationPending) {
				continue
			}
			if err != nil {
				return "", err
			}
			if err := a.Credentials.Set(token); err != nil {
				return "", err
			}
			fmt.Fprintln(a.Out, "Logged in.")
			return token, nil
		}
	}
}

func (a *App) logout(ctx context.Context, client *control.Client, token string) error {
	if token == "" {
		fmt.Fprintln(a.Out, "Already logged out.")
		return nil
	}
	if err := client.Logout(ctx); err != nil {
		fmt.Fprintf(a.ErrOut, "Warning: token could not be revoked remotely: %v\n", err)
	}
	if err := a.Credentials.Delete(); err != nil {
		return err
	}
	fmt.Fprintln(a.Out, "Logged out.")
	return nil
}

func (a *App) domains(ctx context.Context, client *control.Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: opentunnel domains <list|create>")
	}
	switch args[0] {
	case "list":
		domains, err := client.ListDomains(ctx)
		if err != nil {
			return err
		}
		if len(domains) == 0 {
			fmt.Fprintln(a.Out, "No domains yet. Create one with `opentunnel domains create <slug>`.")
			return nil
		}
		for _, domain := range domains {
			fmt.Fprintf(a.Out, "%s\t%s\t%s\n", domain.Slug, domain.Hostname, domain.Status)
		}
		return nil
	case "create":
		if len(args) != 2 {
			return errors.New("usage: opentunnel domains create <slug>")
		}
		domain, err := client.CreateDomain(ctx, args[1])
		if err != nil {
			return err
		}
		fmt.Fprintf(a.Out, "Created https://%s\n", domain.Hostname)
		return nil
	default:
		return fmt.Errorf("unknown domains command %q", args[0])
	}
}

func (a *App) up(ctx context.Context, client *control.Client, token string, cfg *config.Config, args []string) error {
	flags := flag.NewFlagSet("up", flag.ContinueOnError)
	flags.SetOutput(a.ErrOut)
	domainFlag := flags.String("domain", "", "domain slug to connect")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() > 1 {
		return errors.New("usage: opentunnel up [--domain slug] [port]")
	}
	port := cfg.LastPort
	if port == 0 {
		port = 3000
	}
	if flags.NArg() == 1 {
		parsed, err := strconv.Atoi(flags.Arg(0))
		if err != nil || parsed < 1 || parsed > 65535 {
			return fmt.Errorf("invalid port %q", flags.Arg(0))
		}
		port = parsed
	}

	domains, err := client.ListDomains(ctx)
	if err != nil {
		return err
	}
	domain, err := a.selectDomain(ctx, client, domains, *domainFlag, cfg.LastDomainID)
	if err != nil {
		return err
	}
	cfg.LastDomainID = domain.Id.String()
	cfg.LastPort = port
	if err := a.Config.Save(*cfg); err != nil {
		return err
	}

	fmt.Fprintf(a.Out, "Forwarding https://%s -> http://127.0.0.1:%d\n", domain.Hostname, port)
	runner := tunnel.Runner{
		APIURL:        cfg.APIURL,
		Token:         token,
		DomainID:      domain.Id.String(),
		LocalPort:     port,
		ClientVersion: a.Version,
		OnState: func(state tunnel.State) {
			if state.Connected {
				fmt.Fprintf(a.Out, "Connected · requests %d\n", state.RequestCount)
			} else if state.ReconnectAttempts > 0 {
				fmt.Fprintf(a.Out, "Disconnected · reconnect attempt %d · requests %d\n", state.ReconnectAttempts, state.RequestCount)
			} else {
				fmt.Fprintln(a.Out, "Connecting…")
			}
		},
	}
	return runner.Run(ctx)
}

func (a *App) selectDomain(ctx context.Context, client *control.Client, domains []api.Domain, requested, savedID string) (api.Domain, error) {
	if requested != "" {
		for _, domain := range domains {
			if domain.Slug == requested || domain.Hostname == requested {
				return domain, nil
			}
		}
		return client.CreateDomain(ctx, requested)
	}
	if savedID != "" {
		for _, domain := range domains {
			if domain.Id.String() == savedID {
				fmt.Fprintf(a.Out, "Using saved domain %s. Pass --domain to choose another.\n", domain.Hostname)
				return domain, nil
			}
		}
	}
	if len(domains) == 1 {
		return domains[0], nil
	}
	if len(domains) == 0 {
		fmt.Fprint(a.Out, "Choose a subdomain: ")
		reader := bufio.NewReader(a.In)
		slug, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return api.Domain{}, err
		}
		slug = strings.TrimSpace(slug)
		if slug == "" {
			return api.Domain{}, errors.New("a subdomain is required")
		}
		return client.CreateDomain(ctx, slug)
	}
	fmt.Fprintln(a.Out, "Available domains:")
	for index, domain := range domains {
		fmt.Fprintf(a.Out, "  %d) %s\n", index+1, domain.Hostname)
	}
	fmt.Fprint(a.Out, "Select a domain: ")
	var selection int
	if _, err := fmt.Fscan(a.In, &selection); err != nil || selection < 1 || selection > len(domains) {
		return api.Domain{}, errors.New("invalid domain selection")
	}
	return domains[selection-1], nil
}

func (a *App) requests(ctx context.Context, client *control.Client, cfg config.Config, args []string) error {
	flags := flag.NewFlagSet("requests", flag.ContinueOnError)
	flags.SetOutput(a.ErrOut)
	domainFlag := flags.String("domain", "", "domain slug or ID")
	limit := flags.Int("limit", 20, "maximum requests to display")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *limit < 1 || *limit > 100 {
		return errors.New("--limit must be between 1 and 100")
	}
	domains, err := client.ListDomains(ctx)
	if err != nil {
		return err
	}
	domainID, err := resolveDomainID(domains, *domainFlag, cfg.LastDomainID)
	if err != nil {
		return err
	}
	items, err := client.ListRequests(ctx, domainID, *limit)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		fmt.Fprintln(a.Out, "No captured requests.")
		return nil
	}
	for _, item := range items {
		status := "-"
		duration := "-"
		if item.Response != nil {
			status = strconv.Itoa(item.Response.Status)
			duration = fmt.Sprintf("%dms", item.Response.DurationMs)
		}
		path := item.Path
		if item.Query != "" {
			path += "?" + item.Query
		}
		fmt.Fprintf(a.Out, "%s\t%s\t%s\t%s\t%s\n", item.ReceivedAt.Local().Format(time.RFC3339), item.Method, path, status, duration)
	}
	return nil
}

func resolveDomainID(domains []api.Domain, requested, saved string) (uuid.UUID, error) {
	needle := requested
	if needle == "" {
		needle = saved
	}
	for _, domain := range domains {
		if domain.Id.String() == needle || domain.Slug == needle || domain.Hostname == needle {
			return domain.Id, nil
		}
	}
	if parsed, err := uuid.Parse(needle); err == nil {
		return parsed, nil
	}
	if needle == "" {
		return uuid.Nil, errors.New("no saved domain; pass --domain")
	}
	return uuid.Nil, fmt.Errorf("domain %q was not found", needle)
}

func (a *App) usage() {
	fmt.Fprintln(a.ErrOut, `Usage: opentunnel [--api-url URL] <command>

Commands:
  login
  logout
  up [--domain slug] [port]
  domains list
  domains create <slug>
  requests [--domain slug] [--limit 20]`)
}
