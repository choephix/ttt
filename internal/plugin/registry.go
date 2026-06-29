package plugin

import (
	"encoding/json"
	"errors"
	"os"
)

type RegistryEntry struct {
	Name        string        `json:"name"`
	Repo        string        `json:"repo,omitempty"`
	Path        string        `json:"path,omitempty"`
	Version     string        `json:"version"`
	Enabled     bool          `json:"enabled"`
	Permissions PermissionSet `json:"permissions"`
}

type Registry struct {
	Entries []RegistryEntry
	path    string
}

func LoadRegistry(path string) (*Registry, error) {
	r := &Registry{path: path}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return r, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, &r.Entries); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Registry) Save() error {
	data, err := json.MarshalIndent(r.Entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.path, data, 0644)
}

func (r *Registry) Find(name string) *RegistryEntry {
	for i := range r.Entries {
		if r.Entries[i].Name == name {
			return &r.Entries[i]
		}
	}
	return nil
}

func (r *Registry) UpdatePermissions(name string, perms PermissionSet) {
	entry := r.Find(name)
	if entry != nil {
		entry.Permissions = perms
	}
}

func (r *Registry) SetEnabled(name string, enabled bool) {
	entry := r.Find(name)
	if entry != nil {
		entry.Enabled = enabled
	}
}

func (r *Registry) Remove(name string) {
	for i := range r.Entries {
		if r.Entries[i].Name == name {
			r.Entries = append(r.Entries[:i], r.Entries[i+1:]...)
			return
		}
	}
}

func (r *Registry) AddOrUpdate(name, repo, path, version string, perms PermissionSet) {
	entry := r.Find(name)
	if entry != nil {
		entry.Repo = repo
		entry.Path = path
		entry.Version = version
		entry.Permissions = perms
		entry.Enabled = true
		return
	}
	r.Entries = append(r.Entries, RegistryEntry{
		Name:        name,
		Repo:        repo,
		Path:        path,
		Version:     version,
		Enabled:     true,
		Permissions: perms,
	})
}
