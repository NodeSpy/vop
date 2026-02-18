package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/NodeSpy/vop/internal/config"
	"github.com/spf13/cobra"
)

func newCatCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cat",
		Short: "Print the profiles config (redacts service account tokens)",
		RunE:  cmdCat,
	}
}

func cmdCat(_ *cobra.Command, _ []string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	// Deep copy so we can redact without modifying the cached config.
	redacted := &config.Config{
		Profiles: make(map[string]*config.Profile, len(c.Profiles)),
	}
	for name, p := range c.Profiles {
		cp := *p
		if cp.ServiceAccountToken != "" {
			cp.ServiceAccountToken = "***"
		}
		// Copy FieldMap so we don't share the map pointer.
		if p.FieldMap != nil {
			cp.FieldMap = make(map[string]string, len(p.FieldMap))
			for k, v := range p.FieldMap {
				cp.FieldMap[k] = v
			}
		}
		redacted.Profiles[name] = &cp
	}

	data, err := json.MarshalIndent(redacted, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
