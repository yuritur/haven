package aws

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"

	"github.com/havenapp/haven/internal/provider"
	"github.com/havenapp/haven/internal/provider/aws/quota"
)

type probeResult struct {
	cfg      awssdk.Config
	identity provider.Identity
}

// Authenticate runs the full interactive AWS authentication flow.
// It returns a ready-to-use Provider and StateStore, or an error.
func Authenticate(ctx context.Context, prompter provider.Prompter, out io.Writer) (provider.Provider, provider.StateStore, error) {
	pr, err := probe(ctx)
	if err == nil {
		if confirmIdentity(prompter, pr.identity) {
			return initFromProbe(ctx, pr, out)
		}
		pr, err = switchProfile(ctx, prompter)
		if err != nil {
			return nil, nil, err
		}
		return initFromProbe(ctx, pr, out)
	}

	pr, err = onboard(ctx, prompter)
	if err != nil {
		return nil, nil, err
	}
	return initFromProbe(ctx, pr, out)
}

func probe(ctx context.Context) (*probeResult, error) {
	pr, err := probeWithConfig(ctx, loadConfig)
	if err != nil {
		// Fallback: try the "haven" profile (saved by previous onboarding).
		if fallback, ferr := probeWithProfile(ctx, "haven"); ferr == nil {
			return fallback, nil
		}
		return nil, err
	}
	return pr, nil
}

func probeWithConfig(ctx context.Context, loadFn func(context.Context) (awssdk.Config, error)) (*probeResult, error) {
	cfg, err := loadFn(ctx)
	if err != nil {
		return nil, err
	}
	id, err := getIdentity(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &probeResult{
		cfg: cfg,
		identity: provider.Identity{
			AccountID: id.AccountID,
			ARN:       id.ARN,
			Region:    id.Region,
		},
	}, nil
}

func probeWithProfile(ctx context.Context, profile string) (*probeResult, error) {
	return probeWithConfig(ctx, func(ctx context.Context) (awssdk.Config, error) {
		return loadConfigWithProfile(ctx, profile)
	})
}

func confirmIdentity(p provider.Prompter, id provider.Identity) bool {
	p.Print(fmt.Sprintf("\n  AWS Account:  %s", id.AccountID))
	p.Print(fmt.Sprintf("  Region:       %s", id.Region))
	p.Print(fmt.Sprintf("  Identity:     %s\n", id.ARN))
	return p.Confirm("Continue with this account?")
}

func switchProfile(ctx context.Context, p provider.Prompter) (*probeResult, error) {
	profiles := listProfiles()
	if len(profiles) == 0 {
		return collectAndProbe(ctx, p)
	}

	options := make([]string, len(profiles)+1)
	copy(options, profiles)
	options[len(profiles)] = "Enter new credentials"

	idx := p.Select("Available AWS profiles:", options)
	if idx < 0 || idx >= len(options) {
		return nil, fmt.Errorf("selection cancelled")
	}
	if idx == len(profiles) {
		return collectAndProbe(ctx, p)
	}

	pr, err := probeWithProfile(ctx, profiles[idx])
	if err != nil {
		return nil, fmt.Errorf("profile %q: %w", profiles[idx], err)
	}
	if !confirmIdentity(p, pr.identity) {
		return nil, fmt.Errorf("aborted")
	}
	return pr, nil
}

func onboard(ctx context.Context, p provider.Prompter) (*probeResult, error) {
	p.Print("\nNo AWS credentials found.\n")

	if !p.Confirm("Do you have an AWS account?") {
		return nil, fmt.Errorf("no AWS account — create one at https://aws.amazon.com/free/ and run haven again")
	}

	return collectAndProbe(ctx, p)
}

func collectAndProbe(ctx context.Context, p provider.Prompter) (*probeResult, error) {
	accessKey, secretKey, region, err := collectCredentials(p)
	if err != nil {
		return nil, err
	}

	// Validate credentials before saving.
	cfg, err := loadConfigWithStaticCredentials(ctx, accessKey, secretKey, region)
	if err != nil {
		return nil, err
	}
	id, err := getIdentity(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials: %w", err)
	}

	if err := saveCredentials(accessKey, secretKey, region); err != nil {
		return nil, err
	}

	return &probeResult{
		cfg: cfg,
		identity: provider.Identity{
			AccountID: id.AccountID,
			ARN:       id.ARN,
			Region:    id.Region,
		},
	}, nil
}

func collectCredentials(p provider.Prompter) (accessKey, secretKey, region string, err error) {
	p.Print("\nTo get your AWS credentials:")
	p.Print("  1. Open: https://console.aws.amazon.com/iam/home#/security_credentials")
	p.Print("  2. Scroll to \"Access keys\"")
	p.Print("  3. Click \"Create access key\"")
	p.Print("  4. Copy the Access Key ID and Secret Access Key\n")

	accessKey = strings.TrimSpace(p.Input("AWS Access Key ID"))
	if accessKey == "" {
		return "", "", "", fmt.Errorf("access key ID is required")
	}

	secretKey = strings.TrimSpace(p.Secret("Secret Access Key"))
	if secretKey == "" {
		return "", "", "", fmt.Errorf("secret access key is required")
	}

	region = strings.TrimSpace(p.Input("Region [us-east-1]"))
	if region == "" {
		region = "us-east-1"
	}

	return accessKey, secretKey, region, nil
}

func saveCredentials(accessKey, secretKey, region string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("find home directory: %w", err)
	}
	awsDir := filepath.Join(home, ".aws")
	if err := os.MkdirAll(awsDir, 0700); err != nil {
		return fmt.Errorf("create ~/.aws: %w", err)
	}

	// Save credentials to ~/.aws/credentials under [haven] section.
	credPath := filepath.Join(awsDir, "credentials")
	credContent := fmt.Sprintf("aws_access_key_id = %s\naws_secret_access_key = %s\n", accessKey, secretKey)
	if err := upsertINISection(credPath, "haven", credContent); err != nil {
		return fmt.Errorf("write credentials: %w", err)
	}

	// Save region to ~/.aws/config under [profile haven] section.
	cfgPath := filepath.Join(awsDir, "config")
	cfgContent := fmt.Sprintf("region = %s\n", region)
	if err := upsertINISection(cfgPath, "profile haven", cfgContent); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// upsertINISection replaces or appends a [section] in an INI file.
func upsertINISection(path, section, content string) error {
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	header := fmt.Sprintf("[%s]", section)
	lines := strings.Split(string(existing), "\n")
	var result []string
	replaced := false
	skip := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == header {
			// Start of our section — replace it.
			result = append(result, header)
			result = append(result, strings.TrimRight(content, "\n"))
			replaced = true
			skip = true
			continue
		}
		if skip {
			// Skip old section content until next section or EOF.
			if strings.HasPrefix(trimmed, "[") {
				skip = false
				result = append(result, line)
			}
			continue
		}
		result = append(result, line)
	}

	if !replaced {
		// Ensure blank line before new section if file is non-empty.
		text := strings.TrimRight(strings.Join(result, "\n"), "\n")
		if text != "" {
			text += "\n\n"
		}
		text += header + "\n" + content
		result = strings.Split(text, "\n")
	}

	out := strings.Join(result, "\n")
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return os.WriteFile(path, []byte(out), 0600)
}

func listProfiles() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	seen := make(map[string]bool)

	// Parse ~/.aws/credentials for [profile_name] sections.
	credPath := filepath.Join(home, ".aws", "credentials")
	for _, name := range parseINISections(credPath) {
		seen[name] = true
	}

	// Parse ~/.aws/config for [profile name] sections.
	cfgPath := filepath.Join(home, ".aws", "config")
	for _, name := range parseINISections(cfgPath) {
		// In config file, sections are [profile xxx] except [default].
		name = strings.TrimPrefix(name, "profile ")
		seen[name] = true
	}

	profiles := make([]string, 0, len(seen))
	for name := range seen {
		profiles = append(profiles, name)
	}
	sort.Strings(profiles)
	return profiles
}

func parseINISections(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()

	var sections []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			name := line[1 : len(line)-1]
			sections = append(sections, name)
		}
	}
	return sections
}

func initFromProbe(ctx context.Context, pr *probeResult, out io.Writer) (provider.Provider, provider.StateStore, error) {
	store, err := newS3StateStore(ctx, pr.cfg, pr.identity.AccountID)
	if err != nil {
		return nil, nil, err
	}

	p := &AWSProvider{
		cfg:        pr.cfg,
		out:        out,
		bucketName: store.bucketName,
		quotaStore: quota.NewStore(pr.cfg, store.bucketName),
		identity:   pr.identity,
	}
	return p, store, nil
}
