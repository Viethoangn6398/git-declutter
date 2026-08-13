package recovery

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kunmi02/git-declutter/internal/config"
	"github.com/kunmi02/git-declutter/internal/git"
	"github.com/oklog/ulid/v2"
)

const RefPrefix = "refs/git-declutter/recovery/"

type Event struct {
	ID        string     `json:"id"`
	Branch    string     `json:"branch"`
	SHA       string     `json:"sha"`
	DeletedAt time.Time  `json:"deletedAt"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	Reason    string     `json:"reason,omitempty"`
	PRNumber  int        `json:"prNumber,omitempty"`
	Permanent bool       `json:"permanent,omitempty"`
	Restored  bool       `json:"restored,omitempty"`
}

type Store struct {
	Repo *git.Repo
}

func (s Store) Dir(ctx context.Context) (string, error) {
	common, err := s.Repo.CommonDir(ctx)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(common) {
		common = filepath.Join(s.Repo.Path(), common)
	}
	return filepath.Join(common, "git-declutter"), nil
}

func (s Store) metadataPath(ctx context.Context) (string, error) {
	dir, err := s.Dir(ctx)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "recovery.jsonl"), nil
}

func (s Store) EnsureDir(ctx context.Context) error {
	dir, err := s.Dir(ctx)
	if err != nil {
		return err
	}
	return os.MkdirAll(filepath.Join(dir, "cache"), 0o755)
}

func (s Store) Load(ctx context.Context) ([]Event, error) {
	path, err := s.metadataPath(ctx)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	var events []Event
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var ev Event
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}
		events = append(events, ev)
	}
	return events, sc.Err()
}

func (s Store) append(ctx context.Context, ev Event) error {
	if err := s.EnsureDir(ctx); err != nil {
		return err
	}
	path, err := s.metadataPath(ctx)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	_, err = f.Write(append(data, '\n'))
	return err
}

func (s Store) SaveAll(ctx context.Context, events []Event) error {
	return s.rewrite(ctx, events)
}

func (s Store) rewrite(ctx context.Context, events []Event) error {
	if err := s.EnsureDir(ctx); err != nil {
		return err
	}
	path, err := s.metadataPath(ctx)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	for _, ev := range events {
		if err := enc.Encode(ev); err != nil {
			f.Close()
			os.Remove(tmp)
			return err
		}
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

func RecoveryRef(id, branch string) string {
	return RefPrefix + id + "/" + branch
}

func (s Store) Create(ctx context.Context, branch, sha, reason string, prNumber int, cfg config.Config) (Event, error) {
	id := ulid.Make().String()
	now := time.Now().UTC()
	ev := Event{
		ID:        id,
		Branch:    branch,
		SHA:       sha,
		DeletedAt: now,
		Reason:    reason,
		PRNumber:  prNumber,
	}
	if cfg.Recovery.Enabled {
		dur, forever, err := cfg.RetentionDuration()
		if err != nil {
			return Event{}, err
		}
		if !forever {
			exp := now.Add(dur)
			ev.ExpiresAt = &exp
		}
		if err := s.Repo.CreateRef(ctx, RecoveryRef(id, branch), sha); err != nil {
			return Event{}, err
		}
	} else {
		ev.Permanent = true
	}
	if err := s.append(ctx, ev); err != nil {
		return Event{}, err
	}
	return ev, nil
}

func (s Store) CreatePermanent(ctx context.Context, branch, sha, reason string, prNumber int) (Event, error) {
	ev := Event{
		ID:        ulid.Make().String(),
		Branch:    branch,
		SHA:       sha,
		DeletedAt: time.Now().UTC(),
		Reason:    reason,
		PRNumber:  prNumber,
		Permanent: true,
	}
	return ev, s.append(ctx, ev)
}

var (
	ErrNotFound      = errors.New("no recoverable branch found")
	ErrAlreadyExists = errors.New("branch already exists")
)

func (s Store) Restore(ctx context.Context, branch string) (Event, error) {
	events, err := s.Active(ctx)
	if err != nil {
		return Event{}, err
	}
	var match *Event
	for i := range events {
		if events[i].Branch == branch {
			match = &events[i]
		}
	}
	if match == nil {
		return Event{}, ErrNotFound
	}
	return s.restoreEvent(ctx, *match)
}

func (s Store) RestoreLast(ctx context.Context) (Event, error) {
	events, err := s.Active(ctx)
	if err != nil {
		return Event{}, err
	}
	if len(events) == 0 {
		return Event{}, ErrNotFound
	}
	return s.restoreEvent(ctx, events[len(events)-1])
}

func (s Store) restoreEvent(ctx context.Context, ev Event) (Event, error) {
	exists, err := s.Repo.BranchExists(ctx, ev.Branch)
	if err != nil {
		return Event{}, err
	}
	if exists {
		return Event{}, fmt.Errorf("%w: %s", ErrAlreadyExists, ev.Branch)
	}
	ref := RecoveryRef(ev.ID, ev.Branch)
	if err := s.Repo.CreateBranchAt(ctx, ev.Branch, ev.SHA); err != nil {
		return Event{}, err
	}
	_ = s.Repo.DeleteRef(ctx, ref)
	ev.Restored = true
	all, err := s.Load(ctx)
	if err != nil {
		return ev, nil
	}
	for i := range all {
		if all[i].ID == ev.ID {
			all[i].Restored = true
		}
	}
	_ = s.rewrite(ctx, all)
	return ev, nil
}

func (s Store) Active(ctx context.Context) ([]Event, error) {
	events, err := s.Load(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	var active []Event
	for _, ev := range events {
		if ev.Permanent || ev.Restored {
			continue
		}
		if ev.ExpiresAt != nil && !ev.ExpiresAt.After(now) {
			continue
		}
		active = append(active, ev)
	}
	return active, nil
}

func (s Store) CleanupExpired(ctx context.Context) (int, error) {
	events, err := s.Load(ctx)
	if err != nil {
		return 0, err
	}
	now := time.Now().UTC()
	kept := make([]Event, 0, len(events))
	removed := 0
	for _, ev := range events {
		expired := !ev.Permanent && !ev.Restored && ev.ExpiresAt != nil && !ev.ExpiresAt.After(now)
		if expired {
			_ = s.Repo.DeleteRef(ctx, RecoveryRef(ev.ID, ev.Branch))
			removed++
			continue
		}
		kept = append(kept, ev)
	}
	if removed > 0 {
		if err := s.rewrite(ctx, kept); err != nil {
			return removed, err
		}
	}
	return removed, nil
}

func (s Store) ClearAll(ctx context.Context) (int, error) {
	events, err := s.Load(ctx)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, ev := range events {
		if ev.Permanent || ev.Restored {
			continue
		}
		_ = s.Repo.DeleteRef(ctx, RecoveryRef(ev.ID, ev.Branch))
		n++
	}
	if err := s.rewrite(ctx, nil); err != nil {
		return n, err
	}
	return n, nil
}
