package controlapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"open-mihomo-gateway/internal/config"
	"open-mihomo-gateway/internal/mihomo"

	"gopkg.in/yaml.v3"
)

const (
	maxSourceSize   = 10 << 20
	sourceUserAgent = "clash.meta"
)

func parseHTTPSSourceURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return nil, fmt.Errorf("source URL must be an absolute HTTPS URL")
	}
	return parsed, nil
}

func (s *Server) importURL(ctx context.Context, req SourceImportRequest) (Source, error) {
	if _, err := parseHTTPSSourceURL(req.URL); err != nil {
		return Source{}, err
	}
	client := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(next *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("too many redirects")
			}
			if next.URL.Scheme != "https" {
				return fmt.Errorf("redirected source must remain HTTPS")
			}
			return nil
		},
		Transport: &http.Transport{DialContext: safeDialContext},
	}
	httpReq, err := newSourceRequest(ctx, req.URL)
	if err != nil {
		return Source{}, err
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return Source{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Source{}, fmt.Errorf("source returned %s", resp.Status)
	}
	before, _ := s.store.Sources()
	source, err := s.importReaderID(req.ID, req.Name, req.Kind, redactURL(req.URL), io.LimitReader(resp.Body, maxSourceSize+1))
	if err != nil {
		return Source{}, err
	}
	if err := s.credentials.Put(ctx, source.ID, req.URL); err != nil {
		_ = s.store.SaveSources(before)
		_ = os.Remove(source.SnapshotPath)
		return Source{}, err
	}
	return source, nil
}

func newSourceRequest(ctx context.Context, sourceURL string) (*http.Request, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return nil, err
	}
	// Subscription services commonly use the client User-Agent to choose the
	// response format. Identify the requested format as mihomo/Clash Meta while
	// keeping OpenSurge as the product identity everywhere else.
	request.Header.Set("User-Agent", sourceUserAgent)
	return request, nil
}

func safeDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return nil, err
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
			return nil, fmt.Errorf("source URL resolves to a private or local address")
		}
	}
	dialer := &net.Dialer{Timeout: 8 * time.Second}
	return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
}

func redactURL(value string) string {
	parsed, err := url.Parse(value)
	if err != nil {
		return "invalid-url"
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func (s *Server) importReader(name, kind, origin string, reader io.Reader) (Source, error) {
	return s.importReaderID("", name, kind, origin, reader)
}

func (s *Server) importReaderID(id, name, kind, origin string, reader io.Reader) (Source, error) {
	if strings.TrimSpace(name) == "" {
		name = "Imported profile"
	}
	if kind == "" {
		kind = "mihomo_profile"
	}
	if kind != "mihomo_profile" && kind != "rule_provider" {
		return Source{}, fmt.Errorf("unsupported source kind %q", kind)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return Source{}, err
	}
	if len(data) == 0 {
		return Source{}, fmt.Errorf("source is empty")
	}
	if len(data) > maxSourceSize {
		return Source{}, fmt.Errorf("source exceeds 10 MiB limit")
	}
	inventory, err := inspectSource(data, kind)
	valid := err == nil
	validation := "structural validation passed; apply runs mihomo engine validation"
	if err != nil {
		validation = err.Error()
	}
	digestBytes := sha256.Sum256(data)
	digest := hex.EncodeToString(digestBytes[:])
	if id == "" {
		idBytes := sha256.Sum256([]byte(name + "\x00" + origin))
		id = hex.EncodeToString(idBytes[:8])
	} else if !validHexValue(id, 16) {
		return Source{}, fmt.Errorf("source id is invalid")
	}
	dir := filepath.Join(s.store.Dir(), "sources", id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return Source{}, err
	}
	path := filepath.Join(dir, digest+".yaml")
	if err := writeAtomic(path, data, 0o600); err != nil {
		return Source{}, err
	}
	source := Source{
		SchemaVersion: SchemaVersion,
		ID:            id,
		Name:          name,
		Kind:          kind,
		Origin:        origin,
		SnapshotPath:  path,
		Digest:        digest,
		Size:          int64(len(data)),
		Valid:         valid,
		Validation:    validation,
		Inventory:     inventory,
		ImportedAt:    time.Now().UTC(),
		Versions:      []SourceVersion{},
		Diff:          emptySourceDiff(),
	}
	if strings.HasPrefix(origin, "https://") {
		source.Origin = redactURL(origin)
	}
	sources, loadErr := s.store.Sources()
	if loadErr != nil {
		return Source{}, loadErr
	}
	sources = s.decorateSourceStates(sources)
	for i := range sources {
		if sources[i].ID == source.ID {
			previous := sources[i]
			if previous.Digest == source.Digest {
				source.Versions = append([]SourceVersion{}, previous.Versions...)
				source.Desired = previous.Desired
				source.Applied = previous.Applied
				source.Diff.PreviousDigest = previous.Digest
			} else {
				source.Versions = append(append([]SourceVersion{}, previous.Versions...), sourceVersion(previous))
				source.Diff = diffInventory(previous.Digest, previous.Inventory, source.Inventory)
			}
			sources[i] = source
			return source, s.store.SaveSources(sources)
		}
	}
	sources = append(sources, source)
	return source, s.store.SaveSources(sources)
}

func sourceVersion(source Source) SourceVersion {
	return SourceVersion{Digest: source.Digest, Size: source.Size, Valid: source.Valid, Validation: source.Validation, Inventory: source.Inventory, ImportedAt: source.ImportedAt, Desired: source.Desired, Applied: source.Applied, SnapshotPath: source.SnapshotPath}
}

func (s *Server) decorateSourceStates(sources []Source) []Source {
	desiredProfile, appliedProfile, desiredSource := s.profileCompositionDigests()
	_, overlay, _, overlayErr := s.loadProfileOverlay()
	result := append([]Source(nil), sources...)
	for i := range result {
		result[i].Versions = append([]SourceVersion(nil), result[i].Versions...)
		result[i].EffectiveInventory = result[i].Inventory
		result[i].OverlayCompatible = overlayErr == nil
		if overlayErr != nil {
			result[i].OverlayValidation = overlayErr.Error()
		} else if data, _, err := readValidatedSourceSnapshot(s.store.Dir(), result[i]); err != nil {
			result[i].OverlayCompatible = false
			result[i].OverlayValidation = err.Error()
		} else if composition, err := mihomo.ComposeProfileOverlay(data, overlay); err != nil {
			result[i].OverlayCompatible = false
			result[i].OverlayValidation = err.Error()
		} else {
			result[i].EffectiveDigest = composition.Digest
			result[i].EffectiveInventory = inventoryFromInspection(composition.Inspection)
			result[i].OverlayValidation = "compatible with global profile overlay"
		}
		result[i].Desired = desiredSource != "" && result[i].Digest == desiredSource
		result[i].Applied = result[i].Desired && desiredProfile != "" && desiredProfile == appliedProfile
		for j := range result[i].Versions {
			result[i].Versions[j].Desired = desiredSource != "" && result[i].Versions[j].Digest == desiredSource
			result[i].Versions[j].Applied = result[i].Versions[j].Desired && desiredProfile != "" && desiredProfile == appliedProfile
		}
	}
	return result
}

func (s *Server) profileCompositionDigests() (string, string, string) {
	cfg, err := config.LoadRuntime(s.configPath)
	if err != nil {
		return "", "", ""
	}
	desired, _ := config.MihomoProfileDigest(cfg)
	desiredSource := cfg.Mihomo.ProfileSourceDigest
	if desiredSource == "" {
		desiredSource = desired
	}
	applied := ""
	if state, exists := currentBootRuntimeState(cfg); exists {
		applied = state.ProfileDigest
	}
	return desired, applied, desiredSource
}

func emptySourceDiff() SourceDiff {
	return SourceDiff{ProxiesAdded: []string{}, ProxiesRemoved: []string{}, GroupsAdded: []string{}, GroupsRemoved: []string{}, ProxyProvidersAdded: []string{}, ProxyProvidersRemoved: []string{}, RuleProvidersAdded: []string{}, RuleProvidersRemoved: []string{}}
}

func emptyInventory() Inventory {
	return Inventory{
		Proxies:        []string{},
		ProxyProviders: []string{},
		ProxyGroups:    []string{},
		RuleProviders:  []string{},
		Warnings:       []string{},
	}
}

func normalizeInventory(inventory Inventory) Inventory {
	if inventory.Proxies == nil {
		inventory.Proxies = []string{}
	}
	if inventory.ProxyProviders == nil {
		inventory.ProxyProviders = []string{}
	}
	if inventory.ProxyGroups == nil {
		inventory.ProxyGroups = []string{}
	}
	if inventory.RuleProviders == nil {
		inventory.RuleProviders = []string{}
	}
	if inventory.Warnings == nil {
		inventory.Warnings = []string{}
	}
	return inventory
}

func diffInventory(previousDigest string, before, after Inventory) SourceDiff {
	diff := emptySourceDiff()
	diff.PreviousDigest = previousDigest
	diff.ProxiesAdded, diff.ProxiesRemoved = diffNames(before.Proxies, after.Proxies)
	diff.GroupsAdded, diff.GroupsRemoved = diffNames(before.ProxyGroups, after.ProxyGroups)
	diff.ProxyProvidersAdded, diff.ProxyProvidersRemoved = diffNames(before.ProxyProviders, after.ProxyProviders)
	diff.RuleProvidersAdded, diff.RuleProvidersRemoved = diffNames(before.RuleProviders, after.RuleProviders)
	diff.RuleCountDelta = after.RuleCount - before.RuleCount
	return diff
}

func diffNames(before, after []string) (added, removed []string) {
	left, right := map[string]bool{}, map[string]bool{}
	for _, value := range before {
		left[value] = true
	}
	for _, value := range after {
		right[value] = true
	}
	for value := range right {
		if !left[value] {
			added = append(added, value)
		}
	}
	for value := range left {
		if !right[value] {
			removed = append(removed, value)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	if added == nil {
		added = []string{}
	}
	if removed == nil {
		removed = []string{}
	}
	return added, removed
}

func inspectSource(data []byte, kind string) (Inventory, error) {
	inv := emptyInventory()
	if kind == "mihomo_profile" {
		inspection, err := mihomo.InspectImportedProfile(data)
		if err != nil {
			return inv, fmt.Errorf("parse mihomo profile: %w", err)
		}
		inv.Proxies = inspection.Proxies
		inv.ProxyProviders = inspection.ProxyProviders
		inv.ProxyGroups = inspection.ProxyGroups
		inv.RuleProviders = inspection.RuleProviders
		inv.RuleCount = inspection.RuleCount
		inv.TerminalMatch = inspection.TerminalMatch
		inv.Warnings = inspection.Warnings
		targets := append(append([]string{}, inv.Proxies...), inv.ProxyGroups...)
		for _, name := range append(targets, inv.RuleProviders...) {
			if strings.HasPrefix(name, "device/") || strings.HasPrefix(name, "open-surge-ruleset-") || strings.HasPrefix(name, mihomo.LocalRoutingGroupPrefix) {
				return inv, fmt.Errorf("imported source uses reserved OpenSurge name %q", name)
			}
		}
		return inv, nil
	}

	var document yaml.Node
	if err := yaml.NewDecoder(bytes.NewReader(data)).Decode(&document); err != nil {
		return inv, fmt.Errorf("parse YAML: %w", err)
	}
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return inv, fmt.Errorf("top-level YAML must be a mapping")
	}
	root := document.Content[0]
	sections := map[string]*yaml.Node{}
	for i := 0; i < len(root.Content); i += 2 {
		key := root.Content[i]
		if _, exists := sections[key.Value]; exists {
			return inv, fmt.Errorf("duplicate top-level section %q", key.Value)
		}
		sections[key.Value] = root.Content[i+1]
	}
	inv.Proxies = sequenceNames(sections["proxies"])
	inv.ProxyGroups = sequenceNames(sections["proxy-groups"])
	inv.ProxyProviders = mappingKeys(sections["proxy-providers"])
	inv.RuleProviders = mappingKeys(sections["rule-providers"])
	targets := append(append([]string{}, inv.Proxies...), inv.ProxyGroups...)
	for _, name := range append(targets, inv.RuleProviders...) {
		if strings.HasPrefix(name, "device/") || strings.HasPrefix(name, "open-surge-ruleset-") || strings.HasPrefix(name, mihomo.LocalRoutingGroupPrefix) {
			return inv, fmt.Errorf("imported source uses reserved OpenSurge name %q", name)
		}
	}
	return inv, nil
}

func sequenceNames(node *yaml.Node) []string {
	if node == nil || node.Kind != yaml.SequenceNode {
		return []string{}
	}
	var names []string
	for _, item := range node.Content {
		if item.Kind != yaml.MappingNode {
			continue
		}
		for i := 0; i < len(item.Content); i += 2 {
			if item.Content[i].Value == "name" && item.Content[i+1].Kind == yaml.ScalarNode {
				names = append(names, item.Content[i+1].Value)
			}
		}
	}
	sort.Strings(names)
	return names
}

func mappingKeys(node *yaml.Node) []string {
	if node == nil || node.Kind != yaml.MappingNode {
		return []string{}
	}
	keys := make([]string, 0, len(node.Content)/2)
	for i := 0; i < len(node.Content); i += 2 {
		keys = append(keys, node.Content[i].Value)
	}
	sort.Strings(keys)
	return keys
}

func (s *Server) handleSourceUpdate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validHexValue(id, 16) {
		writeError(w, http.StatusBadRequest, "invalid_source_id", "invalid source id")
		return
	}
	var req SourceUpdateRequest
	if err := decodeJSON(r, &req, 64<<10); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if req.Name == nil && req.URL == nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "name or url is required")
		return
	}
	sources, err := s.store.Sources()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "sources_failed", err.Error())
		return
	}
	index := -1
	for i := range sources {
		if sources[i].ID == id {
			index = i
			break
		}
	}
	if index < 0 {
		writeError(w, http.StatusNotFound, "source_not_found", fmt.Sprintf("source %q not found", id))
		return
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "source name is required")
			return
		}
		if len(name) > 128 {
			writeError(w, http.StatusBadRequest, "invalid_request", "source name exceeds 128 characters")
			return
		}
		sources[index].Name = name
	}
	if req.URL != nil {
		raw := strings.TrimSpace(*req.URL)
		if _, err := parseHTTPSSourceURL(raw); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
		if err := s.credentials.Put(r.Context(), id, raw); err != nil {
			writeError(w, http.StatusInternalServerError, "source_credential_failed", err.Error())
			return
		}
		sources[index].Origin = redactURL(raw)
	}
	if err := s.store.SaveSources(sources); err != nil {
		writeError(w, http.StatusInternalServerError, "sources_failed", err.Error())
		return
	}
	updated := s.decorateSourceStates([]Source{sources[index]})[0]
	updated.SnapshotPath = ""
	updated.FetchURL = ""
	public := publicSources([]Source{updated}, s.store.Dir())[0]
	writeJSON(w, http.StatusOK, public)
}

func (s *Server) handleSourceDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validHexValue(id, 16) {
		writeError(w, http.StatusBadRequest, "invalid_source_id", "invalid source id")
		return
	}
	sources, err := s.store.Sources()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "sources_failed", err.Error())
		return
	}
	kept := make([]Source, 0, len(sources))
	found := false
	for _, source := range sources {
		if source.ID == id {
			found = true
			continue
		}
		kept = append(kept, source)
	}
	if !found {
		writeError(w, http.StatusNotFound, "source_not_found", fmt.Sprintf("source %q not found", id))
		return
	}
	if err := s.store.SaveSources(kept); err != nil {
		writeError(w, http.StatusInternalServerError, "sources_failed", err.Error())
		return
	}
	_ = s.credentials.Delete(r.Context(), id)
	_ = os.RemoveAll(filepath.Join(s.store.Dir(), "sources", id))
	w.WriteHeader(http.StatusNoContent)
}
