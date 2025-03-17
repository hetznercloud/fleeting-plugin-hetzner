package hetzner

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/fleeting/fleeting/provider"
)

func TestValidate(t *testing.T) {
	testCases := []struct {
		name   string
		group  InstanceGroup
		env    map[string]string
		assert func(t *testing.T, group InstanceGroup, err error)
	}{
		{
			name: "valid",
			group: InstanceGroup{
				Name:        "fleeting",
				Token:       "dummy",
				Location:    "hel1",
				ServerTypes: []string{"cpx11"},
				Image:       "debian-12",
				VolumeSize:  15,
			},
			assert: func(t *testing.T, group InstanceGroup, err error) {
				assert.NoError(t, err)
				assert.Equal(t, provider.ProtocolSSH, group.settings.Protocol)
				assert.Equal(t, "root", group.settings.Username)
			},
		},
		{
			name: "valid with env",
			group: InstanceGroup{
				Name:        "fleeting",
				Token:       "dummy",
				Location:    "hel1",
				ServerTypes: []string{"cpx11"},
				Image:       "debian-12",
			},
			env: map[string]string{
				"HCLOUD_TOKEN":    "value",
				"HCLOUD_ENDPOINT": "value",
			},
			assert: func(t *testing.T, group InstanceGroup, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "value", group.Token)
				assert.Equal(t, "value", group.Endpoint)
			},
		},
		{
			name:  "empty",
			group: InstanceGroup{},
			assert: func(t *testing.T, group InstanceGroup, err error) {
				assert.Error(t, err)
				assert.Equal(t, `missing required plugin config: name
missing required plugin config: token
missing required plugin config: location
missing required plugin config: server_type
missing required plugin config: image`, err.Error())
			},
		},
		{
			name: "winrm",
			group: InstanceGroup{
				Name:        "fleeting",
				Token:       "dummy",
				Location:    "hel1",
				ServerTypes: []string{"cpx11"},
				Image:       "debian-12",
				settings: provider.Settings{
					ConnectorConfig: provider.ConnectorConfig{
						Protocol: "winrm",
					},
				},
			},
			assert: func(t *testing.T, group InstanceGroup, err error) {
				assert.Error(t, err)
				assert.Equal(t, "unsupported connector config protocol: winrm", err.Error())
			},
		},
		{
			name: "user data",
			group: InstanceGroup{
				Name:         "fleeting",
				Token:        "dummy",
				Location:     "hel1",
				ServerTypes:  []string{"cpx11"},
				Image:        "debian-12",
				UserData:     "dummy",
				UserDataFile: "dummy",
			},
			assert: func(t *testing.T, group InstanceGroup, err error) {
				assert.Error(t, err)
				assert.Equal(t, "mutually exclusive plugin config provided: user_data, user_data_file", err.Error())
			},
		},
		{
			name: "volume size",
			group: InstanceGroup{
				Name:        "fleeting",
				Token:       "dummy",
				Location:    "hel1",
				ServerTypes: []string{"cpx11"},
				Image:       "debian-12",
				VolumeSize:  8,
			},
			assert: func(t *testing.T, group InstanceGroup, err error) {
				assert.Error(t, err)
				assert.Equal(t, "invalid plugin config value: volume_size must be >= 10", err.Error())
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv("HCLOUD_TOKEN", "")
			t.Setenv("HCLOUD_ENDPOINT", "")

			for key, value := range testCase.env {
				t.Setenv(key, value)
			}

			err := testCase.group.validate()
			testCase.assert(t, testCase.group, err)
		})
	}
}

func TestPopulateUserData(t *testing.T) {
	tmp := t.TempDir()
	userDataFile := path.Join(tmp, "user-data.yml")
	require.NoError(t, os.WriteFile(userDataFile, []byte("my-user-data"), 0644))

	group := InstanceGroup{
		Name:         "fleeting",
		UserDataFile: userDataFile,
	}

	require.NoError(t, group.populate())
	require.Equal(t, "my-user-data", group.UserData)
}
