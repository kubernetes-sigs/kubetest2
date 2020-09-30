package deployer

import (
	"reflect"
	"testing"
)

func TestGenerateClusterNames(t *testing.T) {
	testCases := []struct {
		name                 string
		numClusters          int
		uid                  string
		expectedClusterNames []string
	}{
		{
			name:                 "zero clusters",
			uid:                  "foobar",
			expectedClusterNames: []string{},
		},
		{
			name:        "empty uid",
			numClusters: 3,
			expectedClusterNames: []string{
				"kt2-1",
				"kt2-2",
				"kt2-3",
			},
		},
		{
			name:        "3 clusters, 6 character uid",
			numClusters: 3,
			uid:         "foobar",
			expectedClusterNames: []string{
				"kt2-foobar-1",
				"kt2-foobar-2",
				"kt2-foobar-3",
			},
		},
		{
			name:        "20 clusters, 36 character uid",
			numClusters: 20,
			uid:         "abcdefghijklmnopqrstuvwxyz0123456789",
			expectedClusterNames: []string{
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-1",
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-2",
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-3",
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-4",
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-5",
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-6",
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-7",
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-8",
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-9",
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-10",
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-11",
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-12",
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-13",
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-14",
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-15",
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-16",
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-17",
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-18",
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-19",
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-20",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actualClusterNames := generateClusterNames(tc.numClusters, tc.uid)
			if !reflect.DeepEqual(actualClusterNames, tc.expectedClusterNames) {
				t.Errorf("expected cluster names to be: %v\nbut got %v", tc.expectedClusterNames, actualClusterNames)
			}
		})
	}
}
