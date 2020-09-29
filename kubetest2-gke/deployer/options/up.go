package options

import "fmt"

type UpOptions struct {
	NumClusters int `flag:"~num-clusters" desc:"Number of clusters to create, will auto-generate names as (kt2-<run-id>-<index>)"`
}

func (uo *UpOptions) Validate() error {
	// allow max 99 clusters (should be sufficient for most use cases)
	if uo.NumClusters < 1 || uo.NumClusters > 99 {
		return fmt.Errorf("need to specify between 1 and 99 clusters got %q: ", uo.NumClusters)
	}
	return nil
}
